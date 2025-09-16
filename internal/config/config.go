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
//
// Note: initialization of the package-global loader is performed in
// `init()` within `config.go`. For explicit/custom loading use a
// `NewLoader()` instance.
package config

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

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

	// Initialize the global loader once. If createLoader fails, we log
	// the error but continue; Load() will return an error if used before
	// successful initialization.
	globalOnce.Do(func() {
		var e error
		globalLoader, e = createLoader(configDir)
		if e != nil {
			logger.Error("Failed to initialize configuration", "error", e)
		}
	})
}
