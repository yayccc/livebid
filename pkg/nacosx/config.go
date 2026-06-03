package nacosx

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"gopkg.in/yaml.v3"
)

const (
	DefaultGroup   = "LIVEBID"
	DefaultCluster = "default"
)

type Config struct {
	Enabled     bool           `yaml:"enabled"`
	Servers     []ServerConfig `yaml:"servers"`
	NamespaceID string         `yaml:"namespaceId"`
	Username    string         `yaml:"username"`
	Password    string         `yaml:"password"`
	LogDir      string         `yaml:"logDir"`
	CacheDir    string         `yaml:"cacheDir"`
	TimeoutMs   uint64         `yaml:"timeoutMs"`
}

type ServerConfig struct {
	Host        string `yaml:"host"`
	Port        uint64 `yaml:"port"`
	GrpcPort    uint64 `yaml:"grpcPort"`
	Scheme      string `yaml:"scheme"`
	ContextPath string `yaml:"contextPath"`
}

type ConfigCenterConfig struct {
	Enabled bool   `yaml:"enabled"`
	DataID  string `yaml:"dataId"`
	Group   string `yaml:"group"`
}

type RegistryConfig struct {
	Enabled     bool              `yaml:"enabled"`
	Provider    string            `yaml:"provider"`
	ServiceName string            `yaml:"serviceName"`
	Group       string            `yaml:"group"`
	Cluster     string            `yaml:"cluster"`
	IP          string            `yaml:"ip"`
	Port        uint64            `yaml:"port"`
	Weight      float64           `yaml:"weight"`
	Metadata    map[string]string `yaml:"metadata"`
}

type HealthCheckConfig struct {
	Enabled          bool `yaml:"enabled"`
	IntervalSeconds  int  `yaml:"intervalSeconds"`
	TimeoutSeconds   int  `yaml:"timeoutSeconds"`
	FailureThreshold int  `yaml:"failureThreshold"`
}

func DefaultConfig() Config {
	return Config{
		Enabled: false,
		Servers: []ServerConfig{
			{Host: "127.0.0.1", Port: 8848},
		},
		TimeoutMs: 5000,
	}
}

func DefaultConfigCenter(env string, serviceName string) ConfigCenterConfig {
	return ConfigCenterConfig{
		Enabled: false,
		DataID:  fmt.Sprintf("livebid.%s.%s.yaml", normalizeEnv(env), serviceName),
		Group:   DefaultGroup,
	}
}

func DefaultRegistry(serviceName string) RegistryConfig {
	return RegistryConfig{
		Enabled:     false,
		Provider:    "nacos",
		ServiceName: serviceName,
		Group:       DefaultGroup,
		Cluster:     DefaultCluster,
		Weight:      1,
	}
}

func DefaultHealthCheck() HealthCheckConfig {
	return HealthCheckConfig{
		Enabled:          true,
		IntervalSeconds:  5,
		TimeoutSeconds:   2,
		FailureThreshold: 3,
	}
}

func NormalizeConfig(cfg *Config) {
	if cfg.TimeoutMs == 0 {
		cfg.TimeoutMs = 5000
	}
	if len(cfg.Servers) == 0 {
		cfg.Servers = []ServerConfig{{Host: "127.0.0.1", Port: 8848}}
	}
	for i := range cfg.Servers {
		if cfg.Servers[i].Port == 0 {
			cfg.Servers[i].Port = 8848
		}
	}
}

func NormalizeConfigCenter(cfg *ConfigCenterConfig, env string, serviceName string) {
	if cfg.Group == "" {
		cfg.Group = DefaultGroup
	}
	if cfg.DataID == "" {
		cfg.DataID = fmt.Sprintf("livebid.%s.%s.yaml", normalizeEnv(env), serviceName)
	}
}

func NormalizeRegistry(cfg *RegistryConfig, serviceName string) {
	if cfg.Provider == "" {
		cfg.Provider = "nacos"
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = serviceName
	}
	if cfg.Group == "" {
		cfg.Group = DefaultGroup
	}
	if cfg.Cluster == "" {
		cfg.Cluster = DefaultCluster
	}
	if cfg.Weight <= 0 {
		cfg.Weight = 1
	}
}

func NormalizeHealthCheck(cfg *HealthCheckConfig) {
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = 5
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 2
	}
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 3
	}
}

func ApplyEnvOverrides(prefix string, cfg *Config, center *ConfigCenterConfig, registry *RegistryConfig, health *HealthCheckConfig) {
	if cfg != nil {
		setBoolEnv(prefix+"_NACOS_ENABLED", &cfg.Enabled)
		if value := os.Getenv(prefix + "_NACOS_SERVERS"); value != "" {
			cfg.Servers = parseServers(value)
		}
		if value := os.Getenv(prefix + "_NACOS_NAMESPACE_ID"); value != "" {
			cfg.NamespaceID = value
		}
		if value := os.Getenv(prefix + "_NACOS_USERNAME"); value != "" {
			cfg.Username = value
		}
		if value := os.Getenv(prefix + "_NACOS_PASSWORD"); value != "" {
			cfg.Password = value
		}
		if value := os.Getenv(prefix + "_NACOS_LOG_DIR"); value != "" {
			cfg.LogDir = value
		}
		if value := os.Getenv(prefix + "_NACOS_CACHE_DIR"); value != "" {
			cfg.CacheDir = value
		}
		setUint64Env(prefix+"_NACOS_TIMEOUT_MS", &cfg.TimeoutMs)
	}
	if center != nil {
		setBoolEnv(prefix+"_CONFIG_CENTER_ENABLED", &center.Enabled)
		if value := os.Getenv(prefix + "_CONFIG_CENTER_DATA_ID"); value != "" {
			center.DataID = value
		}
		if value := os.Getenv(prefix + "_CONFIG_CENTER_GROUP"); value != "" {
			center.Group = value
		}
	}
	if registry != nil {
		setBoolEnv(prefix+"_REGISTRY_ENABLED", &registry.Enabled)
		if value := os.Getenv(prefix + "_REGISTRY_PROVIDER"); value != "" {
			registry.Provider = value
		}
		if value := os.Getenv(prefix + "_REGISTRY_SERVICE_NAME"); value != "" {
			registry.ServiceName = value
		}
		if value := os.Getenv(prefix + "_REGISTRY_GROUP"); value != "" {
			registry.Group = value
		}
		if value := os.Getenv(prefix + "_REGISTRY_CLUSTER"); value != "" {
			registry.Cluster = value
		}
		if value := os.Getenv(prefix + "_REGISTRY_IP"); value != "" {
			registry.IP = value
		}
		setUint64Env(prefix+"_REGISTRY_PORT", &registry.Port)
		setFloat64Env(prefix+"_REGISTRY_WEIGHT", &registry.Weight)
	}
	if health != nil {
		setBoolEnv(prefix+"_HEALTH_CHECK_ENABLED", &health.Enabled)
		setIntEnv(prefix+"_HEALTH_CHECK_INTERVAL_SECONDS", &health.IntervalSeconds)
		setIntEnv(prefix+"_HEALTH_CHECK_TIMEOUT_SECONDS", &health.TimeoutSeconds)
		setIntEnv(prefix+"_HEALTH_CHECK_FAILURE_THRESHOLD", &health.FailureThreshold)
	}
}

func NewConfigClient(cfg Config) (config_client.IConfigClient, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	NormalizeConfig(&cfg)
	return clients.NewConfigClient(toClientParam(cfg))
}

func NewNamingClient(cfg Config) (naming_client.INamingClient, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	NormalizeConfig(&cfg)
	return clients.NewNamingClient(toClientParam(cfg))
}

func LoadConfigCenterYAML(ctx context.Context, cfg Config, center ConfigCenterConfig, target any) error {
	_ = ctx
	if !cfg.Enabled || !center.Enabled {
		return nil
	}
	NormalizeConfig(&cfg)
	if center.Group == "" || center.DataID == "" {
		return fmt.Errorf("nacos config center requires dataId and group")
	}
	client, err := NewConfigClient(cfg)
	if err != nil {
		return err
	}
	if client == nil {
		return nil
	}
	content, err := client.GetConfig(vo.ConfigParam{DataId: center.DataID, Group: center.Group})
	if err != nil {
		return err
	}
	if strings.TrimSpace(content) == "" {
		return nil
	}
	return yaml.Unmarshal([]byte(content), target)
}

func toClientParam(cfg Config) vo.NacosClientParam {
	clientCfg := constant.ClientConfig{
		NamespaceId: cfg.NamespaceID,
		Username:    cfg.Username,
		Password:    cfg.Password,
		TimeoutMs:   cfg.TimeoutMs,
		LogDir:      cfg.LogDir,
		CacheDir:    cfg.CacheDir,
	}
	servers := make([]constant.ServerConfig, 0, len(cfg.Servers))
	for _, server := range cfg.Servers {
		servers = append(servers, constant.ServerConfig{
			Scheme:      server.Scheme,
			ContextPath: server.ContextPath,
			IpAddr:      server.Host,
			Port:        server.Port,
			GrpcPort:    server.GrpcPort,
		})
	}
	return vo.NacosClientParam{
		ClientConfig:  &clientCfg,
		ServerConfigs: servers,
	}
}

func parseServers(value string) []ServerConfig {
	parts := strings.Split(value, ",")
	servers := make([]ServerConfig, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		host := part
		port := uint64(8848)
		if strings.Contains(part, ":") {
			pieces := strings.Split(part, ":")
			host = pieces[0]
			if parsed, err := strconv.ParseUint(pieces[len(pieces)-1], 10, 64); err == nil {
				port = parsed
			}
		}
		servers = append(servers, ServerConfig{Host: host, Port: port})
	}
	return servers
}

func normalizeEnv(env string) string {
	if env == "" {
		return "local"
	}
	return env
}

func setBoolEnv(key string, target *bool) {
	value := os.Getenv(key)
	if value == "" {
		return
	}
	if parsed, err := strconv.ParseBool(value); err == nil {
		*target = parsed
	}
}

func setIntEnv(key string, target *int) {
	value := os.Getenv(key)
	if value == "" {
		return
	}
	if parsed, err := strconv.Atoi(value); err == nil {
		*target = parsed
	}
}

func setUint64Env(key string, target *uint64) {
	value := os.Getenv(key)
	if value == "" {
		return
	}
	if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
		*target = parsed
	}
}

func setFloat64Env(key string, target *float64) {
	value := os.Getenv(key)
	if value == "" {
		return
	}
	if parsed, err := strconv.ParseFloat(value, 64); err == nil {
		*target = parsed
	}
}
