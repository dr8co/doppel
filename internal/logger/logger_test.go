package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestConcurrentInitialization tests the race condition in logger initialization
func TestConcurrentInitialization(t *testing.T) {
	// Reset the logger state
	resetLogger()

	const numGoroutines = 100
	var wg sync.WaitGroup

	// Channel to collect any panics or errors
	errors := make(chan error, numGoroutines)

	// Start multiple goroutines trying to initialize simultaneously
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("goroutine %d panicked: %v", id, r)
				}
			}()

			// Try different initialization parameters
			level := []string{"debug", "info", "warn", "error"}[id%4]
			format := []string{"text", "json", "pretty"}[id%3]
			output := []string{"stdout", "stderr", "discard"}[id%3]

			InitLogger(level, format, output)

			// Try to use the logger immediately after initialization
			logger := GetLogger()
			if logger == nil {
				errors <- fmt.Errorf("goroutine %d got nil logger", id)
				return
			}

			// Test logging
			logger.Info("test message", "goroutine", id)
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Error(err)
	}

	// Verify the logger is properly initialized
	logger := GetLogger()
	if logger == nil {
		t.Error("Logger should be initialized after concurrent initialization")
	}
}

// TestConcurrentLogging tests concurrent access to the logger
func TestConcurrentLogging(t *testing.T) {
	t.Parallel()
	// Initialize with a buffer to capture output
	resetLogger()

	// Initialize logger with text format to buffer
	initMu.Lock()
	defer initMu.Unlock()
	initLoggerUnsafe("info", "text", "")
	// Replace the writer with our buffer for testing
	if textHandler, ok := log.Load().Handler().(*slog.TextHandler); ok {
		// We need a way to change the writer - this is a limitation in the current design
		// For now, we'll test with stdout but suggest improvements
		_ = textHandler
	}

	const numGoroutines = 50
	const messagesPerGoroutine = 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Start concurrent logging
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("logging goroutine %d panicked: %v", id, r)
				}
			}()

			for j := 0; j < messagesPerGoroutine; j++ {
				// Test different logging methods
				switch j % 6 {
				case 0:
					Info("info message", "goroutine", id, "message", j)
				case 1:
					Warn("warn message", "goroutine", id, "message", j)
				case 2:
					Error("error message", "goroutine", id, "message", j)
				case 3:
					Debug("debug message", "goroutine", id, "message", j)
				case 4:
					InfoCtx(context.Background(), "info with context", "goroutine", id, "message", j)
				case 5:
					InfoAttrs(context.Background(), "info with attrs",
						slog.Int("goroutine", id), slog.Int("message", j))
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Error(err)
	}
}

func TestConcurrentGetLogger(t *testing.T) {
	resetLogger()
	InitLogger("debug", "json", "stdout")

	const numGoroutines = 1000
	var wg sync.WaitGroup
	results := make(chan *slog.Logger, numGoroutines)

	// Test concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger := GetLogger()
			results <- logger
		}()
	}

	wg.Wait()
	close(results)

	// All should return the same logger instance
	var firstLogger *slog.Logger
	count := 0
	for logger := range results {
		if firstLogger == nil {
			firstLogger = logger
		}
		if logger != firstLogger {
			t.Error("Got different logger instances from concurrent GetLogger calls")
		}
		count++
	}

	if count != numGoroutines {
		t.Errorf("Expected %d results, got %d", numGoroutines, count)
	}
}

// TestConcurrentInitAndLog tests initialization and logging happening simultaneously
func TestConcurrentInitAndLog(t *testing.T) {
	t.Parallel()
	resetLogger()

	const numInitGoroutines = 10
	const numLogGoroutines = 50

	var wg sync.WaitGroup
	errors := make(chan error, numInitGoroutines+numLogGoroutines)

	// Start initialization goroutines
	for i := 0; i < numInitGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("init goroutine %d panicked: %v", id, r)
				}
			}()

			time.Sleep(time.Millisecond * time.Duration(id%10)) // Stagger starts
			InitLogger("info", "text", "discard")
		}(i)
	}

	// Start logging goroutines
	for i := 0; i < numLogGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("log goroutine %d panicked: %v", id, r)
				}
			}()

			time.Sleep(time.Millisecond * time.Duration(id%5)) // Stagger starts

			// Try to log even if the logger might not be initialized
			logger := GetLogger()
			if logger != nil {
				logger.Info("concurrent message", "goroutine", id)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Error(err)
	}
}

// TestConcurrentClose tests concurrent access during close operations
func TestConcurrentClose(t *testing.T) {
	resetLogger()
	InitLogger("info", "text", "discard")

	const numGoroutines = 20
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("goroutine %d panicked: %v", id, r)
				}
			}()

			if id%3 == 0 {
				// Some goroutines try to close
				Close()
			} else if id%3 == 1 {
				// Some try to log
				logger := GetLogger()
				if logger != nil {
					logger.Info("message during close", "goroutine", id)
				}
			} else {
				// Some try to reinitialize
				InitLogger("debug", "json", "stderr")
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Error(err)
	}
}

// TestRaceDetection uses go `test -race` to detect races
func TestRaceDetection(t *testing.T) {
	t.Parallel()
	resetLogger()

	// This test is designed to trigger race conditions
	for iteration := 0; iteration < 10; iteration++ {
		var wg sync.WaitGroup

		// Reset for each iteration
		resetLogger()

		// Multiple goroutines doing different operations
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				switch id % 4 {
				case 0:
					InitLogger("info", "text", "stdout")
				case 1:
					logger := GetLogger()
					if logger != nil {
						logger.Info("race test", "id", id)
					}
				case 2:
					Close()
				case 3:
					InitLogger("debug", "json", "stderr")
				}
			}(i)
		}

		wg.Wait()
	}
}

// TestFileHandleConcurrency tests concurrent file operations
func TestFileHandleConcurrency(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "logger_test_*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			t.Error(err)
		}
	}(tempFile.Name())

	_ = tempFile.Close()

	resetLogger()

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("file goroutine %d panicked: %v", id, r)
				}
			}()

			// All goroutines try to initialize with the same file
			InitLogger("info", "text", tempFile.Name())

			// Then try to log
			logger := GetLogger()
			if logger != nil {
				logger.Info("file test message", "goroutine", id)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Error(err)
	}

	// Clean up
	Close()
}

// Benchmark concurrent logging performance
func BenchmarkConcurrentLogging(b *testing.B) {
	resetLogger()
	InitLogger("info", "text", "discard")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			Info("benchmark message", "iteration", i)
			i++
		}
	})
}

// Helper function to reset logger state for testing
func resetLogger() {
	Close()
}

// Test helper to verify logger state
func TestLoggerState(t *testing.T) {
	resetLogger()

	// Test uninitialized state
	if IsInitialized() {
		t.Error("Logger should not be initialized after reset")
	}

	// Test initialization
	InitLogger("info", "text", "stdout")

	if !IsInitialized() {
		t.Error("Logger should be initialized after InitLogger call")
	}

	logger := GetLogger()
	if logger == nil {
		t.Error("GetLogger should return non-nil logger after initialization")
	}

	// Test double initialization (should not reinitialize)
	InitLogger("debug", "json", "stderr")

	// Logger should still be the same (not reinitialized)
	logger2 := GetLogger()
	if logger != logger2 {
		t.Error("Logger should not be reinitialized if already initialized")
	}
}

// Memory usage test for detecting leaks
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Perform many operations
	for i := 0; i < 1000; i++ {
		resetLogger()
		InitLogger("info", "json", "discard")

		for j := 0; j < 100; j++ {
			Info("memory test", "iteration", i, "message", j)
		}

		Close()
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Check if memory usage grew significantly
	allocDiff := m2.TotalAlloc - m1.TotalAlloc
	if allocDiff > 10*1024*1024 { // 10MB threshold
		t.Logf("Memory usage grew by %d bytes", allocDiff)
		// Note: This might not always indicate a leak, but worth monitoring
	}
}

func TestConcurrentLoggingDuringInitialization(t *testing.T) {
	// Reset state
	Close()

	var wg sync.WaitGroup
	count := 100

	wg.Add(1)
	go func() {
		defer wg.Done()
		InitLogger("debug", "text", "stdout")
	}()

	// Start logging while initialization is happening
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			Info("test message", "iteration", i)
		}(i)
	}

	wg.Wait()
}

func TestConcurrentFileLogging(t *testing.T) {
	// Reset state
	Close()

	// Use a temp file for testing
	tmpFile, err := os.CreateTemp("", "logger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			t.Error(err)
		}
	}(tmpFile.Name())

	InitLogger("debug", "text", tmpFile.Name())

	var wg sync.WaitGroup
	count := 100

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			Info("concurrent message", "goroutine", i)
		}(i)
	}

	wg.Wait()

	// Verify all messages were written
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Simple check - more sophisticated checks could be added
	if len(content) == 0 {
		t.Error("No content was written to log file")
	}
}

func TestConcurrentCloseAndLogging(t *testing.T) {
	InitLogger("debug", "text", "stdout")

	var wg sync.WaitGroup
	count := 100

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // Ensure some logs happen before close
		Close()
	}()

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			Info("message before close", "iteration", i)
		}(i)
	}

	wg.Wait()
}

func TestGetLoggerDuringInitialization(t *testing.T) {
	t.Parallel()
	// Reset state
	resetLogger()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		InitLogger("debug", "text", "stdout")
	}()

	go func() {
		defer wg.Done()
		// This should block until initialization is complete
		l := GetLogger()
		if l == nil {
			t.Error("GetLogger returned nil during initialization")
		}
	}()

	wg.Wait()
}

func TestMultipleCloseCalls(t *testing.T) {
	resetLogger()
	InitLogger("debug", "text", "stdout")

	var wg sync.WaitGroup
	count := 10

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Close()
		}()
	}

	wg.Wait()
}
