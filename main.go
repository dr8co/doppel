package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/cmd"
	"github.com/dr8co/doppel/internal/logger"
)

const (
	version = "0.1.0"
)

func main() {
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

	if err := app.Run(context.Background(), os.Args); err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}
