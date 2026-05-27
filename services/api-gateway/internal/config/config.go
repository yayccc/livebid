package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/yayccc/livebid/pkg/logger"
	"gopkg.in/yaml.v3"
)

const ServiceName = "api-gateway"

type Config struct {
	Env         string        `yaml:"env"`
	HTTP        HTTPConfig    `yaml:"http"`
	ShopService ServiceConfig `yaml:"shopService"`
	LiveService ServiceConfig `yaml:"liveService"`
	JWT         JWTConfig     `yaml:"jwt"`
	RPC         RPCConfig     `yaml:"rpc"`
	Log         logger.Config `yaml:"log"`
}

type HTTPConfig struct {
	Addr string `yaml:"addr"`
}

type ServiceConfig struct {
	Addr string `yaml:"addr"`
}

type JWTConfig struct {
	Secret                string `yaml:"secret"`
	Issuer                string `yaml:"issuer"`
	AccessTokenTTLSeconds int    `yaml:"accessTokenTTLSeconds"`
}

type RPCConfig struct {
	TimeoutSeconds int `yaml:"timeoutSeconds"`
}

func (c Config) AccessTokenTTL() time.Duration {
	return time.Duration(c.JWT.AccessTokenTTLSeconds) * time.Second
}

func (c Config) RPCTimeout() time.Duration {
	return time.Duration(c.RPC.TimeoutSeconds) * time.Second
}

func Load() Config {
	cfg := defaultConfig()
	path := getenv("API_GATEWAY_CONFIG", "services/api-gateway/configs/config.local.yaml")
	if fileCfg, err := loadFromFile(path); err == nil {
		cfg = fileCfg
	}
	applyEnvOverrides(&cfg)
	normalize(&cfg)
	return cfg
}

func defaultConfig() Config {
	return Config{
		Env: "local",
		HTTP: HTTPConfig{
			Addr: ":8080",
		},
		ShopService: ServiceConfig{
			Addr: "127.0.0.1:9001",
		},
		LiveService: ServiceConfig{
			Addr: "127.0.0.1:9002",
		},
		JWT: JWTConfig{
			Secret:                "local-dev-jwt-secret-change-me",
			Issuer:                "livebid",
			AccessTokenTTLSeconds: 604800,
		},
		RPC: RPCConfig{
			TimeoutSeconds: 3,
		},
		Log: logger.DevelopmentConfig(ServiceName),
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
	if value := os.Getenv("API_GATEWAY_ENV"); value != "" {
		cfg.Env = value
	}
	if value := os.Getenv("API_GATEWAY_HTTP_ADDR"); value != "" {
		cfg.HTTP.Addr = value
	}
	if value := os.Getenv("API_GATEWAY_SHOP_SERVICE_ADDR"); value != "" {
		cfg.ShopService.Addr = value
	}
	if value := os.Getenv("API_GATEWAY_LIVE_SERVICE_ADDR"); value != "" {
		cfg.LiveService.Addr = value
	}
	if value := os.Getenv("API_GATEWAY_JWT_SECRET"); value != "" {
		cfg.JWT.Secret = value
	}
	if value := os.Getenv("API_GATEWAY_JWT_ISSUER"); value != "" {
		cfg.JWT.Issuer = value
	}
	setIntEnv("API_GATEWAY_JWT_ACCESS_TOKEN_TTL_SECONDS", &cfg.JWT.AccessTokenTTLSeconds)
	setIntEnv("API_GATEWAY_RPC_TIMEOUT_SECONDS", &cfg.RPC.TimeoutSeconds)

	envLog := logger.LoadConfigFromEnv("API_GATEWAY_LOG")
	if os.Getenv("API_GATEWAY_LOG_SERVICE") != "" {
		cfg.Log.ServiceName = envLog.ServiceName
	}
	if os.Getenv("API_GATEWAY_LOG_ENV") != "" {
		cfg.Log.Env = envLog.Env
	}
	if os.Getenv("API_GATEWAY_LOG_LEVEL") != "" {
		cfg.Log.Level = envLog.Level
	}
	if os.Getenv("API_GATEWAY_LOG_ENCODING") != "" {
		cfg.Log.Encoding = envLog.Encoding
	}
	if os.Getenv("API_GATEWAY_LOG_OUTPUTS") != "" {
		cfg.Log.Outputs = envLog.Outputs
	}
	if os.Getenv("API_GATEWAY_LOG_ERROR_OUTPUTS") != "" {
		cfg.Log.ErrorOutputs = envLog.ErrorOutputs
	}
	if os.Getenv("API_GATEWAY_LOG_DEVELOPMENT") != "" {
		cfg.Log.Development = envLog.Development
	}
	if os.Getenv("API_GATEWAY_LOG_DISABLE_CALLER") != "" {
		cfg.Log.DisableCaller = envLog.DisableCaller
	}
	if os.Getenv("API_GATEWAY_LOG_DISABLE_STACK") != "" {
		cfg.Log.DisableStack = envLog.DisableStack
	}
	if os.Getenv("API_GATEWAY_LOG_FILE") != "" {
		cfg.Log.File.Filename = envLog.File.Filename
	}
}

func normalize(cfg *Config) {
	if cfg.Env == "" {
		cfg.Env = "local"
	}
	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = ":8080"
	}
	if cfg.ShopService.Addr == "" {
		cfg.ShopService.Addr = "127.0.0.1:9001"
	}
	if cfg.LiveService.Addr == "" {
		cfg.LiveService.Addr = "127.0.0.1:9002"
	}
	if cfg.JWT.Issuer == "" {
		cfg.JWT.Issuer = "livebid"
	}
	if cfg.JWT.AccessTokenTTLSeconds <= 0 {
		cfg.JWT.AccessTokenTTLSeconds = 604800
	}
	if cfg.RPC.TimeoutSeconds <= 0 {
		cfg.RPC.TimeoutSeconds = 3
	}
	cfg.Log.ServiceName = ServiceName
	cfg.Log.Env = cfg.Env
	if cfg.Env == "local" && cfg.Log.Encoding == "" {
		cfg.Log.Encoding = logger.EncodingConsole
	}
}

func getenv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
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
