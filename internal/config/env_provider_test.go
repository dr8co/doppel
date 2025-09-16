package config

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// TestEnvProvider tests the [EnvProvider] type.
func TestEnvProvider(t *testing.T) {
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
				"TEST_FIND_EXCLUDE_DIRS":       "node_modules,vendor",
				"TEST_FIND_EXCLUDE_FILES":      "*.log",
				"TEST_FIND_EXCLUDE_DIR_REGEX":  "^\\.",
				"TEST_FIND_EXCLUDE_FILE_REGEX": "^\\.",
				"TEST_FIND_MIN_SIZE":           "1MB",
				"TEST_FIND_MAX_SIZE":           "100MB",
				"TEST_FIND_SHOW_FILTERS":       "true",
				"TEST_FIND_OUTPUT_FORMAT":      "json",
				"TEST_FIND_OUTPUT_FILE":        "out.json",
			},
			prefix:   "TEST_",
			priority: 1,
			want: &Config{
				Find: FindConfig{
					Workers:          4,
					Verbose:          true,
					ExcludeDirs:      "node_modules,vendor",
					ExcludeFiles:     "*.log",
					ExcludeDirRegex:  "^\\.",
					ExcludeFileRegex: "^\\.",
					MinSize:          "1MB",
					MaxSize:          "100MB",
					ShowFilters:      true,
					OutputFormat:     "json",
					OutputFile:       "out.json",
				},
			},
		},
		{
			name: "preset configuration",
			env: map[string]string{
				"TEST_PRESET_WORKERS":       "4",
				"TEST_PRESET_VERBOSE":       "true",
				"TEST_PRESET_SHOW_FILTERS":  "true",
				"TEST_PRESET_OUTPUT_FORMAT": "json",
				"TEST_PRESET_OUTPUT_FILE":   "out.json",
			},
			prefix:   "TEST_",
			priority: 1,
			want: &Config{
				Preset: PresetConfig{
					Workers:      4,
					Verbose:      true,
					ShowFilters:  true,
					OutputFormat: "json",
					OutputFile:   "out.json",
				},
			},
		},
		{
			name: "boolean variations",
			env: map[string]string{
				"TEST_FIND_VERBOSE":        "yes",
				"TEST_FIND_SHOW_FILTERS":   "1",
				"TEST_PRESET_VERBOSE":      "on",
				"TEST_PRESET_SHOW_FILTERS": "YES",
			},
			prefix:   "TEST_",
			priority: 1,
			want: &Config{
				Find: FindConfig{
					Verbose:     true,
					ShowFilters: true,
				},
				Preset: PresetConfig{
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
