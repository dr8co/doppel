// Doppel is a fast concurrent CLI tool for finding duplicate files across directories.
// It supports parallel processing and extensive filtering options to skip unwanted files and directories.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/cmd"
	"github.com/dr8co/doppel/internal/config"
	"github.com/dr8co/doppel/internal/logger"
	"github.com/dr8co/doppel/internal/pathutil"
)

const (
	version = "1.0.0"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var closer io.Closer
	closeLogFile := func() {
		if closer != nil {
			_ = closer.Close()
		}
	}
	defer closeLogFile()

	// exit function to handle a graceful shutdown
	exit := func(status int) {
		cancel()
		closeLogFile()
		os.Exit(status)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-c
		logger.InfoAttrs(ctx, "Received signal, shutting down", slog.String("signal", sig.String()))
		exit(1)
	}()

	appConfig, err := config.Load()
	if err != nil {
		logger.Error("failed to load the config", "error", err)
		exit(1)
	}

	app := &cli.Command{
		Name:    "doppel",
		Usage:   "Find duplicate files across directories",
		Version: version,
		Authors: []any{
			"Ian Duncan <dr8co@duck.com>",
		},
		Copyright: "(c) 2025 Ian Duncan",
		Description: `A fast, concurrent duplicate file finder with advanced filtering capabilities.
		
This tool scans directories for duplicate files using efficient hashing algorithms
and supports extensive filtering options to exclude unwanted files and directories.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Set the log level (debug, info, warn, error)",
				Value: "info",
			},
			&cli.StringFlag{
				Name:  "log-format",
				Usage: "Set the log format (text, json, pretty, discard)",
				Value: "pretty",
			},
			&cli.StringFlag{
				Name:  "log-output",
				Usage: "Set the log output (stdout, stderr, null, or file path)",
				Value: "stdout",
			},
			&cli.StringFlag{
				Name:  "config",
				Usage: "Path to the TOML configuration file",
			},
		},
		Commands: []*cli.Command{
			cmd.FindCommand(&appConfig.Find),
			cmd.PresetCommand(&appConfig.Preset),
		},
		DefaultCommand:        "find",
		Suggest:               true,
		EnableShellCompletion: true,
		Before: func(ctx context.Context, command *cli.Command) (context.Context, error) {
			if command.IsSet("config") {
				configPath, err := pathutil.ValidateRegularFile(command.String("config"))
				if err != nil {
					return ctx, fmt.Errorf("failed to parse the config: %w", err)
				}

				loader := config.NewLoader(
					config.WithTimeout(2 * time.Second),
				)
				loader.AddProvider(config.NewFileProvider(configPath, 10))
				loader.AddProvider(config.NewEnvProvider("DOPPEL_", 100))
				customConfig, err := loader.Load(ctx)
				if err != nil {
					return ctx, err
				}
				*appConfig = *customConfig
			}

			logCloser, newCtx, err := initialize(ctx, command, appConfig)
			closer = logCloser
			return newCtx, err
		},
	}

	if err := app.Run(ctx, os.Args); err != nil {
		fmt.Println()
		logger.Error("application error", "error", err)
		exit(1)
	}
}

// initialize sets up the logging system based on CLI flags and configuration.
func initialize(ctx context.Context, command *cli.Command, cfg *config.Config) (io.Closer, context.Context, error) {
	// Override with CLI flags
	if command.IsSet("log-level") {
		cfg.Log.Level = command.String("log-level")
	}
	if command.IsSet("log-format") {
		cfg.Log.Format = command.String("log-format")
	}
	if command.IsSet("log-output") {
		cfg.Log.Output = command.String("log-output")
	}

	// Set up logging
	level := slog.LevelInfo
	addSource := false

	switch strings.ToLower(cfg.Log.Level) {
	case "info", "":
		level = slog.LevelInfo
	case "debug":
		level = slog.LevelDebug
		addSource = true
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		_, _ = fmt.Fprintf(os.Stderr, "Unknown log level '%s'. Using info level.\n", cfg.Log.Level)
	}

	// Initialize the logger
	opts := &slog.HandlerOptions{Level: level, AddSource: addSource}
	logCfg, closer, err := logger.NewConfig(opts, cfg.Log.Format, cfg.Log.Output)
	if err != nil {
		return nil, ctx, err
	}

	err = logger.NewDefault(&logCfg)

	return closer, ctx, err
}
