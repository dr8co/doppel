package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

var (
	Log         *slog.Logger
	logFile     *os.File
	initialized bool
)

// InitLogger configures the global logger using the provided Config struct.
func InitLogger(level string, format string, output string) {
	if initialized {
		return
	}

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

	var writer io.Writer
	switch output {
	case "stdout", "":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	case "null", "discard":
		writer = io.Discard
		format = "null"
	default:
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to open log file %s: %v.\nFalling back to stdout.\n", output, err)
			writer = os.Stdout
		} else {
			// Close the previous log if it exists
			if logFile != nil {
				_ = logFile.Close()
			}
			logFile = file
			writer = file
		}

	}

	var handler slog.Handler
	switch strings.ToLower(format) {
	case "text", "":
		handler = slog.NewTextHandler(writer, opts)
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "null", "discard":
		handler = slog.DiscardHandler
	default:
		// Default to text format for unknown formats
		_, _ = fmt.Fprintf(os.Stderr, "Unknown log format '%s'. Using text format.\n", format)
		handler = slog.NewTextHandler(writer, opts)
	}

	Log = slog.New(handler)
	slog.SetDefault(Log)
	initialized = true
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

// Info logs an informational message.
func Info(args ...any) {
	if Log != nil {
		Log.Info(fmt.Sprint(args...))
	}
}

// Infof logs an informational message with formatting.
func Infof(format string, args ...any) {
	if Log != nil {
		Log.Info(fmt.Sprintf(format, args...))
	}
}

// InfoAttrs logs an informational message with attributes.
func InfoAttrs(message string, attrs ...slog.Attr) {
	if Log != nil {
		Log.LogAttrs(context.TODO(), slog.LevelInfo, message, attrs...)
	}
}

// Warn logs a warning message.
func Warn(args ...any) {
	if Log != nil {
		Log.Warn(fmt.Sprint(args...))
	}
}

// Warnf logs a warning message with formatting.
func Warnf(format string, args ...any) {
	if Log != nil {
		Log.Warn(fmt.Sprintf(format, args...))
	}
}

// WarnAttrs logs a warning message with attributes.
func WarnAttrs(message string, attrs ...slog.Attr) {
	if Log != nil {
		Log.LogAttrs(context.TODO(), slog.LevelWarn, message, attrs...)
	}
}

// Error logs an error message.
func Error(args ...any) {
	if Log != nil {
		Log.Error(fmt.Sprint(args...))
	}
}

// Errorf logs an error message with formatting.
func Errorf(format string, args ...any) {
	if Log != nil {
		Log.Error(fmt.Sprintf(format, args...))
	}
}

// ErrorAttrs logs an error message with attributes.
func ErrorAttrs(message string, attrs ...slog.Attr) {
	if Log != nil {
		Log.LogAttrs(context.TODO(), slog.LevelError, message, attrs...)
	}
}

// Debug logs a debug message.
func Debug(args ...any) {
	if Log != nil {
		Log.Debug(fmt.Sprint(args...))
	}
}

// Debugf logs a debug message with formatting.
func Debugf(format string, args ...any) {
	if Log != nil {
		Log.Debug(fmt.Sprintf(format, args...))
	}
}

// DebugAttrs logs a debug message with attributes.
func DebugAttrs(message string, attrs ...slog.Attr) {
	if Log != nil {
		Log.LogAttrs(context.TODO(), slog.LevelDebug, message, attrs...)
	}
}

// Close closes the log file if it was opened.
func Close() {
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
		initialized = false
	}
}
