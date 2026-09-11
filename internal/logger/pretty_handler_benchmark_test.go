package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"
)

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
