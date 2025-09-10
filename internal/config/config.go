// Package config provides a streamlined, extensible configuration management system.
// It supports multiple configuration sources with priority-based merging and validation.
package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/dr8co/doppel/internal/logger"
)

// Config represents the application configuration structure.
type Config struct {
	Log    LogConfig    `toml:"log" yaml:"log" json:"log"`
	Find   FindConfig   `toml:"find" yaml:"find" json:"find"`
	Preset PresetConfig `toml:"preset" yaml:"preset" json:"preset"`
}

// LogConfig holds the logging configuration.
type LogConfig struct {
	Level  string `toml:"level" yaml:"level" json:"level"`
	Format string `toml:"format" yaml:"format" json:"format"`
	Output string `toml:"output" yaml:"output" json:"output"`
}

// FindConfig holds configuration for the 'find' command.
type FindConfig struct {
	Workers          int    `toml:"workers" yaml:"workers" json:"workers"`
	Verbose          bool   `toml:"verbose" yaml:"verbose" json:"verbose"`
	ExcludeDirs      string `toml:"exclude_dirs" yaml:"exclude_dirs" json:"exclude_dirs"`
	ExcludeFiles     string `toml:"exclude_files" yaml:"exclude_files" json:"exclude_files"`
	ExcludeDirRegex  string `toml:"exclude_dir_regex" yaml:"exclude_dir_regex" json:"exclude_dir_regex"`
	ExcludeFileRegex string `toml:"exclude_file_regex" yaml:"exclude_file_regex" json:"exclude_file_regex"`
	MinSize          string `toml:"min_size" yaml:"min_size" json:"min_size"`
	MaxSize          string `toml:"max_size" yaml:"max_size" json:"max_size"`
	ShowFilters      bool   `toml:"show_filters" yaml:"show_filters" json:"show_filters"`
	OutputFormat     string `toml:"output_format" yaml:"output_format" json:"output_format"`
	OutputFile       string `toml:"output_file" yaml:"output_file" json:"output_file"`
}

// PresetConfig holds configuration for the 'preset' command.
type PresetConfig struct {
	Workers      int    `toml:"workers" yaml:"workers" json:"workers"`
	Verbose      bool   `toml:"verbose" yaml:"verbose" json:"verbose"`
	ShowFilters  bool   `toml:"show_filters" yaml:"show_filters" json:"show_filters"`
	OutputFormat string `toml:"output_format" yaml:"output_format" json:"output_format"`
	OutputFile   string `toml:"output_file" yaml:"output_file" json:"output_file"`
}

// Provider defines the interface for configuration providers.
type Provider interface {
	// Name returns the provider name for identification.
	Name() string
	// Priority returns the provider priority (higher numbers = higher priority).
	Priority() int
	// Load loads configuration from the provider.
	Load(ctx context.Context) (*Config, error)
}

// Validator defines the interface for configuration validation.
type Validator interface {
	Validate(config *Config) error
}

// Merger defines the interface for custom configuration merging.
type Merger interface {
	Merge(base, override *Config) *Config
}

// LoaderOptions configure the configuration loader.
type LoaderOptions struct {
	// Validator for configuration validation.
	Validator Validator
	// Merger for custom merging logic.
	Merger Merger
	// Timeout for provider operations.
	Timeout time.Duration
}

// Loader manages configuration loading from multiple providers.
type Loader struct {
	providers []Provider
	options   LoaderOptions
	mu        sync.RWMutex
}

// Default timeout for provider operations.
const defaultTimeout = 10 * time.Second

// NewLoader creates a new configuration loader.
func NewLoader(opts ...LoaderOption) *Loader {
	options := LoaderOptions{
		Validator: &defaultValidator{},
		Merger:    &defaultMerger{},
		Timeout:   defaultTimeout,
	}

	for _, opt := range opts {
		opt(&options)
	}

	return &Loader{
		providers: make([]Provider, 0),
		options:   options,
	}
}

// LoaderOption is a functional option for configuring the loader.
type LoaderOption func(*LoaderOptions)

// WithValidator sets a custom validator.
func WithValidator(v Validator) LoaderOption {
	return func(opts *LoaderOptions) {
		opts.Validator = v
	}
}

// WithMerger sets a custom merger.
func WithMerger(m Merger) LoaderOption {
	return func(opts *LoaderOptions) {
		opts.Merger = m
	}
}

// WithTimeout sets the operation timeout.
func WithTimeout(timeout time.Duration) LoaderOption {
	return func(opts *LoaderOptions) {
		opts.Timeout = timeout
	}
}

// AddProvider adds a configuration provider.
func (l *Loader) AddProvider(provider Provider) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Insert in priority order (higher priority first)
	inserted := false
	for i, p := range l.providers {
		if provider.Priority() > p.Priority() {
			l.providers = append(l.providers[:i], append([]Provider{provider}, l.providers[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		l.providers = append(l.providers, provider)
	}
}

// Load loads configuration from all providers.
func (l *Loader) Load(ctx context.Context) (*Config, error) {
	if l.options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, l.options.Timeout)
		defer cancel()
	}

	l.mu.RLock()
	providers := make([]Provider, len(l.providers))
	copy(providers, l.providers)
	l.mu.RUnlock()

	config := DefaultConfig()
	var errs []error

	// Load from providers in reverse priority order for merging
	for i := len(providers) - 1; i >= 0; i-- {
		provider := providers[i]
		providerConfig, err := provider.Load(ctx)
		if err != nil {
			logger.Error("Failed to load from provider",
				"provider", provider.Name(),
				"error", err)
			errs = append(errs, fmt.Errorf("provider %s: %w", provider.Name(), err))
			continue
		}

		if providerConfig != nil {
			config = l.options.Merger.Merge(config, providerConfig)
		}
	}

	// Validate the final configuration
	if err := l.options.Validator.Validate(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	if len(errs) > 0 {
		return config, fmt.Errorf("some providers failed to load: %v", errs)
	}

	return config, nil
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
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
	}
}

var (
	// Global loader instance.
	globalLoader *Loader
	globalOnce   sync.Once
)

// Initialize sets up the global configuration loader.
func Initialize(configDir string) error {
	var err error
	globalOnce.Do(func() {
		globalLoader = NewLoader()

		// Add providers in priority order
		tomlPath := filepath.Join(configDir, "config.toml")
		jsonPath := filepath.Join(configDir, "config.json")
		yamlPath := filepath.Join(configDir, "config.yaml")

		globalLoader.AddProvider(NewFileProvider(tomlPath, 10))
		globalLoader.AddProvider(NewFileProvider(jsonPath, 20))
		globalLoader.AddProvider(NewFileProvider(yamlPath, 30))
		globalLoader.AddProvider(NewEnvProvider("DOPPEL_", 40))

		// Load initial configuration
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
		_, err = globalLoader.Load(ctx)
	})
	return err
}

// Load returns the current global configuration.
func Load() (*Config, error) {
	if globalLoader == nil {
		return nil, errors.New("configuration not initialized, call Initialize() first")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return globalLoader.Load(ctx)
}

func init() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.TempDir()
		logger.Error("Could not get the user config directory. Using "+configDir, "error", err)
	}

	configDir = filepath.Join(configDir, "doppel")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		logger.Error("Could not create config directory", "error", err, "path", configDir)
	}

	if err := Initialize(configDir); err != nil {
		logger.Error("Failed to initialize configuration", "error", err)
	}
}
