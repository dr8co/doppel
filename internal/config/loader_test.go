package config

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"time"
)

// mockProvider implements Provider for testing.
type mockProvider struct {
	name     string
	priority int
	config   *Config
	err      error
}

func (p *mockProvider) Name() string  { return p.name }
func (p *mockProvider) Priority() int { return p.priority }
func (p *mockProvider) Load(ctx context.Context) (*Config, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return p.config, p.err
	}
}

// TestLoader tests the Loader type.
func TestLoader(t *testing.T) {
	// Test adding providers
	t.Run("add provider", func(t *testing.T) {
		loader := NewLoader()
		p1 := &mockProvider{name: "p1", priority: 1}
		p2 := &mockProvider{name: "p2", priority: 2}
		p3 := &mockProvider{name: "p3", priority: 3}

		// Add in mixed order, should be sorted by priority
		loader.AddProvider(p2)
		loader.AddProvider(p1)
		loader.AddProvider(p3)

		// Verify order (the highest priority first)
		expectedProviders := []Provider{p3, p2, p1}
		if len(loader.providers) != len(expectedProviders) {
			t.Errorf("AddProvider() got %d providers, want %d", len(loader.providers), len(expectedProviders))
			return
		}
		for i, want := range expectedProviders {
			if loader.providers[i] != want {
				t.Errorf("AddProvider() providers[%d] = %v, want %v", i, loader.providers[i], want)
			}
		}
	})

	// Test loading configurations
	t.Run("load configurations", func(t *testing.T) {
		baseConfig := DefaultConfig()
		overrideConfig := &Config{
			Log: LogConfig{
				Level: "debug",
			},
		}

		tests := []struct {
			name      string
			providers []Provider
			want      *Config
			wantErr   bool
		}{
			{
				name: "successful load",
				providers: []Provider{
					&mockProvider{
						name:     "base",
						priority: 1,
						config:   baseConfig,
					},
					&mockProvider{
						name:     "override",
						priority: 2,
						config:   overrideConfig,
					},
				},
				want: &Config{
					Log: LogConfig{
						Level:  "debug",
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
			{
				name: "provider error",
				providers: []Provider{
					&mockProvider{
						name:     "error",
						priority: 1,
						err:      fmt.Errorf("provider error"),
					},
				},
				wantErr: true,
			},
			{
				name: "context cancellation",
				providers: []Provider{
					&mockProvider{
						name:     "base",
						priority: 1,
						config:   baseConfig,
					},
				},
				wantErr: true, // We'll cancel the context
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				loader := NewLoader()
				for _, p := range tt.providers {
					loader.AddProvider(p)
				}

				var ctx context.Context
				var cancel context.CancelFunc
				if tt.name == "context cancellation" {
					ctx, cancel = context.WithCancel(context.Background())
					cancel()
				} else {
					ctx = context.Background()
				}

				got, err := loader.Load(ctx)
				if tt.wantErr {
					if err == nil {
						t.Error("Load() error = nil, want error")
					}
				} else {
					if err != nil {
						t.Errorf("Load() unexpected error = %v", err)
					} else if !reflect.DeepEqual(got, tt.want) {
						t.Errorf("Load() = %+v\nwant %+v", got, tt.want)
					}
				}

				if cancel != nil {
					cancel()
				}
			})
		}
	})

	// Test loader options
	t.Run("loader options", func(t *testing.T) {
		timeout := 5 * time.Second
		merger := &defaultMerger{}
		validator := &defaultValidator{}

		loader := NewLoader(
			WithTimeout(timeout),
			WithMerger(merger),
			WithValidator(validator),
		)

		if loader.options.Timeout != timeout {
			t.Errorf("WithTimeout() = %v, want %v", loader.options.Timeout, timeout)
		}
		if loader.options.Merger != merger {
			t.Error("WithMerger() did not set the correct merger")
		}
		if loader.options.Validator != validator {
			t.Error("WithValidator() did not set the correct validator")
		}
	})
}

// TestCreateLoader tests the createLoader function.
func TestCreateLoader(t *testing.T) {
	dir, cleanup := testDir(t)
	defer cleanup()

	testConfig := &Config{
		Log: LogConfig{
			Level:  "debug",
			Format: "json",
			Output: "app.log",
		},
	}

	// Write test config files
	writeConfigFile(t, dir, "config", "toml", `[log]
level = "debug"
format = "json"
output = "app.log"`)

	writeConfigFile(t, dir, "config", "json", `{
		"log": {
			"level": "debug",
			"format": "json",
			"output": "app.log"
		}
	}`)

	writeConfigFile(t, dir, "config", "yaml", `log:
  level: debug
  format: json
  output: app.log`)

	// Test createLoader
	loader, err := createLoader(dir)
	if err != nil {
		t.Errorf("createLoader() error = %v", err)
		return
	}

	// Test loading from the created loader
	ctx := context.Background()
	config, err := loader.Load(ctx)
	if err != nil {
		t.Errorf("Load() error = %v", err)
		return
	}
	if !reflect.DeepEqual(config.Log, testConfig.Log) {
		t.Errorf("Load() = %+v\nwant %+v", config.Log, testConfig.Log)
	}

	// Test Load() without initialization
	prevLoader := globalLoader
	globalLoader = nil
	defer func() { globalLoader = prevLoader }()

	if _, err := Load(); err == nil {
		t.Error("Load() without initialization = nil, want error")
	}
}
