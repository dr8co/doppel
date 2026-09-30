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
	maxDepth := 2
	cleanMaxDepth := 0

	sampleConfig := &Config{
		Log: LogConfig{
			Level:  "debug",
			Format: "json",
			Output: "app.log",
		},
		Find: FindConfig{
			Workers:          4,
			Verbose:          true,
			Exclude:          "*.bak",
			MaxDepth:         &maxDepth,
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
		Clean: CleanConfig{
			Workers:         5,
			Verbose:         true,
			Quiet:           true,
			IgnoreHardlinks: true,
			Exclude:         "*.bak",
			MaxDepth:        &cleanMaxDepth,
			ExcludeDirs:     ".git,node_modules",
			ExcludeFiles:    "*.tmp",
			MinSize:         "1KB",
			MaxSize:         "10MB",
			DryRun:          true,
			Mode:            "trash",
			Keep:            "newest",
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
exclude = "*.bak"
max_depth = 2
exclude_dirs = "node_modules,vendor"
exclude_files = "*.log"
min_size = "1MB"
max_size = "100MB"
sort = "size"
sort_reverse = true
paths_only = true
print0 = true
quiet = true
fail_on_duplicates = true
show_filters = true
ignore_hardlinks = true
output_format = "json"
output_file = "out.json"

[clean]
workers = 5
verbose = true
quiet = true
ignore_hardlinks = true
exclude = "*.bak"
max_depth = 0
exclude_dirs = ".git,node_modules"
exclude_files = "*.tmp"
min_size = "1KB"
max_size = "10MB"
dry_run = true
mode = "trash"
keep = "newest"`,
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
		"exclude": "*.bak",
		"max_depth": 2,
		"exclude_dirs": "node_modules,vendor",
		"exclude_files": "*.log",
		"min_size": "1MB",
		"max_size": "100MB",
		"sort": "size",
		"sort_reverse": true,
		"paths_only": true,
		"print0": true,
		"quiet": true,
		"fail_on_duplicates": true,
		"show_filters": true,
		"ignore_hardlinks": true,
		"output_format": "json",
		"output_file": "out.json"
	},
	"clean": {
		"workers": 5,
		"verbose": true,
		"quiet": true,
		"ignore_hardlinks": true,
		"exclude": "*.bak",
		"max_depth": 0,
		"exclude_dirs": ".git,node_modules",
		"exclude_files": "*.tmp",
		"min_size": "1KB",
		"max_size": "10MB",
		"dry_run": true,
		"mode": "trash",
		"keep": "newest"
	}
}`,
		},
		{
			name:   "YAML",
			format: "yaml",
			content: "log:\n" +
				"  level: debug\n" +
				"  format: json\n" +
				"  output: app.log\n" +
				"find:\n" +
				"  workers: 4\n" +
				"  verbose: true\n" +
				"  exclude: \"*.bak\"\n" +
				"  max_depth: 2\n" +
				"  exclude_dirs: node_modules,vendor\n" +
				"  exclude_files: \"*.log\"\n" +
				"  min_size: 1MB\n" +
				"  max_size: 100MB\n" +
				"  sort: size\n" +
				"  sort_reverse: true\n" +
				"  paths_only: true\n" +
				"  print0: true\n" +
				"  quiet: true\n" +
				"  fail_on_duplicates: true\n" +
				"  show_filters: true\n" +
				"  ignore_hardlinks: true\n" +
				"  output_format: json\n" +
				"  output_file: out.json\n" +
				"clean:\n" +
				"  workers: 5\n" +
				"  verbose: true\n" +
				"  quiet: true\n" +
				"  ignore_hardlinks: true\n" +
				"  exclude: \"*.bak\"\n" +
				"  max_depth: 0\n" +
				"  exclude_dirs: \".git,node_modules\"\n" +
				"  exclude_files: \"*.tmp\"\n" +
				"  min_size: 1KB\n" +
				"  max_size: 10MB\n" +
				"  dry_run: true\n" +
				"  mode: trash\n" +
				"  keep: newest\n",
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
