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
