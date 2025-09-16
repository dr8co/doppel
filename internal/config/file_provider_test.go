package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestFileProvider tests the [FileProvider] type.
func TestFileProvider(t *testing.T) {
	// Create a temp directory for test files
	testDir, cleanup := testDir(t)
	defer cleanup()

	sampleConfig := &Config{
		Log: LogConfig{
			Level:  "debug",
			Format: "json",
			Output: "app.log",
		},
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
		Preset: PresetConfig{
			Workers:      4,
			Verbose:      true,
			ShowFilters:  true,
			OutputFormat: "json",
			OutputFile:   "out.json",
		},
	}

	// Test file formats
	formats := []struct {
		name    string
		format  string
		content string
	}{
		{
			name:   "TOML",
			format: "toml",
			content: `[log]
level = "debug"
format = "json"
output = "app.log"

[find]
workers = 4
verbose = true
exclude_dirs = "node_modules,vendor"
exclude_files = "*.log"
exclude_dir_regex = "^\\."
exclude_file_regex = "^\\."
min_size = "1MB"
max_size = "100MB"
show_filters = true
output_format = "json"
output_file = "out.json"

[preset]
workers = 4
verbose = true
show_filters = true
output_format = "json"
output_file = "out.json"`,
		},
		{
			name:   "JSON",
			format: "json",
			content: `{
  "log": {
    "level": "debug",
    "format": "json",
    "output": "app.log"
  },
  "find": {
    "workers": 4,
    "verbose": true,
    "exclude_dirs": "node_modules,vendor",
    "exclude_files": "*.log",
    "exclude_dir_regex": "^\\.",
    "exclude_file_regex": "^\\.",
    "min_size": "1MB",
    "max_size": "100MB",
    "show_filters": true,
    "output_format": "json",
    "output_file": "out.json"
  },
  "preset": {
    "workers": 4,
    "verbose": true,
    "show_filters": true,
    "output_format": "json",
    "output_file": "out.json"
  }
}`,
		},
		{
			name:   "YAML",
			format: "yaml",
			content: `log:
  level: debug
  format: json
  output: app.log
find:
  workers: 4
  verbose: true
  exclude_dirs: node_modules,vendor
  exclude_files: "*.log"
  exclude_dir_regex: "^\\."
  exclude_file_regex: "^\\."
  min_size: 1MB
  max_size: 100MB
  show_filters: true
  output_format: json
  output_file: out.json
preset:
  workers: 4
  verbose: true
  show_filters: true
  output_format: json
  output_file: out.json`,
		},
	}

	for _, tt := range formats {
		t.Run(tt.name, func(t *testing.T) {
			// Create a config file
			path := writeConfigFile(t, testDir, "config", tt.format, tt.content)

			// Create provider
			p := NewFileProvider(path, 1)

			// Verify the provider name and priority
			if name := p.Name(); name != "file:"+path {
				t.Errorf("Name() = %q, want file:%q", name, path)
			}
			if priority := p.Priority(); priority != 1 {
				t.Errorf("Priority() = %d, want 1", priority)
			}

			// Test loading
			ctx := context.Background()
			got, err := p.Load(ctx)
			if err != nil {
				t.Errorf("Load() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, sampleConfig) {
				t.Errorf("Load() = %+v\nwant %+v", got, sampleConfig)
			}
		})
	}

	t.Run("non-existent file", func(t *testing.T) {
		path := filepath.Join(testDir, "nonexistent.toml")
		p := NewFileProvider(path, 1)
		ctx := context.Background()
		got, err := p.Load(ctx)
		if err != nil {
			t.Errorf("Load() error = %v", err)
			return
		}
		if !reflect.DeepEqual(got, &Config{}) {
			t.Errorf("Load() = %+v, want empty config", got)
		}
	})

	t.Run("invalid file content", func(t *testing.T) {
		path := writeConfigFile(t, testDir, "invalid", "json", "{invalid}")
		p := NewFileProvider(path, 1)
		ctx := context.Background()
		if _, err := p.Load(ctx); err == nil {
			t.Error("Load() error = nil, want error for invalid content")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		path := writeConfigFile(t, testDir, "config", "toml", "")
		p := NewFileProvider(path, 1)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := p.Load(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Load() with cancelled context = %v, want %v", err, context.Canceled)
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		path := filepath.Join(testDir, "config.unsupported")
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		p := NewFileProvider(path, 1)
		ctx := context.Background()
		if _, err := p.Load(ctx); err == nil {
			t.Error("Load() error = nil, want error for unsupported format")
		}
	})
}
