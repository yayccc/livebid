package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yayccc/livebid/pkg/auth"
	"github.com/yayccc/livebid/pkg/idgen"
	"github.com/yayccc/livebid/pkg/logger"
	"github.com/yayccc/livebid/pkg/nacosx"
	"github.com/yayccc/livebid/services/ws-gateway/internal/client"
	"github.com/yayccc/livebid/services/ws-gateway/internal/config"
	"github.com/yayccc/livebid/services/ws-gateway/internal/handler"
	"github.com/yayccc/livebid/services/ws-gateway/internal/repository"
	"github.com/yayccc/livebid/services/ws-gateway/internal/router"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type App struct {
	cfg           config.Config
	log           *zap.Logger
	server        *http.Server
	wsHandler     *handler.WebSocketHandler
	online        repository.OnlineStore
	auctionConn   *grpc.ClientConn
	liveConn      *grpc.ClientConn
	eventConsumer client.EventConsumer
	cancel        context.CancelFunc
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	log, err := logger.Init(cfg.Log)
	if err != nil {
		return nil, err
	}

	jwtManager, err := auth.NewJWTManager(cfg.JWT.Secret, auth.WithIssuer(cfg.JWT.UserIssuer))
	if err != nil {
		return nil, err
	}

	namingClient, err := nacosx.NewNamingClient(cfg.Nacos)
	if err != nil {
		return nil, err
	}
	nacosx.SetDefaultNamingClient(namingClient)

	auctionConn, err := client.NewAuctionServiceConn(cfg.AuctionService.Target)
	if err != nil {
		return nil, err
	}
	auctionClient := client.NewAuctionServiceClient(auctionConn)

	liveConn, err := client.NewLiveServiceConn(cfg.LiveService.Target)
	if err != nil {
		_ = auctionConn.Close()
		return nil, err
	}
	liveClient := client.NewLiveServiceClient(liveConn)

	online := repository.NewRedisOnlineStore(cfg.Redis, cfg.HeartbeatTimeout()*2)
	hub := handler.NewHub()
	wsHandler := handler.NewWebSocketHandler(handler.WebSocketHandlerOptions{
		Config:     cfg,
		Hub:        hub,
		Online:     online,
		Auction:    auctionClient,
		Live:       liveClient,
		JWT:        jwtManager,
		IDs:        idgen.New(8),
		Log:        log,
		RPCTimeout: cfg.RPCTimeout(),
	})

	var eventConsumer client.EventConsumer
	if cfg.RocketMQ.Enabled {
		eventConsumer, err = client.NewRocketMQAuctionEventConsumer(cfg.RocketMQ, wsHandler, log)
		if err != nil {
			_ = auctionConn.Close()
			_ = liveConn.Close()
			_ = online.Close()
			return nil, err
		}
	}

	gin.SetMode(gin.ReleaseMode)
	if cfg.Env == "local" {
		gin.SetMode(gin.DebugMode)
	}
	engine := gin.New()
	engine.Use(logger.GinRecovery(log), logger.GinMiddleware(log))
	router.Register(engine, wsHandler)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	appCtx, cancel := context.WithCancel(ctx)
	app := &App{
		cfg:           cfg,
		log:           log,
		server:        server,
		wsHandler:     wsHandler,
		online:        online,
		auctionConn:   auctionConn,
		liveConn:      liveConn,
		eventConsumer: eventConsumer,
		cancel:        cancel,
	}
	app.startBackground(appCtx)
	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	if a.eventConsumer != nil {
		if err := a.eventConsumer.Start(); err != nil {
			return err
		}
		a.log.Info("ws-gateway rocketmq consumer started", zap.String("topic", a.cfg.RocketMQ.Topic), zap.String("group", a.cfg.RocketMQ.ConsumerGroup))
	} else {
		a.log.Info("ws-gateway rocketmq consumer disabled")
	}

	errCh := make(chan error, 1)
	go func() {
		a.log.Info("ws-gateway http server started", zap.String("addr", a.cfg.HTTP.Addr))
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
	if a.cancel != nil {
		a.cancel()
	}
	if a.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = a.server.Shutdown(shutdownCtx)
	}
	if a.wsHandler != nil {
		a.wsHandler.Hub().CloseAll()
	}
	if a.eventConsumer != nil {
		_ = a.eventConsumer.Close()
	}
	if a.online != nil {
		_ = a.online.Close()
	}
	if a.auctionConn != nil {
		_ = a.auctionConn.Close()
	}
	if a.liveConn != nil {
		_ = a.liveConn.Close()
	}
}

func (a *App) startBackground(ctx context.Context) {
	go func() {
		err := a.online.SubscribeOnlineEvents(ctx, func(ctx context.Context, event repository.OnlineEvent) {
			a.wsHandler.QueueOnlineEvent(event)
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			a.log.Warn("ws-gateway online pubsub stopped", zap.Error(err))
		}
	}()

	go func() {
		ticker := time.NewTicker(a.cfg.OnlineBroadcastInterval())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.wsHandler.FlushOnlineEvents(ctx)
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(a.cfg.CleanupInterval())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, roomID := range a.wsHandler.Hub().RoomIDs() {
					a.wsHandler.CleanupRoom(ctx, roomID)
				}
			}
		}
	}()
}
