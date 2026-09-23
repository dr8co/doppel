package output

import (
	"encoding/json"
	"encoding/json/jsontext"
	json2 "encoding/json/v2"
	"io"

	"github.com/dr8co/doppel/internal/model"
)

// JSONLFormatter formats each duplicate group as one JSON object per line.
type JSONLFormatter struct{}

// NewJSONLFormatter creates a new JSONL formatter.
func NewJSONLFormatter() *JSONLFormatter {
	return &JSONLFormatter{}
}

// Format writes one JSON object for each duplicate group.
func (f *JSONLFormatter) Format(report *model.DuplicateReport, w io.Writer) error {
	encoder := jsontext.NewEncoder(w, jsontext.Multiline(false), json.FormatDurationAsNano(true))
	for _, group := range report.Groups {
		if err := json2.MarshalEncode(encoder, group); err != nil {
			return err
		}
	}
	return nil
}

// Name returns the name of the formatter.
func (f *JSONLFormatter) Name() string {
	return "jsonl"
}
