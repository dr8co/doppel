// Package filter provides file and directory filtering capabilities for the doppel duplicate file finder.
//
// This package implements filtering logic to exclude files and directories based on:
//   - Glob patterns for file and directory names
//   - File size constraints (minimum and maximum sizes)
//
// The package supports parsing human-readable file sizes (e.g., "10MB", "1.5GB")
// and provides utilities to display active filter configurations.
package filter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dr8co/doppel/internal/logger"
	"github.com/dr8co/doppel/internal/output"
)

// Config defines criteria for excluding files and directories.
type Config struct {
	// Exclude contains glob patterns that exclude both files and directories.
	Exclude []string `json:"exclude" yaml:"exclude"`

	// ExcludeDirs contains directory names to exclude.
	ExcludeDirs []string `json:"exclude_dirs" yaml:"exclude_dirs"`

	// ExcludeFiles contains file names to exclude.
	ExcludeFiles []string `json:"exclude_files" yaml:"exclude_files"`

	// MinSize is the minimum file size to include (0 means no minimum).
	MinSize int64 `json:"min_size" yaml:"min_size"`

	// MaxSize is the maximum file size to include (0 means no maximum).
	MaxSize int64 `json:"max_size" yaml:"max_size"`
}

// BuildConfig creates a [Config] from command line arguments.
func BuildConfig(excludeDirs, excludeFiles string, minSize, maxSize int64, unifiedExclude ...string) (*Config, error) {
	// Handle negative values
	if minSize < 0 {
		logger.DebugAttrs(context.TODO(), "minSize is negative, setting to 0", slog.Int64("minSize", minSize))
		minSize = 0
	}
	if maxSize < 0 {
		logger.DebugAttrs(context.TODO(), "maxSize is negative, setting to 0", slog.Int64("maxSize", maxSize))
		maxSize = 0
	}

	// Validate min <= max when both are positive
	if minSize > 0 && maxSize > 0 && minSize > maxSize {
		return nil, fmt.Errorf("minimum size (%d) cannot be greater than maximum size (%d)", minSize, maxSize)
	}

	config := &Config{
		MinSize: minSize,
		MaxSize: maxSize,
	}
	if len(unifiedExclude) > 0 && unifiedExclude[0] != "" {
		config.Exclude = parseCommaSeparated(unifiedExclude[0])
		for _, pattern := range config.Exclude {
			if _, err := filepath.Match(pattern, ""); err != nil {
				return nil, fmt.Errorf("invalid exclude glob pattern '%s': %w", pattern, err)
			}
		}
		logger.Debug("Parsed unified exclude patterns", "patterns", config.Exclude)
	}

	// Parse exclude directory patterns
	if excludeDirs != "" {
		config.ExcludeDirs = parseCommaSeparated(excludeDirs)
		logger.Debug("Parsed exclude directories", "dirs", config.ExcludeDirs)
	}

	// Parse exclude file patterns
	if excludeFiles != "" {
		config.ExcludeFiles = parseCommaSeparated(excludeFiles)
		logger.Debug("Parsed exclude files", "files", config.ExcludeFiles)
	}

	return config, nil
}

// parseCommaSeparated splits a comma-separated string and trims whitespace.
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ShouldExcludeDir checks if a directory should be excluded based on filters.
func (fc *Config) ShouldExcludeDir(dirPath string) bool {
	dirName := filepath.Base(dirPath)

	if matchesAnyGlob(fc.Exclude, dirName, dirPath) {
		return true
	}

	// Check exact matches
	for _, pattern := range fc.ExcludeDirs {
		if matched, _ := filepath.Match(pattern, dirName); matched {
			return true
		}
		// Also check if the pattern matches the full path
		if matched, _ := filepath.Match(pattern, dirPath); matched {
			return true
		}
	}

	return false
}

// ShouldExcludeFile checks if a file should be excluded based on filters.
func (fc *Config) ShouldExcludeFile(filePath string, size int64) bool {
	fileName := filepath.Base(filePath)

	// Check size limits
	if fc.MinSize > 0 && size < fc.MinSize {
		return true
	}
	if fc.MaxSize > 0 && size > fc.MaxSize {
		return true
	}

	// If min and max are equal and positive, only include files of exactly that size
	if fc.MinSize > 0 && fc.MinSize == fc.MaxSize && size != fc.MinSize {
		return true
	}
	if matchesAnyGlob(fc.Exclude, fileName, filePath) {
		return true
	}

	// Check exact matches
	for _, pattern := range fc.ExcludeFiles {
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return true
		}
		// Also check if the pattern matches the full path
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
	}

	return false
}

func matchesAnyGlob(patterns []string, values ...string) bool {
	for _, pattern := range patterns {
		for _, value := range values {
			if matched, _ := filepath.Match(pattern, value); matched {
				return true
			}
		}
	}
	return false
}

// DisplayActiveFilters prints the currently active file and directory filters from the provided configuration.
func DisplayActiveFilters(config *Config) {
	DisplayActiveFiltersTo(config, os.Stdout)
}

// DisplayActiveFiltersTo writes the currently active filters to w.
func DisplayActiveFiltersTo(config *Config, w io.Writer) {
	_, _ = fmt.Fprintln(w, "🔧 Active filters:")
	if len(config.Exclude) > 0 {
		_, _ = fmt.Fprintf(w, "  🚫 Exclude: %s\n", strings.Join(config.Exclude, ", "))
	}
	if len(config.ExcludeDirs) > 0 {
		_, _ = fmt.Fprintf(w, "  📁 Exclude directories: %s\n", strings.Join(config.ExcludeDirs, ", "))
	}

	if len(config.ExcludeFiles) > 0 {
		_, _ = fmt.Fprintf(w, "  📄 Exclude files: %s\n", strings.Join(config.ExcludeFiles, ", "))
	}

	if config.MinSize > 0 {
		_, _ = fmt.Fprintf(w, "  📏 Minimum file size: %s\n", output.FormatBytes(config.MinSize))
	}

	if config.MaxSize > 0 {
		_, _ = fmt.Fprintf(w, "  📏 Maximum file size: %s\n", output.FormatBytes(config.MaxSize))
	}

	if len(config.Exclude) == 0 && len(config.ExcludeDirs) == 0 && len(config.ExcludeFiles) == 0 &&
		config.MinSize == 0 && config.MaxSize == 0 {
		_, _ = fmt.Fprintln(w, "  ✅ No filters active")
	}

	_, _ = fmt.Fprintln(w)
}

// ParseFileSize parses a file size string with optional suffix and returns size in bytes.
// Supported formats:
//   - Plain numbers: "1024", "500" (treated as bytes)
//   - With suffixes: "10MB", "5.5GB", "1.5KiB", "2TB"
//   - Case insensitive: "10mb", "5GB", "1kib"
//   - Spaces allowed: "10 MB", "5.5 GB"
//
// Note:
//   - Negatives are treated as 0.
//
// Returns the size in bytes as int64, or error if parsing fails.
func ParseFileSize(s string) (int64, error) {
	// Empty string is treated as 0
	if s == "" {
		return 0, nil
	}

	// Trim spaces
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("invalid file size format")
	}

	// Escape the leading '+'
	if s[0] == '+' {
		if len(s) > 1 {
			s = s[1:]
		}
	}

	// Negatives are treated as 0
	if s[0] == '-' {
		if len(s) > 1 {
			return 0, nil
		}
	}

	// Split number and unit
	i := 0
	for ; i < len(s); i++ {
		if (s[i] < '0' || s[i] > '9') && s[i] != '.' {
			break
		}
	}

	if i == 0 {
		return 0, errors.New("invalid file size format")
	}

	// Parse numeric part
	val, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric part: %w", err)
	}

	unit := normalizeUnit(strings.TrimSpace(s[i:]))

	var multiplier int64
	switch unit {
	case "b", "":
		multiplier = 1
	case "kb":
		multiplier = 1000
	case "mb":
		multiplier = 1000 * 1000
	case "gb":
		multiplier = 1000 * 1000 * 1000
	case "tb":
		multiplier = 1000 * 1000 * 1000 * 1000
	case "pb":
		multiplier = 1000 * 1000 * 1000 * 1000 * 1000
	case "eb":
		multiplier = 1000 * 1000 * 1000 * 1000 * 1000 * 1000

	case "kib":
		multiplier = 1024
	case "mib":
		multiplier = 1024 * 1024
	case "gib":
		multiplier = 1024 * 1024 * 1024
	case "tib":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "pib":
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	case "eib":
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024 * 1024
	default:
		return 0, errors.New("invalid unit")
	}

	// Convert
	res := val * float64(multiplier)
	if res > float64(^uint64(0)>>1) {
		return 0, errors.New("size overflow")
	}
	return int64(res), nil
}

// normalizeUnit converts unit to lowercase, handling only the first 3 characters for performance
func normalizeUnit(unit string) string {
	if len(unit) == 0 {
		return unit
	}

	// Manually convert to lowercase to avoid allocation
	var unitBytes [3]byte
	j := 0
	for range unit {
		if j >= 3 {
			break
		}
		c := unit[j]
		if c >= 'A' && c <= 'Z' {
			// //nolint:gosec // false positive: unitBytes is fixed size array
			unitBytes[j] = c + ('a' - 'A')
		} else {
			// //nolint:gosec
			unitBytes[j] = c
		}
		j++
	}

	return string(unitBytes[:j])
}
