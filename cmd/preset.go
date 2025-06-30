package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/internal/config"
	"github.com/dr8co/doppel/internal/display"
	"github.com/dr8co/doppel/internal/scanner"
	"github.com/dr8co/doppel/internal/stats"
	"github.com/dr8co/doppel/pkg/duplicate"
)

// PresetCommand returns the preset command configuration
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
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "workers",
				Aliases: []string{"w"},
				Value:   4,
				Usage:   "Number of worker goroutines for parallel hashing",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose output",
			},
			&cli.BoolFlag{
				Name:  "stats",
				Usage: "Show detailed statistics",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "dev",
				Usage: "Development preset - skip build dirs, temp files, version control",
				Action: func(ctx context.Context, c *cli.Command) error {
					return findDuplicatesWithPreset(ctx, c, "dev")
				},
			},
			{
				Name:  "media",
				Usage: "Media preset - focus on images/videos, skip small files",
				Action: func(ctx context.Context, c *cli.Command) error {
					return findDuplicatesWithPreset(ctx, c, "media")
				},
			},
			{
				Name:  "docs",
				Usage: "Documents preset - focus on document files",
				Action: func(ctx context.Context, c *cli.Command) error {
					return findDuplicatesWithPreset(ctx, c, "docs")
				},
			},
			{
				Name:  "clean",
				Usage: "Clean preset - skip temporary and cache files",
				Action: func(ctx context.Context, c *cli.Command) error {
					return findDuplicatesWithPreset(ctx, c, "clean")
				},
			},
		},
	}
}

func findDuplicatesWithPreset(_ context.Context, c *cli.Command, preset string) error {
	filterConfig := config.GetPresetConfig(preset)

	directories := scanner.GetDirectoriesFromArgs(c)

	verbose := c.Bool("verbose")
	showStats := c.Bool("stats")
	workers := c.Int("workers")

	s := &stats.Stats{StartTime: time.Now()}

	if verbose {
		fmt.Printf("🔍 Using preset '%s' to scan directories: %v\n", preset, directories)
		config.DisplayFilterConfig(filterConfig)
	}

	// Phase 1: Group files by size
	sizeGroups, err := scanner.GroupFilesBySize(directories, filterConfig, s, verbose)
	if err != nil {
		return fmt.Errorf("error scanning files: %w", err)
	}

	if verbose {
		fmt.Printf("📊 Found %d files, %d size groups\n", s.TotalFiles, len(sizeGroups))
	}

	// Phase 2: Hash files that have potential duplicates
	duplicates, err := duplicate.FindDuplicatesByHash(sizeGroups, workers, s, verbose)
	if err != nil {
		return fmt.Errorf("error finding duplicates: %w", err)
	}

	// Phase 3: Display results
	display.ShowResults(duplicates, s, showStats || verbose)

	return nil
}
