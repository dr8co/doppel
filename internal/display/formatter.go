package display

import (
	"fmt"
	"io"
)

// OutputFormatter formats duplicate reports to different output formats
type OutputFormatter interface {
	Format(report *DuplicateReport, w io.Writer) error
}

// OutputFormatterRegistry manages available output formatters
type OutputFormatterRegistry struct {
	formatters map[string]OutputFormatter
}

// NewOutputFormatterRegistry creates a new OutputFormatterRegistry
func NewOutputFormatterRegistry() *OutputFormatterRegistry {
	return &OutputFormatterRegistry{
		formatters: make(map[string]OutputFormatter),
	}
}

// Register adds a new formatter to the registry
func (r *OutputFormatterRegistry) Register(name string, formatter OutputFormatter) error {
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
func (r *OutputFormatterRegistry) Get(name string) (OutputFormatter, bool) {
	formatter, exists := r.formatters[name]
	return formatter, exists
}

// List returns a list of registered formatter names
func (r *OutputFormatterRegistry) List() []string {
	names := make([]string, 0, len(r.formatters))
	for name := range r.formatters {
		names = append(names, name)
	}
	return names
}

// Format formats the duplicate report using the specified formatter and writes it to the provided writer
func (r *OutputFormatterRegistry) Format(name string, report *DuplicateReport, w io.Writer) error {
	formatter, exists := r.formatters[name]
	if !exists {
		return fmt.Errorf("formatter '%s' not found", name)
	}
	return formatter.Format(report, w)
}

// InitFormatters initializes the default output formatters and returns a registry
func InitFormatters() (*OutputFormatterRegistry, error) {
	registry := NewOutputFormatterRegistry()

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
