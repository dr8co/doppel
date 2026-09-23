package output

import (
	"encoding/json"
	"encoding/json/jsontext"
	json2 "encoding/json/v2"
	"io"

	"github.com/dr8co/doppel/internal/model"
)

// JSONFormatter formats duplicate reports as JSON.
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSON formatter.
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// Format writes the duplicate report as formatted JSON to the writer.
func (f *JSONFormatter) Format(report *model.DuplicateReport, w io.Writer) error {
	// Pretty print with 2-space indentation
	encoder := jsontext.NewEncoder(w, jsontext.WithIndent("  "), jsontext.Multiline(true), json.FormatDurationAsNano(true))
	return json2.MarshalEncode(encoder, report)
}

// Name returns the name of the formatter.
func (f *JSONFormatter) Name() string {
	return "json"
}
