package logger

import (
	"context"

	"go.uber.org/zap"
)

// NoopLogger реализует интерфейс Logger, игнорируя все операции логирования.
// Используется для тестов или отключения логирования.
type NoopLogger struct{}

// NewNoopLogger создает новый no-op логгер.
func NewNoopLogger() *Logger {
	return &Logger{zapLogger: zap.NewNop()}
}

// Debug игнорирует debug-сообщения.
func (l *NoopLogger) Debug(ctx context.Context, msg string, fields ...zap.Field) {}

// Info игнорирует info-сообщения.
func (l *NoopLogger) Info(ctx context.Context, msg string, fields ...zap.Field) {}

// Warn игнорирует warn-сообщения.
func (l *NoopLogger) Warn(ctx context.Context, msg string, fields ...zap.Field) {}

// Error игнорирует error-сообщения.
func (l *NoopLogger) Error(ctx context.Context, msg string, fields ...zap.Field) {}

// Fatal игнорирует fatal-сообщения (не завершает программу).
func (l *NoopLogger) Fatal(ctx context.Context, msg string, fields ...zap.Field) {}

// With возвращает тот же NoopLogger.
func (l *NoopLogger) With(fields ...zap.Field) *Logger {
	return &Logger{zapLogger: zap.NewNop()}
}

// WithContext возвращает тот же NoopLogger.
func (l *NoopLogger) WithContext(ctx context.Context) *Logger {
	return &Logger{zapLogger: zap.NewNop()}
}

// SetLevel игнорирует изменение уровня.
func (l *NoopLogger) SetLevel(levelStr string) error {
	return nil
}

// Sync игнорирует синхронизацию.
func (l *NoopLogger) Sync() error {
	return nil
}

// Close игнорирует закрытие.
func (l *NoopLogger) Close() error {
	return nil
}

// Sugar возвращает sugared no-op логгер.
func (l *NoopLogger) Sugar() *zap.SugaredLogger {
	return zap.NewNop().Sugar()
}
