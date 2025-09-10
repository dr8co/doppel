// Package config provides a streamlined, extensible configuration management system.
// It supports multiple configuration sources with priority-based merging and validation.
//
// The default configuration sources include environment variables and configuration files in the following formats:
//   - TOML
//   - JSON
//   - YAML
//
// Configuration is loaded from multiple providers, merged based on priority, and validated before use.
//
// The package offers sensible defaults but allows extensive customization through interfaces for providers,
// validators, and mergers.
//
// Custom configuration providers, validators, and mergers can be easily integrated.
// The package ensures thread-safe operations and is designed for high performance in concurrent environments.
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
	// Log holds the logging configuration.
	Log LogConfig `toml:"log" yaml:"log" json:"log"`

	// Find holds the 'find' command configuration.
	Find FindConfig `toml:"find" yaml:"find" json:"find"`

	// Preset holds the 'preset' command configuration.
	Preset PresetConfig `toml:"preset" yaml:"preset" json:"preset"`
}

// LogConfig holds the logging configuration.
type LogConfig struct {
	// Level sets the logging level (e.g., "debug", "info", "warn", "error").
	Level string `toml:"level" yaml:"level" json:"level"`

	// Format sets the logging format (e.g., "text", "json", "pretty", "discard").
	Format string `toml:"format" yaml:"format" json:"format"`

	// Output sets the logging output (e.g., "stdout", "stderr", "null", or file path).
	Output string `toml:"output" yaml:"output" json:"output"`
}

// FindConfig holds configuration for the 'find' command.
type FindConfig struct {
	// Workers sets the number of concurrent workers for file processing.
	// Default is the number of CPU cores.
	Workers int `toml:"workers" yaml:"workers" json:"workers"`

	// Verbose enables verbose output.
	Verbose bool `toml:"verbose" yaml:"verbose" json:"verbose"`

	// ExcludeDirs holds the glob patterns to exclude directories from searching.
	// This is a comma-separated list of patterns, which should be escaped as needed.
	ExcludeDirs string `toml:"exclude_dirs" yaml:"exclude_dirs" json:"exclude_dirs"`

	// ExcludeFiles holds the glob patterns to exclude files from searching.
	// This is a comma-separated list of patterns, which should be escaped as needed.
	ExcludeFiles string `toml:"exclude_files" yaml:"exclude_files" json:"exclude_files"`

	// ExcludeDirRegex holds regex patterns to exclude directories.
	// This is a comma-separated list of patterns, which should be escaped as needed.
	ExcludeDirRegex string `toml:"exclude_dir_regex" yaml:"exclude_dir_regex" json:"exclude_dir_regex"`

	// ExcludeFileRegex holds regex patterns to exclude files.
	// This is a comma-separated list of patterns, which should be escaped as needed.
	ExcludeFileRegex string `toml:"exclude_file_regex" yaml:"exclude_file_regex" json:"exclude_file_regex"`

	// MinSize sets the minimum file size to consider (e.g., "10KB", "5MB").
	MinSize string `toml:"min_size" yaml:"min_size" json:"min_size"`

	// MaxSize sets the maximum file size to consider (e.g., "100MB", "1GB").
	MaxSize string `toml:"max_size" yaml:"max_size" json:"max_size"`

	// ShowFilters enables displaying the active filters.
	ShowFilters bool `toml:"show_filters" yaml:"show_filters" json:"show_filters"`

	// OutputFormat sets the output format (e.g., "pretty", "json", "yaml").
	OutputFormat string `toml:"output_format" yaml:"output_format" json:"output_format"`

	// OutputFile sets the file to write output to (default is stdout).
	OutputFile string `toml:"output_file" yaml:"output_file" json:"output_file"`
}

// PresetConfig holds configuration for the 'preset' command.
type PresetConfig struct {
	// Workers sets the number of concurrent workers for file processing.
	// Default is the number of CPU cores.
	Workers int `toml:"workers" yaml:"workers" json:"workers"`

	// Verbose enables verbose output.
	Verbose bool `toml:"verbose" yaml:"verbose" json:"verbose"`

	// ShowFilters enables displaying the active filters.
	ShowFilters bool `toml:"show_filters" yaml:"show_filters" json:"show_filters"`

	// OutputFormat sets the output format (e.g., "pretty", "json", "yaml").
	OutputFormat string `toml:"output_format" yaml:"output_format" json:"output_format"`

	// OutputFile sets the file to write output to (default is stdout).
	OutputFile string `toml:"output_file" yaml:"output_file" json:"output_file"`
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

// defaultFindConfig returns a FindConfig instance with default settings.
func defaultFindConfig() FindConfig {
	return FindConfig{
		Workers:      runtime.NumCPU(),
		OutputFormat: "pretty",
	}
}

// defaultPresetConfig returns a PresetConfig instance with default settings.
func defaultPresetConfig() PresetConfig {
	return PresetConfig{
		Workers:      runtime.NumCPU(),
		OutputFormat: "pretty",
	}
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Log: LogConfig{
			Level:  "info",
			Format: "pretty",
			Output: "stdout",
		},
		Find:   defaultFindConfig(),
		Preset: defaultPresetConfig(),
	}
}

var (
	// Global loader instance.
	globalLoader *Loader
	globalOnce   sync.Once
)

// Initialize sets up the global configuration loader.
// It should be called once at application startup.
// The configuration providers are set up in the following priority order (highest to lowest):
//  1. Environment Variables (prefix "DOPPEL_")
//  2. JSON file (config.json)
//  3. TOML file (config.toml)
//  4. YAML file (config.yaml)
//
// The configDir parameter specifies the directory to look for configuration files.
func Initialize(configDir string) error {
	var err error
	globalOnce.Do(func() {
		globalLoader = NewLoader()

		// Add providers in priority order
		tomlPath := filepath.Join(configDir, "config.toml")
		jsonPath := filepath.Join(configDir, "config.json")
		yamlPath := filepath.Join(configDir, "config.yaml")

		globalLoader.AddProvider(NewFileProvider(yamlPath, 10))
		globalLoader.AddProvider(NewFileProvider(tomlPath, 20))
		globalLoader.AddProvider(NewFileProvider(jsonPath, 30))
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
