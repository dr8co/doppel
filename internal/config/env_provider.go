package config

import (
	"context"
	"os"
	"strconv"
	"strings"
)

// EnvProvider provides configuration from environment variables.
type EnvProvider struct {
	prefix   string
	priority int
}

// NewEnvProvider creates a new environment provider.
func NewEnvProvider(prefix string, priority int) *EnvProvider {
	return &EnvProvider{
		prefix:   prefix,
		priority: priority,
	}
}

// Name returns the provider name.
func (p *EnvProvider) Name() string {
	return "env:" + p.prefix
}

// Priority returns the provider priority.
func (p *EnvProvider) Priority() int {
	return p.priority
}

// Load loads configuration from the environment.
func (p *EnvProvider) Load(ctx context.Context) (*Config, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	config := &Config{
		Log:    LogConfig{},
		Find:   FindConfig{},
		Preset: PresetConfig{},
	}

	// Load log configuration
	p.loadStringFromEnv("LOG_LEVEL", &config.Log.Level)
	p.loadStringFromEnv("LOG_FORMAT", &config.Log.Format)
	p.loadStringFromEnv("LOG_OUTPUT", &config.Log.Output)

	// Load find configuration
	p.loadIntFromEnv("FIND_WORKERS", &config.Find.Workers)
	p.loadBoolFromEnv("FIND_VERBOSE", &config.Find.Verbose)
	p.loadStringFromEnv("FIND_EXCLUDE_DIRS", &config.Find.ExcludeDirs)
	p.loadStringFromEnv("FIND_EXCLUDE_FILES", &config.Find.ExcludeFiles)
	p.loadStringFromEnv("FIND_EXCLUDE_DIR_REGEX", &config.Find.ExcludeDirRegex)
	p.loadStringFromEnv("FIND_EXCLUDE_FILE_REGEX", &config.Find.ExcludeFileRegex)
	p.loadStringFromEnv("FIND_MIN_SIZE", &config.Find.MinSize)
	p.loadStringFromEnv("FIND_MAX_SIZE", &config.Find.MaxSize)
	p.loadBoolFromEnv("FIND_SHOW_FILTERS", &config.Find.ShowFilters)
	p.loadStringFromEnv("FIND_OUTPUT_FORMAT", &config.Find.OutputFormat)
	p.loadStringFromEnv("FIND_OUTPUT_FILE", &config.Find.OutputFile)

	// Load preset configuration
	p.loadIntFromEnv("PRESET_WORKERS", &config.Preset.Workers)
	p.loadBoolFromEnv("PRESET_VERBOSE", &config.Preset.Verbose)
	p.loadBoolFromEnv("PRESET_SHOW_FILTERS", &config.Preset.ShowFilters)
	p.loadStringFromEnv("PRESET_OUTPUT_FORMAT", &config.Preset.OutputFormat)
	p.loadStringFromEnv("PRESET_OUTPUT_FILE", &config.Preset.OutputFile)

	return config, nil
}

// loadStringFromEnv loads a string from the environment.
func (p *EnvProvider) loadStringFromEnv(key string, target *string) {
	if value := os.Getenv(p.prefix + key); value != "" {
		*target = value
	}
}

// loadIntFromEnv loads an int from the environment.
func (p *EnvProvider) loadIntFromEnv(key string, target *int) {
	if value := os.Getenv(p.prefix + key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			*target = parsed
		}
	}
}

// loadBoolFromEnv loads a bool from the environment.
func (p *EnvProvider) loadBoolFromEnv(key string, target *bool) {
	if value := os.Getenv(p.prefix + key); value != "" {
		lower := strings.ToLower(value)
		*target = lower == "true" || value == "1" || lower == "yes" || lower == "on"
	}
}
