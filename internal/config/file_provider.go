package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// FileProvider provides configuration from files.
type FileProvider struct {
	path     string
	format   string
	priority int
}

// NewFileProvider creates a new file provider.
// The path is expected to be sanitized.
// The file format is inferred from the file extension, or assumed to be TOML if not available.
func NewFileProvider(path string, priority int) *FileProvider {
	ext := strings.ToLower(filepath.Ext(path))
	//nolint:goconst
	format := "toml"
	switch ext {
	case ".yaml", ".yml":
		format = "yaml"
	case ".json":
		format = "json"
	case ".toml":
		format = "toml"
	}

	return &FileProvider{
		path:     path,
		format:   format,
		priority: priority,
	}
}

// Name returns the provider name.
func (p *FileProvider) Name() string {
	return "file:" + p.path
}

// Priority returns the provider priority.
func (p *FileProvider) Priority() int {
	return p.priority
}

// Load loads configuration from the file.
func (p *FileProvider) Load(ctx context.Context) (*Config, error) {
	// Return empty config if the file doesn't exist (not an error)
	if _, err := os.Stat(p.path); os.IsNotExist(err) {
		return &Config{}, nil
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := os.ReadFile(p.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", p.path, err)
	}

	config := &Config{}
	switch p.format {
	case "toml":
		if err := toml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to decode TOML: %w", err)
		}
	case "yaml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to decode YAML: %w", err)
		}
	case "json":
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to decode JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported file format: %s", p.format)
	}

	return config, nil
}
