package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	log         *slog.Logger
	logFile     *os.File
	initialized bool
	mu          sync.RWMutex
)

// init initializes the logger with safe defaults for testing.
func init() {
	if !initialized {
		initLoggerUnsafe("", "", "")
		initialized = false
	}
}

// InitLogger configures the global logger with thread safety.
func InitLogger(level string, format string, output string) {
	if !initialized {
		mu.Lock()
		defer mu.Unlock()
		initLoggerUnsafe(level, format, output)
	}

}

// initLoggerUnsafe is the internal implementation that initializes the logger without locks.
func initLoggerUnsafe(level string, format string, output string) {
	lvl := parseLogLevel(level)

	var opts *slog.HandlerOptions
	if lvl == slog.LevelDebug {
		opts = &slog.HandlerOptions{
			Level:     lvl,
			AddSource: true,
		}
	} else {
		opts = &slog.HandlerOptions{
			Level: lvl,
		}
	}

	writer, err := createWriter(output)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to create writer for %s: %v.\nFalling back to stdout.\n", output, err)
		writer = os.Stdout
	}

	handler := createHandler(writer, format, opts)

	log = slog.New(handler)
	initialized = true
}

// createWriter creates an io.Writer based on the output string.
func createWriter(output string) (io.Writer, error) {
	switch output {
	case "stdout", "":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "null", "discard":
		return io.Discard, nil
	default:
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", output, err)
		}

		if logFile != nil {
			_ = logFile.Close()
		}
		logFile = file
		return file, nil
	}
}

// createHandler creates a slog.Handler based on the format string.
func createHandler(writer io.Writer, format string, opts *slog.HandlerOptions) slog.Handler {
	switch strings.ToLower(format) {
	case "text", "":
		return slog.NewTextHandler(writer, opts)
	case "json":
		return slog.NewJSONHandler(writer, opts)
	case "null", "discard":
		return slog.DiscardHandler
	case "pretty", "color", "terminal", "human":
		return NewPrettyHandler(writer, opts)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "Unknown log format '%s'. Using text format.\n", format)
		return slog.NewTextHandler(writer, opts)
	}
}

// parseLogLevel converts string log level to slog.Level
func parseLogLevel(levelStr string) slog.Level {
	switch strings.ToLower(levelStr) {
	case "info", "":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		_, _ = fmt.Fprintf(os.Stderr, "Unknown log level '%s'. Using info level.\n", levelStr)
		return slog.LevelInfo
	}
}

// GetLogger returns the global logger instance (thread-safe).
func GetLogger() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return log
}

// Info logs an informational message.
func Info(msg string, args ...any) {
	GetLogger().Info(msg, args...)
}

// InfoCtx logs an informational message with context.
func InfoCtx(ctx context.Context, msg string, args ...any) {
	GetLogger().InfoContext(ctx, msg, args...)
}

// InfoAttrs logs an informational message with attributes.
func InfoAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	GetLogger().LogAttrs(ctx, slog.LevelInfo, message, attrs...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	GetLogger().Warn(msg, args...)
}

// WarnCtx logs a warning message with context.
func WarnCtx(ctx context.Context, msg string, args ...any) {
	GetLogger().WarnContext(ctx, msg, args...)
}

// WarnAttrs logs a warning message with attributes.
func WarnAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	GetLogger().LogAttrs(ctx, slog.LevelWarn, message, attrs...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	GetLogger().Error(msg, args...)
}

// ErrorCtx logs an error message with context.
func ErrorCtx(ctx context.Context, msg string, args ...any) {
	GetLogger().ErrorContext(ctx, msg, args...)
}

// ErrorAttrs logs an error message with attributes.
func ErrorAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	GetLogger().LogAttrs(ctx, slog.LevelError, message, attrs...)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	GetLogger().Debug(msg, args...)
}

// DebugCtx logs a debug message with context.
func DebugCtx(ctx context.Context, msg string, args ...any) {
	GetLogger().DebugContext(ctx, msg, args...)
}

// DebugAttrs logs a debug message with attributes.
func DebugAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	GetLogger().LogAttrs(ctx, slog.LevelDebug, message, attrs...)
}

// Close closes the log file if it was opened.
func Close() {
	mu.Lock()
	defer mu.Unlock()

	// Close the log file if it was opened
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}

	initialized = false
	log = nil
}
