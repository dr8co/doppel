package config

import (
	"regexp"
	"testing"
)

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
			name:             "empty config",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := BuildFilterConfig(tt.excludeDirs, tt.excludeFiles, tt.excludeDirRegex, tt.excludeFileRegex, tt.minSize, tt.maxSize)

			if (err != nil) != tt.wantErr {
				t.Errorf("BuildFilterConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Verify the config was built correctly
			if config.MinSize != tt.minSize {
				t.Errorf("MinSize = %v, want %v", config.MinSize, tt.minSize)
			}

			if config.MaxSize != tt.maxSize {
				t.Errorf("MaxSize = %v, want %v", config.MaxSize, tt.maxSize)
			}

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

func TestShouldExcludeDir(t *testing.T) {
	tests := []struct {
		name       string
		config     *FilterConfig
		dirPath    string
		shouldSkip bool
	}{
		{
			name: "exact match",
			config: &FilterConfig{
				ExcludeDirs: []string{"node_modules", ".git"},
			},
			dirPath:    "/path/to/node_modules",
			shouldSkip: true,
		},
		{
			name: "pattern match",
			config: &FilterConfig{
				ExcludeDirs: []string{"*.git"},
			},
			dirPath:    "/path/to/project.git",
			shouldSkip: true,
		},
		{
			name: "regex match",
			config: &FilterConfig{
				ExcludeDirRegex: []*regexp.Regexp{regexp.MustCompile("^\\.")},
			},
			dirPath:    "/path/to/.hidden",
			shouldSkip: true,
		},
		{
			name: "no match",
			config: &FilterConfig{
				ExcludeDirs:     []string{"node_modules", ".git"},
				ExcludeDirRegex: []*regexp.Regexp{regexp.MustCompile("^\\.")},
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

func TestShouldExcludeFile(t *testing.T) {
	tests := []struct {
		name       string
		config     *FilterConfig
		filePath   string
		fileSize   int64
		shouldSkip bool
	}{
		{
			name: "too small",
			config: &FilterConfig{
				MinSize: 1000,
			},
			filePath:   "/path/to/small.txt",
			fileSize:   500,
			shouldSkip: true,
		},
		{
			name: "too large",
			config: &FilterConfig{
				MaxSize: 1000,
			},
			filePath:   "/path/to/large.txt",
			fileSize:   1500,
			shouldSkip: true,
		},
		{
			name: "exact match",
			config: &FilterConfig{
				ExcludeFiles: []string{"temp.txt", "log.txt"},
			},
			filePath:   "/path/to/temp.txt",
			fileSize:   1000,
			shouldSkip: true,
		},
		{
			name: "pattern match",
			config: &FilterConfig{
				ExcludeFiles: []string{"*.tmp", "*.log"},
			},
			filePath:   "/path/to/file.tmp",
			fileSize:   1000,
			shouldSkip: true,
		},
		{
			name: "regex match",
			config: &FilterConfig{
				ExcludeFileRegex: []*regexp.Regexp{regexp.MustCompile("\\.bak$")},
			},
			filePath:   "/path/to/file.bak",
			fileSize:   1000,
			shouldSkip: true,
		},
		{
			name: "no match",
			config: &FilterConfig{
				MinSize:          100,
				MaxSize:          10000,
				ExcludeFiles:     []string{"*.tmp", "*.log"},
				ExcludeFileRegex: []*regexp.Regexp{regexp.MustCompile("\\.bak$")},
			},
			filePath:   "/path/to/document.txt",
			fileSize:   1000,
			shouldSkip: false,
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
