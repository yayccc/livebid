package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/yayccc/livebid/pkg/logger"
	"github.com/yayccc/livebid/services/auction-service/internal/bootstrap"
	"github.com/yayccc/livebid/services/auction-service/internal/config"
	"go.uber.org/zap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	app, err := bootstrap.New(ctx, cfg)
	if err != nil {
		panic(err)
	}
	defer app.Stop()
	defer func() {
		if err := logger.Sync(); err != nil {
			logger.L().Warn("sync logger failed", zap.Error(err))
		}
	}()

	if err := app.Run(ctx); err != nil {
		logger.L().Fatal("auction-service exited", zap.Error(err))
	}
}
