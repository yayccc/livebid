package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yayccc/livebid/pkg/auth"
	"github.com/yayccc/livebid/pkg/logger"
	"github.com/yayccc/livebid/services/api-gateway/internal/client"
	"github.com/yayccc/livebid/services/api-gateway/internal/config"
	"github.com/yayccc/livebid/services/api-gateway/internal/handler"
	"github.com/yayccc/livebid/services/api-gateway/internal/router"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type App struct {
	cfg       config.Config
	log       *zap.Logger
	server    *http.Server
	shopConn  *grpc.ClientConn
	goodsConn *grpc.ClientConn
	liveConn  *grpc.ClientConn
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	log, err := logger.Init(cfg.Log)
	if err != nil {
		return nil, err
	}

	jwtManager, err := auth.NewJWTManager(cfg.JWT.Secret, auth.WithIssuer(cfg.JWT.Issuer))
	if err != nil {
		return nil, err
	}

	shopConn, err := client.NewShopServiceConn(cfg.ShopService.Addr)
	if err != nil {
		return nil, err
	}
	shopClient := client.NewShopServiceClient(shopConn)
	shopHandler := handler.NewShopHandler(shopClient, jwtManager, cfg.AccessTokenTTL(), cfg.RPCTimeout())

	goodsConn, err := client.NewGoodsServiceConn(cfg.GoodsService.Addr)
	if err != nil {
		_ = shopConn.Close()
		return nil, err
	}
	goodsClient := client.NewGoodsServiceClient(goodsConn)
	goodsHandler := handler.NewGoodsHandler(goodsClient, cfg.RPCTimeout())

	liveConn, err := client.NewLiveServiceConn(cfg.LiveService.Addr)
	if err != nil {
		_ = shopConn.Close()
		return nil, err
	}
	liveClient := client.NewLiveServiceClient(liveConn)
	liveHandler := handler.NewLiveHandler(liveClient, jwtManager, cfg.RPCTimeout())

	gin.SetMode(gin.ReleaseMode)
	if cfg.Env == "local" {
		gin.SetMode(gin.DebugMode)
	}
	engine := gin.New()
	engine.Use(logger.GinRecovery(log), logger.GinMiddleware(log))
	router.Register(engine, shopHandler, goodsHandler, liveHandler)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{
		cfg:       cfg,
		log:       log,
		server:    server,
		shopConn:  shopConn,
		goodsConn: goodsConn,
		liveConn:  liveConn,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		a.log.Info("api-gateway http server started", zap.String("addr", a.cfg.HTTP.Addr))
		errCh <- a.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return a.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (a *App) Stop(ctx context.Context) {
	if a.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = a.server.Shutdown(shutdownCtx)
	}
	if a.shopConn != nil {
		_ = a.shopConn.Close()
	}
	if a.goodsConn != nil {
		_ = a.goodsConn.Close()
	}
	if a.liveConn != nil {
		_ = a.liveConn.Close()
	}
}
