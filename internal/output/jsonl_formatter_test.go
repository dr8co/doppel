package output

import (
	"bufio"
	"bytes"
	"encoding/json"
	json2 "encoding/json/v2"
	"reflect"
	"testing"

	"github.com/dr8co/doppel/internal/model"
)

func TestJSONLFormatterFormat(t *testing.T) {
	report := &model.DuplicateReport{
		Groups: []model.DuplicateGroup{
			{ID: 1, Count: 2, Size: 12, WastedSpace: 12, Files: []string{"one", "two"}},
			{ID: 2, Count: 2, Size: 24, WastedSpace: 24, Files: []string{"three", "four"}},
		},
	}

	var buf bytes.Buffer
	if err := NewJSONLFormatter().Format(report, &buf); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var got []model.DuplicateGroup
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		var group model.DuplicateGroup
		if err := json2.Unmarshal(scanner.Bytes(), &group, json.FormatDurationAsNano(true)); err != nil {
			t.Fatalf("output line is not valid JSON: %v", err)
		}
		got = append(got, group)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("reading output: %v", err)
	}

	if !reflect.DeepEqual(got, report.Groups) {
		t.Fatalf("groups = %#v, want %#v", got, report.Groups)
	}
}

func TestJSONLFormatterEmptyReport(t *testing.T) {
	var buf bytes.Buffer
	if err := NewJSONLFormatter().Format(&model.DuplicateReport{}, &buf); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("output = %q, want empty output", buf.String())
	}
}
