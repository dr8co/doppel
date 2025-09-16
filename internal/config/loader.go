package config

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/dr8co/doppel/internal/logger"
)

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
const defaultTimeout = 3 * time.Second

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

var (
	// Global loader instance.
	globalLoader *Loader
	globalOnce   sync.Once
)

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

// createLoader creates and configures a new loader with the given config directory.
// This is the internal implementation used by Initialize and init.
func createLoader(configDir string) (*Loader, error) {
	loader := NewLoader()

	// Add providers in priority order
	tomlPath := filepath.Join(configDir, "config.toml")
	jsonPath := filepath.Join(configDir, "config.json")
	yamlPath := filepath.Join(configDir, "config.yaml")

	loader.AddProvider(NewFileProvider(yamlPath, 10))
	loader.AddProvider(NewFileProvider(tomlPath, 20))
	loader.AddProvider(NewFileProvider(jsonPath, 30))
	loader.AddProvider(NewEnvProvider("DOPPEL_", 40))

	// Load initial configuration
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	_, err := loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	return loader, nil
}

// Load returns the current global configuration.
func Load() (*Config, error) {
	if globalLoader == nil {
		return nil, errors.New("configuration not initialized, use custom loader via NewLoader()")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return globalLoader.Load(ctx)
}
