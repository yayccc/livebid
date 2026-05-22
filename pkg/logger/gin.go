package logger

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	HeaderRequestID = "X-Request-Id"
	HeaderTraceID   = "X-Trace-Id"
	GinContextKey   = "logger"
)

type GinOption func(*ginConfig)

type ginConfig struct {
	skipPaths map[string]struct{}
}

func WithGinSkipPaths(paths ...string) GinOption {
	return func(cfg *ginConfig) {
		if cfg.skipPaths == nil {
			cfg.skipPaths = make(map[string]struct{}, len(paths))
		}
		for _, path := range paths {
			cfg.skipPaths[path] = struct{}{}
		}
	}
}

func GinMiddleware(base *zap.Logger, opts ...GinOption) gin.HandlerFunc {
	if base == nil {
		base = L()
	}
	cfg := ginConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(c *gin.Context) {
		if _, ok := cfg.skipPaths[c.Request.URL.Path]; ok {
			c.Next()
			return
		}

		start := time.Now()
		requestID := c.GetHeader(HeaderRequestID)
		traceID := c.GetHeader(HeaderTraceID)

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}
		if requestID != "" {
			fields = append(fields, RequestID(requestID))
			c.Writer.Header().Set(HeaderRequestID, requestID)
		}
		if traceID != "" {
			fields = append(fields, TraceID(traceID))
		}

		reqCtx := WithContextFields(c.Request.Context(), fields...)
		c.Request = c.Request.WithContext(reqCtx)
		c.Set(GinContextKey, base.With(fields...))

		c.Next()

		status := c.Writer.Status()
		logFields := append(fields,
			zap.Int("status", status),
			zap.Int("body_size", c.Writer.Size()),
			zap.Duration("latency", time.Since(start)),
		)
		if len(c.Errors) > 0 {
			logFields = append(logFields, zap.String("errors", c.Errors.String()))
		}

		switch {
		case status >= http.StatusInternalServerError:
			base.Error("http request completed", logFields...)
		case status >= http.StatusBadRequest:
			base.Warn("http request completed", logFields...)
		default:
			base.Info("http request completed", logFields...)
		}
	}
}

func GinRecovery(base *zap.Logger) gin.HandlerFunc {
	if base == nil {
		base = L()
	}
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				fields := []zap.Field{
					zap.Any("panic", recovered),
					zap.ByteString("stack", debug.Stack()),
				}
				contextFields := FieldsFromContext(c.Request.Context())
				if len(contextFields) > 0 {
					fields = append(fields, contextFields...)
				} else {
					fields = append(fields,
						zap.String("method", c.Request.Method),
						zap.String("path", c.Request.URL.Path),
						zap.String("client_ip", c.ClientIP()),
					)
				}
				base.Error("http panic recovered", fields...)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

func GinLogger(c *gin.Context) *zap.Logger {
	if c == nil {
		return L()
	}
	if value, ok := c.Get(GinContextKey); ok {
		if log, ok := value.(*zap.Logger); ok && log != nil {
			return log
		}
	}
	return FromContext(c.Request.Context())
}

func GinContext(c *gin.Context) context.Context {
	if c == nil || c.Request == nil {
		return context.Background()
	}
	return c.Request.Context()
}
