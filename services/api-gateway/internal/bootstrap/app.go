package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yayccc/livebid/pkg/auth"
	"github.com/yayccc/livebid/pkg/logger"
	"github.com/yayccc/livebid/pkg/nacosx"
	"github.com/yayccc/livebid/services/api-gateway/internal/client"
	"github.com/yayccc/livebid/services/api-gateway/internal/config"
	"github.com/yayccc/livebid/services/api-gateway/internal/handler"
	"github.com/yayccc/livebid/services/api-gateway/internal/router"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type App struct {
	cfg         config.Config
	log         *zap.Logger
	server      *http.Server
	shopConn    *grpc.ClientConn
	userConn    *grpc.ClientConn
	goodsConn   *grpc.ClientConn
	liveConn    *grpc.ClientConn
	auctionConn *grpc.ClientConn
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	log, err := logger.Init(cfg.Log)
	if err != nil {
		return nil, err
	}

	shopJWTManager, err := auth.NewJWTManager(cfg.JWT.Secret, auth.WithIssuer(cfg.JWT.ShopIssuer))
	if err != nil {
		return nil, err
	}
	userJWTManager, err := auth.NewJWTManager(cfg.JWT.Secret, auth.WithIssuer(cfg.JWT.UserIssuer))
	if err != nil {
		return nil, err
	}
	namingClient, err := nacosx.NewNamingClient(cfg.Nacos)
	if err != nil {
		return nil, err
	}
	nacosx.SetDefaultNamingClient(namingClient)

	shopConn, err := client.NewShopServiceConn(cfg.ShopService.Target)
	if err != nil {
		return nil, err
	}
	shopClient := client.NewShopServiceClient(shopConn)
	shopHandler := handler.NewShopHandler(shopClient, shopJWTManager, cfg.AccessTokenTTL(), cfg.RPCTimeout())

	userConn, err := client.NewUserServiceConn(cfg.UserService.Target)
	if err != nil {
		_ = shopConn.Close()
		return nil, err
	}
	userClient := client.NewUserServiceClient(userConn)
	userHandler := handler.NewUserHandler(userClient, cfg.RPCTimeout())

	goodsConn, err := client.NewGoodsServiceConn(cfg.GoodsService.Target)
	if err != nil {
		_ = shopConn.Close()
		_ = userConn.Close()
		return nil, err
	}
	goodsClient := client.NewGoodsServiceClient(goodsConn)
	goodsHandler := handler.NewGoodsHandler(goodsClient, cfg.RPCTimeout())
	fileStorage, err := handler.NewS3ObjectStorage(ctx, cfg.Storage)
	if err != nil {
		_ = shopConn.Close()
		_ = userConn.Close()
		_ = goodsConn.Close()
		return nil, err
	}
	fileHandler := handler.NewFileHandler(fileStorage)

	liveConn, err := client.NewLiveServiceConn(cfg.LiveService.Target)
	if err != nil {
		_ = shopConn.Close()
		_ = userConn.Close()
		_ = goodsConn.Close()
		return nil, err
	}
	liveClient := client.NewLiveServiceClient(liveConn)

	auctionConn, err := client.NewAuctionServiceConn(cfg.AuctionService.Target)
	if err != nil {
		_ = shopConn.Close()
		_ = userConn.Close()
		_ = goodsConn.Close()
		_ = liveConn.Close()
		return nil, err
	}
	auctionClient := client.NewAuctionServiceClient(auctionConn)
	auctionHandler := handler.NewAuctionHandlerWithGoods(auctionClient, goodsClient, cfg.RPCTimeout())
	liveHandler := handler.NewLiveHandlerWithAggregates(liveClient, shopClient, goodsClient, auctionClient, cfg.UserLive, cfg.RPCTimeout())

	gin.SetMode(gin.ReleaseMode)
	if cfg.Env == "local" {
		gin.SetMode(gin.DebugMode)
	}
	engine := gin.New()
	engine.Use(logger.GinRecovery(log), logger.GinMiddleware(log))
	router.Register(engine, shopHandler, userHandler, goodsHandler, fileHandler, liveHandler, auctionHandler, shopJWTManager, userJWTManager)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{
		cfg:         cfg,
		log:         log,
		server:      server,
		shopConn:    shopConn,
		userConn:    userConn,
		goodsConn:   goodsConn,
		liveConn:    liveConn,
		auctionConn: auctionConn,
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
	if a.userConn != nil {
		_ = a.userConn.Close()
	}
	if a.goodsConn != nil {
		_ = a.goodsConn.Close()
	}
	if a.liveConn != nil {
		_ = a.liveConn.Close()
	}
	if a.auctionConn != nil {
		_ = a.auctionConn.Close()
	}
}
