package bootstrap

import (
	"context"
	"net"

	"github.com/yayccc/livebid/pkg/idgen"
	"github.com/yayccc/livebid/pkg/logger"
	"github.com/yayccc/livebid/services/auction-service/internal/client"
	"github.com/yayccc/livebid/services/auction-service/internal/config"
	"github.com/yayccc/livebid/services/auction-service/internal/handler"
	"github.com/yayccc/livebid/services/auction-service/internal/repository"
	"github.com/yayccc/livebid/services/auction-service/internal/router"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

type App struct {
	cfg           config.Config
	log           *zap.Logger
	db            *gorm.DB
	grpcServer    *grpc.Server
	stateStore    repository.AuctionStateStore
	goods         client.GoodsClient
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

	goodsClient, err := client.NewGRPCGoodsClient(cfg.Goods.Addr)
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
		eventPublisher,
		idgen.New(cfg.WorkerID),
		cfg.RocketMQ.DelayLevel,
	)

	grpcServer := grpc.NewServer()
	router.RegisterGRPC(grpcServer, auctionHandler)

	_ = ctx
	return &App{
		cfg:           cfg,
		log:           log,
		db:            db,
		grpcServer:    grpcServer,
		stateStore:    stateStore,
		goods:         goodsClient,
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

	errCh := make(chan error, 1)
	go func() {
		a.log.Info("auction-service grpc server started", zap.String("addr", a.cfg.GRPC.Addr))
		errCh <- a.grpcServer.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		a.grpcServer.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}

func (a *App) Stop() {
	a.grpcServer.GracefulStop()
	if a.stateStore != nil {
		_ = a.stateStore.Close()
	}
	if a.goods != nil {
		_ = a.goods.Close()
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
