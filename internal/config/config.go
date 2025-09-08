// Package config provides configuration management functionality for doppel.
// It handles initialization and access to YAML and TOML configuration files
// located in the user's configuration directory.
package config

import (
	"os"
	"path/filepath"

	altsrc "github.com/urfave/cli-altsrc/v3"

	"github.com/dr8co/doppel/internal/logger"
)

var (
	// Yaml is the source for YAML configuration files.
	Yaml altsrc.Sourcer

	// Toml is the source for TOML configuration files.
	Toml altsrc.Sourcer
)

// Init initializes the configuration system by setting up the configuration directory
// and file paths for YAML and TOML configuration files. It performs the following steps:
//  1. Determines the user's configuration directory
//  2. Creates a "doppel" subdirectory if it doesn't exist
//  3. Sets up paths for YAML and TOML configuration files
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
	if err == nil {
		Yaml = altsrc.StringSourcer(filepath.Join(configDir, "config.yaml"))
		Toml = altsrc.StringSourcer(filepath.Join(configDir, "config.toml"))
	} else {
		logger.Error("Could not create the user config directory", "error", err, "path", configDir)
	}
}
