// Package config provides configuration management functionality for doppel.
// It handles initialization and access to YAML and TOML configuration files
// located in the user's configuration directory.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/dr8co/doppel/internal/logger"
)

// tomlConfig is the path to the TOML configuration file.
var (
	tomlConfig string
	mu         sync.RWMutex
)

// DoppelConfig holds the configuration for doppel.
type DoppelConfig struct {
	LogLevel  string `toml:"log_level" json:"log_level" yaml:"log_level"`
	LogFormat string `toml:"log_format" json:"log_format" yaml:"log_format"`
	LogOutput string `toml:"log_output" json:"log_output" yaml:"log_output"`
	Find      *FindConfig
	Preset    *PresetConfig
}

// FindConfig holds the configuration for the find command.
type FindConfig struct {
	Workers          int    `toml:"workers" json:"workers" yaml:"workers"`
	Verbose          bool   `toml:"verbose" json:"verbose" yaml:"verbose"`
	ExcludeDirs      string `toml:"exclude_dirs" json:"exclude_dirs" yaml:"exclude_dirs"`
	ExcludeFiles     string `toml:"exclude_files" json:"exclude_files" yaml:"exclude_files"`
	ExcludeDirRegex  string `toml:"exclude_dir_regex" json:"exclude_dir_regex" yaml:"exclude_dir_regex"`
	ExcludeFileRegex string `toml:"exclude_file_regex" json:"exclude_file_regex" yaml:"exclude_file_regex"`
	MinSize          string `toml:"min_size" json:"min_size" yaml:"min_size"`
	MaxSize          string `toml:"max_size" json:"max_size" yaml:"max_size"`
	ShowFilters      bool   `toml:"show_filters" json:"show_filters" yaml:"show_filters"`
	OutputFormat     string `toml:"output_format" json:"output_format" yaml:"output_format"`
	OutputFile       string `toml:"output_file" json:"output_file" yaml:"output_file"`
}

// PresetConfig holds the configuration for the preset command.
type PresetConfig struct {
	Workers      int    `toml:"workers" json:"workers" yaml:"workers"`
	Verbose      bool   `toml:"verbose" json:"verbose" yaml:"verbose"`
	ShowFilters  bool   `toml:"show_filters" json:"show_filters" yaml:"show_filters"`
	OutputFormat string `toml:"output_format" json:"output_format" yaml:"output_format"`
	OutputFile   string `toml:"output_file" json:"output_file" yaml:"output_file"`
}

// LoadConfig loads the configuration from the TOML configuration file.
func LoadConfig() (*DoppelConfig, error) {
	config := Default()
	_, err := toml.DecodeFile(getConfigFile(), config)
	if err != nil {
		return Default(), fmt.Errorf("error decoding config file: %w", err)
	}
	return config, nil
}

// Default returns the default configuration.
func Default() *DoppelConfig {
	return &DoppelConfig{
		LogLevel:  "info",
		LogFormat: "pretty",
		LogOutput: "stdout",
		Find: &FindConfig{
			Workers:      runtime.NumCPU(),
			OutputFormat: "pretty",
		},
		Preset: &PresetConfig{
			Workers:      runtime.NumCPU(),
			OutputFormat: "pretty",
		},
	}
}

// SetConfigFile sets the path to the TOML configuration file.
func SetConfigFile(path string) {
	mu.Lock()
	defer mu.Unlock()
	tomlConfig = filepath.Clean(path)
}

// getConfigFile returns the path to the TOML configuration file.
func getConfigFile() string {
	mu.RLock()
	defer mu.RUnlock()
	return tomlConfig
}

// Init initializes the configuration system by setting up the configuration file.
// It performs the following steps:
//  1. Determines the user's configuration directory
//  2. Creates a "doppel" subdirectory if it doesn't exist
//  3. Sets up the path for the TOML configuration file
//
// The function logs errors if it cannot:
//   - Get the user configuration directory
//   - Create the "doppel" configuration directory
func init() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		logger.Error("Could not get the user config directory", "error", err)
		return
	}

	configDir = filepath.Join(configDir, "doppel")
	err = os.MkdirAll(configDir, 0o750)
	if err != nil {
		logger.Error("Could not create the user config directory", "error", err, "path", configDir)
	}
	// We still want to set the config file path even if we can't create the directory
	tomlConfig = filepath.Join(configDir, "config.toml")
}
