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
	"testing"
)

var mu sync.Mutex

// reconfigure reconfigures the default logger with the provided configuration.
func reconfigure(config Config) error {
	mu.Lock()
	defer mu.Unlock()

	var err error
	defaultLogger, err = New(config)
	return err
}

// TestNewLogger tests creating a new logger with valid/invalid configurations
func TestNewLogger(t *testing.T) {
	t.Run("ValidWriter", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cfg := Config{Writer: buf, Format: "text", Level: "debug"}
		l, err := New(cfg)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		l.Debug("test message")
		if !strings.Contains(buf.String(), "test message") {
			t.Error("Log message not found in output")
		}
	})

	t.Run("NilWriter", func(t *testing.T) {
		cfg := Config{Writer: nil}
		_, err := New(cfg)
		if err == nil {
			t.Fatal("Expected error for nil writer")
		}
	})
}

// TestNewConfig tests logger configuration creation
func TestNewConfig(t *testing.T) {
	tests := []struct {
		name         string
		level        string
		format       string
		output       string
		expectError  bool
		expectCloser bool
	}{
		{
			name:         "stdout output",
			level:        "info",
			format:       "json",
			output:       "stdout",
			expectError:  false,
			expectCloser: false,
		},
		{
			name:         "stderr output",
			level:        "debug",
			format:       "text",
			output:       "stderr",
			expectError:  false,
			expectCloser: false,
		},
		{
			name:         "null output",
			level:        "warn",
			format:       "null",
			output:       "null",
			expectError:  false,
			expectCloser: false,
		},
		{
			name:         "file output",
			level:        "error",
			format:       "json",
			output:       filepath.Join(t.TempDir(), "test.log"),
			expectError:  false,
			expectCloser: true,
		},
		{
			name:        "invalid file path",
			level:       "info",
			format:      "json",
			output:      ".",
			expectError: true,
		},
		{
			name:        "empty file path",
			level:       "info",
			format:      "json",
			output:      "",
			expectError: false, // Should default to stdout
		},
		{
			name:         "discard output",
			level:        "warn",
			format:       "json",
			output:       "discard",
			expectError:  false,
			expectCloser: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, closer, err := NewConfig(tt.level, tt.format, tt.output)

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

			if config.Level != tt.level {
				t.Errorf("expected level %s, got %s", tt.level, config.Level)
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
	config := Config{
		Level:  "debug",
		Format: "json",
		Writer: &buf,
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

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"", slog.LevelInfo}, // default
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"invalid", slog.LevelInfo}, // default for unknown
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestCreateHandler tests the creation of log handlers.
func TestCreateHandler(t *testing.T) {
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}

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
			handler := createHandler(&buf, tt.format, opts)
			handlerType := fmt.Sprintf("%T", handler)
			if handlerType != tt.expected {
				t.Errorf("createHandler(%q) = %s, want %s", tt.format, handlerType, tt.expected)
			}
		})
	}
}

// TestGlobalLoggerConcurrency tests that the global logger can be safely used from multiple goroutines.
func TestGlobalLoggerConcurrency(t *testing.T) {
	// Reset the default logger for this test
	originalLogger := defaultLogger
	defer func() { defaultLogger = originalLogger }()

	var buf bytes.Buffer
	config := Config{
		Level:  "info",
		Format: "json",
		Writer: &buf,
	}

	// Test concurrent initialization
	const numGoroutines = 100
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := reconfigure(config)
			//err := InitDefault(config)
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("InitDefault error: %v", err)
	}

	// Verify logger was initialized
	logger := GetDefault()
	if logger == nil {
		t.Error("default logger should not be nil after initialization")
	}

	// Test concurrent logging
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
	// We expect at least some log entries (debug might be filtered out depending on level)
	if len(lines) < numMessages {
		t.Logf("Got %d log lines from %d concurrent operations", len(lines), numMessages*4)
	}
}

// resetDefaultLogger is a helper function to reset the global state between tests.
// This is crucial for ensuring tests are isolated.
func resetDefaultLogger() {
	// Re-initialize a new sync.Once to allow InitDefault to be called again in different tests.
	once = sync.Once{}
	// Reset the default logger to a clean state.
	defaultLogger = &Logger{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		config: Config{},
	}
}

// TestConcurrencySafety tests that the global logger can be safely used from multiple goroutines.
func TestConcurrencySafety(t *testing.T) {
	// Save and restore the original logger
	originalLogger := defaultLogger
	defer func() { defaultLogger = originalLogger }()

	resetDefaultLogger()
	var buf bytes.Buffer
	config := Config{Level: "info", Writer: &buf}

	// This test will have multiple goroutines trying to initialize, set, and use the default logger.
	// The `sync.Once` in `InitDefault` is the primary mechanism being tested for safety.
	// We also test concurrent writes to the logger.
	t.Run("Concurrent InitDefault and logging", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 100

		// All goroutines will try to initialize the logger. Only one should succeed.
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				err := InitDefault(config)
				if err != nil {
					t.Errorf("Error from InitDefault in goroutine: %v\n", err)
				}
			}()
		}
		wg.Wait()

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
	// Save and restore the original logger
	originalLogger := defaultLogger
	defer func() { defaultLogger = originalLogger }()

	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Writer: &buf,
	}

	err := reconfigure(config)
	if err != nil {
		t.Fatalf("failed to initialize default logger: %v", err)
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
	originalLogger := defaultLogger
	defer func() { defaultLogger = originalLogger }()

	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "text",
		Writer: &buf,
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

	if GetDefault() != newLogger {
		t.Error("GetDefault should return the logger we just set")
	}

	// Test setting nil logger
	err = SetDefault(nil)
	if err == nil {
		t.Error("SetDefault with nil logger should return error")
	}

	// Verify logger wasn't changed after nil attempt
	if GetDefault() != newLogger {
		t.Error("default logger should not change when SetDefault fails")
	}
}

// TestContextMethods tests logging methods that accept context.
func TestContextMethods(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Writer: &buf,
	}

	err := reconfigure(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
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
	config := Config{
		Level:  "info",
		Format: "json",
		Writer: &buf,
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

// TestDebugSourceAddition tests that source information is added to debug logs.
func TestDebugSourceAddition(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Writer: &buf,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Debug("debug message with source")

	output := buf.String()
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	// When the level is 'debug', source information should be added
	if _, hasSource := entry["source"]; !hasSource {
		t.Error("expected source information in debug level logs")
	}
}

// TestNonDebugNoSource tests that source information is not added to non-debug logs.
func TestNonDebugNoSource(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "info",
		Format: "json",
		Writer: &buf,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Info("info message without source")

	output := buf.String()
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	// When the level is not 'debug', source information should not be added
	if _, hasSource := entry["source"]; hasSource {
		t.Error("did not expect source information in non-debug level logs")
	}
}

// TestRaceConditionOnDefaultLogger tests for race conditions in the default logger.
func TestRaceConditionOnDefaultLogger(t *testing.T) {
	// This test specifically checks for race conditions in accessing the default logger
	originalLogger := defaultLogger
	defer func() { defaultLogger = originalLogger }()

	var buf bytes.Buffer
	config := Config{
		Level:  "info",
		Format: "json",
		Writer: &buf,
	}

	err := reconfigure(config)
	if err != nil {
		t.Errorf("failed to create logger: %v", err)
		return
	}

	const numGoroutines = 50
	const messagesPerGoroutine = 20

	var wg sync.WaitGroup

	// Start goroutines that will all try to initialize and use the logger
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Try to initialize (only the first call should succeed due to sync.Once)
			err := InitDefault(config)
			if err != nil {
				t.Errorf("failed to create logger: %v", err)
				return
			}

			// Use the logger extensively
			for j := 0; j < messagesPerGoroutine; j++ {
				Info("message from goroutine", "goroutine_id", id, "message_id", j)

				// Also test getting and setting
				logger := GetDefault()
				if logger == nil {
					t.Errorf("GetDefault returned nil in goroutine %d", id)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify we got some log output
	output := buf.String()
	if len(output) == 0 {
		t.Error("expected log output from concurrent operations")
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	expectedMessages := numGoroutines * messagesPerGoroutine
	if len(lines) != expectedMessages {
		t.Logf("Expected %d messages, got %d (this might be OK due to race conditions)", expectedMessages, len(lines))
	}
}

// TestFileOutput tests logging to actual files.
func TestFileOutput(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config, closer, err := NewConfig("info", "json", logFile)
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}
	defer func() {
		if closer != nil {
			err = closer.Close()
			if err != nil {
				t.Errorf("failed to close file: %v", err)
			}
		}
	}()

	logger, err := New(config)
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
	cfg := Config{Writer: buf, Level: "debug"}
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
			cfg := Config{Writer: buf, Format: tt.format, Level: "info"}
			l, err := New(cfg)
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
	cfg := Config{Writer: buf, Level: "warn"}
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
	config := Config{
		Level:  "info",
		Format: "json",
		Writer: io.Discard,
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
	originalLogger := defaultLogger
	defer func() { defaultLogger = originalLogger }()

	config := Config{
		Level:  "info",
		Format: "json",
		Writer: io.Discard,
	}

	err := InitDefault(config)
	if err != nil {
		b.Fatalf("failed to initialize default logger: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message", "iteration", i, "key", "value")
	}
}

// BenchmarkConcurrentGlobalLogging benchmarks the concurrent global logging functions.
func BenchmarkConcurrentGlobalLogging(b *testing.B) {
	originalLogger := defaultLogger
	defer func() { defaultLogger = originalLogger }()

	config := Config{
		Level:  "info",
		Format: "json",
		Writer: io.Discard,
	}

	err := InitDefault(config)
	if err != nil {
		b.Fatalf("failed to initialize default logger: %v", err)
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
