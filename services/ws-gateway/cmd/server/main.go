package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/yayccc/livebid/pkg/logger"
	"github.com/yayccc/livebid/services/ws-gateway/internal/bootstrap"
	"github.com/yayccc/livebid/services/ws-gateway/internal/config"
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
	defer func() {
		app.Stop(ctx)
		if err := logger.Sync(); err != nil {
			logger.L().Warn("sync logger failed", zap.Error(err))
		}
	}()

	if err := app.Run(ctx); err != nil {
		logger.L().Fatal("ws-gateway exited", zap.Error(err))
	}
}
