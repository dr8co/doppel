package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
		Aliases: []string{"search", "f"},
		Usage:   "Find duplicate files in specified directories",
		Description: `Scan directories for duplicate files. If no directories are specified, 
the current directory is used. Files are compared using Blake3 hashes after 
initial size-based filtering.`,
		ArgsUsage:             "[directories...]",
		EnableShellCompletion: true,
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
			&cli.StringFlag{
				Name:  "output-format",
				Usage: "Output format: pretty, json",
				Value: "pretty",
			},
			&cli.StringFlag{
				Name:  "output-file",
				Usage: "Write output to file (default: stdout)",
				Value: "",
			},
		},
		Action: findDuplicatesCmd,
	}
}

// findDuplicatesCmd is the action function for the find command.
func findDuplicatesCmd(_ context.Context, c *cli.Command) error {
	directories, err := scanner.GetDirectoriesFromArgs(c)
	if err != nil {
		return err
	}

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

	return findDuplicates(c, directories, filterConfig)
}

// findDuplicates performs the main logic of finding duplicate files.
func findDuplicates(c *cli.Command, directories []string, filterConfig *config.FilterConfig) error {
	if c.Bool("show-filters") {
		config.DisplayFilterConfig(filterConfig)
		return nil
	}

	verbose := c.Bool("verbose")
	if verbose {
		fmt.Printf("🔍 Scanning directories: %v\n", directories)
		config.DisplayFilterConfig(filterConfig)
	}

	s := &stats.Stats{StartTime: time.Now()}

	// Phase 1: Group files by size
	sizeGroups, err := scanner.GroupFilesBySize(directories, filterConfig, s, verbose)
	if err != nil {
		return fmt.Errorf("error scanning files: %w", err)
	}

	if verbose {
		fmt.Printf("📊 Found %d files, %d size groups\n", s.TotalFiles, len(sizeGroups))
	}

	// Phase 2: Hash files that have potential duplicates
	workers := c.Int("workers")
	report, err := duplicate.FindDuplicatesByHash(sizeGroups, workers, s, verbose)
	if err != nil {
		return fmt.Errorf("error finding duplicates: %w", err)
	}
	s.Duration = time.Since(s.StartTime)

	// Phase 3: Output the results
	reg, err := display.InitFormatters()
	if err != nil {
		return fmt.Errorf("error initializing formatters: %w", err)
	}

	format := c.String("output-format")

	outputFile := c.String("output-file")
	var out io.Writer = os.Stdout

	if outputFile != "" {
		outputFile = filepath.Clean(outputFile)
		if outputFile == "." {
			outputFile = "doppel-report.txt"
		}

		outputFile, err = filepath.Abs(outputFile)
		if err != nil {
			return fmt.Errorf("error getting absolute path for output file: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
			return fmt.Errorf("error creating output directory: %w", err)
		}

		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("error opening output file: %w", err)
		}
		defer func(file *os.File) {
			_ = file.Close()
		}(file)

		out = file
	}

	err = reg.Format(format, report, out)
	if err != nil {
		return fmt.Errorf("error formatting report: %w", err)
	}

	if out != os.Stdout {
		fmt.Println("\n✅ Results written to", outputFile)
	}

	return nil
}
