package config

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// testDir creates a temporary directory for testing and returns its path and a cleanup function.
func testDir(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	return dir, func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("Failed to cleanup temp dir: %v", err)
		}
	}
}

// writeConfigFile writes config content to a file with the given name and format.
func writeConfigFile(t *testing.T, dir, name string, format string, content string) string {
	t.Helper()

	path := filepath.Join(dir, name+"."+format)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	return path
}

// withEnv sets environment variables for the duration of the test and restores them after.
func withEnv(t *testing.T, env map[string]string) func() {
	t.Helper()

	// Store original values
	originalValues := make(map[string]string)
	for k := range env {
		originalValues[k] = os.Getenv(k)
	}

	// Set new values
	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("Failed to set environment variable %s: %v", k, err)
		}
	}

	// Return cleanup function
	return func() {
		for k, v := range originalValues {
			var err error
			if v == "" {
				err = os.Unsetenv(k)
			} else {
				err = os.Setenv(k, v)
			}
			if err != nil {
				t.Errorf("Failed to restore environment variable %s: %v", k, err)
			}
		}
	}
}

// TestDefaultConfig tests the default configuration.
func TestDefaultConfig(t *testing.T) {
	tests := []struct {
		name string
		want *Config
	}{
		{
			name: "default configuration",
			want: &Config{
				Log: LogConfig{
					Level:  "info",
					Format: "pretty",
					Output: "stdout",
				},
				Find: FindConfig{
					Workers:      runtime.NumCPU(),
					OutputFormat: "pretty",
				},
				Preset: PresetConfig{
					Workers:      runtime.NumCPU(),
					OutputFormat: "pretty",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultConfig()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultConfig() = %+v\nwant %+v", got, tt.want)
			}
		})
	}
}
