package output

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/dr8co/doppel/internal/model"
	"gopkg.in/yaml.v3"
)

func TestYAMLFormatter_Format(t *testing.T) {

	report := &model.DuplicateReport{
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
		ScanDate:         time.Now().UTC(),
		TotalWastedSpace: 1024,
		Groups: []model.DuplicateGroup{
			{
				Id:          1,
				Count:       2,
				Size:        512,
				WastedSpace: 256,
				Files:       []string{"file1.txt", "file2.txt"},
			},
			{
				Id:          2,
				Count:       2,
				Size:        512,
				WastedSpace: 256,
				Files:       []string{"file3.txt", "file4.txt"},
			},
		},
	}

	var buf bytes.Buffer
	formatter := NewYAMLFormatter()
	err := formatter.Format(report, &buf)
	if err != nil {
		t.Fatalf("failed to format report: %v", err)
	}

	var got model.DuplicateReport
	err = yaml.Unmarshal(buf.Bytes(), &got)
	if err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
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
