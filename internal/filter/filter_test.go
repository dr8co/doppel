package filter

import (
	"regexp"
	"strings"
	"testing"
)

// TestBuildFilterConfig tests the [BuildConfig] function with various configurations for filter building.
// It verifies the function's behavior with different input patterns, size constraints, and edge cases.
// Additionally, it ensures expected errors are returned for invalid inputs.
func TestBuildFilterConfig(t *testing.T) {
	tests := []struct {
		name             string
		excludeDirs      string
		excludeFiles     string
		excludeDirRegex  string
		excludeFileRegex string
		minSize          int64
		maxSize          int64
		wantErr          bool
	}{
		{
			name:             "empty filter",
			excludeDirs:      "",
			excludeFiles:     "",
			excludeDirRegex:  "",
			excludeFileRegex: "",
			minSize:          0,
			maxSize:          0,
			wantErr:          false,
		},
		{
			name:             "with size limits",
			excludeDirs:      "",
			excludeFiles:     "",
			excludeDirRegex:  "",
			excludeFileRegex: "",
			minSize:          1000,
			maxSize:          5000,
			wantErr:          false,
		},
		{
			name:             "with exclude patterns",
			excludeDirs:      "node_modules,.git",
			excludeFiles:     "*.tmp,*.log",
			excludeDirRegex:  "",
			excludeFileRegex: "",
			minSize:          0,
			maxSize:          0,
			wantErr:          false,
		},
		{
			name:             "with regex patterns",
			excludeDirs:      "",
			excludeFiles:     "",
			excludeDirRegex:  "^\\.",
			excludeFileRegex: "^.*\\.bak$",
			minSize:          0,
			maxSize:          0,
			wantErr:          false,
		},
		{
			name:             "invalid dir regex",
			excludeDirs:      "",
			excludeFiles:     "",
			excludeDirRegex:  "[",
			excludeFileRegex: "",
			minSize:          0,
			maxSize:          0,
			wantErr:          true,
		},
		{
			name:             "invalid file regex",
			excludeDirs:      "",
			excludeFiles:     "",
			excludeDirRegex:  "",
			excludeFileRegex: "[",
			minSize:          0,
			maxSize:          0,
			wantErr:          true,
		},
		{
			name:             "whitespace patterns",
			excludeDirs:      "   ",
			excludeFiles:     "\t",
			excludeDirRegex:  "   ",
			excludeFileRegex: "\n",
			minSize:          0,
			maxSize:          0,
			wantErr:          false,
		},
		{
			name:             "overlapping patterns",
			excludeDirs:      "foo,foo",
			excludeFiles:     "bar,bar",
			excludeDirRegex:  "baz|baz",
			excludeFileRegex: "qux|qux",
			minSize:          0,
			maxSize:          0,
			wantErr:          false,
		},
		{
			name:             "very large min/max",
			excludeDirs:      "",
			excludeFiles:     "",
			excludeDirRegex:  "",
			excludeFileRegex: "",
			minSize:          1 << 40,
			maxSize:          1 << 41,
			wantErr:          false,
		},
		{
			name:             "min > max",
			excludeDirs:      "",
			excludeFiles:     "",
			excludeDirRegex:  "",
			excludeFileRegex: "",
			minSize:          100,
			maxSize:          10,
			wantErr:          true,
		},
		{
			name:             "negative min",
			excludeDirs:      "",
			excludeFiles:     "",
			excludeDirRegex:  "",
			excludeFileRegex: "",
			minSize:          -1,
			maxSize:          100,
			wantErr:          false,
		},
		{
			name:             "negative max",
			excludeDirs:      "",
			excludeFiles:     "",
			excludeDirRegex:  "",
			excludeFileRegex: "",
			minSize:          -1,
			maxSize:          -100,
			wantErr:          false,
		},
	}

	// Helper function to validate size field
	validateSize := func(actual, expected int64, fieldName string) {
		if expected < 0 {
			if actual != 0 {
				t.Errorf("%s = %v, want 0 for negative %s", fieldName, actual, strings.ToLower(fieldName))
			}
			return
		}
		if actual != expected {
			t.Errorf("%s = %v, want %v", fieldName, actual, expected)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := BuildConfig(tt.excludeDirs, tt.excludeFiles, tt.excludeDirRegex, tt.excludeFileRegex, tt.minSize, tt.maxSize)

			if (err != nil) != tt.wantErr {
				t.Errorf("BuildConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Verify the filter was built correctly
			validateSize(config.MinSize, tt.minSize, "MinSize")
			validateSize(config.MaxSize, tt.maxSize, "MaxSize")

			// Check exclude dirs
			expectedDirs := parseCommaSeparated(tt.excludeDirs)
			if len(config.ExcludeDirs) != len(expectedDirs) {
				t.Errorf("ExcludeDirs length = %v, want %v", len(config.ExcludeDirs), len(expectedDirs))
			}

			// Check exclude files
			expectedFiles := parseCommaSeparated(tt.excludeFiles)
			if len(config.ExcludeFiles) != len(expectedFiles) {
				t.Errorf("ExcludeFiles length = %v, want %v", len(config.ExcludeFiles), len(expectedFiles))
			}

			// Check regex counts
			expectedDirRegexCount := len(parseCommaSeparated(tt.excludeDirRegex))
			if !tt.wantErr && len(config.ExcludeDirRegex) != expectedDirRegexCount {
				t.Errorf("ExcludeDirRegex length = %v, want %v", len(config.ExcludeDirRegex), expectedDirRegexCount)
			}

			expectedFileRegexCount := len(parseCommaSeparated(tt.excludeFileRegex))
			if !tt.wantErr && len(config.ExcludeFileRegex) != expectedFileRegexCount {
				t.Errorf("ExcludeFileRegex length = %v, want %v", len(config.ExcludeFileRegex), expectedFileRegexCount)
			}
		})
	}
}

// TestShouldExcludeDir validates the behavior of ShouldExcludeDir by testing various exclusion configurations and scenarios.
func TestShouldExcludeDir(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		dirPath    string
		shouldSkip bool
	}{
		{
			name: "exact match",
			config: &Config{
				ExcludeDirs: []string{"node_modules", ".git"},
			},
			dirPath:    "/path/to/node_modules",
			shouldSkip: true,
		},
		{
			name: "pattern match",
			config: &Config{
				ExcludeDirs: []string{"*.git"},
			},
			dirPath:    "/path/to/project.git",
			shouldSkip: true,
		},
		{
			name: "regex match",
			config: &Config{
				ExcludeDirRegex: []*regexp.Regexp{regexp.MustCompile(`^\.`)},
			},
			dirPath:    "/path/to/.hidden",
			shouldSkip: true,
		},
		{
			name: "no match",
			config: &Config{
				ExcludeDirs:     []string{"node_modules", ".git"},
				ExcludeDirRegex: []*regexp.Regexp{regexp.MustCompile(`^\.`)},
			},
			dirPath:    "/path/to/src",
			shouldSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ShouldExcludeDir(tt.dirPath)
			if result != tt.shouldSkip {
				t.Errorf("ShouldExcludeDir(%s) = %v, want %v", tt.dirPath, result, tt.shouldSkip)
			}
		})
	}
}

// TestShouldExcludeFile verifies the behavior of ShouldExcludeFile based on various file attributes and filter criteria.
func TestShouldExcludeFile(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		filePath   string
		fileSize   int64
		shouldSkip bool
	}{
		{
			name: "too small",
			config: &Config{
				MinSize: 1000,
			},
			filePath:   "/path/to/small.txt",
			fileSize:   500,
			shouldSkip: true,
		},
		{
			name: "too large",
			config: &Config{
				MaxSize: 1000,
			},
			filePath:   "/path/to/large.txt",
			fileSize:   1500,
			shouldSkip: true,
		},
		{
			name: "exact match",
			config: &Config{
				ExcludeFiles: []string{"temp.txt", "log.txt"},
			},
			filePath:   "/path/to/temp.txt",
			fileSize:   1000,
			shouldSkip: true,
		},
		{
			name: "pattern match",
			config: &Config{
				ExcludeFiles: []string{"*.tmp", "*.log"},
			},
			filePath:   "/path/to/file.tmp",
			fileSize:   1000,
			shouldSkip: true,
		},
		{
			name: "regex match",
			config: &Config{
				ExcludeFileRegex: []*regexp.Regexp{regexp.MustCompile(`\.bak$`)},
			},
			filePath:   "/path/to/file.bak",
			fileSize:   1000,
			shouldSkip: true,
		},
		{
			name: "no match",
			config: &Config{
				MinSize:          100,
				MaxSize:          10000,
				ExcludeFiles:     []string{"*.tmp", "*.log"},
				ExcludeFileRegex: []*regexp.Regexp{regexp.MustCompile(`\.bak$`)},
			},
			filePath:   "/path/to/document.txt",
			fileSize:   1000,
			shouldSkip: false,
		},
		{
			name: "negative min size treated as 0",
			config: &Config{
				MinSize: -100,
			},
			filePath:   "file.txt",
			fileSize:   50,
			shouldSkip: false,
		},
		{
			name: "negative max size treated as no maximum",
			config: &Config{
				MaxSize: -100,
			},
			filePath:   "file.txt",
			fileSize:   1000,
			shouldSkip: false,
		},
		{
			name: "min equals max - exact match",
			config: &Config{
				MinSize: 1000,
				MaxSize: 1000,
			},
			filePath:   "file.txt",
			fileSize:   1000,
			shouldSkip: false,
		},
		{
			name: "min equals max - not exact match",
			config: &Config{
				MinSize: 1000,
				MaxSize: 1000,
			},
			filePath:   "file.txt",
			fileSize:   999,
			shouldSkip: true,
		},
		{
			name: "min exceeds max - all files excluded",
			config: &Config{
				MinSize: 2000,
				MaxSize: 1000,
			},
			filePath:   "file.txt",
			fileSize:   1500,
			shouldSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ShouldExcludeFile(tt.filePath, tt.fileSize)
			if result != tt.shouldSkip {
				t.Errorf("ShouldExcludeFile(%s, %d) = %v, want %v", tt.filePath, tt.fileSize, result, tt.shouldSkip)
			}
		})
	}
}

// TestParseCommaSeparated tests the parseCommaSeparated function for various input cases,
// verifying correct splitting and trimming.
func TestParseCommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single value",
			input:    "value",
			expected: []string{"value"},
		},
		{
			name:     "multiple values",
			input:    "value1,value2,value3",
			expected: []string{"value1", "value2", "value3"},
		},
		{
			name:     "with whitespace",
			input:    " value1 , value2 , value3 ",
			expected: []string{"value1", "value2", "value3"},
		},
		{
			name:     "with empty parts",
			input:    "value1,,value3",
			expected: []string{"value1", "value3"},
		},
		{
			name:     "empty parts",
			input:    ",,,,,",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparated(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("parseCommaSeparated(%s) length = %v, want %v", tt.input, len(result), len(tt.expected))
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parseCommaSeparated(%s)[%d] = %v, want %v", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}
