package config

import (
	"runtime"
	"strings"
	"testing"
)

// TestDefaultValidator tests the defaultValidator.
func TestDefaultValidator(t *testing.T) {
	validator := &defaultValidator{}

	tests := []struct {
		name     string
		config   *Config
		wantErr  bool
		errField string
	}{
		{
			name: "valid config",
			config: &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "text",
					Output: "stdout",
				},
				Find: FindConfig{
					Workers:      runtime.NumCPU(),
					OutputFormat: "json",
				},
				Preset: PresetConfig{
					Workers:      runtime.NumCPU(),
					OutputFormat: "pretty",
				},
			},
			wantErr: false,
		},
		{
			name: "missing log level",
			config: &Config{
				Log: LogConfig{
					Format: "text",
					Output: "stdout",
				},
			},
			wantErr:  true,
			errField: "log level",
		},
		{
			name: "invalid log level",
			config: &Config{
				Log: LogConfig{
					Level:  "invalid",
					Format: "text",
					Output: "stdout",
				},
			},
			wantErr:  true,
			errField: "log level",
		},
		{
			name: "invalid log format",
			config: &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "invalid",
					Output: "stdout",
				},
			},
			wantErr:  true,
			errField: "log format",
		},
		{
			name: "too few workers in find config",
			config: &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "text",
				},
				Find: FindConfig{
					Workers: 0,
				},
			},
			wantErr:  true,
			errField: "too few workers",
		},
		{
			name: "too many workers in find config",
			config: &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "text",
				},
				Find: FindConfig{
					Workers: max(maxWorkers, runtime.NumCPU()) + 1,
				},
			},
			wantErr:  true,
			errField: "too many workers",
		},
		{
			name: "invalid output format in find config",
			config: &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "text",
				},
				Find: FindConfig{
					Workers:      runtime.NumCPU(),
					OutputFormat: "invalid",
				},
			},
			wantErr:  true,
			errField: "output format",
		},
		{
			name: "too few workers in preset config",
			config: &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "text",
				},
				Find: FindConfig{
					Workers: runtime.NumCPU(),
				},
				Preset: PresetConfig{
					Workers: 0,
				},
			},
			wantErr:  true,
			errField: "too few workers",
		},
		{
			name: "too many workers in preset config",
			config: &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "text",
				},
				Find: FindConfig{
					Workers: runtime.NumCPU(),
				},
				Preset: PresetConfig{
					Workers: max(maxWorkers, runtime.NumCPU()) + 1,
				},
			},
			wantErr:  true,
			errField: "too many workers",
		},
		{
			name: "invalid output format in preset config",
			config: &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "text",
				},
				Find: FindConfig{
					Workers: runtime.NumCPU(),
				},
				Preset: PresetConfig{
					Workers:      runtime.NumCPU(),
					OutputFormat: "invalid",
				},
			},
			wantErr:  true,
			errField: "output format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("%s: expected error but got none", tt.name)
				}
				if tt.errField != "" && !strings.Contains(err.Error(), tt.errField) {
					t.Errorf("%s: expected error containing %q, got %q", tt.name, tt.errField, err.Error())
				}
			} else if err != nil {
				t.Errorf("%s: unexpected error: %v", tt.name, err)
			}
		})
	}
}
