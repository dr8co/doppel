package display

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dr8co/doppel/internal/stats"
)

func TestConvertToReport(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "convert_report_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tempDir)

	// Create files of different sizes
	fileA := filepath.Join(tempDir, "a.txt")
	fileB := filepath.Join(tempDir, "b.txt")
	fileC := filepath.Join(tempDir, "c.txt")
	fileD := filepath.Join(tempDir, "d.txt")

	content1 := []byte("1234567890")       // 10 bytes
	content2 := []byte("abcdefghij123456") // 16 bytes

	if err := os.WriteFile(fileA, content1, 0644); err != nil {
		t.Fatalf("Failed to write fileA: %v", err)
	}
	if err := os.WriteFile(fileB, content1, 0644); err != nil {
		t.Fatalf("Failed to write fileB: %v", err)
	}
	if err := os.WriteFile(fileC, content2, 0644); err != nil {
		t.Fatalf("Failed to write fileC: %v", err)
	}
	if err := os.WriteFile(fileD, content2, 0644); err != nil {
		t.Fatalf("Failed to write fileD: %v", err)
	}

	duplicates := map[string][]string{
		"hash1": {fileA, fileB},
		"hash2": {fileC, fileD},
	}

	s := &stats.Stats{
		TotalFiles:      4,
		ProcessedFiles:  4,
		DuplicateGroups: 2,
		DuplicateFiles:  4,
		StartTime:       time.Now().Add(-2 * time.Second),
	}

	report := ConvertToReport(duplicates, s)
	if report == nil {
		t.Fatal("ConvertToReport returned nil")
	}

	if report.Stats != s {
		t.Errorf("Expected stats pointer to be preserved")
	}

	if len(report.Groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(report.Groups))
	}

	// Check group details
	for _, group := range report.Groups {
		switch group.Count {
		case 2:
			if len(group.Files) != 2 {
				t.Errorf("Expected 2 files in group, got %d", len(group.Files))
			}
			if group.Size != 10 && group.Size != 16 {
				t.Errorf("Unexpected group size: %d", group.Size)
			}
			expectedWasted := uint64(group.Size)
			if group.WastedSpace != expectedWasted {
				t.Errorf("Expected wasted space %d, got %d", expectedWasted, group.WastedSpace)
			}
		default:
			t.Errorf("Unexpected group count: %d", group.Count)
		}
	}

	expectedTotalWasted := uint64(10 + 16)
	if report.TotalWastedSpace != expectedTotalWasted {
		t.Errorf("Expected total wasted space %d, got %d", expectedTotalWasted, report.TotalWastedSpace)
	}

	// Test with empty duplicates
	emptyReport := ConvertToReport(map[string][]string{}, s)
	if emptyReport == nil {
		t.Fatal("ConvertToReport returned nil for empty input")
	}
	if len(emptyReport.Groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(emptyReport.Groups))
	}
	if emptyReport.TotalWastedSpace != 0 {
		t.Errorf("Expected 0 wasted space for empty input, got %d", emptyReport.TotalWastedSpace)
	}

	// Test with a group with only 1 file (should be skipped)
	oneFile := map[string][]string{
		"hash3": {fileA},
	}
	reportOne := ConvertToReport(oneFile, s)
	if len(reportOne.Groups) != 0 {
		t.Errorf("Expected 0 groups for single-file group, got %d", len(reportOne.Groups))
	}
}
