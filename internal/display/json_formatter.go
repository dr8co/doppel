package display

import (
	"encoding/json"
	"io"
)

// JSONFormatter formats duplicate reports as JSON
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// Format writes the duplicate report as formatted JSON to the writer
func (f *JSONFormatter) Format(report *DuplicateReport, w io.Writer) error {
	encoder := json.NewEncoder(w)
	// Pretty print with 2-space indentation
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
