package bootstrap

import (
	"context"
	"net"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/yayccc/livebid/pkg/grpcx"
	"github.com/yayccc/livebid/pkg/identity"
	"github.com/yayccc/livebid/pkg/idgen"
	"github.com/yayccc/livebid/pkg/logger"
	"github.com/yayccc/livebid/pkg/nacosx"
	"github.com/yayccc/livebid/services/auction-service/internal/client"
	"github.com/yayccc/livebid/services/auction-service/internal/config"
	"github.com/yayccc/livebid/services/auction-service/internal/handler"
	"github.com/yayccc/livebid/services/auction-service/internal/repository"
	"github.com/yayccc/livebid/services/auction-service/internal/router"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"gorm.io/gorm"
)

type App struct {
	cfg           config.Config
	log           *zap.Logger
	db            *gorm.DB
	grpcServer    *grpc.Server
	health        *health.Server
	naming        naming_client.INamingClient
	registration  *nacosx.Registration
	stateStore    repository.AuctionStateStore
	goods         client.GoodsClient
	live          client.LiveClient
	events        client.EventPublisher
	eventConsumer client.EventConsumer
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	log, err := logger.Init(cfg.Log)
	if err != nil {
		return nil, err
	}

	db, err := repository.OpenDB(cfg.MySQL)
	if err != nil {
		return nil, err
	}
	namingClient, err := nacosx.NewNamingClient(cfg.Nacos)
	if err != nil {
		return nil, err
	}
	nacosx.SetDefaultNamingClient(namingClient)

	goodsClient, err := client.NewGRPCGoodsClient(cfg.Goods.Target)
	if err != nil {
		return nil, err
	}
	liveClient, err := client.NewGRPCLiveClient(cfg.Live.Target)
	if err != nil {
		return nil, err
	}
	eventPublisher, err := client.NewRocketMQEventPublisher(cfg.RocketMQ)
	if err != nil {
		return nil, err
	}

	auctionRepo := repository.NewGormAuctionRepository(db)
	bidRepo := repository.NewGormBidRecordRepository(db)
	stateStore := repository.NewRedisAuctionStateStore(cfg.Redis)
	eventProcessor := client.NewAuctionEventProcessor(
		auctionRepo,
		bidRepo,
		stateStore,
		log,
		cfg.RocketMQ.MaxReconsumeTimes,
	)
	eventConsumer, err := client.NewRocketMQEventConsumer(cfg.RocketMQ, eventProcessor)
	if err != nil {
		return nil, err
	}
	// handler 负责同步入口，consumer 负责异步事件补偿，两者共用同一组仓储和 Redis 状态。
	auctionHandler := handler.NewAuctionGRPCHandler(
		auctionRepo,
		bidRepo,
		stateStore,
		goodsClient,
		liveClient,
		eventPublisher,
		idgen.New(cfg.WorkerID),
		cfg.RocketMQ.DelayLevel,
	)

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(identity.UnaryServerInterceptor()))
	healthServer := router.RegisterGRPC(grpcServer, auctionHandler)

	_ = ctx
	return &App{
		cfg:           cfg,
		log:           log,
		db:            db,
		grpcServer:    grpcServer,
		health:        healthServer,
		naming:        namingClient,
		stateStore:    stateStore,
		goods:         goodsClient,
		live:          liveClient,
		events:        eventPublisher,
		eventConsumer: eventConsumer,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	if a.eventConsumer != nil {
		if err := a.eventConsumer.Start(); err != nil {
			return err
		}
		a.log.Info("auction-service rocketmq consumer started", zap.String("topic", a.cfg.RocketMQ.Topic), zap.String("group", a.cfg.RocketMQ.ConsumerGroup))
	}

	listener, err := net.Listen("tcp", a.cfg.GRPC.Addr)
	if err != nil {
		return err
	}
	registration, err := nacosx.RegisterInstance(ctx, a.naming, a.cfg.Registry, listener.Addr(), a.cfg.Env, a.log)
	if err != nil {
		_ = listener.Close()
		return err
	}
	a.registration = registration
	if a.registration != nil {
		a.registration.StartHealthCheck(ctx, a.cfg.HealthCheck)
	}

	errCh := make(chan error, 1)
	go func() {
		a.log.Info("auction-service grpc server started", zap.String("addr", listener.Addr().String()))
		errCh <- a.grpcServer.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		grpcx.SetNotServing(a.health, router.HealthServiceName)
		_ = a.registration.Deregister(context.Background())
		a.grpcServer.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}

func (a *App) Stop() {
	grpcx.SetNotServing(a.health, router.HealthServiceName)
	_ = a.registration.Deregister(context.Background())
	a.grpcServer.GracefulStop()
	if a.stateStore != nil {
		_ = a.stateStore.Close()
	}
	if a.goods != nil {
		_ = a.goods.Close()
	}
	if a.live != nil {
		_ = a.live.Close()
	}
	if a.events != nil {
		_ = a.events.Close()
	}
	if a.eventConsumer != nil {
		_ = a.eventConsumer.Close()
	}
	if a.db != nil {
		sqlDB, err := a.db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
}
