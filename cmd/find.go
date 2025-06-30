package cmd

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/internal/config"
	"github.com/dr8co/doppel/internal/display"
	"github.com/dr8co/doppel/internal/scanner"
	"github.com/dr8co/doppel/internal/stats"
	"github.com/dr8co/doppel/pkg/duplicate"
)

// FindCommand returns the find command configuration
func FindCommand() *cli.Command {
	return &cli.Command{
		Name:    "find",
		Aliases: []string{"f"},
		Usage:   "Find duplicate files in specified directories",
		Description: `Scan directories for duplicate files. If no directories are specified, 
the current directory is used. Files are compared using SHA-256 hashes after 
initial size-based filtering.`,
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
			&cli.StringFlag{
				Name:  "exclude-dirs",
				Usage: "Comma-separated list of directory patterns to exclude (glob patterns)",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "exclude-files",
				Usage: "Comma-separated list of file patterns to exclude (glob patterns)",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "exclude-dir-regex",
				Usage: "Comma-separated list of regex patterns for directories to exclude",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "exclude-file-regex",
				Usage: "Comma-separated list of regex patterns for files to exclude",
				Value: "",
			},
			&cli.Int64Flag{
				Name:  "min-size",
				Usage: "Minimum file size in bytes (0 = no limit)",
				Value: 0,
			},
			&cli.Int64Flag{
				Name:  "max-size",
				Usage: "Maximum file size in bytes (0 = no limit)",
				Value: 0,
			},
			&cli.BoolFlag{
				Name:  "show-filters",
				Usage: "Show active filters and exit without scanning",
			},
			&cli.BoolFlag{
				Name:  "stats",
				Usage: "Show detailed statistics at the end",
			},
		},
		Action: findDuplicates,
	}
}

func findDuplicates(_ context.Context, c *cli.Command) error {
	directories := scanner.GetDirectoriesFromArgs(c)

	// Build filter configuration
	filterConfig, err := config.BuildFilterConfig(
		c.String("exclude-dirs"),
		c.String("exclude-files"),
		c.String("exclude-dir-regex"),
		c.String("exclude-file-regex"),
		c.Int64("min-size"),
		c.Int64("max-size"),
	)
	if err != nil {
		return fmt.Errorf("error building filter configuration: %w", err)
	}

	if c.Bool("show-filters") {
		config.DisplayFilterConfig(filterConfig)
		return nil
	}

	verbose := c.Bool("verbose")
	showStats := c.Bool("stats")
	workers := c.Int("workers")

	s := &stats.Stats{StartTime: time.Now()}

	if verbose {
		fmt.Printf("🔍 Scanning directories: %v\n", directories)
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
