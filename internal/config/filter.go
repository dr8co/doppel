package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dr8co/doppel/internal/stats"
)

// FilterConfig defines criteria for excluding files and directories
type FilterConfig struct {
	ExcludeDirs      []string
	ExcludeFiles     []string
	ExcludeDirRegex  []*regexp.Regexp
	ExcludeFileRegex []*regexp.Regexp
	MinSize          int64
	MaxSize          int64
}

// BuildFilterConfig creates a FilterConfig from command line arguments
func BuildFilterConfig(excludeDirs, excludeFiles, excludeDirRegex, excludeFileRegex string, minSize, maxSize int64) (*FilterConfig, error) {
	config := &FilterConfig{
		MinSize: minSize,
		MaxSize: maxSize,
	}

	// Parse exclude directory patterns
	if excludeDirs != "" {
		config.ExcludeDirs = parseCommaSeparated(excludeDirs)
	}

	// Parse exclude file patterns
	if excludeFiles != "" {
		config.ExcludeFiles = parseCommaSeparated(excludeFiles)
	}

	// Parse exclude directory regex patterns
	if excludeDirRegex != "" {
		patterns := parseCommaSeparated(excludeDirRegex)
		for _, pattern := range patterns {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid directory regex pattern '%s': %w", pattern, err)
			}
			config.ExcludeDirRegex = append(config.ExcludeDirRegex, regex)
		}
	}

	// Parse exclude file regex patterns
	if excludeFileRegex != "" {
		patterns := parseCommaSeparated(excludeFileRegex)
		for _, pattern := range patterns {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid file regex pattern '%s': %w", pattern, err)
			}
			config.ExcludeFileRegex = append(config.ExcludeFileRegex, regex)
		}
	}

	return config, nil
}

// parseCommaSeparated splits a comma-separated string and trims whitespace
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

// ShouldExcludeDir checks if a directory should be excluded based on filters
func (fc *FilterConfig) ShouldExcludeDir(dirPath string) bool {
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
	for _, regex := range fc.ExcludeDirRegex {
		if regex.MatchString(dirName) || regex.MatchString(dirPath) {
			return true
		}
	}

	return false
}

// ShouldExcludeFile checks if a file should be excluded based on filters
func (fc *FilterConfig) ShouldExcludeFile(filePath string, size int64) bool {
	fileName := filepath.Base(filePath)

	// Check size limits
	if fc.MinSize > 0 && size < fc.MinSize {
		return true
	}
	if fc.MaxSize > 0 && size > fc.MaxSize {
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
	for _, regex := range fc.ExcludeFileRegex {
		if regex.MatchString(fileName) || regex.MatchString(filePath) {
			return true
		}
	}

	return false
}

// DisplayFilterConfig shows the current filter configuration
func DisplayFilterConfig(config *FilterConfig) {
	fmt.Println("🔧 Active filters:")
	if len(config.ExcludeDirs) > 0 {
		fmt.Printf("  📁 Exclude directories: %s\n", strings.Join(config.ExcludeDirs, ", "))
	}
	if len(config.ExcludeFiles) > 0 {
		fmt.Printf("  📄 Exclude files: %s\n", strings.Join(config.ExcludeFiles, ", "))
	}
	if len(config.ExcludeDirRegex) > 0 {
		patterns := make([]string, len(config.ExcludeDirRegex))
		for i, regex := range config.ExcludeDirRegex {
			patterns[i] = regex.String()
		}
		fmt.Printf("  📁 Exclude directory regex: %s\n", strings.Join(patterns, ", "))
	}
	if len(config.ExcludeFileRegex) > 0 {
		patterns := make([]string, len(config.ExcludeFileRegex))
		for i, regex := range config.ExcludeFileRegex {
			patterns[i] = regex.String()
		}
		fmt.Printf("  📄 Exclude file regex: %s\n", strings.Join(patterns, ", "))
	}
	if config.MinSize > 0 {
		fmt.Printf("  📏 Minimum file size: %s\n", stats.FormatBytes(config.MinSize))
	}
	if config.MaxSize > 0 {
		fmt.Printf("  📏 Maximum file size: %s\n", stats.FormatBytes(config.MaxSize))
	}
	if len(config.ExcludeDirs) == 0 && len(config.ExcludeFiles) == 0 &&
		len(config.ExcludeDirRegex) == 0 && len(config.ExcludeFileRegex) == 0 &&
		config.MinSize == 0 && config.MaxSize == 0 {
		fmt.Println("  ✅ No filters active")
	}
	fmt.Println()
}
