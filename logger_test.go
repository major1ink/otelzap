package logger

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewLogger(t *testing.T) {
	ctx := context.Background()
	l, err := NewLogger(
		ctx,
		WithLevel("info"),
		WithAsJSON(true),
		WithEnableStdout(true),
		WithShutdownTimeout(2*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	l.Info(ctx, "Test message", zap.String("key", "value"))
	if err := l.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}
}

func TestNoopLogger(t *testing.T) {
	l := NewNoopLogger()
	l.Info(context.Background(), "Test message", zap.String("key", "value"))
	if err := l.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}
}

func TestFieldsFromContext(t *testing.T) {
	l := NewNoopLogger()
	ctx := context.WithValue(context.Background(), traceIDKey, "12345")
	ctx = context.WithValue(ctx, userIDKey, "user1")
	fields := l.fieldsFromContext(ctx)
	if len(fields) != 2 {
		t.Fatalf("Expected 2 fields, got %d", len(fields))
	}
	if fields[0].Key != string(traceIDKey) || fields[0].String != "12345" {
		t.Errorf("Expected trace_id=12345, got %v", fields[0])
	}
	if fields[1].Key != string(userIDKey) || fields[1].String != "user1" {
		t.Errorf("Expected user_id=user1, got %v", fields[1])
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zapcore.Level
		hasError bool
	}{
		{"debug", zapcore.DebugLevel, false},
		{"info", zapcore.InfoLevel, false},
		{"warn", zapcore.WarnLevel, false},
		{"error", zapcore.ErrorLevel, false},
		{"fatal", zapcore.FatalLevel, false},
		{"invalid", zapcore.InfoLevel, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, err := parseLevel(tt.input)
			if (err != nil) != tt.hasError {
				t.Errorf("Expected error: %v, got: %v", tt.hasError, err)
			}
			if level != tt.expected {
				t.Errorf("Expected level: %v, got: %v", tt.expected, level)
			}
		})
	}
}
