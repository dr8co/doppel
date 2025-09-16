package config

import (
	"reflect"
	"testing"
)

// TestDefaultMerger tests the defaultMerger.
func TestDefaultMerger(t *testing.T) {
	merger := &defaultMerger{}

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
				},
			},
			override: &Config{
				Find: FindConfig{
					Workers:      8,
					Verbose:      true,
					ExcludeFiles: "*.log",
				},
			},
			want: &Config{
				Find: FindConfig{
					Workers:      8,
					Verbose:      true,
					ExcludeDirs:  "node_modules",
					ExcludeFiles: "*.log",
					OutputFormat: "pretty",
				},
			},
		},
		{
			name: "merge preset config",
			base: &Config{
				Preset: PresetConfig{
					Workers:      4,
					Verbose:      false,
					ShowFilters:  false,
					OutputFormat: "pretty",
				},
			},
			override: &Config{
				Preset: PresetConfig{
					Workers:     8,
					Verbose:     true,
					ShowFilters: true,
				},
			},
			want: &Config{
				Preset: PresetConfig{
					Workers:      8,
					Verbose:      true,
					ShowFilters:  true,
					OutputFormat: "pretty",
				},
			},
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
				Preset: PresetConfig{
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
				Preset: PresetConfig{
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
				Preset: PresetConfig{
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
				Preset: PresetConfig{
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
