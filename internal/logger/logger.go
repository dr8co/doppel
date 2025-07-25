package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

var defaultLogger atomic.Pointer[Logger]

// Config holds the configuration for the logger.
type Config struct {
	// Format specifies the log format (e.g., "text", "json", "pretty", etc.)
	Format string

	// Writer is the output destination for the logs (e.g., os.Stdout, os.Stderr, or a file)
	Writer io.Writer

	// Options holds additional options for the slog.Handler
	Options *slog.HandlerOptions
}

// Logger is a wrapper around slog.Logger with additional configuration
// and convenience methods for logging at different levels.
type Logger struct {
	logger *slog.Logger
}

// init initializes the default logger with a basic configuration.
func init() {
	ResetDefault()
}

// New creates a new Logger instance with the provided configuration.
func New(config *Config) (*Logger, error) {
	if config == nil {
		config = &Config{Writer: os.Stdout}
	}

	if config.Writer == nil {
		return nil, fmt.Errorf("writer cannot be nil")
	}

	if config.Options == nil {
		config.Options = &slog.HandlerOptions{}
	}

	handler := createHandler(config)

	return &Logger{logger: slog.New(handler)}, nil

}

// Logger returns the underlying slog.Logger instance.
func (l *Logger) Logger() *slog.Logger {
	return l.logger
}

// LogAttrs logs a message with attributes at the specified level.
func (l *Logger) LogAttrs(ctx context.Context, level slog.Level, message string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, level, message, attrs...)
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

// Log logs a message at the specified level.
func (l *Logger) Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	l.logger.Log(ctx, level, msg, args...)
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info logs an informational message.
func (l *Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// DebugContext logs a debug message with context.
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, args...)
}

// InfoContext logs an informational message with context.
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, args...)
}

// WarnContext logs a warning message with context.
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, msg, args...)
}

// ErrorContext logs an error message with context.
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, args...)
}

// Handler returns the underlying slog.Handler used by the logger.
func (l *Logger) Handler() slog.Handler {
	return l.logger.Handler()
}

// With returns a Logger that includes the given attributes in each output operation.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{l.logger.With(args...)}
}

// WithGroup returns a Logger that starts a group
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{l.logger.WithGroup(name)}
}

// Default returns the default logger.
func Default() *Logger {
	return defaultLogger.Load()
}

// SetDefault sets the default logger to the provided logger.
func SetDefault(logger *Logger) error {
	if logger == nil {
		return fmt.Errorf("logger cannot be nil")
	}
	defaultLogger.Store(logger)
	return nil
}

// ResetDefault resets the default logger to the standard slog.Default logger.
func ResetDefault() {
	defaultLogger.Store(&Logger{slog.Default()})
}

// NewDefault creates a new default logger with the provided configuration.
func NewDefault(config *Config) error {
	newLogger, err := New(config)
	if err != nil {
		return err
	}
	defaultLogger.Store(newLogger)
	return nil
}

// createHandler creates a slog.Handler based on the format string.
func createHandler(config *Config) slog.Handler {
	switch strings.ToLower(config.Format) {
	case "text", "":
		// TODO: Fix frame detection.
		return slog.NewTextHandler(config.Writer, config.Options)
	case "json":
		// TODO: Fix frame detection here, too.
		return slog.NewJSONHandler(config.Writer, config.Options)
	case "null", "discard":
		return slog.DiscardHandler
	case "pretty", "color", "terminal", "human":
		return NewPrettyHandler(config.Writer, config.Options)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "Unknown log format '%s'. Using text format.\n", config.Format)
		return slog.NewTextHandler(config.Writer, config.Options)
	}
}

// NewConfig creates a new Config instance based on the provided parameters.
// If the output is a file, it is opened and a closer is returned.
// The closer can be used to close the file when done.
func NewConfig(opts *slog.HandlerOptions, format, output string) (Config, io.Closer, error) {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	config := Config{
		Options: opts,
		Format:  format,
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
		if outFile == "." {
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
	defaultLogger.Load().logger.Info(msg, args...)
}

// InfoCtx logs an informational message with context.
func InfoCtx(ctx context.Context, msg string, args ...any) {
	defaultLogger.Load().logger.InfoContext(ctx, msg, args...)
}

// InfoAttrs logs an informational message with attributes.
func InfoAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	defaultLogger.Load().logger.LogAttrs(ctx, slog.LevelInfo, message, attrs...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	defaultLogger.Load().logger.Warn(msg, args...)
}

// WarnCtx logs a warning message with context.
func WarnCtx(ctx context.Context, msg string, args ...any) {
	defaultLogger.Load().logger.WarnContext(ctx, msg, args...)
}

// WarnAttrs logs a warning message with attributes.
func WarnAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	defaultLogger.Load().logger.LogAttrs(ctx, slog.LevelWarn, message, attrs...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	defaultLogger.Load().logger.Error(msg, args...)
}

// ErrorCtx logs an error message with context.
func ErrorCtx(ctx context.Context, msg string, args ...any) {
	defaultLogger.Load().logger.ErrorContext(ctx, msg, args...)
}

// ErrorAttrs logs an error message with attributes.
func ErrorAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	defaultLogger.Load().logger.LogAttrs(ctx, slog.LevelError, message, attrs...)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	defaultLogger.Load().logger.Debug(msg, args...)
}

// DebugCtx logs a debug message with context.
func DebugCtx(ctx context.Context, msg string, args ...any) {
	defaultLogger.Load().logger.DebugContext(ctx, msg, args...)
}

// DebugAttrs logs a debug message with attributes.
func DebugAttrs(ctx context.Context, message string, attrs ...slog.Attr) {
	defaultLogger.Load().logger.LogAttrs(ctx, slog.LevelDebug, message, attrs...)
}
