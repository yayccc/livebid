package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func New(cfg Config, opts ...Option) (*zap.Logger, error) {
	for _, opt := range opts {
		opt(&cfg)
	}
	cfg = normalizeConfig(cfg)

	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	ws, err := buildWriteSyncer(cfg)
	if err != nil {
		return nil, err
	}

	errWS, err := buildErrorWriteSyncer(cfg)
	if err != nil {
		return nil, err
	}

	encoder, err := buildEncoder(cfg.Encoding)
	if err != nil {
		return nil, err
	}

	core := zapcore.NewCore(encoder, ws, level)
	options := []zap.Option{
		zap.ErrorOutput(errWS),
	}
	if !cfg.DisableCaller {
		options = append(options, zap.AddCaller())
	}
	if cfg.Development {
		options = append(options, zap.Development())
	}
	if !cfg.DisableStack {
		options = append(options, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	fields := make([]zap.Field, 0, 2+len(cfg.Fields))
	if cfg.ServiceName != "" {
		fields = append(fields, zap.String(FieldService, cfg.ServiceName))
	}
	if cfg.Env != "" {
		fields = append(fields, zap.String(FieldEnv, cfg.Env))
	}
	for key, value := range cfg.Fields {
		fields = append(fields, zap.String(key, value))
	}
	if len(fields) > 0 {
		options = append(options, zap.Fields(fields...))
	}

	return zap.New(core, options...), nil
}

func Init(cfg Config, opts ...Option) (*zap.Logger, error) {
	log, err := New(cfg, opts...)
	if err != nil {
		return nil, err
	}
	zap.ReplaceGlobals(log)
	return log, nil
}

func MustInit(cfg Config, opts ...Option) *zap.Logger {
	log, err := Init(cfg, opts...)
	if err != nil {
		panic(err)
	}
	return log
}

func L() *zap.Logger {
	return zap.L()
}

func S() *zap.SugaredLogger {
	return zap.S()
}

func Named(name string) *zap.Logger {
	return L().Named(name)
}

func With(fields ...zap.Field) *zap.Logger {
	return L().With(fields...)
}

func Sync() error {
	err := L().Sync()
	if err == nil {
		return nil
	}
	message := err.Error()
	if strings.Contains(message, "invalid argument") || strings.Contains(message, "inappropriate ioctl") {
		return nil
	}
	return err
}

func normalizeConfig(cfg Config) Config {
	defaults := DefaultConfig()
	fileLooksUninitialized := cfg.File.MaxSizeMB <= 0 &&
		cfg.File.MaxBackups <= 0 &&
		cfg.File.MaxAgeDays <= 0 &&
		!cfg.File.Compress &&
		!cfg.File.LocalTime

	if cfg.Level == "" {
		cfg.Level = defaults.Level
	}
	if cfg.Encoding == "" {
		cfg.Encoding = defaults.Encoding
	}
	if len(cfg.Outputs) == 0 && cfg.File.Filename == "" {
		cfg.Outputs = defaults.Outputs
	}
	if len(cfg.ErrorOutputs) == 0 {
		cfg.ErrorOutputs = defaults.ErrorOutputs
	}
	if cfg.File.MaxSizeMB <= 0 {
		cfg.File.MaxSizeMB = defaults.File.MaxSizeMB
	}
	if cfg.File.MaxBackups <= 0 {
		cfg.File.MaxBackups = defaults.File.MaxBackups
	}
	if cfg.File.MaxAgeDays <= 0 {
		cfg.File.MaxAgeDays = defaults.File.MaxAgeDays
	}
	if fileLooksUninitialized {
		cfg.File.Compress = defaults.File.Compress
		cfg.File.LocalTime = defaults.File.LocalTime
	}
	return cfg
}

func parseLevel(level string) (zapcore.LevelEnabler, error) {
	atomicLevel := zap.NewAtomicLevel()
	if err := atomicLevel.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(level)))); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}
	return atomicLevel, nil
}

func buildEncoder(encoding string) (zapcore.Encoder, error) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case EncodingConsole:
		return zapcore.NewConsoleEncoder(encoderConfig()), nil
	case "", EncodingJSON:
		return zapcore.NewJSONEncoder(encoderConfig()), nil
	default:
		return nil, fmt.Errorf("unsupported log encoding %q", encoding)
	}
}

func buildWriteSyncer(cfg Config) (zapcore.WriteSyncer, error) {
	writers := make([]zapcore.WriteSyncer, 0, len(cfg.Outputs)+1)
	for _, output := range cfg.Outputs {
		writer, err := outputWriter(output)
		if err != nil {
			return nil, err
		}
		writers = append(writers, writer)
	}
	if cfg.File.Filename != "" {
		writers = append(writers, zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.File.Filename,
			MaxSize:    cfg.File.MaxSizeMB,
			MaxBackups: cfg.File.MaxBackups,
			MaxAge:     cfg.File.MaxAgeDays,
			Compress:   cfg.File.Compress,
			LocalTime:  cfg.File.LocalTime,
		}))
	}
	if len(writers) == 0 {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}
	return zapcore.Lock(zapcore.NewMultiWriteSyncer(writers...)), nil
}

func buildErrorWriteSyncer(cfg Config) (zapcore.WriteSyncer, error) {
	writers := make([]zapcore.WriteSyncer, 0, len(cfg.ErrorOutputs))
	for _, output := range cfg.ErrorOutputs {
		writer, err := outputWriter(output)
		if err != nil {
			return nil, err
		}
		writers = append(writers, writer)
	}
	if len(writers) == 0 {
		writers = append(writers, zapcore.AddSync(os.Stderr))
	}
	return zapcore.Lock(zapcore.NewMultiWriteSyncer(writers...)), nil
}

func outputWriter(output string) (zapcore.WriteSyncer, error) {
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "", OutputStdout:
		return zapcore.AddSync(os.Stdout), nil
	case OutputStderr:
		return zapcore.AddSync(os.Stderr), nil
	default:
		return nil, fmt.Errorf("unsupported log output %q", output)
	}
}
