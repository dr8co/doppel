package display

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/dr8co/doppel/internal/stats"
)

func TestPrettyFormatter_Format(t *testing.T) {
	report := &DuplicateReport{
		ScanDate: time.Date(2025, 7, 10, 12, 0, 0, 0, time.UTC),
		Stats: &stats.Stats{
			TotalFiles:      10,
			ProcessedFiles:  8,
			SkippedDirs:     1,
			SkippedFiles:    1,
			ErrorCount:      0,
			DuplicateFiles:  4,
			DuplicateGroups: 2,
			Duration:        2 * time.Second,
		},
		TotalWastedSpace: 2048,
		Groups: []DuplicateGroup{
			{
				Id:          1,
				Count:       2,
				Size:        1024,
				WastedSpace: 1024,
				Files:       []string{"/tmp/foo1.txt", "/tmp/foo2.txt"},
			},
			{
				Id:          2,
				Count:       2,
				Size:        1024,
				WastedSpace: 1024,
				Files:       []string{"/tmp/bar1.txt", "/tmp/bar2.txt"},
			},
		},
	}

	var buf bytes.Buffer
	formatter := NewPrettyFormatter()
	err := formatter.Format(report, &buf)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	// Check for key phrases in the output
	checks := []string{
		"Duplicate group 1 (2 files):",
		"/tmp/foo1.txt",
		"/tmp/foo2.txt",
		"Duplicate group 2 (2 files):",
		"/tmp/bar1.txt",
		"/tmp/bar2.txt",
		"Summary:",
		"Duplicate files found: 4 (in 2 groups)",
		"Total wasted space:",
		"Detailed Statistics:",
		"Total files scanned: 10",
		"Files processed for hashing: 8",
		"Directories skipped: 1",
		"Files skipped: 1",
		"Files with errors: 0",
		"Processing time: 2s",
		"Processing rate: 4.0 files/second",
	}
	for _, phrase := range checks {
		if !strings.Contains(output, phrase) {
			t.Errorf("Output missing expected phrase: %q", phrase)
		}
	}
}
