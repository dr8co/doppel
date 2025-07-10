package stats

import (
	"fmt"
	"sync/atomic"
	"time"
)

// Stats tracks various statistics during the duplicate file finding process
type Stats struct {
	TotalFiles      uint64        `json:"total_files"`
	ProcessedFiles  uint64        `json:"processed_files"`
	SkippedDirs     uint64        `json:"skipped_dirs"`
	SkippedFiles    uint64        `json:"skipped_files"`
	ErrorCount      uint64        `json:"error_count"`
	DuplicateGroups uint64        `json:"duplicate_groups"`
	DuplicateFiles  uint64        `json:"duplicate_files"`
	StartTime       time.Time     `json:"start_time"`
	Duration        time.Duration `json:"duration"`
}

func (s *Stats) IncrementErrorCount() {
	atomic.AddUint64(&s.ErrorCount, 1)
}

func (s *Stats) IncrementProcessedFiles() {
	atomic.AddUint64(&s.ProcessedFiles, 1)
}

func (s *Stats) IncrementDuplicateGroups() {
	atomic.AddUint64(&s.DuplicateGroups, 1)
}

func (s *Stats) AddDuplicateFiles(count uint64) {
	atomic.AddUint64(&s.DuplicateFiles, count)
}

func (s *Stats) GetErrorCount() uint64 {
	return atomic.LoadUint64(&s.ErrorCount)
}

func (s *Stats) GetProcessedFiles() uint64 {
	return atomic.LoadUint64(&s.ProcessedFiles)
}

func (s *Stats) GetDuplicateGroups() uint64 {
	return atomic.LoadUint64(&s.DuplicateGroups)
}

func (s *Stats) GetDuplicateFiles() uint64 {
	return atomic.LoadUint64(&s.DuplicateFiles)
}

// FormatBytes converts a byte count to a human-readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
