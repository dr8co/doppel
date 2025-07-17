package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
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

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-c
		logger.InfoAttrs(ctx, "received signal, shutting down", slog.String("signal", sig.String()))
		cancel()
		logger.Close()
		os.Exit(1)
	}()

	app := &cli.Command{
		Name:    "doppel",
		Usage:   "Find duplicate files across directories",
		Version: version,
		Authors: []any{
			"Ian Duncan",
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
				Usage: "Set the log format (text, json)",
				Value: "text",
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

			logger.InitLogger(logLevel, logFormat, logOutput)

			return ctx, nil
		},
	}

	defer logger.Close()

	if err := app.Run(ctx, os.Args); err != nil {
		logger.Error("error running the application", err)
		cancel()
		logger.Close()
		os.Exit(1)
	}
}
