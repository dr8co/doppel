package logger

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestConcurrentHandle tests concurrent calls to Handle method.
func TestConcurrentHandle(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	const numGoroutines = 100
	const numLogsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines that log concurrently
	for i := range numGoroutines {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numLogsPerGoroutine; j++ {
				record := slog.NewRecord(time.Now(), slog.LevelInfo,
					fmt.Sprintf("Message from goroutine %d, log %d", goroutineID, j), 0)
				record.AddAttrs(slog.String("goroutine", fmt.Sprintf("%d", goroutineID)))

				err := handler.Handle(context.Background(), record)
				if err != nil {
					t.Errorf("Handle failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify we got all expected log entries
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// We should have numGoroutines * numLogsPerGoroutine * 3 lines
	// (each log entry spans 3 lines due to attribute formatting)
	expectedMinLines := numGoroutines * numLogsPerGoroutine
	if len(lines) < expectedMinLines {
		t.Errorf("Expected at least %d lines, got %d", expectedMinLines, len(lines))
	}

	// Check for garbled output (lines mixing)
	for i, line := range lines {
		if strings.Count(line, "INFO") > 1 {
			t.Errorf("Line %d appears to contain mixed output: %q", i, line)
		}
	}
}

// TestConcurrentWithAttrs tests concurrent calls to WithAttrs.
func TestConcurrentWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := NewPrettyHandler(&buf, nil)

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	handlers := make(chan slog.Handler, numGoroutines)

	// Create handlers with attributes concurrently
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			attrs := []slog.Attr{
				slog.String("handler_id", fmt.Sprintf("handler_%d", id)),
				slog.Int("value", id),
			}
			handler := baseHandler.WithAttrs(attrs)
			handlers <- handler
		}(i)
	}

	wg.Wait()
	close(handlers)

	// Use each handler to log something
	wg.Add(numGoroutines)
	for handler := range handlers {
		go func(h slog.Handler) {
			defer wg.Done()
			record := slog.NewRecord(time.Now(), slog.LevelInfo, "Test message", 0)
			err := h.Handle(context.Background(), record)
			if err != nil {
				t.Errorf("Handle failed: %v", err)
			}
		}(handler)
	}

	wg.Wait()
}

// TestConcurrentWithGroup tests concurrent calls to WithGroup.
func TestConcurrentWithGroup(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := NewPrettyHandler(&buf, nil)

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	handlers := make(chan slog.Handler, numGoroutines)

	// Create grouped handlers concurrently
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			handler := baseHandler.WithGroup(fmt.Sprintf("group_%d", id))
			handlers <- handler
		}(i)
	}

	wg.Wait()
	close(handlers)

	// Use each handler
	wg.Add(numGoroutines)
	for handler := range handlers {
		go func(h slog.Handler) {
			defer wg.Done()
			record := slog.NewRecord(time.Now(), slog.LevelInfo, "Group test", 0)
			err := h.Handle(context.Background(), record)
			if err != nil {
				t.Errorf("Handle failed: %v", err)
			}
		}(handler)
	}

	wg.Wait()
}

// TestSharedBufferConcurrency tests the shared [strings.Builder] issue.
func TestSharedBufferConcurrency(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, nil)

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// This test specifically targets the strings.Builder reuse issue
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			record := slog.NewRecord(time.Now(), slog.LevelInfo,
				fmt.Sprintf("Unique message %d", id), 0)
			record.AddAttrs(
				slog.String("unique_key", fmt.Sprintf("unique_value_%d", id)),
				slog.Int("id", id),
			)

			err := handler.Handle(context.Background(), record)
			if err != nil {
				t.Errorf("Handle failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	output := buf.String()
	re := regexp.MustCompile(`Unique message \d+`)
	matches := re.FindAllString(output, -1)

	found := make(map[string]int, numGoroutines)
	for _, match := range matches {
		found[match]++
	}

	// Check that each unique message appears exactly once
	for i := range numGoroutines {
		expected := fmt.Sprintf("Unique message %d", i)
		if count := found[expected]; count != 1 {
			t.Errorf("Expected message %q to appear exactly once, found %d times", expected, count)
		}
	}
}

// TestRaceDetectorStress is a stress test designed to trigger race conditions.
func TestRaceDetectorStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, &slog.HandlerOptions{
		AddSource: true, // This exercises the getFrame function
	})

	const duration = 2 * time.Second
	const numGoroutines = 20

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wg sync.WaitGroup

	// Goroutines that continuously log
	for i := range numGoroutines / 2 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			counter := 0
			for {
				select {
				case <-ctx.Done():
					return
				default:
					record := slog.NewRecord(time.Now(), slog.LevelInfo,
						fmt.Sprintf("Stress test %d:%d", id, counter), 0)
					record.AddAttrs(slog.Int("counter", counter))

					err := handler.Handle(context.Background(), record)
					if err != nil {
						t.Errorf("Handle failed: %v", err)
						return
					}
					counter++
					runtime.Gosched() // Yield to increase the chance of races
				}
			}
		}(i)
	}

	// Goroutines that continuously create new handlers with attributes
	for i := numGoroutines / 2; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			counter := 0
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Create the handler with attributes
					attrs := []slog.Attr{
						slog.String("creator", fmt.Sprintf("goroutine_%d", id)),
						slog.Int("iteration", counter),
					}
					newHandler := handler.WithAttrs(attrs)

					// Use it immediately
					record := slog.NewRecord(time.Now(), slog.LevelWarn,
						fmt.Sprintf("New handler test %d:%d", id, counter), 0)
					err := newHandler.Handle(context.Background(), record)
					if err != nil {
						t.Errorf("Handle failed: %v", err)
						return
					}
					counter++
					runtime.Gosched()
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestColorOutputConcurrency tests concurrent access to color package globals.
func TestColorOutputConcurrency(t *testing.T) {
	// Create multiple handlers that might modify the color package state
	const numHandlers = 10
	var wg sync.WaitGroup
	wg.Add(numHandlers)

	for i := range numHandlers {
		go func(id int) {
			defer wg.Done()
			var buf bytes.Buffer
			handler := NewPrettyHandler(&buf, nil)

			// Log something to trigger color usage
			record := slog.NewRecord(time.Now(), slog.LevelError,
				fmt.Sprintf("Color test %d", id), 0)
			err := handler.Handle(context.Background(), record)
			if err != nil {
				t.Errorf("Handle failed: %v", err)
			}
		}(i)
	}

	wg.Wait()
}

// Benchmark to measure the performance impact of concurrency.
func BenchmarkConcurrentPrettyLogging(b *testing.B) {
	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, nil)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			record := slog.NewRecord(time.Now(), slog.LevelInfo, "Benchmark message", 0)
			record.AddAttrs(slog.String("key", "value"))
			err := handler.Handle(context.Background(), record)
			if err != nil {
				b.Errorf("Handle failed: %v", err)
			}
		}
	})
}

// Helper function to create a logger with our handler for integration testing
func createTestLogger() *slog.Logger {
	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
	return slog.New(handler)
}

// TestIntegrationWithSlog tests integration with the slog package.
func TestIntegrationWithSlog(t *testing.T) {
	logger := createTestLogger()

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()

			// Use various slog methods
			logger.Debug("Debug message", "id", id)
			logger.Info("Info message", "id", id)
			logger.Warn("Warning message", "id", id)
			logger.Error("Error message", "id", id)

			// Test with context
			type ctxKey string
			ctx := context.WithValue(context.Background(), ctxKey("test"), "value")
			logger.InfoContext(ctx, "Context message", "id", id)

			// Test with groups
			grouped := logger.WithGroup(fmt.Sprintf("group_%d", id))
			grouped.Info("Grouped message", "data", "test")
		}(i)
	}

	wg.Wait()
}
