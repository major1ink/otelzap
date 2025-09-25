package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otelLog "go.opentelemetry.io/otel/log"
	otelLogSdk "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger обёртка над zap.Logger с поддержкой контекста и OTLP.
type Logger struct {
	zapLogger    *zap.Logger
	otelProvider *otelLogSdk.LoggerProvider
	config       Config
}

// NewLogger создает новый экземпляр логгера.
func NewLogger(ctx context.Context, opts ...Option) (*Logger, error) {
	cfg := Config{
		AsJSON:          true,
		EnableOTLP:      false,
		EnableStdout:    true,
		Level:           "info",
		ShutdownTimeout: 2 * time.Second,
	}

	for _, o := range opts {
		o(&cfg)
	}

	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	cores, otelProvider, err := buildCores(ctx, cfg, level)
	if err != nil {
		return nil, fmt.Errorf("failed to build cores: %w", err)
	}

	zapLogger := zap.New(
		zapcore.NewTee(cores...),
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)

	return &Logger{
		zapLogger:    zapLogger,
		otelProvider: otelProvider,
		config:       cfg,
	}, nil
}

// buildCores создает слайс cores для zapcore.Tee.
func buildCores(ctx context.Context, cfg Config, level zapcore.Level) ([]zapcore.Core, *otelLogSdk.LoggerProvider, error) {
	var cores []zapcore.Core
	var otelProvider *otelLogSdk.LoggerProvider

	if cfg.EnableStdout {
		cores = append(cores, createStdoutCore(cfg.AsJSON, level))
	}

	if cfg.EnableOTLP {
		otlpCore, provider, err := createOTLPCore(ctx, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create OTLP core: %v\n", err)
		} else {
			cores = append(cores, otlpCore)
			otelProvider = provider
		}
	}

	if len(cores) == 0 {
		return nil, nil, fmt.Errorf("no cores configured")
	}

	return cores, otelProvider, nil
}

// createStdoutCore создает core для вывода в stdout.
func createStdoutCore(asJSON bool, level zapcore.Level) zapcore.Core {
	config := buildEncoderConfig()
	var encoder zapcore.Encoder
	if asJSON {
		encoder = zapcore.NewJSONEncoder(config)
	} else {
		encoder = zapcore.NewConsoleEncoder(config)
	}
	return zapcore.NewCore(encoder, &noSyncWriter{os.Stdout}, level)
}

func createOTLPCore(ctx context.Context, cfg Config) (*SimpleOTLPCore, *otelLogSdk.LoggerProvider, error) {
	otlpLogger, provider, processor, err := createOTLPLogger(ctx, cfg.OtlpEndpoint, cfg.ServiceName, cfg.ServiceEnvironment, cfg.OtlpUseTLS)
	if err != nil {
		return nil, nil, err
	}
	return NewSimpleOTLPCore(otlpLogger, processor, zap.NewAtomicLevelAt(parseLevelDefault(cfg.Level)), cfg.ShutdownTimeout), provider, nil
}

// createOTLPLogger создает OTLP логгер.
func createOTLPLogger(ctx context.Context, endpoint, serviceName, serviceEnvironment string, useTLS bool) (otelLog.Logger, *otelLogSdk.LoggerProvider, *otelLogSdk.BatchProcessor, error) {
	exporter, err := createOTLPExporter(ctx, endpoint, useTLS)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}
	rs, err := createResource(ctx, serviceName, serviceEnvironment)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}
	processor := otelLogSdk.NewBatchProcessor(exporter)
	provider := otelLogSdk.NewLoggerProvider(
		otelLogSdk.WithResource(rs),
		otelLogSdk.WithProcessor(processor),
	)
	return provider.Logger("app"), provider, processor, nil
}

// createOTLPExporter создает gRPC экспортер для OTLP.
func createOTLPExporter(ctx context.Context, endpoint string, useTLS bool) (*otlploggrpc.Exporter, error) {
	opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(endpoint)}
	if !useTLS {
		opts = append(opts, otlploggrpc.WithInsecure())
	}
	return otlploggrpc.New(ctx, opts...)
}

// createResource создает метаданные сервиса.
func createResource(ctx context.Context, serviceName, serviceEnvironment string) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			attribute.String("deployment.environment", serviceEnvironment),
		),
	)
}

// buildEncoderConfig настраивает Zap encoder.
func buildEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
}

// SetLevel динамически меняет уровень логирования.
func (l *Logger) SetLevel(levelStr string) error {
	level, err := parseLevel(levelStr)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	l.zapLogger.Core().Enabled(level)
	return nil
}

// Sync сбрасывает буферы логгера.
func (l *Logger) Sync() error {
	return l.zapLogger.Sync()
}

// Close завершает работу логгера и OTLP провайдера.
func (l *Logger) Close() error {
	var errs []error
	if err := l.zapLogger.Sync(); err != nil {
		errs = append(errs, fmt.Errorf("failed to sync zap: %w", err))
	}
	if l.otelProvider != nil {
		ctx, cancel := context.WithTimeout(context.Background(), l.config.ShutdownTimeout)
		defer cancel()
		if err := l.otelProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown OTLP: %w", err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// With создает новый логгер с дополнительными полями.
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{
		zapLogger:    l.zapLogger.With(fields...),
		otelProvider: l.otelProvider,
		config:       l.config,
	}
}

// WithContext создает логгер с полями из контекста.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		zapLogger:    l.zapLogger.With(l.fieldsFromContext(ctx)...),
		otelProvider: l.otelProvider,
		config:       l.config,
	}
}

// Debug логирует на уровне Debug.
func (l *Logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	l.zapLogger.Debug(msg, append(l.fieldsFromContext(ctx), fields...)...)
}

// Info логирует на уровне Info.
func (l *Logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	l.zapLogger.Info(msg, append(l.fieldsFromContext(ctx), fields...)...)
}

// Warn логирует на уровне Warn.
func (l *Logger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	l.zapLogger.Warn(msg, append(l.fieldsFromContext(ctx), fields...)...)
}

// Error логирует на уровне Error.
func (l *Logger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	l.zapLogger.Error(msg, append(l.fieldsFromContext(ctx), fields...)...)
}

// Fatal логирует на уровне Fatal и завершает программу.
func (l *Logger) Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	l.zapLogger.Fatal(msg, append(l.fieldsFromContext(ctx), fields...)...)
}

// Sugar возвращает sugared логгер.
func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.zapLogger.Sugar()
}

// NewNopLogger создает no-op логгер для тестов.
func NewNopLogger() *Logger {
	return &Logger{
		zapLogger: zap.NewNop(),
	}
}

// NewBenchmarkLogger создает no-op логгер для бенчмарков.
func NewBenchmarkLogger() *Logger {
	return &Logger{
		zapLogger: zap.New(zapcore.NewNopCore()),
	}
}

// noSyncWriter оборачивает io.Writer, игнорируя Sync.
type noSyncWriter struct {
	io.Writer
}

func (w *noSyncWriter) Sync() error {
	return nil // Игнорируем Sync
}
