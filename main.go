package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/cmd"
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
		Commands: []*cli.Command{
			cmd.FindCommand(),
			cmd.PresetCommand(),
		},
		DefaultCommand: "find",
		Suggest:        true,
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
