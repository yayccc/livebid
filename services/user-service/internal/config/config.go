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

const ServiceName = "user-service"

type Config struct {
	Env          string                    `yaml:"env"`
	GRPC         GRPCConfig                `yaml:"grpc"`
	MySQL        MySQLConfig               `yaml:"mysql"`
	JWT          JWTConfig                 `yaml:"jwt"`
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

type JWTConfig struct {
	Secret                string `yaml:"secret"`
	Issuer                string `yaml:"issuer"`
	AccessTokenTTLSeconds int64  `yaml:"accessTokenTTLSeconds"`
}

func Load() Config {
	cfg := defaultConfig()
	path := getenv("USER_SERVICE_CONFIG", "services/user-service/configs/config.local.yaml")
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
		JWT: JWTConfig{
			Secret:                "livebid-local-dev-secret",
			Issuer:                "livebid",
			AccessTokenTTLSeconds: 604800,
		},
		Log:          logger.DevelopmentConfig(ServiceName),
		Nacos:        nacosx.DefaultConfig(),
		ConfigCenter: nacosx.DefaultConfigCenter("local", ServiceName),
		Registry:     nacosx.DefaultRegistry(ServiceName),
		HealthCheck:  nacosx.DefaultHealthCheck(),
		WorkerID:     2,
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
	if value := os.Getenv("USER_SERVICE_ENV"); value != "" {
		cfg.Env = value
	}
	if value := os.Getenv("USER_SERVICE_GRPC_ADDR"); value != "" {
		cfg.GRPC.Addr = value
	}
	if value := os.Getenv("USER_SERVICE_MYSQL_DSN"); value != "" {
		cfg.MySQL.DSN = value
	}
	if value := os.Getenv("USER_SERVICE_JWT_SECRET"); value != "" {
		cfg.JWT.Secret = value
	}
	if value := os.Getenv("USER_SERVICE_JWT_ISSUER"); value != "" {
		cfg.JWT.Issuer = value
	}
	setBoolEnv("USER_SERVICE_MYSQL_AUTO_MIGRATE", &cfg.MySQL.AutoMigrate)
	setIntEnv("USER_SERVICE_MYSQL_MAX_OPEN_CONNS", &cfg.MySQL.MaxOpenConns)
	setIntEnv("USER_SERVICE_MYSQL_MAX_IDLE_CONNS", &cfg.MySQL.MaxIdleConns)
	setIntEnv("USER_SERVICE_MYSQL_CONN_MAX_LIFETIME_SECONDS", &cfg.MySQL.ConnMaxLifetimeSeconds)
	setInt64Env("USER_SERVICE_WORKER_ID", &cfg.WorkerID)
	setInt64Env("USER_SERVICE_JWT_ACCESS_TOKEN_TTL_SECONDS", &cfg.JWT.AccessTokenTTLSeconds)

	envLog := logger.LoadConfigFromEnv("USER_SERVICE_LOG")
	if os.Getenv("USER_SERVICE_LOG_SERVICE") != "" {
		cfg.Log.ServiceName = envLog.ServiceName
	}
	if os.Getenv("USER_SERVICE_LOG_ENV") != "" {
		cfg.Log.Env = envLog.Env
	}
	if os.Getenv("USER_SERVICE_LOG_LEVEL") != "" {
		cfg.Log.Level = envLog.Level
	}
	if os.Getenv("USER_SERVICE_LOG_ENCODING") != "" {
		cfg.Log.Encoding = envLog.Encoding
	}
	if os.Getenv("USER_SERVICE_LOG_OUTPUTS") != "" {
		cfg.Log.Outputs = envLog.Outputs
	}
	if os.Getenv("USER_SERVICE_LOG_ERROR_OUTPUTS") != "" {
		cfg.Log.ErrorOutputs = envLog.ErrorOutputs
	}
	if os.Getenv("USER_SERVICE_LOG_DEVELOPMENT") != "" {
		cfg.Log.Development = envLog.Development
	}
	if os.Getenv("USER_SERVICE_LOG_DISABLE_CALLER") != "" {
		cfg.Log.DisableCaller = envLog.DisableCaller
	}
	if os.Getenv("USER_SERVICE_LOG_DISABLE_STACK") != "" {
		cfg.Log.DisableStack = envLog.DisableStack
	}
	if os.Getenv("USER_SERVICE_LOG_FILE") != "" {
		cfg.Log.File.Filename = envLog.File.Filename
	}
	nacosx.ApplyEnvOverrides("USER_SERVICE", &cfg.Nacos, &cfg.ConfigCenter, &cfg.Registry, &cfg.HealthCheck)
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
	if cfg.WorkerID <= 0 {
		cfg.WorkerID = 2
	}
	if cfg.JWT.Issuer == "" {
		cfg.JWT.Issuer = "livebid"
	}
	if cfg.JWT.AccessTokenTTLSeconds <= 0 {
		cfg.JWT.AccessTokenTTLSeconds = 604800
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
