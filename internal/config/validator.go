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

	if err := v.validateCleanConfig(&config.Clean); err != nil {
		return fmt.Errorf("clean config validation failed: %w", err)
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
		validFormats := []string{"text", "json", "pretty", "discard"}
		if !contains(validFormats, config.Format) {
			return fmt.Errorf("invalid log format: %s, must be one of %v", config.Format, validFormats)
		}
	}

	return nil
}

// validateFindConfig validates the find configuration.
func (v *defaultValidator) validateFindConfig(config *FindConfig) error {
	if config.MaxDepth != nil && *config.MaxDepth < 0 {
		return fmt.Errorf("invalid max depth: %d (must be zero or greater)", *config.MaxDepth)
	}
	return validate(config.Workers, config.OutputFormat)
}

func (v *defaultValidator) validateCleanConfig(config *CleanConfig) error {
	validModes := []string{"delete", "trash", "replace-with-hardlink"}
	if config.Mode != "" && !contains(validModes, config.Mode) {
		return fmt.Errorf("invalid clean mode: %s, must be one of %v", config.Mode, validModes)
	}
	if config.Keep != "" {
		validPolicies := []string{"newest", "oldest", "shortest-path", "first"}
		if !contains(validPolicies, config.Keep) {
			return fmt.Errorf("invalid clean keep policy: %s, must be one of %v", config.Keep, validPolicies)
		}
	}
	return nil
}

// validate checks worker counts and output formats.
func validate(workers int, outputFormat string) error {
	if workers < minWorkers {
		return fmt.Errorf("too few workers: %d (min %d)", workers, minWorkers)
	}

	if workers > max(maxWorkers, runtime.NumCPU()) {
		return fmt.Errorf("too many workers: %d (max %d)", workers, max(maxWorkers, runtime.NumCPU()))
	}

	if outputFormat != "" {
		validFormats := []string{"json", "jsonl", "pretty", "yaml"}
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
