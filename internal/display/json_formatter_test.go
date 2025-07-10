package display

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/dr8co/doppel/internal/stats"
)

func TestJSONFormatter_Format(t *testing.T) {
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
			StartTime:       time.Date(2025, 7, 10, 12, 0, 0, 0, time.UTC),
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
	formatter := NewJSONFormatter()
	err := formatter.Format(report, &buf)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Validate output is valid JSON and matches the expected structure
	var decoded DuplicateReport
	err = json.Unmarshal(buf.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if decoded.TotalWastedSpace != report.TotalWastedSpace {
		t.Errorf("TotalWastedSpace mismatch: got %d, want %d", decoded.TotalWastedSpace, report.TotalWastedSpace)
	}
	if len(decoded.Groups) != len(report.Groups) {
		t.Errorf("Groups length mismatch: got %d, want %d", len(decoded.Groups), len(report.Groups))
	}
	if decoded.Stats.TotalFiles != report.Stats.TotalFiles {
		t.Errorf("Stats.TotalFiles mismatch: got %d, want %d", decoded.Stats.TotalFiles, report.Stats.TotalFiles)
	}
}
