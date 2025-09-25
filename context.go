package logger

import (
	"context"

	"go.uber.org/zap"
)

// contextKey используется для ключей контекста, чтобы избежать коллизий.
type contextKey string

const (
	traceIDKey contextKey = "trace_id"
	userIDKey  contextKey = "user_id"
)

// fieldsFromContext извлекает поля из контекста.
func (l *Logger) fieldsFromContext(ctx context.Context) []zap.Field {
	var fields []zap.Field

	// Стандартные поля
	if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
		fields = append(fields, zap.String(string(traceIDKey), traceID))
	}
	if userID, ok := ctx.Value(userIDKey).(string); ok && userID != "" {
		fields = append(fields, zap.String(string(userIDKey), userID))
	}

	// Кастомные extractors
	for _, fn := range l.config.FieldExtractors {
		fields = append(fields, fn(ctx)...)
	}

	return fields
}
