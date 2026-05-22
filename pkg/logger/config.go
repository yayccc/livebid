package logger

import (
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	EncodingJSON    = "json"
	EncodingConsole = "console"

	OutputStdout = "stdout"
	OutputStderr = "stderr"
)

// Config describes a service logger. It is intentionally small enough to keep in
// each service config file, while still covering production defaults.
type Config struct {
	ServiceName  string            `json:"serviceName" yaml:"serviceName" mapstructure:"serviceName"`
	Env          string            `json:"env" yaml:"env" mapstructure:"env"`
	Level        string            `json:"level" yaml:"level" mapstructure:"level"`
	Encoding     string            `json:"encoding" yaml:"encoding" mapstructure:"encoding"`
	Outputs      []string          `json:"outputs" yaml:"outputs" mapstructure:"outputs"`
	ErrorOutputs []string          `json:"errorOutputs" yaml:"errorOutputs" mapstructure:"errorOutputs"`
	Fields       map[string]string `json:"fields" yaml:"fields" mapstructure:"fields"`

	Development   bool `json:"development" yaml:"development" mapstructure:"development"`
	DisableCaller bool `json:"disableCaller" yaml:"disableCaller" mapstructure:"disableCaller"`
	DisableStack  bool `json:"disableStack" yaml:"disableStack" mapstructure:"disableStack"`

	File FileConfig `json:"file" yaml:"file" mapstructure:"file"`
}

type FileConfig struct {
	Filename   string `json:"filename" yaml:"filename" mapstructure:"filename"`
	MaxSizeMB  int    `json:"maxSizeMB" yaml:"maxSizeMB" mapstructure:"maxSizeMB"`
	MaxBackups int    `json:"maxBackups" yaml:"maxBackups" mapstructure:"maxBackups"`
	MaxAgeDays int    `json:"maxAgeDays" yaml:"maxAgeDays" mapstructure:"maxAgeDays"`
	Compress   bool   `json:"compress" yaml:"compress" mapstructure:"compress"`
	LocalTime  bool   `json:"localTime" yaml:"localTime" mapstructure:"localTime"`
}

type Option func(*Config)

func DefaultConfig() Config {
	return Config{
		Env:          "local",
		Level:        "info",
		Encoding:     EncodingJSON,
		Outputs:      []string{OutputStdout},
		ErrorOutputs: []string{OutputStderr},
		File: FileConfig{
			MaxSizeMB:  100,
			MaxBackups: 10,
			MaxAgeDays: 30,
			Compress:   true,
			LocalTime:  true,
		},
	}
}

func DevelopmentConfig(serviceName string) Config {
	cfg := DefaultConfig()
	cfg.ServiceName = serviceName
	cfg.Env = "dev"
	cfg.Level = "debug"
	cfg.Encoding = EncodingConsole
	cfg.Development = true
	cfg.File.Compress = false
	return cfg
}

func ProductionConfig(serviceName string) Config {
	cfg := DefaultConfig()
	cfg.ServiceName = serviceName
	cfg.Env = "prod"
	cfg.Level = "info"
	cfg.Encoding = EncodingJSON
	return cfg
}

func LoadConfigFromEnv(prefix string) Config {
	cfg := DefaultConfig()
	if prefix == "" {
		prefix = "LIVEBID_LOG"
	}
	prefix = strings.TrimRight(prefix, "_")

	setString := func(key string, target *string) {
		if v := os.Getenv(prefix + "_" + key); v != "" {
			*target = v
		}
	}
	setBool := func(key string, target *bool) {
		if v := os.Getenv(prefix + "_" + key); v != "" {
			if parsed, err := strconv.ParseBool(v); err == nil {
				*target = parsed
			}
		}
	}
	setInt := func(key string, target *int) {
		if v := os.Getenv(prefix + "_" + key); v != "" {
			if parsed, err := strconv.Atoi(v); err == nil {
				*target = parsed
			}
		}
	}
	setList := func(key string, target *[]string) {
		if v := os.Getenv(prefix + "_" + key); v != "" {
			parts := strings.Split(v, ",")
			values := make([]string, 0, len(parts))
			for _, part := range parts {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					values = append(values, trimmed)
				}
			}
			if len(values) > 0 {
				*target = values
			}
		}
	}

	setString("SERVICE", &cfg.ServiceName)
	setString("ENV", &cfg.Env)
	setString("LEVEL", &cfg.Level)
	setString("ENCODING", &cfg.Encoding)
	setList("OUTPUTS", &cfg.Outputs)
	setList("ERROR_OUTPUTS", &cfg.ErrorOutputs)
	setBool("DEVELOPMENT", &cfg.Development)
	setBool("DISABLE_CALLER", &cfg.DisableCaller)
	setBool("DISABLE_STACK", &cfg.DisableStack)
	setString("FILE", &cfg.File.Filename)
	setInt("FILE_MAX_SIZE_MB", &cfg.File.MaxSizeMB)
	setInt("FILE_MAX_BACKUPS", &cfg.File.MaxBackups)
	setInt("FILE_MAX_AGE_DAYS", &cfg.File.MaxAgeDays)
	setBool("FILE_COMPRESS", &cfg.File.Compress)
	setBool("FILE_LOCAL_TIME", &cfg.File.LocalTime)

	return cfg
}

func WithServiceName(serviceName string) Option {
	return func(cfg *Config) {
		cfg.ServiceName = serviceName
	}
}

func WithLevel(level string) Option {
	return func(cfg *Config) {
		cfg.Level = level
	}
}

func WithEncoding(encoding string) Option {
	return func(cfg *Config) {
		cfg.Encoding = encoding
	}
}

func WithOutputs(outputs ...string) Option {
	return func(cfg *Config) {
		cfg.Outputs = outputs
	}
}

func WithFile(filename string) Option {
	return func(cfg *Config) {
		cfg.File.Filename = filename
	}
}

func encoderConfig() zapcore.EncoderConfig {
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "time"
	cfg.LevelKey = "level"
	cfg.NameKey = "logger"
	cfg.CallerKey = "caller"
	cfg.MessageKey = "msg"
	cfg.StacktraceKey = "stacktrace"
	cfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format(time.RFC3339Nano))
	}
	cfg.EncodeLevel = zapcore.LowercaseLevelEncoder
	cfg.EncodeCaller = zapcore.ShortCallerEncoder
	cfg.EncodeDuration = zapcore.MillisDurationEncoder
	return cfg
}
