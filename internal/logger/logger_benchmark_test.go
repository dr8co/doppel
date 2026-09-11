package logger

import (
	"io"
	"log/slog"
	"testing"
)

// BenchmarkLoggerCreation benchmarks the creation of a new logger.
func BenchmarkLoggerCreation(b *testing.B) {
	config := &Config{
		Format:  "json",
		Writer:  io.Discard,
		Options: &slog.HandlerOptions{Level: slog.LevelInfo},
	}

	for b.Loop() {
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

	for i := 0; b.Loop(); i++ {
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

// BenchmarkAtomicLoggerAccess benchmarks access to the global logger.
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
