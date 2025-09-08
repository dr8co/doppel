package cmd

import (
	"context"
	"runtime"

	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/internal/config"
	"github.com/dr8co/doppel/internal/filter"
	"github.com/dr8co/doppel/internal/scanner"
)

// PresetCommand returns the preset command configuration.
func PresetCommand(cfg *config.PresetConfig) *cli.Command {
	return &cli.Command{
		Name:    "preset",
		Aliases: []string{"p"},
		Usage:   "Use predefined filter presets",
		Description: `Apply common filter presets for different scenarios:
  - dev: Skip development directories and files
  - media: Focus on media files, skip small files
  - docs: Focus on document files
  - clean: Skip temporary and cache files`,
		ArgsUsage:             "[directories...]",
		EnableShellCompletion: true,
		Suggest:               true,

		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "workers",
				Aliases: []string{"w"},
				Value:   runtime.NumCPU(),
				Usage:   "Number of worker goroutines for parallel hashing",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose output with detailed progress information",
			},
			&cli.BoolFlag{
				Name:  "show-filters",
				Usage: "Show active filters and exit without scanning",
			},
			&cli.StringFlag{
				Name:  "output-format",
				Usage: "Output format: pretty, json, yaml",
				Value: "pretty",
			},
			&cli.StringFlag{
				Name:  "output-file",
				Usage: "Write output to file (default: stdout)",
				Value: "",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "dev",
				Usage: "Development preset - skip build dirs, temp files, version control",
				Action: func(ctx context.Context, c *cli.Command) error {
					return findDuplicatesWithPreset(ctx, c, cfg, "dev")
				},
				Suggest:               true,
				EnableShellCompletion: true,
			},
			{
				Name:  "media",
				Usage: "Media preset - focus on images/videos, skip small files",
				Action: func(ctx context.Context, c *cli.Command) error {
					return findDuplicatesWithPreset(ctx, c, cfg, "media")
				},
				Suggest:               true,
				EnableShellCompletion: true,
			},
			{
				Name:  "docs",
				Usage: "Documents preset - focus on document files",
				Action: func(ctx context.Context, c *cli.Command) error {
					return findDuplicatesWithPreset(ctx, c, cfg, "docs")
				},
				Suggest:               true,
				EnableShellCompletion: true,
			},
			{
				Name:  "clean",
				Usage: "Clean preset - skip temporary and cache files",
				Action: func(ctx context.Context, c *cli.Command) error {
					return findDuplicatesWithPreset(ctx, c, cfg, "clean")
				},
				Suggest:               true,
				EnableShellCompletion: true,
			},
		},
	}
}

// findDuplicatesWithPreset finds duplicates using a specific preset configuration.
func findDuplicatesWithPreset(ctx context.Context, c *cli.Command, cfg *config.PresetConfig, preset string) error {
	directories, err := scanner.GetDirectoriesFromArgs(c)
	if err != nil {
		return err
	}
	filterConfig := filter.GetPresetConfig(preset)

	// Override with CLI flags
	if c.IsSet("workers") {
		cfg.Workers = c.Int("workers")
	}
	if c.IsSet("verbose") {
		cfg.Verbose = c.Bool("verbose")
	}
	if c.IsSet("show-filters") {
		cfg.ShowFilters = c.Bool("show-filters")
	}
	if c.IsSet("output-file") {
		cfg.OutputFile = c.String("output-file")
	}
	if c.IsSet("output-format") {
		cfg.OutputFormat = c.String("output-format")
	}

	cfg2 := config.FindConfig{
		Workers:      cfg.Workers,
		Verbose:      cfg.Verbose,
		ShowFilters:  cfg.ShowFilters,
		OutputFile:   cfg.OutputFile,
		OutputFormat: cfg.OutputFormat,
	}

	return findDuplicates(ctx, &cfg2, directories, filterConfig)
}
