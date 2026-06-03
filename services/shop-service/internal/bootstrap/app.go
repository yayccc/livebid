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
	"github.com/yayccc/livebid/services/shop-service/internal/config"
	"github.com/yayccc/livebid/services/shop-service/internal/handler"
	"github.com/yayccc/livebid/services/shop-service/internal/repository"
	"github.com/yayccc/livebid/services/shop-service/internal/router"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"gorm.io/gorm"
)

type App struct {
	cfg          config.Config
	log          *zap.Logger
	db           *gorm.DB
	grpcServer   *grpc.Server
	health       *health.Server
	naming       naming_client.INamingClient
	registration *nacosx.Registration
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

	shopRepo := repository.NewGormShopRepository(db)
	loginLogRepo := repository.NewGormShopLoginLogRepository(db)
	shopHandler := handler.NewShopGRPCHandler(shopRepo, loginLogRepo, idgen.New(cfg.WorkerID))

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(identity.UnaryServerInterceptor()))
	healthServer := router.RegisterGRPC(grpcServer, shopHandler)
	namingClient, err := nacosx.NewNamingClient(cfg.Nacos)
	if err != nil {
		return nil, err
	}

	_ = ctx
	return &App{
		cfg:        cfg,
		log:        log,
		db:         db,
		grpcServer: grpcServer,
		health:     healthServer,
		naming:     namingClient,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
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
		a.log.Info("shop-service grpc server started", zap.String("addr", listener.Addr().String()))
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
	if a.db != nil {
		sqlDB, err := a.db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
}
