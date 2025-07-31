package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// TestNew tests creating a new logger with valid/invalid configurations
func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config with stdout",
			config: &Config{
				Format:  "json",
				Writer:  os.Stdout,
				Options: &slog.HandlerOptions{Level: slog.LevelInfo},
			},
			wantErr: false,
		},
		{
			name:    "nil config uses defaults",
			config:  nil,
			wantErr: false,
		},
		{
			name: "nil writer should fail",
			config: &Config{
				Format:  "json",
				Writer:  nil,
				Options: &slog.HandlerOptions{Level: slog.LevelInfo},
			},
			wantErr: true,
		},
		{
			name: "nil options gets default",
			config: &Config{
				Format:  "text",
				Writer:  os.Stdout,
				Options: nil,
			},
			wantErr: false,
		},
		{
			name: "empty config with writer",
			config: &Config{
				Writer: os.Stderr,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if logger != nil {
					t.Error("expected logger to be nil on error")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if logger == nil {
				t.Error("expected logger to be non-nil")
				return
			}

			// Test that the logger is functional
			logger.Info("test message")
		})
	}
}

// TestNewConfig tests logger configuration creation
func TestNewConfig(t *testing.T) {
	tests := []struct {
		name         string
		opts         *slog.HandlerOptions
		format       string
		output       string
		expectError  bool
		expectCloser bool
	}{
		{
			name:         "stdout output",
			opts:         &slog.HandlerOptions{Level: slog.LevelInfo},
			format:       "json",
			output:       "stdout",
			expectError:  false,
			expectCloser: false,
		},
		{
			name:         "stderr output",
			opts:         &slog.HandlerOptions{Level: slog.LevelDebug},
			format:       "text",
			output:       "stderr",
			expectError:  false,
			expectCloser: false,
		},
		{
			name:         "null output",
			opts:         &slog.HandlerOptions{Level: slog.LevelWarn},
			format:       "null",
			output:       "null",
			expectError:  false,
			expectCloser: false,
		},
		{
			name:         "file output",
			opts:         &slog.HandlerOptions{Level: slog.LevelError},
			format:       "json",
			output:       filepath.Join(t.TempDir(), "test.log"),
			expectError:  false,
			expectCloser: true,
		},
		{
			name:        "invalid file path",
			opts:        &slog.HandlerOptions{Level: slog.LevelInfo},
			format:      "json",
			output:      ".",
			expectError: true,
		},
		{
			name:        "empty file path",
			opts:        &slog.HandlerOptions{Level: slog.LevelInfo},
			format:      "json",
			output:      "",
			expectError: false, // Should default to stdout
		},
		{
			name:         "discard output",
			opts:         &slog.HandlerOptions{Level: slog.LevelError},
			format:       "json",
			output:       "discard",
			expectError:  false,
			expectCloser: false,
		},

		{
			name:         "nil options",
			opts:         nil,
			format:       "json",
			output:       "stdout",
			expectError:  false,
			expectCloser: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, closer, err := NewConfig(tt.opts, tt.format, tt.output)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.expectCloser && closer == nil {
				t.Error("expected closer to be non-nil")
			}
			if !tt.expectCloser && closer != nil {
				t.Error("expected closer to be nil")
			}

			if config.Format != tt.format {
				t.Errorf("expected format %s, got %s", tt.format, config.Format)
			}
			if config.Writer == nil {
				t.Error("expected writer to be non-nil")
			}

			if closer != nil {
				err = closer.Close()
				if err != nil {
					t.Errorf("expected nil, got %v", err)
				}
			}
		})
	}
}

// TestLogLevels tests all log levels work correctly
func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Format:  "json",
		Writer:  &buf,
		Options: &slog.HandlerOptions{Level: slog.LevelDebug},
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	ctx := context.Background()
	testMessage := "test message"
	testArgs := []any{"key", "value"}

	// Test all logging methods
	logger.Debug(testMessage, testArgs...)
	logger.Info(testMessage, testArgs...)
	logger.Warn(testMessage, testArgs...)
	logger.Error(testMessage, testArgs...)

	// Test context methods
	logger.DebugContext(ctx, testMessage, testArgs...)
	logger.InfoContext(ctx, testMessage, testArgs...)
	logger.WarnContext(ctx, testMessage, testArgs...)
	logger.ErrorContext(ctx, testMessage, testArgs...)

	// Test attr methods
	attrs := []slog.Attr{slog.String("attr_key", "attr_value")}
	logger.DebugAttrs(ctx, testMessage, attrs...)
	logger.InfoAttrs(ctx, testMessage, attrs...)
	logger.WarnAttrs(ctx, testMessage, attrs...)
	logger.ErrorAttrs(ctx, testMessage, attrs...)

	output := buf.String()
	if output == "" {
		t.Error("expected log output, got empty string")
	}

	// Count log entries (each line should be a JSON object)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	expectedLines := 12 // Basic 4 + 4 context + 4 attrs
	if len(lines) != expectedLines {
		t.Errorf("expected %d log lines, got %d", expectedLines, len(lines))
	}

	// Verify JSON format
	for i, line := range lines {
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i+1, err)
		}
		if logEntry["msg"] != testMessage {
			t.Errorf("line %d: expected message %q, got %q", i+1, testMessage, logEntry["msg"])
		}
	}
}

// TestCreateHandler tests the creation of log handlers.
func TestCreateHandler(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Writer:  &buf,
		Options: &slog.HandlerOptions{Level: slog.LevelDebug},
	}

	tests := []struct {
		format   string
		expected string
	}{
		{"text", "*slog.TextHandler"},
		{"TEXT", "*slog.TextHandler"},
		{"", "*slog.TextHandler"}, // default
		{"json", "*slog.JSONHandler"},
		{"JSON", "*slog.JSONHandler"},
		{"null", "slog.discardHandler"},
		{"discard", "slog.discardHandler"},
		{"pretty", "*logger.PrettyHandler"},
		{"color", "*logger.PrettyHandler"},
		{"terminal", "*logger.PrettyHandler"},
		{"human", "*logger.PrettyHandler"},
		{"invalid", "*slog.TextHandler"}, // fallback to text
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			config.Format = tt.format
			handler := createHandler(config)
			handlerType := fmt.Sprintf("%T", handler)
			if handlerType != tt.expected {
				t.Errorf("createHandler(%q) = %s, want %s", tt.format, handlerType, tt.expected)
			}
		})
	}
}

// TestGlobalLoggerConcurrency tests that the global logger can be safely used from multiple goroutines.
func TestGlobalLoggerConcurrency(t *testing.T) {
	// Save the original default logger
	originalLogger := defaultLogger.Load()
	defer defaultLogger.Store(originalLogger)

	var buf bytes.Buffer
	config := &Config{
		Format:  "json",
		Writer:  &buf,
		Options: &slog.HandlerOptions{Level: slog.LevelInfo},
	}

	newLogger, err := New(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Test concurrent setting of default logger
	const numGoroutines = 100
	var wg sync.WaitGroup

	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Create a unique logger for this goroutine
			var goroutineBuf bytes.Buffer
			goroutineConfig := &Config{
				Format:  "json",
				Writer:  &goroutineBuf,
				Options: &slog.HandlerOptions{Level: slog.LevelInfo},
			}

			goroutineLogger, err := New(goroutineConfig)
			if err != nil {
				t.Errorf("failed to create logger in goroutine %d: %v", id, err)
				return
			}

			// Set this logger as default (last one wins)
			err = SetDefault(goroutineLogger)
			if err != nil {
				t.Errorf("failed to set logger in goroutine %d: %v", id, err)
				return
			}

			// Use the current default logger
			currentDefault := Default()
			if currentDefault == nil {
				t.Errorf("default logger is nil in goroutine %d", id)
				return
			}
		}(i)
	}

	wg.Wait()

	// Verify we have a valid default logger
	finalDefault := Default()
	if finalDefault == nil {
		t.Error("final default logger should not be nil")
	}

	// Test concurrent logging
	err = SetDefault(newLogger)
	if err != nil {
		t.Fatalf("failed to set default logger: %v", err)
	}
	const numMessages = 1000
	wg = sync.WaitGroup{}

	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			Info("concurrent message", "id", id)
			Warn("concurrent warning", "id", id)
			Error("concurrent error", "id", id)
			Debug("concurrent debug", "id", id)
		}(i)
	}

	wg.Wait()

	// Verify logs were written (should have some output)
	output := buf.String()
	if len(output) == 0 {
		t.Error("expected log output from concurrent logging")
	}

	// Count lines to ensure we got some reasonable number of log entries
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < numMessages {
		t.Logf("Got %d log lines from %d concurrent operations", len(lines), numMessages*4)
	}
}

// TestLoggerImmutability tests if the individual logger instances are truly immutable.
func TestLoggerImmutability(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Format:  "json",
		Writer:  &buf,
		Options: &slog.HandlerOptions{Level: slog.LevelInfo},
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Log a message
	logger.logger.Info("immutable test message")

	// Get the original underlying logger
	originalSlogger := logger.Logger()

	// Create another logger with a different config
	var buf2 bytes.Buffer
	config2 := &Config{
		Format:  "text",
		Writer:  &buf2,
		Options: &slog.HandlerOptions{Level: slog.LevelDebug},
	}

	logger2, err := New(config2)
	if err != nil {
		t.Fatalf("failed to create second logger: %v", err)
	}

	// Verify original logger is unchanged
	currentSlogger := logger.Logger()
	if originalSlogger != currentSlogger {
		t.Error("original logger's internal state should not change")
	}

	// Verify they log to different destinations
	logger.logger.Info("first logger message")
	logger2.logger.Info("second logger message")

	output1 := buf.String()
	output2 := buf2.String()

	if !strings.Contains(output1, "first logger message") {
		t.Error("first logger should log to its configured writer")
	}

	if !strings.Contains(output2, "second logger message") {
		t.Error("second logger should log to its configured writer")
	}

	// Verify formats are different (JSON vs TEXT)
	lines1 := strings.Split(strings.TrimSpace(output1), "\n")
	lines2 := strings.Split(strings.TrimSpace(output2), "\n")

	// The first should be JSON
	var jsonEntry map[string]interface{}
	if err := json.Unmarshal([]byte(lines1[0]), &jsonEntry); err != nil {
		t.Error("first logger should produce JSON output")
	}

	// The second should be text (not valid JSON)
	if json.Unmarshal([]byte(lines2[0]), &jsonEntry) == nil {
		t.Error("second logger should produce text output, not JSON")
	}
}

// TestConcurrencySafety tests that the global logger can be safely used from multiple goroutines.
func TestConcurrencySafety(t *testing.T) {
	// Reset the default logger for this test
	originalLogger := defaultLogger.Load()
	defer defaultLogger.Store(originalLogger)

	var buf bytes.Buffer
	config := &Config{Writer: &buf, Options: &slog.HandlerOptions{Level: slog.LevelInfo}}
	logger, err := New(config)
	if err != nil {
		t.Fatalf("failed to initialize logger: %v", err)
	}

	err = SetDefault(logger)
	if err != nil {
		t.Fatalf("failed to set the default logger: %v", err)
	}

	t.Run("Concurrent logging", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 100

		// All goroutines will now log concurrently to the initialized default logger.
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				Info("message from goroutine", "id", id)
			}(i)
		}
		wg.Wait()

		// Check if all messages were logged.
		output := buf.String()
		for i := 0; i < numGoroutines; i++ {
			expectedMsg := fmt.Sprintf("id=%d", i)
			if !strings.Contains(output, expectedMsg) {
				t.Errorf("Log output is missing message from goroutine %d", i)
			}
		}
	})
}

// TestGlobalFunctions tests all global logging functions.
func TestGlobalFunctions(t *testing.T) {
	// Reset the default logger for this test
	originalLogger := defaultLogger.Load()
	defer defaultLogger.Store(originalLogger)

	var buf bytes.Buffer
	config := &Config{
		Format:  "json",
		Writer:  &buf,
		Options: &slog.HandlerOptions{Level: slog.LevelDebug},
	}

	err := NewDefault(config)
	if err != nil {
		t.Fatalf("failed to set the default logger: %v", err)
	}

	ctx := context.Background()

	// Test all global logging functions
	Info("info message", "key", "value")
	Warn("warn message", "key", "value")
	Error("error message", "key", "value")
	Debug("debug message", "key", "value")

	InfoCtx(ctx, "info ctx message", "key", "value")
	WarnCtx(ctx, "warn ctx message", "key", "value")
	ErrorCtx(ctx, "error ctx message", "key", "value")
	DebugCtx(ctx, "debug ctx message", "key", "value")

	InfoAttrs(ctx, "info attrs message", slog.String("attr", "value"))
	WarnAttrs(ctx, "warn attrs message", slog.String("attr", "value"))
	ErrorAttrs(ctx, "error attrs message", slog.String("attr", "value"))
	DebugAttrs(ctx, "debug attrs message", slog.String("attr", "value"))

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	expectedLines := 12 // Basic 4 + 4 context + 4 attrs
	if len(lines) != expectedLines {
		t.Errorf("expected %d log lines, got %d", expectedLines, len(lines))
	}

	// Verify all lines are valid JSON
	for i, line := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i+1, err)
		}
	}
}

// TestSetDefault tests setting a custom default logger.
func TestSetDefault(t *testing.T) {
	// Save original
	originalLogger := defaultLogger.Load()
	defer defaultLogger.Store(originalLogger)

	var buf bytes.Buffer
	config := &Config{
		Format:  "text",
		Writer:  &buf,
		Options: &slog.HandlerOptions{Level: slog.LevelDebug},
	}

	newLogger, err := New(config)
	if err != nil {
		t.Fatalf("failed to create new logger: %v", err)
	}

	// Test setting valid logger
	err = SetDefault(newLogger)
	if err != nil {
		t.Errorf("SetDefault with valid logger should not error: %v", err)
	}

	if Default() != newLogger {
		t.Error("GetDefault should return the logger we just set")
	}

	// Test setting nil logger
	err = SetDefault(nil)
	if err == nil {
		t.Error("SetDefault with nil logger should return error")
	}

	// Verify logger wasn't changed after nil attempt
	if Default() != newLogger {
		t.Error("default logger should not change when SetDefault fails")
	}

	// Test that the logger works
	Info("test message after setting default")
	output := buf.String()
	if !strings.Contains(output, "test message after setting default") {
		t.Error("expected log message in output")
	}
}

// TestContextMethods tests logging methods that accept context.
func TestContextMethods(t *testing.T) {
	// Save the original default logger
	originalLogger := defaultLogger.Load()
	defer defaultLogger.Store(originalLogger)

	var buf bytes.Buffer
	config := &Config{
		Format:  "json",
		Writer:  &buf,
		Options: &slog.HandlerOptions{Level: slog.LevelDebug},
	}

	err := NewDefault(config)
	if err != nil {
		t.Fatalf("failed to set default logger: %v", err)
	}

	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("test_key"), "test_value")

	// Test global context methods
	InfoCtx(ctx, "info message", "key", "value")
	WarnCtx(ctx, "warn message", "key", "value")
	ErrorCtx(ctx, "error message", "key", "value")
	DebugCtx(ctx, "debug message", "key", "value")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 4 {
		t.Errorf("expected 4 log lines, got %d", len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i+1, err)
		}
	}

	// Reset buffer and test instance methods
	buf.Reset()

	InfoAttrs(ctx, "info attrs message", slog.String("attr", "value"))
	WarnAttrs(ctx, "warn attrs message", slog.String("attr", "value"))
	ErrorAttrs(ctx, "error attrs message", slog.String("attr", "value"))
	DebugAttrs(ctx, "debug attrs message", slog.String("attr", "value"))

	output = buf.String()
	lines = strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 4 {
		t.Errorf("expected 4 log lines for attrs methods, got %d", len(lines))
	}
}

// TestLoggerInstance tests the logger instance methods.
func TestLoggerInstance(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Format:  "json",
		Writer:  &buf,
		Options: &slog.HandlerOptions{Level: slog.LevelInfo},
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Test that Logger() returns the underlying slog.Logger
	slogLogger := logger.Logger()
	if slogLogger == nil {
		t.Fatal("Logger() should return non-nil slog.Logger")
	}

	// Test that we can use the underlying logger directly
	slogLogger.Info("direct slog message", "key", "value")

	output := buf.String()
	if output == "" {
		t.Error("expected output from direct slog usage")
	}

	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	if entry["msg"] != "direct slog message" {
		t.Errorf("expected message 'direct slog message', got %q", entry["msg"])
	}
}

// TestSourceInformation tests the inclusion of the source information in the logs.
func TestSourceInformation(t *testing.T) {
	tests := []struct {
		name         string
		level        slog.Level
		addSource    bool
		expectSource bool
	}{
		{
			name:         "debug level with AddSource true",
			level:        slog.LevelDebug,
			addSource:    true,
			expectSource: true,
		},
		{
			name:         "debug level with AddSource false",
			level:        slog.LevelDebug,
			addSource:    false,
			expectSource: false,
		},
		{
			name:         "info level with AddSource true",
			level:        slog.LevelInfo,
			addSource:    true,
			expectSource: true,
		},
		{
			name:         "info level with AddSource false",
			level:        slog.LevelInfo,
			addSource:    false,
			expectSource: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := &Config{
				Format: "json",
				Writer: &buf,
				Options: &slog.HandlerOptions{
					Level:     tt.level,
					AddSource: tt.addSource,
				},
			}

			logger, err := New(config)
			if err != nil {
				t.Fatalf("failed to create logger: %v", err)
			}

			logger.logger.Log(context.Background(), tt.level, "test message")

			output := buf.String()
			var entry map[string]interface{}
			if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
				t.Errorf("output is not valid JSON: %v", err)
			}

			_, hasSource := entry["source"]
			if tt.expectSource && !hasSource {
				t.Error("expected source information in log entry")
			}
			if !tt.expectSource && hasSource {
				t.Error("did not expect source information in log entry")
			}
		})
	}
}

// TestAtomicDefaultLogger verifies the atomic nature of the default logger.
func TestAtomicDefaultLogger(t *testing.T) {
	originalLogger := defaultLogger.Load()
	defer defaultLogger.Store(originalLogger)

	var buf1, buf2 bytes.Buffer

	config1 := &Config{
		Format:  "json",
		Writer:  &buf1,
		Options: &slog.HandlerOptions{Level: slog.LevelInfo},
	}

	config2 := &Config{
		Format:  "text",
		Writer:  &buf2,
		Options: &slog.HandlerOptions{Level: slog.LevelInfo},
	}

	logger1, err := New(config1)
	if err != nil {
		t.Fatalf("failed to create logger1: %v", err)
	}

	logger2, err := New(config2)
	if err != nil {
		t.Fatalf("failed to create logger2: %v", err)
	}

	const numGoroutines = 50
	const messagesPerGoroutine = 20

	var wg sync.WaitGroup
	var swapCount int64

	// Start goroutines that will swap between loggers and log messages
	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < messagesPerGoroutine; j++ {
				// Randomly swap between loggers
				if id%2 == 0 {
					err := SetDefault(logger1)
					if err != nil {
						t.Errorf("SetDefault() failed in goroutine %d: %v", id, err)
						return
					}
					atomic.AddInt64(&swapCount, 1)
				} else {
					err := SetDefault(logger2)
					if err != nil {
						t.Errorf("SetDefault() failed in goroutine %d: %v", id, err)
						return
					}
					atomic.AddInt64(&swapCount, 1)
				}

				// Log a message with the current default
				Info("message from goroutine", "goroutine_id", id, "message_id", j)

				// Verify we can always get a valid logger
				current := Default()
				if current == nil {
					t.Errorf("Default() returned nil in goroutine %d", id)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify we performed swaps
	if atomic.LoadInt64(&swapCount) == 0 {
		t.Error("expected some logger swaps to occur")
	}

	// Verify we got some output in at least one buffer
	totalOutput := len(buf1.String()) + len(buf2.String())
	if totalOutput == 0 {
		t.Error("expected some log output from concurrent operations")
	}

	t.Logf("Total swaps: %d, Output lengths: buf1=%d, buf2=%d",
		atomic.LoadInt64(&swapCount), len(buf1.String()), len(buf2.String()))
}

// TestFileOutput tests logging to actual files.
func TestFileOutput(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config, closer, err := NewConfig(&slog.HandlerOptions{Level: slog.LevelInfo}, "json", logFile)
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}
	defer func() {
		if closer != nil {
			if err := closer.Close(); err != nil {
				t.Errorf("failed to close file: %v", err)
			}
		}
	}()

	logger, err := New(&config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	testMessage := "test file output"
	logger.Info(testMessage, "key", "value")

	// Close the file to ensure data is flushed
	if closer != nil {
		err = closer.Close()
		if err != nil {
			t.Errorf("failed to close file: %v", err)
		}
		closer = nil
	}

	// Read the file and verify content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Errorf("log file content is not valid JSON: %v", err)
	}

	if entry["msg"] != testMessage {
		t.Errorf("expected message %q, got %q", testMessage, entry["msg"])
	}
}

// TestConcurrentLogging tests concurrent logging safety
func TestConcurrentLogging(t *testing.T) {
	buf := new(bytes.Buffer)
	cfg := &Config{Writer: buf, Options: &slog.HandlerOptions{Level: slog.LevelDebug}}
	l, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			l.Info(fmt.Sprintf("message %d", i))
		}(i)
	}
	wg.Wait()

	lines := strings.Count(buf.String(), "\n")
	if lines != 100 {
		t.Errorf("Expected 100 log lines, got %d", lines)
	}
}

// TestLogFormat tests different log formats
func TestLogFormat(t *testing.T) {
	tests := []struct {
		format   string
		validate func(t *testing.T, output string)
	}{
		{
			"text",
			func(t *testing.T, output string) {
				if !strings.Contains(output, "INFO") || !strings.Contains(output, "test") {
					t.Error("Text format validation failed")
				}
			},
		},
		{
			"json",
			func(t *testing.T, output string) {
				if !strings.Contains(output, `"level":"INFO"`) || !strings.Contains(output, `"msg":"test"`) {
					t.Error("JSON format validation failed")
				}
			},
		},
		{
			"pretty",
			func(t *testing.T, output string) {
				if !strings.Contains(output, "INFO") || !strings.Contains(output, "test") {
					t.Error("Pretty format validation failed")
				}
			},
		},
		{
			"unknown",
			func(t *testing.T, output string) {
				if !strings.Contains(output, "INFO") || !strings.Contains(output, "test") {
					t.Error("Fallback format validation failed")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cfg := Config{Writer: buf, Format: tt.format, Options: &slog.HandlerOptions{Level: slog.LevelInfo}}
			l, err := New(&cfg)
			if err != nil {
				t.Fatal(err)
			}

			l.Info("test")
			tt.validate(t, buf.String())
		})
	}
}

// TestLogLevelFiltering tests log level filtering
func TestLogLevelFiltering(t *testing.T) {
	buf := new(bytes.Buffer)
	cfg := &Config{Writer: buf, Options: &slog.HandlerOptions{Level: slog.LevelWarn}}
	l, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	l.Info("should not appear")
	l.Warn("should appear")

	output := buf.String()
	if strings.Contains(output, "should not appear") {
		t.Error("Info message should be filtered")
	}
	if !strings.Contains(output, "should appear") {
		t.Error("Warn message should appear")
	}
}

// Benchmark tests

// BenchmarkLoggerCreation benchmarks the creation of a new logger.
func BenchmarkLoggerCreation(b *testing.B) {
	config := &Config{
		Format:  "json",
		Writer:  io.Discard,
		Options: &slog.HandlerOptions{Level: slog.LevelInfo},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger, err := New(config)
		if err != nil {
			b.Fatalf("failed to create logger: %v", err)
		}
		_ = logger
	}
}

// BenchmarkGlobalLogging benchmarks the global logging functions.
func BenchmarkGlobalLogging(b *testing.B) {
	originalLogger := defaultLogger.Load()
	defer defaultLogger.Store(originalLogger)

	config := &Config{
		Format:  "json",
		Writer:  io.Discard,
		Options: &slog.HandlerOptions{Level: slog.LevelInfo},
	}

	err := NewDefault(config)
	if err != nil {
		b.Fatalf("failed to set default logger: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message", "iteration", i, "key", "value")
	}
}

// BenchmarkConcurrentGlobalLogging benchmarks the concurrent global logging functions.
func BenchmarkConcurrentGlobalLogging(b *testing.B) {
	originalLogger := defaultLogger.Load()
	defer defaultLogger.Store(originalLogger)

	config := &Config{
		Format:  "json",
		Writer:  io.Discard,
		Options: &slog.HandlerOptions{Level: slog.LevelInfo},
	}

	err := NewDefault(config)
	if err != nil {
		b.Fatalf("failed to set default logger: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			Info("concurrent benchmark message", "iteration", i, "key", "value")
			i++
		}
	})
}

// BenchmarkAtomicLoggerAccess benchmarks access to the global logger
func BenchmarkAtomicLoggerAccess(b *testing.B) {
	originalLogger := defaultLogger.Load()
	defer defaultLogger.Store(originalLogger)

	config := &Config{
		Format:  "json",
		Writer:  io.Discard,
		Options: &slog.HandlerOptions{Level: slog.LevelInfo},
	}

	logger, err := New(config)
	if err != nil {
		b.Fatalf("failed to create logger: %v", err)
	}

	err = SetDefault(logger)
	if err != nil {
		b.Fatalf("failed to set default logger: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Measure the cost of atomic access
			current := Default()
			_ = current
		}
	})
}
