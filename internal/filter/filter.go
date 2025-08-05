package filter

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dr8co/doppel/internal/logger"
	"github.com/dr8co/doppel/internal/output"
)

// Config defines criteria for excluding files and directories.
type Config struct {
	// ExcludeDirs contains directory names to exclude.
	ExcludeDirs []string `json:"exclude_dirs" yaml:"exclude_dirs"`

	// ExcludeFiles contains file names to exclude.
	ExcludeFiles []string `json:"exclude_files" yaml:"exclude_files"`

	// ExcludeDirRegexRaw contains the raw regex patterns for directories to exclude.
	ExcludeDirRegexRaw []string `json:"exclude_dir_regex" yaml:"exclude_dir_regex"`

	// ExcludeFileRegexRaw contains the raw regex patterns for files to exclude.
	ExcludeFileRegexRaw []string `json:"exclude_file_regex" yaml:"exclude_file_regex"`

	// MinSize is the minimum file size to include (0 means no minimum).
	MinSize int64 `json:"min_size" yaml:"min_size"`

	// MaxSize is the maximum file size to include (0 means no maximum).
	MaxSize int64 `json:"max_size" yaml:"max_size"`

	// excludeFileRegex contains compiled regex patterns for files to exclude.
	excludeFileRegex []*regexp.Regexp

	// excludeDirRegex contains compiled regex patterns for directories to exclude.
	excludeDirRegex []*regexp.Regexp
}

// BuildConfig creates a [Config] from command line arguments.
func BuildConfig(excludeDirs, excludeFiles, excludeDirRegex, excludeFileRegex string, minSize, maxSize int64) (*Config, error) {
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

	// Parse exclude directory regex patterns
	if excludeDirRegex != "" {
		patterns := parseCommaSeparated(excludeDirRegex)
		if len(patterns) > 0 {
			logger.Debug("Parsing exclude directory regex", "patterns", patterns)

			for _, pattern := range patterns {
				regex, err := regexp.Compile(pattern)
				if err != nil {
					return nil, fmt.Errorf("invalid directory regex pattern '%s': %w", pattern, err)
				}
				config.excludeDirRegex = append(config.excludeDirRegex, regex)
			}

			config.ExcludeDirRegexRaw = patterns
			logger.Debug("Parsed exclude directory regex", "regex", config.excludeDirRegex)
		}
	}

	// Parse exclude file regex patterns
	if excludeFileRegex != "" {
		patterns := parseCommaSeparated(excludeFileRegex)
		if len(patterns) > 0 {
			logger.Debug("Parsing exclude file regex", "patterns", patterns)

			for _, pattern := range patterns {
				regex, err := regexp.Compile(pattern)
				if err != nil {
					return nil, fmt.Errorf("invalid file regex pattern '%s': %w", pattern, err)
				}
				config.excludeFileRegex = append(config.excludeFileRegex, regex)
			}

			config.ExcludeFileRegexRaw = patterns
			logger.Debug("Parsed exclude file regex", "regex", config.excludeFileRegex)
		}
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

	// Check regex patterns
	for _, regex := range fc.excludeDirRegex {
		if regex.MatchString(dirName) || regex.MatchString(dirPath) {
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

	// Check regex patterns
	for _, regex := range fc.excludeFileRegex {
		if regex.MatchString(fileName) || regex.MatchString(filePath) {
			return true
		}
	}

	return false
}

// DisplayActiveFilters prints the currently active file and directory filters from the provided configuration.
func DisplayActiveFilters(config *Config) {
	fmt.Println("🔧 Active filters:")
	if len(config.ExcludeDirs) > 0 {
		fmt.Printf("  📁 Exclude directories: %s\n", strings.Join(config.ExcludeDirs, ", "))
	}

	if len(config.ExcludeFiles) > 0 {
		fmt.Printf("  📄 Exclude files: %s\n", strings.Join(config.ExcludeFiles, ", "))
	}

	if len(config.excludeDirRegex) > 0 {
		fmt.Printf("  📁 Exclude directory regex: %q\n", config.ExcludeDirRegexRaw)
	}

	if len(config.excludeFileRegex) > 0 {
		fmt.Printf("  📄 Exclude file regex: %q\n", config.ExcludeFileRegexRaw)
	}

	if config.MinSize > 0 {
		fmt.Printf("  📏 Minimum file size: %s\n", output.FormatBytes(config.MinSize))
	}

	if config.MaxSize > 0 {
		fmt.Printf("  📏 Maximum file size: %s\n", output.FormatBytes(config.MaxSize))
	}

	if len(config.ExcludeDirs) == 0 && len(config.ExcludeFiles) == 0 &&
		len(config.excludeDirRegex) == 0 && len(config.excludeFileRegex) == 0 &&
		config.MinSize == 0 && config.MaxSize == 0 {
		fmt.Println("  ✅ No filters active")
	}

	fmt.Println()
}
