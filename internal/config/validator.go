package config

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// defaultValidator provides comprehensive validation.
type defaultValidator struct{}

const (
	maxWorkers = 64
	minWorkers = 1
)

// Validate validates the configuration.
func (v *defaultValidator) Validate(config *Config) error {
	if err := v.validateLogConfig(&config.Log); err != nil {
		return fmt.Errorf("log config validation failed: %w", err)
	}

	if err := v.validateFindConfig(&config.Find); err != nil {
		return fmt.Errorf("find config validation failed: %w", err)
	}

	if err := v.validatePresetConfig(&config.Preset); err != nil {
		return fmt.Errorf("preset config validation failed: %w", err)
	}

	return nil
}

// validateLogConfig validates the log configuration.
func (v *defaultValidator) validateLogConfig(config *LogConfig) error {
	if config.Level == "" {
		return errors.New("log level is required")
	}

	validLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLevels, config.Level) {
		return fmt.Errorf("invalid log level: %s, must be one of %v", config.Level, validLevels)
	}

	if config.Format != "" {
		validFormats := []string{"json", "pretty"}
		if !contains(validFormats, config.Format) {
			return fmt.Errorf("invalid log format: %s, must be one of %v", config.Format, validFormats)
		}
	}

	return nil
}

// validateFindConfig validates the find configuration.
func (v *defaultValidator) validateFindConfig(config *FindConfig) error {
	return validate(config.Workers, config.OutputFormat)
}

// validatePresetConfig validates the preset configuration.
func (v *defaultValidator) validatePresetConfig(config *PresetConfig) error {
	return validate(config.Workers, config.OutputFormat)
}

// validate is a common validation function for both preset and find config.
func validate(workers int, outputFormat string) error {
	if workers < minWorkers {
		return fmt.Errorf("workers must be positive, got: %d", workers)
	}

	if workers > max(maxWorkers, runtime.NumCPU()) {
		return fmt.Errorf("workers too high: %d (max %d)", workers, max(maxWorkers, runtime.NumCPU()))
	}

	if outputFormat != "" {
		validFormats := []string{"json", "pretty", "yaml"}
		if !contains(validFormats, outputFormat) {
			return fmt.Errorf("invalid output format: %s, must be one of %v", outputFormat, validFormats)
		}
	}
	return nil
}

// contains returns true if the given string is in the slice.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
