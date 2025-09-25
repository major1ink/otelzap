package logger

import (
	"context"
	"fmt"
	"os"
	"time"

	otelLog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap/zapcore"
)

// SimpleOTLPCore реализует zapcore. Core для отправки логов в OTLP.
type SimpleOTLPCore struct {
	otlpLogger  otelLog.Logger
	processor   *log.BatchProcessor // Для вызова ForceFlush в Sync.
	level       zapcore.LevelEnabler
	emitTimeout time.Duration
}

// NewSimpleOTLPCore создает новый OTLP core.
func NewSimpleOTLPCore(otlpLogger otelLog.Logger, processor *log.BatchProcessor, level zapcore.LevelEnabler, emitTimeout time.Duration) *SimpleOTLPCore {
	if emitTimeout == 0 {
		emitTimeout = 500 * time.Millisecond
	}
	return &SimpleOTLPCore{
		otlpLogger:  otlpLogger,
		processor:   processor,
		level:       level,
		emitTimeout: emitTimeout,
	}
}

// Enabled проверяет, включен ли уровень логирования.
func (c *SimpleOTLPCore) Enabled(level zapcore.Level) bool {
	return c.level.Enabled(level)
}

// With добавляет поля в новый core.
func (c *SimpleOTLPCore) With(fields []zapcore.Field) zapcore.Core {
	return &SimpleOTLPCore{
		otlpLogger:  c.otlpLogger,
		processor:   c.processor,
		level:       c.level,
		emitTimeout: c.emitTimeout,
	}
}

// Check добавляет core в CheckedEntry, если уровень включен.
func (c *SimpleOTLPCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

// Write записывает лог в OTLP.
func (c *SimpleOTLPCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	severity := mapZapToOtelSeverity(entry.Level)
	record := makeBaseRecord(entry, severity)
	if len(fields) > 0 {
		attrs := encodeFieldsToAttrs(fields)
		if len(attrs) > 0 {
			record.AddAttributes(attrs...)
		}
	}
	// Добавляем caller и stacktrace, если есть.
	if entry.Caller.Defined {
		record.AddAttributes(otelLog.String("caller", entry.Caller.String()))
	}
	if entry.Stack != "" {
		record.AddAttributes(otelLog.String("stacktrace", entry.Stack))
	}

	if err := c.emitWithTimeout(record); err != nil {
		// Fallback на stderr при timeout.
		fmt.Fprintf(os.Stderr, "failed to emit OTLP log: %v, message: %s\n", err, entry.Message)
	}
	return nil
}

// Sync вызывает flush для OTLP batch processor.
func (c *SimpleOTLPCore) Sync() error {
	if c.processor == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.emitTimeout)
	defer cancel()
	if err := c.processor.ForceFlush(ctx); err != nil {
		return fmt.Errorf("failed to flush OTLP processor: %w", err)
	}
	return nil
}

// mapZapToOtelSeverity маппит Zap уровни на OTLP severity.
func mapZapToOtelSeverity(level zapcore.Level) otelLog.Severity {
	switch level {
	case zapcore.DebugLevel:
		return otelLog.SeverityDebug
	case zapcore.InfoLevel:
		return otelLog.SeverityInfo
	case zapcore.WarnLevel:
		return otelLog.SeverityWarn
	case zapcore.ErrorLevel:
		return otelLog.SeverityError
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return otelLog.SeverityFatal
	default:
		return otelLog.SeverityInfo
	}
}

// makeBaseRecord создает базовую OTLP запись.
func makeBaseRecord(entry zapcore.Entry, sev otelLog.Severity) otelLog.Record {
	r := otelLog.Record{}
	r.SetSeverity(sev)
	r.SetBody(otelLog.StringValue(entry.Message))
	r.SetTimestamp(entry.Time)
	return r
}

// encodeFieldsToAttrs конвертирует Zap поля в OTLP атрибуты.
func encodeFieldsToAttrs(fields []zapcore.Field) []otelLog.KeyValue {
	if len(fields) == 0 {
		return nil
	}

	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fields {
		f.AddTo(enc)
	}

	attrs := make([]otelLog.KeyValue, 0, len(enc.Fields))
	for k, v := range enc.Fields {
		switch val := v.(type) {
		case string:
			attrs = append(attrs, otelLog.String(k, val))
		case bool:
			attrs = append(attrs, otelLog.Bool(k, val))
		case int64:
			attrs = append(attrs, otelLog.Int64(k, val))
		case float64:
			attrs = append(attrs, otelLog.Float64(k, val))
		case []interface{}:
			// Конвертируем массив в строку, так как Slice не поддерживается в текущей версии.
			attrs = append(attrs, otelLog.String(k, fmt.Sprintf("%v", val)))
		case map[string]interface{}:
			// Конвертируем map в строку, так как Map не поддерживается в текущей версии.
			attrs = append(attrs, otelLog.String(k, fmt.Sprintf("%v", val)))
		default:
			attrs = append(attrs, otelLog.String(k, fmt.Sprintf("%v", val)))
		}
	}

	return attrs
}

// emitWithTimeout отправляет лог с таймаутом.
func (c *SimpleOTLPCore) emitWithTimeout(record otelLog.Record) error {
	if c.otlpLogger == nil {
		return fmt.Errorf("otlp logger is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.emitTimeout)
	defer cancel()
	c.otlpLogger.Emit(ctx, record)
	return nil
}
