package config

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// TestEnvProvider tests the [EnvProvider] type.
func TestEnvProvider(t *testing.T) {
	zero := 0
	tests := []struct {
		name     string
		env      map[string]string
		prefix   string
		priority int
		want     *Config
		wantErr  bool
	}{
		{
			name:     "empty environment",
			env:      map[string]string{},
			prefix:   "TEST_",
			priority: 1,
			want:     &Config{},
		},
		{
			name: "log configuration",
			env: map[string]string{
				"TEST_LOG_LEVEL":  "debug",
				"TEST_LOG_FORMAT": "json",
				"TEST_LOG_OUTPUT": "file.log",
			},
			prefix:   "TEST_",
			priority: 1,
			want: &Config{
				Log: LogConfig{
					Level:  "debug",
					Format: "json",
					Output: "file.log",
				},
			},
		},
		{
			name: "find configuration",
			env: map[string]string{
				"TEST_FIND_WORKERS":            "4",
				"TEST_FIND_VERBOSE":            "true",
				"TEST_FIND_EXCLUDE":            "*.bak",
				"TEST_FIND_MAX_DEPTH":          "0",
				"TEST_FIND_EXCLUDE_DIRS":       "node_modules,vendor",
				"TEST_FIND_EXCLUDE_FILES":      "*.log",
				"TEST_FIND_MIN_SIZE":           "1MB",
				"TEST_FIND_MAX_SIZE":           "100MB",
				"TEST_FIND_SORT":               "size",
				"TEST_FIND_SORT_REVERSE":       "true",
				"TEST_FIND_PATHS_ONLY":         "true",
				"TEST_FIND_PRINT0":             "true",
				"TEST_FIND_QUIET":              "true",
				"TEST_FIND_FAIL_ON_DUPLICATES": "true",
				"TEST_FIND_SHOW_FILTERS":       "true",
				"TEST_FIND_IGNORE_HARDLINKS":   "true",
				"TEST_FIND_OUTPUT_FORMAT":      "json",
				"TEST_FIND_OUTPUT_FILE":        "out.json",
			},
			prefix:   "TEST_",
			priority: 1,
			want: &Config{
				Find: FindConfig{
					Workers:          4,
					Verbose:          true,
					Exclude:          "*.bak",
					MaxDepth:         &zero,
					ExcludeDirs:      "node_modules,vendor",
					ExcludeFiles:     "*.log",
					MinSize:          "1MB",
					MaxSize:          "100MB",
					Sort:             "size",
					SortReverse:      true,
					PathsOnly:        true,
					Print0:           true,
					Quiet:            true,
					FailOnDuplicates: true,
					ShowFilters:      true,
					IgnoreHardlinks:  true,
					OutputFormat:     "json",
					OutputFile:       "out.json",
				},
			},
		},
		{
			name: "clean configuration",
			env: map[string]string{
				"TEST_CLEAN_WORKERS":          "5",
				"TEST_CLEAN_VERBOSE":          "true",
				"TEST_CLEAN_QUIET":            "true",
				"TEST_CLEAN_IGNORE_HARDLINKS": "true",
				"TEST_CLEAN_EXCLUDE":          "*.bak",
				"TEST_CLEAN_MAX_DEPTH":        "0",
				"TEST_CLEAN_EXCLUDE_DIRS":     ".git,node_modules",
				"TEST_CLEAN_EXCLUDE_FILES":    "*.tmp",
				"TEST_CLEAN_MIN_SIZE":         "1KB",
				"TEST_CLEAN_MAX_SIZE":         "10MB",
				"TEST_CLEAN_DRY_RUN":          "true",
				"TEST_CLEAN_MODE":             "trash",
				"TEST_CLEAN_KEEP":             "newest",
			},
			prefix:   "TEST_",
			priority: 1,
			want: &Config{
				Clean: CleanConfig{
					Workers:         5,
					Verbose:         true,
					Quiet:           true,
					IgnoreHardlinks: true,
					Exclude:         "*.bak",
					MaxDepth:        &zero,
					ExcludeDirs:     ".git,node_modules",
					ExcludeFiles:    "*.tmp",
					MinSize:         "1KB",
					MaxSize:         "10MB",
					DryRun:          true,
					Mode:            "trash",
					Keep:            "newest",
				},
			},
		},
		{
			name: "boolean variations",
			env: map[string]string{
				"TEST_FIND_VERBOSE":      "yes",
				"TEST_FIND_SHOW_FILTERS": "1",
			},
			prefix:   "TEST_",
			priority: 1,
			want: &Config{
				Find: FindConfig{
					Verbose:     true,
					ShowFilters: true,
				},
			},
		},
		{
			name: "invalid integer",
			env: map[string]string{
				"TEST_FIND_WORKERS": "not_a_number",
			},
			prefix:   "TEST_",
			priority: 1,
			want:     &Config{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the environment
			cleanup := withEnv(t, tt.env)
			defer cleanup()

			// Create provider
			p := NewEnvProvider(tt.prefix, tt.priority)

			// Check the provider name
			if name := p.Name(); name != "env:"+tt.prefix {
				t.Errorf("Name() = %q, want env:%q", name, tt.prefix)
			}
			if priority := p.Priority(); priority != tt.priority {
				t.Errorf("Priority() = %d, want %d", priority, tt.priority)
			}

			// Test loading
			ctx := context.Background()
			got, err := p.Load(ctx)
			if tt.wantErr && err == nil {
				t.Error("Load() expected error but got none")
			} else if !tt.wantErr && err != nil {
				t.Errorf("Load() unexpected error: %v", err)
			} else if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load() = %+v\nwant %+v", got, tt.want)
			}
		})
	}

	t.Run("context cancellation", func(t *testing.T) {
		p := NewEnvProvider("TEST_", 1)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := p.Load(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Load() with cancelled context = %v, want %v", err, context.Canceled)
		}
	})
}
