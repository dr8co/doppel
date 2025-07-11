package logger

import (
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
	case "null":
		writer = io.Discard
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
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "text", "":
		handler = slog.NewTextHandler(writer, opts)
	default:
		// Default to text format for unknown formats
		_, _ = fmt.Fprintf(os.Stderr, "Unknown log format '%s'. Using text format.\n", format)
		handler = slog.NewTextHandler(writer, opts)
	}

	Log = slog.New(handler)
	initialized = true
}

// parseLogLevel converts string log level to slog.Level
func parseLogLevel(levelStr string) slog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		_, _ = fmt.Fprintf(os.Stderr, "Unknown log level '%s'. Using info level.\n", levelStr)
		return slog.LevelInfo
	}
}

func Info(args ...interface{}) {
	if Log != nil {
		Log.Info(fmt.Sprint(args...))
	}
}

func Infof(format string, args ...interface{}) {
	if Log != nil {
		Log.Info(fmt.Sprintf(format, args...))
	}
}

func Warn(args ...interface{}) {
	if Log != nil {
		Log.Warn(fmt.Sprint(args...))
	}
}

func Warnf(format string, args ...interface{}) {
	if Log != nil {
		Log.Warn(fmt.Sprintf(format, args...))
	}
}

func Error(args ...interface{}) {
	if Log != nil {
		Log.Error(fmt.Sprint(args...))
	}
}

func Errorf(format string, args ...interface{}) {
	if Log != nil {
		Log.Error(fmt.Sprintf(format, args...))
	}
}

func Debug(args ...interface{}) {
	if Log != nil {
		Log.Debug(fmt.Sprint(args...))
	}
}

func Debugf(format string, args ...interface{}) {
	if Log != nil {
		Log.Debug(fmt.Sprintf(format, args...))
	}
}

func Close() {
	if logFile != nil {
		_ = logFile.Close()
	}
}
