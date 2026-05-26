package bootstrap

import (
	"context"
	"net"

	"github.com/yayccc/livebid/pkg/idgen"
	"github.com/yayccc/livebid/pkg/logger"
	"github.com/yayccc/livebid/services/live-service/internal/config"
	"github.com/yayccc/livebid/services/live-service/internal/handler"
	"github.com/yayccc/livebid/services/live-service/internal/repository"
	"github.com/yayccc/livebid/services/live-service/internal/router"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

type App struct {
	cfg        config.Config
	log        *zap.Logger
	db         *gorm.DB
	grpcServer *grpc.Server
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

	liveRepo := repository.NewGormLiveRoomRepository(db)
	liveHandler := handler.NewLiveGRPCHandler(liveRepo, idgen.New(cfg.WorkerID), cfg.SRS)

	grpcServer := grpc.NewServer()
	router.RegisterGRPC(grpcServer, liveHandler)

	_ = ctx
	return &App{
		cfg:        cfg,
		log:        log,
		db:         db,
		grpcServer: grpcServer,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", a.cfg.GRPC.Addr)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		a.log.Info("live-service grpc server started", zap.String("addr", a.cfg.GRPC.Addr))
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
	if a.db != nil {
		sqlDB, err := a.db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
}
