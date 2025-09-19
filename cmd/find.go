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

	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/internal/config"
	"github.com/dr8co/doppel/internal/filter"
	"github.com/dr8co/doppel/internal/finder"
	"github.com/dr8co/doppel/internal/model"
	"github.com/dr8co/doppel/internal/output"
	"github.com/dr8co/doppel/internal/scanner"
)

// FindCommand returns the find command configuration.
func FindCommand(cfg *config.FindConfig) *cli.Command {
	return &cli.Command{
		Name:    "find",
		Aliases: []string{"search", "f"},
		Usage:   "Find duplicate files in specified directories",
		Description: `Scan directories for duplicate files. If no directories are specified, 
only the current working directory is scanned.
Files are compared by their hashes after filtration.`,
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
			&cli.StringFlag{
				Name:    "exclude-dirs",
				Aliases: []string{"skip-dirs"},
				Usage:   "Comma-separated list of directory patterns to exclude (glob patterns)",
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "exclude-files",
				Aliases: []string{"skip-files"},
				Usage:   "Comma-separated list of file patterns to exclude (glob patterns)",
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "exclude-dirs-regex",
				Aliases: []string{"skip-dirs-regex"},
				Usage:   "Comma-separated list of regex patterns for directories to exclude",
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "exclude-files-regex",
				Aliases: []string{"skip-files-regex"},
				Usage:   "Comma-separated list of regex patterns for files to exclude",
				Value:   "",
			},
			&cli.StringFlag{
				Name:  "min-size",
				Usage: "Minimum file size (e.g., 10MB, 1.5GB, 500KiB) (0 = no limit)",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "max-size",
				Usage: "Maximum file size (e.g., 100MB, 2GB, 1TiB) (0 = no limit)",
				Value: "",
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
		Action: func(ctx context.Context, c *cli.Command) error {
			return findDuplicatesCmd(ctx, c, cfg)
		},
	}
}

// findDuplicatesCmd is the action function for the find command.
func findDuplicatesCmd(ctx context.Context, c *cli.Command, cfg *config.FindConfig) error {
	// Override with CLI flags
	if c.IsSet("workers") {
		cfg.Workers = c.Int("workers")
	}
	if c.IsSet("verbose") {
		cfg.Verbose = c.Bool("verbose")
	}
	if c.IsSet("exclude-dirs") {
		cfg.ExcludeDirs = c.String("exclude-dirs")
	}
	if c.IsSet("exclude-files") {
		cfg.ExcludeFiles = c.String("exclude-files")
	}
	if c.IsSet("exclude-dir-regex") {
		cfg.ExcludeDirRegex = c.String("exclude-dir-regex")
	}
	if c.IsSet("exclude-file-regex") {
		cfg.ExcludeFileRegex = c.String("exclude-file-regex")
	}
	if c.IsSet("min-size") {
		cfg.MinSize = c.String("min-size")
	}
	if c.IsSet("max-size") {
		cfg.MaxSize = c.String("max-size")
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

	directories, err := scanner.GetDirectoriesFromArgs(c)
	if err != nil {
		return err
	}

	// Parse size strings to int64 bytes
	var minSize, maxSize int64
	if cfg.MinSize != "" {
		minSize, err = filter.ParseFileSize(cfg.MinSize)
		if err != nil {
			return fmt.Errorf("invalid min-size: %w", err)
		}
	}

	if cfg.MaxSize != "" {
		maxSize, err = filter.ParseFileSize(cfg.MaxSize)
		if err != nil {
			return fmt.Errorf("invalid max-size: %w", err)
		}
	}

	// Build filter configuration
	filterConfig, err := filter.BuildConfig(
		cfg.ExcludeDirs,
		cfg.ExcludeFiles,
		cfg.ExcludeDirRegex,
		cfg.ExcludeFileRegex,
		minSize,
		maxSize,
	)
	if err != nil {
		return fmt.Errorf("error building filter configuration: %w", err)
	}

	return findDuplicates(ctx, cfg, directories, filterConfig)
}

// findDuplicates performs the main logic of finding duplicate files.
func findDuplicates(ctx context.Context, cfg *config.FindConfig, directories []string, filterConfig *filter.Config) error {
	if cfg.ShowFilters {
		filter.DisplayActiveFilters(filterConfig)
		return nil
	}

	if cfg.Verbose {
		fmt.Printf("🔍 Scanning directories: %v\n", directories)
		filter.DisplayActiveFilters(filterConfig)
	}

	s := &model.Stats{StartTime: time.Now()}

	// Phase 1: Group files by size
	sizeGroups, err := scanner.GroupFilesBySize(ctx, directories, filterConfig, s, cfg.Verbose)
	if err != nil {
		return fmt.Errorf("error scanning files: %w", err)
	}

	if cfg.Verbose {
		if s.TotalFiles > 0 {
			n := len(sizeGroups)
			fmt.Printf("📊 Found %d file%s, %d size group%s.\n", s.TotalFiles, pluralize(s.TotalFiles), n, pluralize(n))
		} else {
			fmt.Println(" Did not find any regular files.")
		}
	}

	// Phase 2: Hash files that have potential duplicates
	report, err := finder.FindDuplicatesByHash(ctx, sizeGroups, cfg.Workers, s, cfg.Verbose)
	s.Duration = time.Since(s.StartTime)
	if err != nil {
		return fmt.Errorf("error finding duplicates: %w", err)
	}

	// Phase 3: Output the results
	reg, err := output.InitFormatters()
	if err != nil {
		return fmt.Errorf("error initializing formatters: %w", err)
	}

	outputFile := cfg.OutputFile
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

	err = reg.Format(cfg.OutputFormat, report, out)
	if err != nil {
		return fmt.Errorf("error formatting report: %w", err)
	}

	if out != os.Stdout {
		fmt.Printf("\n✅ Results written to \"%s\"", outputFile)
	}
	fmt.Println()

	return nil
}

type integral interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func pluralize[T integral](num T) string {
	if num < 2 {
		return ""
	}
	return "s"
}
