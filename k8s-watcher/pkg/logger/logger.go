package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/wasim-nihal/k8s-watcher/pkg/config"
)

var (
	defaultLogger *slog.Logger
)

// Initialize sets up the logging system based on configuration
func Initialize(cfg config.LoggingConfig) error {
	var level slog.Level
	switch strings.ToUpper(cfg.Level) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		return fmt.Errorf("invalid log level: %s", cfg.Level)
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	switch strings.ToUpper(cfg.Format) {
	case "JSON":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "LOGFMT":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		return fmt.Errorf("invalid log format: %s", cfg.Format)
	}

	// Set timezone
	if strings.ToUpper(cfg.Timezone) == "UTC" {
		handler = handler.WithAttrs([]slog.Attr{
			slog.Time("time", time.Now().UTC()),
		})
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)

	return nil
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// With creates a new logger with the given attributes
func With(args ...any) *slog.Logger {
	return defaultLogger.With(args...)
}
