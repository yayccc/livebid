package logger

import (
	"context"

	"go.uber.org/zap"
)

type contextFieldsKey struct{}

func WithContextFields(ctx context.Context, fields ...zap.Field) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	existing := FieldsFromContext(ctx)
	merged := make([]zap.Field, 0, len(existing)+len(fields))
	merged = append(merged, existing...)
	merged = append(merged, fields...)
	return context.WithValue(ctx, contextFieldsKey{}, merged)
}

func FieldsFromContext(ctx context.Context) []zap.Field {
	if ctx == nil {
		return nil
	}
	fields, ok := ctx.Value(contextFieldsKey{}).([]zap.Field)
	if !ok || len(fields) == 0 {
		return nil
	}
	copied := make([]zap.Field, len(fields))
	copy(copied, fields)
	return copied
}

func FromContext(ctx context.Context) *zap.Logger {
	fields := FieldsFromContext(ctx)
	if len(fields) == 0 {
		return L()
	}
	return L().With(fields...)
}
