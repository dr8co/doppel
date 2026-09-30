package config

import (
	"reflect"
	"testing"
)

// TestDefaultMerger tests the defaultMerger.
func TestDefaultMerger(t *testing.T) {
	merger := &defaultMerger{}
	cleanDepth := 0

	tests := []struct {
		name     string
		base     *Config
		override *Config
		want     *Config
	}{
		{
			name: "merge log config",
			base: &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "text",
					Output: "stdout",
				},
			},
			override: &Config{
				Log: LogConfig{
					Level:  "debug",
					Format: "json",
				},
			},
			want: &Config{
				Log: LogConfig{
					Level:  "debug",
					Format: "json",
					Output: "stdout",
				},
			},
		},
		{
			name: "merge find config",
			base: &Config{
				Find: FindConfig{
					Workers:      4,
					Verbose:      false,
					ExcludeDirs:  "node_modules",
					OutputFormat: "pretty",
					Sort:         "path",
				},
			},
			override: &Config{
				Find: FindConfig{
					Workers:          8,
					Verbose:          true,
					ExcludeFiles:     "*.log",
					IgnoreHardlinks:  true,
					Sort:             "size",
					SortReverse:      true,
					PathsOnly:        true,
					Print0:           true,
					Quiet:            true,
					FailOnDuplicates: true,
				},
			},
			want: &Config{
				Find: FindConfig{
					Workers:          8,
					Verbose:          true,
					ExcludeDirs:      "node_modules",
					ExcludeFiles:     "*.log",
					IgnoreHardlinks:  true,
					OutputFormat:     "pretty",
					Sort:             "size",
					SortReverse:      true,
					PathsOnly:        true,
					Print0:           true,
					Quiet:            true,
					FailOnDuplicates: true,
				},
			},
		},
		{
			name: "merge clean config",
			base: &Config{Clean: CleanConfig{Mode: "delete"}},
			override: &Config{Clean: CleanConfig{
				Workers:         4,
				Verbose:         true,
				Quiet:           true,
				IgnoreHardlinks: true,
				Exclude:         "*.bak",
				MaxDepth:        &cleanDepth,
				ExcludeDirs:     ".git",
				ExcludeFiles:    "*.tmp",
				MinSize:         "1KB",
				MaxSize:         "10MB",
				DryRun:          true,
				Mode:            "trash",
				Keep:            "newest",
			}},
			want: &Config{Clean: CleanConfig{
				Workers:         4,
				Verbose:         true,
				Quiet:           true,
				IgnoreHardlinks: true,
				Exclude:         "*.bak",
				MaxDepth:        &cleanDepth,
				ExcludeDirs:     ".git",
				ExcludeFiles:    "*.tmp",
				MinSize:         "1KB",
				MaxSize:         "10MB",
				DryRun:          true,
				Mode:            "trash",
				Keep:            "newest",
			}},
		},
		{
			name: "empty override",
			base: &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "text",
					Output: "stdout",
				},
				Find: FindConfig{
					Workers: 4,
				},
			},
			override: &Config{},
			want: &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "text",
					Output: "stdout",
				},
				Find: FindConfig{
					Workers: 4,
				},
			},
		},
		{
			name: "empty base",
			base: &Config{},
			override: &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "text",
					Output: "stdout",
				},
				Find: FindConfig{
					Workers: 4,
				},
			},
			want: &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "text",
					Output: "stdout",
				},
				Find: FindConfig{
					Workers: 4,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := merger.Merge(tt.base, tt.override)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Merge() = %+v\nwant %+v", got, tt.want)
			}
		})
	}
}
