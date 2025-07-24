package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	defaultLogger *Logger
	once          sync.Once
)

// Config holds the configuration for the logger
type Config struct {
	Level  string
	Format string
	Writer io.Writer
}

// Logger is a wrapper around slog.Logger with additional configuration
// and convenience methods for logging at different levels.
type Logger struct {
	logger *slog.Logger
	config Config
}

// init initializes the default logger with a basic configuration.
// It is intended for use in tests or when no specific configuration is provided.
func init() {
	defaultLogger = &Logger{
		logger: slog.Default(),
		config: Config{
			Level:  "",
			Format: "",
			Writer: nil,
		},
	}
}

// New creates a new Logger instance with the provided configuration.
func New(config Config) (*Logger, error) {
	if config.Writer == nil {
		return nil, fmt.Errorf("writer cannot be nil")
	}

	level := parseLogLevel(config.Level)

	var opts *slog.HandlerOptions
	if level == slog.LevelDebug {
		opts = &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		}
	} else {
		opts = &slog.HandlerOptions{
			Level: level,
		}
	}

	handler := createHandler(config.Writer, config.Format, opts)

	return &Logger{logger: slog.New(handler), config: config}, nil

}

// Config returns the current configuration of the logger.
func (l *Logger) Config() Config {
	return l.config
}

// Logger returns the underlying slog.Logger instance.
func (l *Logger) Logger() *slog.Logger {
	return l.logger
}

// InfoAttrs logs an informational message with attributes.
func (l *Logger) InfoAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelInfo, message, attrs...)
}

// WarnAttrs logs a warning message with attributes.
func (l *Logger) WarnAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelWarn, message, attrs...)
}

// ErrorAttrs logs an error message with attributes.
func (l *Logger) ErrorAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelError, message, attrs...)
}

// DebugAttrs logs a debug message with attributes.
func (l *Logger) DebugAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelDebug, message, attrs...)
}

// InitDefault initializes the default logger with the provided configuration.
func InitDefault(config Config) error {
	var err error
	once.Do(func() {
		defaultLogger, err = New(config)
	})
	return err
}

func GetDefault() *Logger {
	return defaultLogger
}

func SetDefault(logger *Logger) error {
	if logger == nil {
		return fmt.Errorf("logger cannot be nil")
	}
	defaultLogger = logger
	return nil
}

// createHandler creates a slog.Handler based on the format string.
func createHandler(writer io.Writer, format string, opts *slog.HandlerOptions) slog.Handler {
	switch strings.ToLower(format) {
	case "text", "":
		// TODO: Fix frame detection.
		return slog.NewTextHandler(writer, opts)
	case "json":
		// TODO: Fix frame detection here, too.
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

// NewConfig creates a new Config instance based on the provided parameters.
// If the output is a file, it is opened and a closer is returned.
// The closer can be used to close the file when done.
func NewConfig(level, format, output string) (Config, io.Closer, error) {
	config := Config{
		Level:  level,
		Format: format,
	}
	var closer io.Closer = nil

	switch output {
	case "stdout", "":
		config.Writer = os.Stdout
	case "stderr":
		config.Writer = os.Stderr
	case "null", "discard":
		config.Writer = io.Discard
	default:
		outFile := filepath.Clean(output)
		if outFile == "." || outFile == "" {
			return Config{}, nil, fmt.Errorf("invalid file path")
		}
		
		var err error
		outFile, err = filepath.Abs(outFile)
		if err != nil {
			return Config{}, nil, fmt.Errorf("failed to get absolute path for log file: %w", err)
		}

		if err = os.MkdirAll(filepath.Dir(outFile), 0755); err != nil {
			return Config{}, nil, fmt.Errorf("error creating log directory: %w", err)
		}

		file, err := os.OpenFile(outFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return Config{}, nil, fmt.Errorf("failed to open log file %s: %w", outFile, err)
		}

		config.Writer = file
		closer = file
	}
	return config, closer, nil
}

// Info logs an informational message.
func Info(msg string, args ...any) {
	defaultLogger.logger.Info(msg, args...)
}

// InfoCtx logs an informational message with context.
func InfoCtx(ctx context.Context, msg string, args ...any) {
	defaultLogger.logger.InfoContext(ctx, msg, args...)
}

// InfoAttrs logs an informational message with attributes.
func InfoAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	defaultLogger.logger.LogAttrs(ctx, slog.LevelInfo, message, attrs...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	defaultLogger.logger.Warn(msg, args...)
}

// WarnCtx logs a warning message with context.
func WarnCtx(ctx context.Context, msg string, args ...any) {
	defaultLogger.logger.WarnContext(ctx, msg, args...)
}

// WarnAttrs logs a warning message with attributes.
func WarnAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	defaultLogger.logger.LogAttrs(ctx, slog.LevelWarn, message, attrs...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	defaultLogger.logger.Error(msg, args...)
}

// ErrorCtx logs an error message with context.
func ErrorCtx(ctx context.Context, msg string, args ...any) {
	defaultLogger.logger.ErrorContext(ctx, msg, args...)
}

// ErrorAttrs logs an error message with attributes.
func ErrorAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	defaultLogger.logger.LogAttrs(ctx, slog.LevelError, message, attrs...)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	defaultLogger.logger.Debug(msg, args...)
}

// DebugCtx logs a debug message with context.
func DebugCtx(ctx context.Context, msg string, args ...any) {
	defaultLogger.logger.DebugContext(ctx, msg, args...)
}

// DebugAttrs logs a debug message with attributes.
func DebugAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	defaultLogger.logger.LogAttrs(ctx, slog.LevelDebug, message, attrs...)
}
