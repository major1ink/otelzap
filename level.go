package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap/zapcore"
)

// parseLevel конвертирует строку в zapcore.Level.
func parseLevel(levelStr string) (zapcore.Level, error) {
	switch strings.ToLower(levelStr) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unknown level: %s", levelStr)
	}
}

// parseLevelDefault возвращает уровень без ошибки (для OTLP).
func parseLevelDefault(levelStr string) zapcore.Level {
	level, _ := parseLevel(levelStr)
	return level
}
