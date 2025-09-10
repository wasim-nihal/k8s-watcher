package logger_test

import (
	"log/slog"
	"testing"

	"github.com/wasim-nihal/k8s-watcher/pkg/config"
	"github.com/wasim-nihal/k8s-watcher/pkg/logger"
)

func TestLoggerInitialization(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.LoggingConfig
		wantErr bool
	}{
		{
			name: "valid config - JSON format",
			cfg: config.LoggingConfig{
				Level:    "INFO",
				Format:   "JSON",
				Timezone: "UTC",
			},
			wantErr: false,
		},
		{
			name: "valid config - LOGFMT format",
			cfg: config.LoggingConfig{
				Level:    "DEBUG",
				Format:   "LOGFMT",
				Timezone: "LOCAL",
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			cfg: config.LoggingConfig{
				Level:    "INVALID",
				Format:   "JSON",
				Timezone: "UTC",
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			cfg: config.LoggingConfig{
				Level:    "INFO",
				Format:   "INVALID",
				Timezone: "UTC",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := logger.Initialize(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoggingLevels(t *testing.T) {
	// Set up a test logger
	cfg := config.LoggingConfig{
		Level:    "DEBUG",
		Format:   "JSON",
		Timezone: "UTC",
	}
	err := logger.Initialize(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Test all logging levels
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")

}

func TestLoggerWith(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:    "INFO",
		Format:   "JSON",
		Timezone: "UTC",
	}
	err := logger.Initialize(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Create a logger with additional context
	contextLogger := logger.With(
		"component", "test",
		"requestID", "123",
	)

	if contextLogger == nil {
		t.Error("With() returned nil logger")
	}

	// Log with the context logger
	contextLogger.Info("test message")
}

func BenchmarkLoggerInfo(b *testing.B) {
	cfg := config.LoggingConfig{
		Level:    "INFO",
		Format:   "JSON",
		Timezone: "UTC",
	}
	logger.Initialize(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "iteration", i)
	}
}

func BenchmarkLoggerWith(b *testing.B) {
	cfg := config.LoggingConfig{
		Level:    "INFO",
		Format:   "JSON",
		Timezone: "UTC",
	}
	logger.Initialize(cfg)
	attrs := []any{"component", "test", "requestID", "123"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.With(attrs...).Info("benchmark message", "iteration", i)
	}
}

func TestLogLevelParsing(t *testing.T) {
	tests := []struct {
		level   string
		want    slog.Level
		wantErr bool
	}{
		{"DEBUG", slog.LevelDebug, false},
		{"INFO", slog.LevelInfo, false},
		{"WARN", slog.LevelWarn, false},
		{"ERROR", slog.LevelError, false},
		{"debug", slog.LevelDebug, false}, // Test case insensitivity
		{"info", slog.LevelInfo, false},   // Test case insensitivity
		{"INVALID", slog.LevelInfo, true}, // Test invalid level
		{"", slog.LevelInfo, true},        // Test empty level
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			cfg := config.LoggingConfig{
				Level:    tt.level,
				Format:   "JSON",
				Timezone: "UTC",
			}

			err := logger.Initialize(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTimezoneHandling(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
	}{
		{"UTC timezone", "UTC"},
		{"Local timezone", "LOCAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.LoggingConfig{
				Level:    "INFO",
				Format:   "JSON",
				Timezone: tt.timezone,
			}

			err := logger.Initialize(cfg)
			if err != nil {
				t.Fatalf("Initialize() failed: %v", err)
			}

			// Log a test message
			logger.Info("test message")
		})
	}
}
