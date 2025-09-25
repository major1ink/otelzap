package logger

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Config определяет настройки логгера.
type Config struct {
	AsJSON             bool                                // Формат вывода: JSON (true) или консоль (false).
	EnableOTLP         bool                                // Включить экспорт в OTLP.
	EnableStdout       bool                                // Включить вывод в stdout.
	Level              string                              // Уровень логирования (debug, info, warn, error).
	OtlpEndpoint       string                              // Эндпоинт OTLP коллектора.
	OtlpUseTLS         bool                                // Использовать TLS для OTLP.
	ServiceName        string                              // Имя сервиса для телеметрии.
	ServiceEnvironment string                              // Окружение сервиса (prod, dev).
	ShutdownTimeout    time.Duration                       // Таймаут для shutdown OTLP.
	FieldExtractors    []func(context.Context) []zap.Field // Кастомные функции для извлечения полей из контекста.
}

// Option настраивает Config.
type Option func(*Config)

// WithAsJSON включает/выключает JSON формат.
func WithAsJSON(v bool) Option { return func(c *Config) { c.AsJSON = v } }

// WithEnableOTLP включает/выключает OTLP.
func WithEnableOTLP(v bool) Option { return func(c *Config) { c.EnableOTLP = v } }

// WithEnableStdout включает/выключает вывод в stdout.
func WithEnableStdout(v bool) Option { return func(c *Config) { c.EnableStdout = v } }

// WithLevel устанавливает уровень логирования.
func WithLevel(level string) Option { return func(c *Config) { c.Level = level } }

// WithOTLPEndpoint устанавливает эндпоинт OTLP.
func WithOTLPEndpoint(endpoint string) Option { return func(c *Config) { c.OtlpEndpoint = endpoint } }

// WithOTLPUseTLS включает TLS для OTLP.
func WithOTLPUseTLS(v bool) Option { return func(c *Config) { c.OtlpUseTLS = v } }

// WithServiceName устанавливает имя сервиса.
func WithServiceName(name string) Option { return func(c *Config) { c.ServiceName = name } }

// WithServiceEnvironment устанавливает окружение.
func WithServiceEnvironment(env string) Option { return func(c *Config) { c.ServiceEnvironment = env } }

// WithShutdownTimeout устанавливает таймаут для shutdown.
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(c *Config) { c.ShutdownTimeout = timeout }
}

// WithFieldExtractor добавляет кастомный extractor полей из контекста.
func WithFieldExtractor(fn func(context.Context) []zap.Field) Option {
	return func(c *Config) { c.FieldExtractors = append(c.FieldExtractors, fn) }
}
