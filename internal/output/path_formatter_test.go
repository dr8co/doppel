package output

import (
	"bytes"
	"testing"

	"github.com/dr8co/doppel/internal/model"
)

func TestWritePaths(t *testing.T) {
	report := &model.DuplicateReport{
		Groups: []model.DuplicateGroup{
			{Files: []string{"path with spaces", "quoted\"path"}},
			{Files: []string{"second"}},
		},
	}

	tests := []struct {
		name          string
		nullDelimited bool
		want          string
	}{
		{name: "newline", want: "path with spaces\nquoted\"path\nsecond\n"},
		{name: "nul", nullDelimited: true, want: "path with spaces\x00quoted\"path\x00second\x00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WritePaths(report, &buf, tt.nullDelimited); err != nil {
				t.Fatalf("WritePaths() error = %v", err)
			}
			if buf.String() != tt.want {
				t.Fatalf("WritePaths() = %q, want %q", buf.String(), tt.want)
			}
		})
	}
}

func TestWritePathsEmptyReport(t *testing.T) {
	var buf bytes.Buffer
	if err := WritePaths(&model.DuplicateReport{}, &buf, true); err != nil {
		t.Fatalf("WritePaths() error = %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("output = %q, want empty output", buf.String())
	}
}

func TestSortReportDeterministicOrder(t *testing.T) {
	report := &model.DuplicateReport{
		Groups: []model.DuplicateGroup{
			{ID: 2, Size: 200, WastedSpace: 50, Count: 2, Files: []string{"/tmp/z.txt", "/tmp/a.txt"}},
			{ID: 1, Size: 100, WastedSpace: 10, Count: 3, Files: []string{"/tmp/m.txt", "/tmp/b.txt"}},
		},
	}

	SortReport(report, "path", false)
	if report.Groups[0].ID != 2 || report.Groups[1].ID != 1 {
		t.Fatalf("path sort mismatch: got group IDs %d,%d want 2,1", report.Groups[0].ID, report.Groups[1].ID)
	}
	if report.Groups[0].Files[0] != "/tmp/a.txt" || report.Groups[0].Files[1] != "/tmp/z.txt" {
		t.Fatalf("path sort in group: got %v want [/tmp/a.txt /tmp/z.txt]", report.Groups[0].Files)
	}

	SortReport(report, "size", true)
	if report.Groups[0].ID != 2 || report.Groups[1].ID != 1 {
		t.Fatalf("size reverse sort mismatch: got group IDs %d,%d want 2,1", report.Groups[0].ID, report.Groups[1].ID)
	}
	if err := SortReport(report, "bogus", false); err == nil {
		t.Fatal("SortReport() error = nil, want invalid sort error")
	}
}
