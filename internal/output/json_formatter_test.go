package output

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/dr8co/doppel/internal/model"
)

// TestJSONFormatter_Format validates the JSON formatting of a DuplicateReport using the [JSONFormatter].
// It ensures the output is valid JSON and matches the original report structure and data.
func TestJSONFormatter_Format(t *testing.T) {
	report := &model.DuplicateReport{
		ScanDate: time.Now().UTC(),
		Stats: &model.Stats{
			TotalFiles:      10,
			ProcessedFiles:  8,
			SkippedDirs:     1,
			SkippedFiles:    1,
			ErrorCount:      0,
			DuplicateFiles:  4,
			DuplicateGroups: 2,
			StartTime:       time.Now().UTC(),
			Duration:        2 * time.Millisecond,
		},
		TotalWastedSpace: 2048,
		Groups: []model.DuplicateGroup{
			{
				ID:          1,
				Count:       2,
				Size:        1024,
				WastedSpace: 1024,
				Files:       []string{"/tmp/foo1.txt", "/tmp/foo2.txt"},
			},
			{
				ID:          2,
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
	var got model.DuplicateReport
	err = json.Unmarshal(buf.Bytes(), &got)
	if err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if !reflect.DeepEqual(got.Stats, report.Stats) {
		t.Errorf("Stats mismatch: got %v, want %v", got.Stats, report.Stats)
	}

	if !reflect.DeepEqual(got.Groups, report.Groups) {
		t.Errorf("Groups mismatch: got %v, want %v", got.Groups, report.Groups)
	}

	if got.TotalWastedSpace != report.TotalWastedSpace {
		t.Errorf("TotalWastedSpace mismatch: got %d, want %d", got.TotalWastedSpace, report.TotalWastedSpace)
	}

	if got.ScanDate != report.ScanDate {
		t.Errorf("ScanDate mismatch: got %v, want %v", got.ScanDate, report.ScanDate)
	}
}
