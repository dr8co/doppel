// Package cmd provides the command-line interface commands for the doppel duplicate file finder.
//
// This package implements the CLI commands using the urfave/cli framework, including
//   - find: The main command for finding duplicate files with extensive filtering options
//   - preset: Command for using predefined filter configurations for common scenarios
//
// Each command supports various flags for controlling worker threads, output formats,
// filtering criteria, and other operational parameters.
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/urfave/cli-altsrc/v3/toml"
	"github.com/urfave/cli-altsrc/v3/yaml"
	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/internal/config"
	"github.com/dr8co/doppel/internal/filter"
	"github.com/dr8co/doppel/internal/finder"
	"github.com/dr8co/doppel/internal/model"
	"github.com/dr8co/doppel/internal/output"
	"github.com/dr8co/doppel/internal/scanner"
)

// FindCommand returns the find command configuration.
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
			&cli.StringFlag{
				Name:    "exclude-dirs",
				Usage:   "Comma-separated list of directory patterns to exclude (glob patterns)",
				Value:   "",
				Sources: cli.NewValueSourceChain(toml.TOML("exclude-dirs", config.Toml), yaml.YAML("exclude-dirs", config.Yaml)),
			},
			&cli.StringFlag{
				Name:    "exclude-files",
				Usage:   "Comma-separated list of file patterns to exclude (glob patterns)",
				Value:   "",
				Sources: cli.NewValueSourceChain(toml.TOML("exclude-files", config.Toml), yaml.YAML("exclude-files", config.Yaml)),
			},
			&cli.StringFlag{
				Name:    "exclude-dir-regex",
				Usage:   "Comma-separated list of regex patterns for directories to exclude",
				Value:   "",
				Sources: cli.NewValueSourceChain(toml.TOML("exclude-dir-regex", config.Toml), yaml.YAML("exclude-dir-regex", config.Yaml)),
			},
			&cli.StringFlag{
				Name:    "exclude-file-regex",
				Usage:   "Comma-separated list of regex patterns for files to exclude",
				Value:   "",
				Sources: cli.NewValueSourceChain(toml.TOML("exclude-file-regex", config.Toml), yaml.YAML("exclude-file-regex", config.Yaml)),
			},
			&cli.StringFlag{
				Name:    "min-size",
				Usage:   "Minimum file size (e.g., 10MB, 1.5GB, 500KiB) (0 = no limit)",
				Value:   "",
				Sources: cli.NewValueSourceChain(toml.TOML("min-size", config.Toml), yaml.YAML("min-size", config.Yaml)),
			},
			&cli.StringFlag{
				Name:    "max-size",
				Usage:   "Maximum file size (e.g., 100MB, 2GB, 1TiB) (0 = no limit)",
				Value:   "",
				Sources: cli.NewValueSourceChain(toml.TOML("max-size", config.Toml), yaml.YAML("max-size", config.Yaml)),
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
		Action: findDuplicatesCmd,
	}
}

// findDuplicatesCmd is the action function for the find command.
func findDuplicatesCmd(ctx context.Context, c *cli.Command) error {
	directories, err := scanner.GetDirectoriesFromArgs(c)
	if err != nil {
		return err
	}

	// Parse size strings to int64 bytes
	var minSize, maxSize int64
	if minSizeStr := c.String("min-size"); minSizeStr != "" {
		minSize, err = filter.ParseFileSize(minSizeStr)
		if err != nil {
			return fmt.Errorf("invalid min-size: %w", err)
		}
	}

	if maxSizeStr := c.String("max-size"); maxSizeStr != "" {
		maxSize, err = filter.ParseFileSize(maxSizeStr)
		if err != nil {
			return fmt.Errorf("invalid max-size: %w", err)
		}
	}

	// Build filter configuration
	filterConfig, err := filter.BuildConfig(
		c.String("exclude-dirs"),
		c.String("exclude-files"),
		c.String("exclude-dir-regex"),
		c.String("exclude-file-regex"),
		minSize,
		maxSize,
	)
	if err != nil {
		return fmt.Errorf("error building filter configuration: %w", err)
	}

	return findDuplicates(ctx, c, directories, filterConfig)
}

// findDuplicates performs the main logic of finding duplicate files.
func findDuplicates(ctx context.Context, c *cli.Command, directories []string, filterConfig *filter.Config) error {
	if c.Bool("show-filters") {
		filter.DisplayActiveFilters(filterConfig)
		return nil
	}

	verbose := c.Bool("verbose")
	if verbose {
		fmt.Printf("🔍 Scanning directories: %v\n", directories)
		filter.DisplayActiveFilters(filterConfig)
	}

	s := &model.Stats{StartTime: time.Now()}

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
	report, err := finder.FindDuplicatesByHash(ctx, sizeGroups, workers, s, verbose)
	s.Duration = time.Since(s.StartTime)
	if err != nil {
		return fmt.Errorf("error finding duplicates: %w", err)
	}

	// Phase 3: Output the results
	reg, err := output.InitFormatters()
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

		if err := os.MkdirAll(filepath.Dir(outputFile), 0o750); err != nil {
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
		fmt.Printf("\n✅ Results written to \"%s\"", outputFile)
	}
	fmt.Println()

	return nil
}
