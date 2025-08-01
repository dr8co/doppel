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

	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/cmd"
	"github.com/dr8co/doppel/internal/logger"
)

const (
	version = "0.1.0"
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

	app := &cli.Command{
		Name:    "doppel",
		Usage:   "Find duplicate files across directories",
		Version: version,
		Authors: []any{
			"Ian Duncan <dr8co@duck.com>",
		},
		Copyright: "(c) 2025 Ian Duncan",
		Description: `A fast, concurrent duplicate file finder with advanced filtering capabilities.
		
This tool scans directories for duplicate files by comparing file sizes first, 
then computing Blake3 hashes for files of the same size. It supports parallel 
processing and extensive filtering options to skip unwanted files and directories.`,
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
		},
		Commands: []*cli.Command{
			cmd.FindCommand(),
			cmd.PresetCommand(),
		},
		DefaultCommand:        "find",
		Suggest:               true,
		EnableShellCompletion: true,
		Before: func(ctx context.Context, command *cli.Command) (context.Context, error) {
			logLevel := command.String("log-level")
			logFormat := command.String("log-format")
			logOutput := command.String("log-output")

			level := slog.LevelInfo
			addSource := false

			switch strings.ToLower(logLevel) {
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
				_, _ = fmt.Fprintf(os.Stderr, "Unknown log level '%s'. Using info level.\n", logLevel)
			}

			var err error
			var cfg logger.Config
			opts := &slog.HandlerOptions{Level: level, AddSource: addSource}

			cfg, closer, err = logger.NewConfig(opts, logFormat, logOutput)
			if err != nil {
				return ctx, err
			}

			err = logger.NewDefault(&cfg)

			return ctx, err
		},
	}

	if err := app.Run(ctx, os.Args); err != nil {
		fmt.Println()
		logger.Error("error running the application", "error", err)
		exit(1)
	}
}
