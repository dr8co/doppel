// Package output provides formatting and output capabilities for duplicate file scan results.
//
// This package implements a pluggable formatter system that supports multiple output formats:
//   - Pretty: Human-readable formatted output with colors and alignment
//   - JSON: Structured JSON output for programmatic consumption
//   - YAML: YAML format for configuration-style output
//
// The package uses a registry pattern to manage formatters and provides utilities
// for formatting file sizes and other display elements.
package output

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/dr8co/doppel/internal/model"
)

// Formatter formats duplicate reports to different output formats.
type Formatter interface {
	Name() string
	Format(report *model.DuplicateReport, w io.Writer) error
}

// FormatterRegistry manages available output formatters.
type FormatterRegistry struct {
	formatters map[string]Formatter
}

// NewFormatterRegistry creates a new OutputFormatterRegistry.
func NewFormatterRegistry() *FormatterRegistry {
	return &FormatterRegistry{
		formatters: make(map[string]Formatter),
	}
}

// Register adds a new formatter to the registry.
func (r *FormatterRegistry) Register(name string, formatter Formatter) error {
	if name == "" {
		return errors.New("formatter name cannot be empty")
	}
	if formatter == nil {
		return errors.New("formatter cannot be nil")
	}
	r.formatters[name] = formatter
	return nil
}

// Get retrieves a formatter by name from the registry.
func (r *FormatterRegistry) Get(name string) (Formatter, bool) {
	formatter, exists := r.formatters[name]
	return formatter, exists
}

// List returns a list of registered formatter names.
func (r *FormatterRegistry) List() []string {
	names := make([]string, 0, len(r.formatters))
	for name := range r.formatters {
		names = append(names, name)
	}
	return names
}

// Format formats the duplicate report using the specified formatter and writes it to the provided writer.
func (r *FormatterRegistry) Format(name string, report *model.DuplicateReport, w io.Writer) error {
	formatter, exists := r.formatters[name]
	if !exists {
		return fmt.Errorf("formatter '%s' not found", name)
	}
	return formatter.Format(report, w)
}

// InitFormatters initializes the default output formatters and returns a registry.
func InitFormatters() (*FormatterRegistry, error) {
	registry := NewFormatterRegistry()

	err := registry.Register("json", NewJSONFormatter())
	if err != nil {
		return nil, err
	}

	err = registry.Register("jsonl", NewJSONLFormatter())
	if err != nil {
		return nil, err
	}

	err = registry.Register("pretty", NewPrettyFormatter())
	if err != nil {
		return nil, err
	}

	err = registry.Register("yaml", NewYAMLFormatter())
	if err != nil {
		return nil, err
	}

	return registry, nil
}

// FormatBytes converts a byte count to a human-readable string.
func FormatBytes(bytes int64) string {
	const unit = 1000
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

// SortReport sorts duplicate groups and member paths deterministically.
func SortReport(report *model.DuplicateReport, sortBy string, reverse bool) error {
	if report == nil {
		return nil
	}

	const defaultSortBy = "path"

	sortBy = strings.TrimSpace(strings.ToLower(sortBy))
	if sortBy == "" {
		sortBy = defaultSortBy
	}

	for i := range report.Groups {
		sort.Strings(report.Groups[i].Files)
	}

	switch sortBy {
	case defaultSortBy:
		sort.SliceStable(report.Groups, func(i, j int) bool {
			left, right := "", ""
			if len(report.Groups[i].Files) > 0 {
				left = report.Groups[i].Files[0]
			}
			if len(report.Groups[j].Files) > 0 {
				right = report.Groups[j].Files[0]
			}
			if left == right {
				return len(report.Groups[i].Files) < len(report.Groups[j].Files)
			}
			return left < right
		})
	case "size":
		sort.SliceStable(report.Groups, func(i, j int) bool {
			return report.Groups[i].Size < report.Groups[j].Size
		})
	case "wasted-space":
		sort.SliceStable(report.Groups, func(i, j int) bool {
			return report.Groups[i].WastedSpace < report.Groups[j].WastedSpace
		})
	case "count":
		sort.SliceStable(report.Groups, func(i, j int) bool {
			return report.Groups[i].Count < report.Groups[j].Count
		})
	default:
		return fmt.Errorf("unsupported sort mode %q", sortBy)
	}

	if reverse {
		sort.SliceStable(report.Groups, func(i, j int) bool {
			left, right := report.Groups[i], report.Groups[j]
			if sortBy == defaultSortBy {
				leftPath, rightPath := "", ""
				if len(left.Files) > 0 {
					leftPath = left.Files[0]
				}
				if len(right.Files) > 0 {
					rightPath = right.Files[0]
				}
				if leftPath == rightPath {
					return len(left.Files) > len(right.Files)
				}
				return leftPath > rightPath
			}
			if sortBy == "size" {
				return left.Size > right.Size
			}
			if sortBy == "wasted-space" {
				return left.WastedSpace > right.WastedSpace
			}
			return left.Count > right.Count
		})
	}

	return nil
}
