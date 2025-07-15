package output

import (
	"fmt"
	"io"

	"github.com/dr8co/doppel/internal/model"
)

// Formatter formats duplicate reports to different output formats
type Formatter interface {
	Name() string
	Format(report *model.DuplicateReport, w io.Writer) error
}

// FormatterRegistry manages available output formatters
type FormatterRegistry struct {
	formatters map[string]Formatter
}

// NewFormatterRegistry creates a new OutputFormatterRegistry
func NewFormatterRegistry() *FormatterRegistry {
	return &FormatterRegistry{
		formatters: make(map[string]Formatter),
	}
}

// Register adds a new formatter to the registry
func (r *FormatterRegistry) Register(name string, formatter Formatter) error {
	if name == "" {
		return fmt.Errorf("formatter name cannot be empty")
	}
	if formatter == nil {
		return fmt.Errorf("formatter cannot be nil")
	}
	r.formatters[name] = formatter
	return nil
}

// Get retrieves a formatter by name from the registry
func (r *FormatterRegistry) Get(name string) (Formatter, bool) {
	formatter, exists := r.formatters[name]
	return formatter, exists
}

// List returns a list of registered formatter names
func (r *FormatterRegistry) List() []string {
	names := make([]string, 0, len(r.formatters))
	for name := range r.formatters {
		names = append(names, name)
	}
	return names
}

// Format formats the duplicate report using the specified formatter and writes it to the provided writer
func (r *FormatterRegistry) Format(name string, report *model.DuplicateReport, w io.Writer) error {
	formatter, exists := r.formatters[name]
	if !exists {
		return fmt.Errorf("formatter '%s' not found", name)
	}
	return formatter.Format(report, w)
}

// InitFormatters initializes the default output formatters and returns a registry
func InitFormatters() (*FormatterRegistry, error) {
	registry := NewFormatterRegistry()

	err := registry.Register("json", NewJSONFormatter())
	if err != nil {
		return nil, err
	}
	err = registry.Register("pretty", NewPrettyFormatter())
	if err != nil {
		return nil, err
	}
	return registry, nil
}

// FormatBytes converts a byte count to a human-readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
