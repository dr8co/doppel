package output

import (
	"io"

	"github.com/dr8co/doppel/internal/model"
	"gopkg.in/yaml.v3"
)

// YAMLFormatter formats duplicate reports as YAML.
type YAMLFormatter struct{}

// NewYAMLFormatter creates a new YAML formatter.
func NewYAMLFormatter() *YAMLFormatter {
	return &YAMLFormatter{}
}

// Format writes the duplicate report as formatted YAML to the writer.
func (f *YAMLFormatter) Format(report *model.DuplicateReport, w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	defer func(encoder *yaml.Encoder) {
		_ = encoder.Close()
	}(encoder)

	return encoder.Encode(report)
}

// Name returns the name of the formatter.
func (f *YAMLFormatter) Name() string {
	return "yaml"
}
