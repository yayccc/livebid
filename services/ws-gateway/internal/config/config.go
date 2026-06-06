package config

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yayccc/livebid/pkg/logger"
	"github.com/yayccc/livebid/pkg/nacosx"
	"gopkg.in/yaml.v3"
)

const ServiceName = "ws-gateway"

type Config struct {
	Env            string                    `yaml:"env"`
	HTTP           HTTPConfig                `yaml:"http"`
	Redis          RedisConfig               `yaml:"redis"`
	RocketMQ       RocketMQConfig            `yaml:"rocketmq"`
	AuctionService ServiceConfig             `yaml:"auctionService"`
	LiveService    ServiceConfig             `yaml:"liveService"`
	JWT            JWTConfig                 `yaml:"jwt"`
	WebSocket      WebSocketConfig           `yaml:"websocket"`
	RPC            RPCConfig                 `yaml:"rpc"`
	Log            logger.Config             `yaml:"log"`
	Nacos          nacosx.Config             `yaml:"nacos"`
	ConfigCenter   nacosx.ConfigCenterConfig `yaml:"configCenter"`
}

type HTTPConfig struct {
	Addr string `yaml:"addr"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type RocketMQConfig struct {
	NameServers       []string `yaml:"nameServers"`
	Topic             string   `yaml:"topic"`
	ConsumerGroup     string   `yaml:"consumerGroup"`
	Enabled           bool     `yaml:"enabled"`
	MaxReconsumeTimes int32    `yaml:"maxReconsumeTimes"`
}

type ServiceConfig struct {
	Addr   string `yaml:"addr"`
	Target string `yaml:"target"`
}

type JWTConfig struct {
	Secret     string `yaml:"secret"`
	UserIssuer string `yaml:"userIssuer"`
}

type WebSocketConfig struct {
	HeartbeatIntervalSeconds       int    `yaml:"heartbeatIntervalSeconds"`
	HeartbeatTimeoutSeconds        int    `yaml:"heartbeatTimeoutSeconds"`
	MaxMessageBytes                int64  `yaml:"maxMessageBytes"`
	WriteQueueSize                 int    `yaml:"writeQueueSize"`
	InstanceID                     string `yaml:"instanceId"`
	AllowOrigins                   string `yaml:"allowOrigins"`
	RequireLivingRoom              bool   `yaml:"requireLivingRoom"`
	CleanupIntervalSeconds         int    `yaml:"cleanupIntervalSeconds"`
	OnlineBroadcastIntervalSeconds int    `yaml:"onlineBroadcastIntervalSeconds"`
}

type RPCConfig struct {
	TimeoutSeconds int `yaml:"timeoutSeconds"`
}

func (c Config) HeartbeatInterval() time.Duration {
	return time.Duration(c.WebSocket.HeartbeatIntervalSeconds) * time.Second
}

func (c Config) HeartbeatTimeout() time.Duration {
	return time.Duration(c.WebSocket.HeartbeatTimeoutSeconds) * time.Second
}

func (c Config) CleanupInterval() time.Duration {
	return time.Duration(c.WebSocket.CleanupIntervalSeconds) * time.Second
}

func (c Config) OnlineBroadcastInterval() time.Duration {
	return time.Duration(c.WebSocket.OnlineBroadcastIntervalSeconds) * time.Second
}

func (c Config) RPCTimeout() time.Duration {
	return time.Duration(c.RPC.TimeoutSeconds) * time.Second
}

func Load() Config {
	cfg := defaultConfig()
	path := getenv("WS_GATEWAY_CONFIG", "services/ws-gateway/configs/config.local.yaml")
	if fileCfg, err := loadFromFile(path); err == nil {
		cfg = fileCfg
	}
	applyEnvOverrides(&cfg)
	normalize(&cfg)
	_ = nacosx.LoadConfigCenterYAML(context.Background(), cfg.Nacos, cfg.ConfigCenter, &cfg)
	applyEnvOverrides(&cfg)
	normalize(&cfg)
	return cfg
}

func defaultConfig() Config {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "local"
	}
	return Config{
		Env: "local",
		HTTP: HTTPConfig{
			Addr: ":58081",
		},
		Redis: RedisConfig{
			Addr: "127.0.0.1:6379",
		},
		RocketMQ: RocketMQConfig{
			NameServers:       nil,
			Topic:             "auction_event",
			ConsumerGroup:     "ws-gateway-auction-broadcast",
			Enabled:           false,
			MaxReconsumeTimes: 3,
		},
		AuctionService: ServiceConfig{
			Addr:   "127.0.0.1:9003",
			Target: "127.0.0.1:9003",
		},
		LiveService: ServiceConfig{
			Addr:   "127.0.0.1:9007",
			Target: "127.0.0.1:9007",
		},
		JWT: JWTConfig{
			Secret:     "local-dev-jwt-secret-change-me",
			UserIssuer: "livebid-user",
		},
		WebSocket: WebSocketConfig{
			HeartbeatIntervalSeconds:       30,
			HeartbeatTimeoutSeconds:        90,
			MaxMessageBytes:                16 * 1024,
			WriteQueueSize:                 128,
			InstanceID:                     hostname,
			AllowOrigins:                   "*",
			RequireLivingRoom:              true,
			CleanupIntervalSeconds:         15,
			OnlineBroadcastIntervalSeconds: 3,
		},
		RPC: RPCConfig{
			TimeoutSeconds: 3,
		},
		Log:          logger.DevelopmentConfig(ServiceName),
		Nacos:        nacosx.DefaultConfig(),
		ConfigCenter: nacosx.DefaultConfigCenter("local", ServiceName),
	}
}

func loadFromFile(path string) (Config, error) {
	cfg := defaultConfig()
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, err
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if value := os.Getenv("WS_GATEWAY_ENV"); value != "" {
		cfg.Env = value
	}
	if value := os.Getenv("WS_GATEWAY_HTTP_ADDR"); value != "" {
		cfg.HTTP.Addr = value
	}
	if value := os.Getenv("WS_GATEWAY_REDIS_ADDR"); value != "" {
		cfg.Redis.Addr = value
	}
	if value := os.Getenv("WS_GATEWAY_REDIS_PASSWORD"); value != "" {
		cfg.Redis.Password = value
	}
	setIntEnv("WS_GATEWAY_REDIS_DB", &cfg.Redis.DB)
	if value := os.Getenv("WS_GATEWAY_ROCKETMQ_NAME_SERVER"); value != "" {
		cfg.RocketMQ.NameServers = splitCSV(value)
		cfg.RocketMQ.Enabled = true
	}
	if value := os.Getenv("WS_GATEWAY_AUCTION_EVENT_TOPIC"); value != "" {
		cfg.RocketMQ.Topic = value
	}
	if value := os.Getenv("WS_GATEWAY_AUCTION_CONSUMER_GROUP"); value != "" {
		cfg.RocketMQ.ConsumerGroup = value
	}
	setBoolEnv("WS_GATEWAY_ROCKETMQ_ENABLED", &cfg.RocketMQ.Enabled)
	setInt32Env("WS_GATEWAY_ROCKETMQ_MAX_RECONSUME_TIMES", &cfg.RocketMQ.MaxReconsumeTimes)
	if value := os.Getenv("WS_GATEWAY_AUCTION_SERVICE_ADDR"); value != "" {
		cfg.AuctionService.Addr = value
	}
	if value := os.Getenv("WS_GATEWAY_AUCTION_SERVICE_TARGET"); value != "" {
		cfg.AuctionService.Target = value
	}
	if value := os.Getenv("WS_GATEWAY_LIVE_SERVICE_ADDR"); value != "" {
		cfg.LiveService.Addr = value
	}
	if value := os.Getenv("WS_GATEWAY_LIVE_SERVICE_TARGET"); value != "" {
		cfg.LiveService.Target = value
	}
	if value := os.Getenv("WS_GATEWAY_JWT_SECRET"); value != "" {
		cfg.JWT.Secret = value
	}
	if value := os.Getenv("WS_GATEWAY_JWT_USER_ISSUER"); value != "" {
		cfg.JWT.UserIssuer = value
	}
	setIntEnv("WS_GATEWAY_HEARTBEAT_INTERVAL_SECONDS", &cfg.WebSocket.HeartbeatIntervalSeconds)
	setIntEnv("WS_GATEWAY_HEARTBEAT_TIMEOUT_SECONDS", &cfg.WebSocket.HeartbeatTimeoutSeconds)
	setInt64Env("WS_GATEWAY_MAX_MESSAGE_BYTES", &cfg.WebSocket.MaxMessageBytes)
	setIntEnv("WS_GATEWAY_WRITE_QUEUE_SIZE", &cfg.WebSocket.WriteQueueSize)
	if value := os.Getenv("WS_GATEWAY_INSTANCE_ID"); value != "" {
		cfg.WebSocket.InstanceID = value
	}
	if value := os.Getenv("WS_GATEWAY_ALLOW_ORIGINS"); value != "" {
		cfg.WebSocket.AllowOrigins = value
	}
	setBoolEnv("WS_GATEWAY_REQUIRE_LIVING_ROOM", &cfg.WebSocket.RequireLivingRoom)
	setIntEnv("WS_GATEWAY_CLEANUP_INTERVAL_SECONDS", &cfg.WebSocket.CleanupIntervalSeconds)
	setIntEnv("WS_GATEWAY_ONLINE_BROADCAST_INTERVAL_SECONDS", &cfg.WebSocket.OnlineBroadcastIntervalSeconds)
	setIntEnv("WS_GATEWAY_RPC_TIMEOUT_SECONDS", &cfg.RPC.TimeoutSeconds)

	envLog := logger.LoadConfigFromEnv("WS_GATEWAY_LOG")
	if os.Getenv("WS_GATEWAY_LOG_SERVICE") != "" {
		cfg.Log.ServiceName = envLog.ServiceName
	}
	if os.Getenv("WS_GATEWAY_LOG_ENV") != "" {
		cfg.Log.Env = envLog.Env
	}
	if os.Getenv("WS_GATEWAY_LOG_LEVEL") != "" {
		cfg.Log.Level = envLog.Level
	}
	if os.Getenv("WS_GATEWAY_LOG_ENCODING") != "" {
		cfg.Log.Encoding = envLog.Encoding
	}
	if os.Getenv("WS_GATEWAY_LOG_OUTPUTS") != "" {
		cfg.Log.Outputs = envLog.Outputs
	}
	if os.Getenv("WS_GATEWAY_LOG_ERROR_OUTPUTS") != "" {
		cfg.Log.ErrorOutputs = envLog.ErrorOutputs
	}
	if os.Getenv("WS_GATEWAY_LOG_DEVELOPMENT") != "" {
		cfg.Log.Development = envLog.Development
	}
	if os.Getenv("WS_GATEWAY_LOG_DISABLE_CALLER") != "" {
		cfg.Log.DisableCaller = envLog.DisableCaller
	}
	if os.Getenv("WS_GATEWAY_LOG_DISABLE_STACK") != "" {
		cfg.Log.DisableStack = envLog.DisableStack
	}
	if os.Getenv("WS_GATEWAY_LOG_FILE") != "" {
		cfg.Log.File.Filename = envLog.File.Filename
	}
	nacosx.ApplyEnvOverrides("WS_GATEWAY", &cfg.Nacos, &cfg.ConfigCenter, nil, nil)
}

func normalize(cfg *Config) {
	if cfg.Env == "" {
		cfg.Env = "local"
	}
	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = ":58081"
	}
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "127.0.0.1:6379"
	}
	if cfg.RocketMQ.Topic == "" {
		cfg.RocketMQ.Topic = "auction_event"
	}
	if cfg.RocketMQ.ConsumerGroup == "" {
		cfg.RocketMQ.ConsumerGroup = "ws-gateway-auction-broadcast"
	}
	if cfg.RocketMQ.MaxReconsumeTimes <= 0 {
		cfg.RocketMQ.MaxReconsumeTimes = 3
	}
	if len(cfg.RocketMQ.NameServers) == 0 {
		cfg.RocketMQ.Enabled = false
	}
	if cfg.AuctionService.Addr == "" {
		cfg.AuctionService.Addr = "127.0.0.1:9003"
	}
	if cfg.AuctionService.Target == "" {
		cfg.AuctionService.Target = cfg.AuctionService.Addr
	} else if cfg.AuctionService.Addr == "" {
		cfg.AuctionService.Addr = cfg.AuctionService.Target
	}
	if cfg.LiveService.Addr == "" {
		cfg.LiveService.Addr = "127.0.0.1:9007"
	}
	if cfg.LiveService.Target == "" {
		cfg.LiveService.Target = cfg.LiveService.Addr
	} else if cfg.LiveService.Addr == "" {
		cfg.LiveService.Addr = cfg.LiveService.Target
	}
	if cfg.JWT.UserIssuer == "" {
		cfg.JWT.UserIssuer = "livebid-user"
	}
	if cfg.WebSocket.HeartbeatIntervalSeconds <= 0 {
		cfg.WebSocket.HeartbeatIntervalSeconds = 15
	}
	if cfg.WebSocket.HeartbeatTimeoutSeconds <= 0 {
		cfg.WebSocket.HeartbeatTimeoutSeconds = 45
	}
	if cfg.WebSocket.HeartbeatTimeoutSeconds < cfg.WebSocket.HeartbeatIntervalSeconds {
		cfg.WebSocket.HeartbeatTimeoutSeconds = cfg.WebSocket.HeartbeatIntervalSeconds * 3
	}
	if cfg.WebSocket.MaxMessageBytes <= 0 {
		cfg.WebSocket.MaxMessageBytes = 16 * 1024
	}
	if cfg.WebSocket.WriteQueueSize <= 0 {
		cfg.WebSocket.WriteQueueSize = 128
	}
	if strings.TrimSpace(cfg.WebSocket.InstanceID) == "" {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "local"
		}
		cfg.WebSocket.InstanceID = hostname
	}
	if cfg.WebSocket.AllowOrigins == "" {
		cfg.WebSocket.AllowOrigins = "*"
	}
	if cfg.WebSocket.CleanupIntervalSeconds <= 0 {
		cfg.WebSocket.CleanupIntervalSeconds = 15
	}
	if cfg.WebSocket.OnlineBroadcastIntervalSeconds <= 0 {
		cfg.WebSocket.OnlineBroadcastIntervalSeconds = 3
	}
	if cfg.RPC.TimeoutSeconds <= 0 {
		cfg.RPC.TimeoutSeconds = 3
	}
	cfg.Log.ServiceName = ServiceName
	cfg.Log.Env = cfg.Env
	if cfg.Env == "local" && cfg.Log.Encoding == "" {
		cfg.Log.Encoding = logger.EncodingConsole
	}
	nacosx.NormalizeConfig(&cfg.Nacos)
	nacosx.NormalizeConfigCenter(&cfg.ConfigCenter, cfg.Env, ServiceName)
}

func getenv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func setIntEnv(key string, target *int) {
	value := os.Getenv(key)
	if value == "" {
		return
	}
	parsed, err := strconv.Atoi(value)
	if err == nil {
		*target = parsed
	}
}

func setInt32Env(key string, target *int32) {
	value := os.Getenv(key)
	if value == "" {
		return
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err == nil {
		*target = int32(parsed)
	}
}

func setInt64Env(key string, target *int64) {
	value := os.Getenv(key)
	if value == "" {
		return
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err == nil {
		*target = parsed
	}
}

func setBoolEnv(key string, target *bool) {
	value := os.Getenv(key)
	if value == "" {
		return
	}
	parsed, err := strconv.ParseBool(value)
	if err == nil {
		*target = parsed
	}
}
