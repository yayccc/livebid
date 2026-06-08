package config

import (
	"context"
	"errors"
	"os"
	"strconv"

	"github.com/yayccc/livebid/pkg/logger"
	"github.com/yayccc/livebid/pkg/nacosx"
	"gopkg.in/yaml.v3"
)

const ServiceName = "auction-service"

type Config struct {
	Env          string                    `yaml:"env"`
	GRPC         GRPCConfig                `yaml:"grpc"`
	MySQL        MySQLConfig               `yaml:"mysql"`
	Redis        RedisConfig               `yaml:"redis"`
	RocketMQ     RocketMQConfig            `yaml:"rocketmq"`
	Auction      AuctionConfig             `yaml:"auction"`
	Goods        GoodsConfig               `yaml:"goods"`
	Live         LiveConfig                `yaml:"live"`
	Log          logger.Config             `yaml:"log"`
	Nacos        nacosx.Config             `yaml:"nacos"`
	ConfigCenter nacosx.ConfigCenterConfig `yaml:"configCenter"`
	Registry     nacosx.RegistryConfig     `yaml:"registry"`
	HealthCheck  nacosx.HealthCheckConfig  `yaml:"healthCheck"`
	WorkerID     int64                     `yaml:"workerID"`
}

type GRPCConfig struct {
	Addr string `yaml:"addr"`
}

type MySQLConfig struct {
	DSN                    string `yaml:"dsn"`
	AutoMigrate            bool   `yaml:"autoMigrate"`
	MaxOpenConns           int    `yaml:"maxOpenConns"`
	MaxIdleConns           int    `yaml:"maxIdleConns"`
	ConnMaxLifetimeSeconds int    `yaml:"connMaxLifetimeSeconds"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type RocketMQConfig struct {
	NameServers       []string `yaml:"nameServers"`
	ProducerGroup     string   `yaml:"producerGroup"`
	ConsumerGroup     string   `yaml:"consumerGroup"`
	Topic             string   `yaml:"topic"`
	DelayLevel        int      `yaml:"delayLevel"`
	MaxReconsumeTimes int32    `yaml:"maxReconsumeTimes"`
}

type AuctionConfig struct {
	BidCountdownSeconds int `yaml:"bidCountdownSeconds"`
}

type GoodsConfig struct {
	Addr   string `yaml:"addr"`
	Target string `yaml:"target"`
}

type LiveConfig struct {
	Addr   string `yaml:"addr"`
	Target string `yaml:"target"`
}

func Load() Config {
	cfg := defaultConfig()
	path := getenv("AUCTION_SERVICE_CONFIG", "services/auction-service/configs/config.local.yaml")
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
	return Config{
		Env: "local",
		GRPC: GRPCConfig{
			Addr: ":9003",
		},
		MySQL: MySQLConfig{
			DSN:                    "root:123456@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local",
			AutoMigrate:            true,
			MaxOpenConns:           20,
			MaxIdleConns:           10,
			ConnMaxLifetimeSeconds: 1800,
		},
		Redis: RedisConfig{
			Addr: "127.0.0.1:6379",
		},
		RocketMQ: RocketMQConfig{
			NameServers:       []string{"127.0.0.1:9876"},
			ProducerGroup:     "auction-service-producer",
			ConsumerGroup:     "auction-service-consumer",
			Topic:             "auction_event",
			DelayLevel:        4,
			MaxReconsumeTimes: 5,
		},
		Auction: AuctionConfig{
			BidCountdownSeconds: 30,
		},
		Goods: GoodsConfig{
			Addr: "127.0.0.1:9002",
		},
		Live: LiveConfig{
			Addr: "127.0.0.1:9007",
		},
		Log:          logger.DevelopmentConfig(ServiceName),
		Nacos:        nacosx.DefaultConfig(),
		ConfigCenter: nacosx.DefaultConfigCenter("local", ServiceName),
		Registry:     nacosx.DefaultRegistry(ServiceName),
		HealthCheck:  nacosx.DefaultHealthCheck(),
		WorkerID:     3,
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
	if value := os.Getenv("AUCTION_SERVICE_ENV"); value != "" {
		cfg.Env = value
	}
	if value := os.Getenv("AUCTION_SERVICE_GRPC_ADDR"); value != "" {
		cfg.GRPC.Addr = value
	}
	if value := os.Getenv("AUCTION_SERVICE_MYSQL_DSN"); value != "" {
		cfg.MySQL.DSN = value
	}
	if value := os.Getenv("AUCTION_SERVICE_REDIS_ADDR"); value != "" {
		cfg.Redis.Addr = value
	}
	if value := os.Getenv("AUCTION_SERVICE_REDIS_PASSWORD"); value != "" {
		cfg.Redis.Password = value
	}
	if value := os.Getenv("AUCTION_SERVICE_GOODS_ADDR"); value != "" {
		cfg.Goods.Addr = value
	}
	if value := os.Getenv("AUCTION_SERVICE_GOODS_TARGET"); value != "" {
		cfg.Goods.Target = value
	}
	if value := os.Getenv("AUCTION_SERVICE_LIVE_ADDR"); value != "" {
		cfg.Live.Addr = value
	}
	if value := os.Getenv("AUCTION_SERVICE_LIVE_TARGET"); value != "" {
		cfg.Live.Target = value
	}
	if value := os.Getenv("AUCTION_SERVICE_ROCKETMQ_NAME_SERVER"); value != "" {
		cfg.RocketMQ.NameServers = []string{value}
	}
	if value := os.Getenv("AUCTION_SERVICE_ROCKETMQ_TOPIC"); value != "" {
		cfg.RocketMQ.Topic = value
	}
	if value := os.Getenv("AUCTION_SERVICE_ROCKETMQ_PRODUCER_GROUP"); value != "" {
		cfg.RocketMQ.ProducerGroup = value
	}
	if value := os.Getenv("AUCTION_SERVICE_ROCKETMQ_CONSUMER_GROUP"); value != "" {
		cfg.RocketMQ.ConsumerGroup = value
	}
	setBoolEnv("AUCTION_SERVICE_MYSQL_AUTO_MIGRATE", &cfg.MySQL.AutoMigrate)
	setIntEnv("AUCTION_SERVICE_MYSQL_MAX_OPEN_CONNS", &cfg.MySQL.MaxOpenConns)
	setIntEnv("AUCTION_SERVICE_MYSQL_MAX_IDLE_CONNS", &cfg.MySQL.MaxIdleConns)
	setIntEnv("AUCTION_SERVICE_MYSQL_CONN_MAX_LIFETIME_SECONDS", &cfg.MySQL.ConnMaxLifetimeSeconds)
	setIntEnv("AUCTION_SERVICE_REDIS_DB", &cfg.Redis.DB)
	setIntEnv("AUCTION_SERVICE_ROCKETMQ_DELAY_LEVEL", &cfg.RocketMQ.DelayLevel)
	setIntEnv("AUCTION_SERVICE_BID_COUNTDOWN_SECONDS", &cfg.Auction.BidCountdownSeconds)
	setInt32Env("AUCTION_SERVICE_ROCKETMQ_MAX_RECONSUME_TIMES", &cfg.RocketMQ.MaxReconsumeTimes)
	setInt64Env("AUCTION_SERVICE_WORKER_ID", &cfg.WorkerID)

	envLog := logger.LoadConfigFromEnv("AUCTION_SERVICE_LOG")
	if os.Getenv("AUCTION_SERVICE_LOG_SERVICE") != "" {
		cfg.Log.ServiceName = envLog.ServiceName
	}
	if os.Getenv("AUCTION_SERVICE_LOG_ENV") != "" {
		cfg.Log.Env = envLog.Env
	}
	if os.Getenv("AUCTION_SERVICE_LOG_LEVEL") != "" {
		cfg.Log.Level = envLog.Level
	}
	if os.Getenv("AUCTION_SERVICE_LOG_ENCODING") != "" {
		cfg.Log.Encoding = envLog.Encoding
	}
	if os.Getenv("AUCTION_SERVICE_LOG_OUTPUTS") != "" {
		cfg.Log.Outputs = envLog.Outputs
	}
	if os.Getenv("AUCTION_SERVICE_LOG_ERROR_OUTPUTS") != "" {
		cfg.Log.ErrorOutputs = envLog.ErrorOutputs
	}
	if os.Getenv("AUCTION_SERVICE_LOG_DEVELOPMENT") != "" {
		cfg.Log.Development = envLog.Development
	}
	if os.Getenv("AUCTION_SERVICE_LOG_DISABLE_CALLER") != "" {
		cfg.Log.DisableCaller = envLog.DisableCaller
	}
	if os.Getenv("AUCTION_SERVICE_LOG_DISABLE_STACK") != "" {
		cfg.Log.DisableStack = envLog.DisableStack
	}
	if os.Getenv("AUCTION_SERVICE_LOG_FILE") != "" {
		cfg.Log.File.Filename = envLog.File.Filename
	}
	nacosx.ApplyEnvOverrides("AUCTION_SERVICE", &cfg.Nacos, &cfg.ConfigCenter, &cfg.Registry, &cfg.HealthCheck)
}

func normalize(cfg *Config) {
	if cfg.Env == "" {
		cfg.Env = "local"
	}
	if cfg.GRPC.Addr == "" {
		cfg.GRPC.Addr = ":9003"
	}
	if cfg.MySQL.DSN == "" {
		cfg.MySQL.DSN = "root:123456@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local"
	}
	if cfg.MySQL.MaxOpenConns <= 0 {
		cfg.MySQL.MaxOpenConns = 20
	}
	if cfg.MySQL.MaxIdleConns <= 0 {
		cfg.MySQL.MaxIdleConns = 10
	}
	if cfg.MySQL.ConnMaxLifetimeSeconds <= 0 {
		cfg.MySQL.ConnMaxLifetimeSeconds = 1800
	}
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "127.0.0.1:6379"
	}
	if len(cfg.RocketMQ.NameServers) == 0 {
		cfg.RocketMQ.NameServers = []string{"127.0.0.1:9876"}
	}
	if cfg.RocketMQ.ProducerGroup == "" {
		cfg.RocketMQ.ProducerGroup = "auction-service-producer"
	}
	if cfg.RocketMQ.ConsumerGroup == "" {
		cfg.RocketMQ.ConsumerGroup = "auction-service-consumer"
	}
	if cfg.RocketMQ.Topic == "" {
		cfg.RocketMQ.Topic = "auction_event"
	}
	if cfg.RocketMQ.DelayLevel <= 0 {
		cfg.RocketMQ.DelayLevel = 4
	}
	if cfg.RocketMQ.MaxReconsumeTimes <= 0 {
		cfg.RocketMQ.MaxReconsumeTimes = 5
	}
	if cfg.Auction.BidCountdownSeconds <= 0 {
		cfg.Auction.BidCountdownSeconds = 30
	}
	if cfg.Goods.Addr == "" {
		cfg.Goods.Addr = "127.0.0.1:9002"
	}
	if cfg.Goods.Target == "" {
		cfg.Goods.Target = cfg.Goods.Addr
	} else if cfg.Goods.Addr == "" {
		cfg.Goods.Addr = cfg.Goods.Target
	}
	if cfg.Live.Addr == "" {
		cfg.Live.Addr = "127.0.0.1:9007"
	}
	if cfg.Live.Target == "" {
		cfg.Live.Target = cfg.Live.Addr
	} else if cfg.Live.Addr == "" {
		cfg.Live.Addr = cfg.Live.Target
	}
	if cfg.WorkerID <= 0 {
		cfg.WorkerID = 3
	}
	cfg.Log.ServiceName = ServiceName
	cfg.Log.Env = cfg.Env
	if cfg.Env == "local" && cfg.Log.Encoding == "" {
		cfg.Log.Encoding = logger.EncodingConsole
	}
	nacosx.NormalizeConfig(&cfg.Nacos)
	nacosx.NormalizeConfigCenter(&cfg.ConfigCenter, cfg.Env, ServiceName)
	nacosx.NormalizeRegistry(&cfg.Registry, ServiceName)
	nacosx.NormalizeHealthCheck(&cfg.HealthCheck)
}

func getenv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
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
