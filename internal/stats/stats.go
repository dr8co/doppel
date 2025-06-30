package stats

import (
	"fmt"
	"time"
)

// Stats tracks various statistics during the duplicate file finding process
type Stats struct {
	TotalFiles      int
	ProcessedFiles  int
	SkippedDirs     int
	SkippedFiles    int
	ErrorCount      int
	DuplicateGroups int
	DuplicateFiles  int
	StartTime       time.Time
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
