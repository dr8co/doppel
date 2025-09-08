package cmd

import (
	"context"
	"runtime"

	"github.com/urfave/cli-altsrc/v3/toml"
	"github.com/urfave/cli-altsrc/v3/yaml"
	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/internal/config"
	"github.com/dr8co/doppel/internal/filter"
	"github.com/dr8co/doppel/internal/scanner"
)

// PresetCommand returns the preset command configuration.
func PresetCommand() *cli.Command {
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
				Sources: cli.NewValueSourceChain(toml.TOML("workers", config.Toml), yaml.YAML("workers", config.Yaml)),
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose output with detailed progress information",
				Sources: cli.NewValueSourceChain(toml.TOML("verbose", config.Toml), yaml.YAML("verbose", config.Yaml)),
			},
			&cli.BoolFlag{
				Name:    "show-filters",
				Usage:   "Show active filters and exit without scanning",
				Sources: cli.NewValueSourceChain(toml.TOML("show-filters", config.Toml), yaml.YAML("show-filters", config.Yaml)),
			},
			&cli.StringFlag{
				Name:    "output-format",
				Usage:   "Output format: pretty, json, yaml",
				Value:   "pretty",
				Sources: cli.NewValueSourceChain(toml.TOML("output-format", config.Toml), yaml.YAML("output-format", config.Yaml)),
			},
			&cli.StringFlag{
				Name:    "output-file",
				Usage:   "Write output to file (default: stdout)",
				Value:   "",
				Sources: cli.NewValueSourceChain(toml.TOML("output-file", config.Toml), yaml.YAML("output-file", config.Yaml)),
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "dev",
				Usage: "Development preset - skip build dirs, temp files, version control",
				Action: func(ctx context.Context, c *cli.Command) error {
					return findDuplicatesWithPreset(ctx, c, "dev")
				},
				Suggest:               true,
				EnableShellCompletion: true,
			},
			{
				Name:  "media",
				Usage: "Media preset - focus on images/videos, skip small files",
				Action: func(ctx context.Context, c *cli.Command) error {
					return findDuplicatesWithPreset(ctx, c, "media")
				},
				Suggest:               true,
				EnableShellCompletion: true,
			},
			{
				Name:  "docs",
				Usage: "Documents preset - focus on document files",
				Action: func(ctx context.Context, c *cli.Command) error {
					return findDuplicatesWithPreset(ctx, c, "docs")
				},
				Suggest:               true,
				EnableShellCompletion: true,
			},
			{
				Name:  "clean",
				Usage: "Clean preset - skip temporary and cache files",
				Action: func(ctx context.Context, c *cli.Command) error {
					return findDuplicatesWithPreset(ctx, c, "clean")
				},
				Suggest:               true,
				EnableShellCompletion: true,
			},
		},
	}
}

// findDuplicatesWithPreset finds duplicates using a specific preset configuration.
func findDuplicatesWithPreset(ctx context.Context, c *cli.Command, preset string) error {
	directories, err := scanner.GetDirectoriesFromArgs(c)
	if err != nil {
		return err
	}
	filterConfig := filter.GetPresetConfig(preset)

	return findDuplicates(ctx, c, directories, filterConfig)
}
