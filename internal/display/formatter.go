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

func NewOutputFormatterRegistry() *OutputFormatterRegistry {
	return &OutputFormatterRegistry{
		formatters: make(map[string]OutputFormatter),
	}
}

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

func (r *OutputFormatterRegistry) Get(name string) (OutputFormatter, bool) {
	formatter, exists := r.formatters[name]
	return formatter, exists
}

func (r *OutputFormatterRegistry) List() []string {
	names := make([]string, 0, len(r.formatters))
	for name := range r.formatters {
		names = append(names, name)
	}
	return names
}

func (r *OutputFormatterRegistry) Format(name string, report *DuplicateReport, w io.Writer) error {
	formatter, exists := r.formatters[name]
	if !exists {
		return fmt.Errorf("formatter '%s' not found", name)
	}
	return formatter.Format(report, w)
}
