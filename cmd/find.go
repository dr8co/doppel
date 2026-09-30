// Package cmd provides the command-line interface commands for the doppel duplicate file finder.
//
// This package implements the CLI commands using the urfave/cli framework, including
//   - find: The main command for finding duplicate files with extensive filtering options
//
// Each command supports various flags for controlling worker threads, output formats,
// filtering criteria, and other operational parameters.
//
//nolint:goconst
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/internal/config"
	"github.com/dr8co/doppel/internal/filter"
	"github.com/dr8co/doppel/internal/finder"
	"github.com/dr8co/doppel/internal/model"
	"github.com/dr8co/doppel/internal/output"
	"github.com/dr8co/doppel/internal/scanner"
)

// ConfigLoader loads the application configuration for a command action.
type ConfigLoader func(context.Context, *cli.Command) (*config.Config, error)

// ErrDuplicatesFound indicates that a scan found duplicates and the caller requested a failing status.
var ErrDuplicatesFound = errors.New("duplicates found")

// FindCommand returns the find command configuration.
func FindCommand(cfg *config.FindConfig, loadConfig ConfigLoader) *cli.Command {
	return &cli.Command{
		Name:    "find",
		Aliases: []string{"search", "f"},
		Usage:   "Find duplicate files in specified directories or files",
		Description: `Scan directories for duplicate files. If no directories are specified, 
only the current working directory is scanned.
With --files, positional arguments are treated as an explicit list of regular files and filters are ignored.
With --files-from, paths are read from a file or stdin and filters are ignored.
Files are compared by their hashes after filtration.`,
		ArgsUsage:             "[directories...] or --files [files...] or --files-from <file>",
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
				Name:  "files",
				Usage: "Treat positional arguments as explicit regular files and ignore all filters",
			},
			&cli.StringFlag{
				Name:  "files-from",
				Usage: "Read explicit regular file paths from a file, or '-' for stdin",
			},
			&cli.BoolFlag{
				Name:  "null",
				Usage: "Read NUL-delimited paths from --files-from",
			},
			&cli.BoolFlag{
				Name:  "ignore-empty-paths",
				Usage: "Ignore empty or whitespace-only paths from --files-from",
			},
			&cli.StringFlag{
				Name:  "exclude",
				Usage: "Comma-separated glob patterns to exclude files and directories",
			},
			&cli.IntFlag{
				Name:        "max-depth",
				Usage:       "Maximum containing-directory depth to scan (0 = root level)",
				DefaultText: "unlimited",
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
			&cli.BoolFlag{
				Name:  "ignore-hardlinks",
				Usage: "Treat hard-linked paths as one underlying file",
			},
			&cli.StringFlag{
				Name:  "output-format",
				Usage: "Output format: pretty, json, jsonl, yaml",
				Value: "pretty",
			},
			&cli.StringFlag{
				Name:  "sort",
				Usage: "Sort duplicate groups by path, size, wasted-space, or count",
				Value: "path",
			},
			&cli.BoolFlag{
				Name:  "reverse",
				Usage: "Reverse the selected sort order",
			},
			&cli.BoolFlag{
				Name:  "paths-only",
				Usage: "Output only paths from duplicate groups",
			},
			&cli.BoolFlag{
				Name:  "print0",
				Usage: "Terminate paths with NUL instead of newline",
			},
			&cli.BoolFlag{
				Name:  "quiet",
				Usage: "Suppress progress and informational output",
			},
			&cli.BoolFlag{
				Name:  "fail-on-duplicates",
				Usage: "Return a nonzero status when duplicates are found",
			},
			&cli.StringFlag{
				Name:  "output-file",
				Usage: "Write output to file (default: stdout)",
				Value: "",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			loaded, err := loadConfig(ctx, c)
			if err != nil {
				return err
			}
			*cfg = loaded.Find
			return findDuplicatesCmd(ctx, c, cfg)
		},
	}
}

// findDuplicatesCmd is the action function for the find command.
//
//nolint:gocyclo
func findDuplicatesCmd(ctx context.Context, c *cli.Command, cfg *config.FindConfig) error {
	// Override with CLI flags
	if c.IsSet("workers") {
		cfg.Workers = c.Int("workers")
	}
	if c.IsSet("verbose") {
		cfg.Verbose = c.Bool("verbose")
	}
	if c.IsSet("exclude") {
		cfg.Exclude = c.String("exclude")
	}
	if c.IsSet("max-depth") {
		maxDepth := c.Int("max-depth")
		cfg.MaxDepth = &maxDepth
	}
	if c.IsSet("exclude-dirs") {
		cfg.ExcludeDirs = c.String("exclude-dirs")
	}
	if c.IsSet("exclude-files") {
		cfg.ExcludeFiles = c.String("exclude-files")
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
	if c.IsSet("ignore-hardlinks") {
		cfg.IgnoreHardlinks = c.Bool("ignore-hardlinks")
	}
	if c.IsSet("output-file") {
		cfg.OutputFile = c.String("output-file")
	}
	if c.IsSet("output-format") {
		cfg.OutputFormat = c.String("output-format")
	}
	if c.IsSet("sort") {
		cfg.Sort = c.String("sort")
	}
	if c.IsSet("reverse") {
		cfg.SortReverse = c.Bool("reverse")
	}
	quiet := c.Bool("quiet")
	if quiet && cfg.Verbose {
		return errors.New("--quiet cannot be combined with --verbose")
	}
	if err := validateExclusionMode(cfg); err != nil {
		return err
	}
	if cfg.MaxDepth != nil && *cfg.MaxDepth < 0 {
		return fmt.Errorf("invalid --max-depth: %d (must be zero or greater)", *cfg.MaxDepth)
	}

	sortMode := strings.TrimSpace(strings.ToLower(cfg.Sort))
	if sortMode == "" {
		sortMode = "path"
	}
	if err := output.SortReport(&model.DuplicateReport{}, sortMode, false); err != nil {
		return fmt.Errorf("invalid --sort value %q: %w", cfg.Sort, err)
	}

	explicitFiles := c.Bool("files")
	var directories, files []string
	var err error
	filesFrom := c.String("files-from")
	filesFromSet := c.IsSet("files-from")
	if c.Bool("null") && !filesFromSet {
		return errors.New("--null requires --files-from")
	}
	//nolint:gocritic
	if filesFromSet {
		if explicitFiles {
			return errors.New("--files-from cannot be combined with --files")
		}
		if c.Args().Len() > 0 {
			return errors.New("--files-from cannot be combined with positional arguments")
		}
		if filesFrom == "" {
			return errors.New("--files-from requires a file path or '-' for stdin")
		}

		var input io.Reader = os.Stdin
		var inputFile *os.File
		if filesFrom != "-" {
			//nolint:gosec
			inputFile, err = os.Open(filesFrom)
			if err != nil {
				return fmt.Errorf("error opening file list %s: %w", filesFrom, err)
			}
			defer func() {
				_ = inputFile.Close()
			}()
			input = inputFile
		}
		files, err = scanner.GetFilesFromReader(input, c.Bool("null"), c.Bool("ignore-empty-paths"))
		if err != nil {
			return err
		}
		explicitFiles = true
	} else if explicitFiles {
		files, err = scanner.GetFilesFromArgs(c)
	} else {
		directories, err = scanner.GetDirectoriesFromArgs(c)
	}
	if err != nil {
		return err
	}

	pathsOnly := c.Bool("paths-only")
	print0 := c.Bool("print0")
	if print0 && !pathsOnly {
		return errors.New("--print0 requires --paths-only")
	}
	if pathsOnly && !strings.EqualFold(cfg.OutputFormat, "pretty") {
		return errors.New("--paths-only cannot be combined with --output-format")
	}
	outputOptions := outputOptions{
		pathsOnly:        pathsOnly,
		print0:           print0,
		quiet:            quiet,
		failOnDuplicates: c.Bool("fail-on-duplicates"),
	}

	if explicitFiles {
		return findDuplicates(ctx, cfg, directories, files, true, &filter.Config{}, outputOptions)
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
		minSize,
		maxSize,
		cfg.Exclude,
	)
	if err != nil {
		return fmt.Errorf("error building filter configuration: %w", err)
	}

	return findDuplicates(ctx, cfg, directories, nil, false, filterConfig, outputOptions)
}

func validateExclusionMode(cfg *config.FindConfig) error {
	if strings.TrimSpace(cfg.Exclude) == "" {
		return nil
	}
	for name, value := range map[string]string{
		"--exclude-dirs":  cfg.ExcludeDirs,
		"--exclude-files": cfg.ExcludeFiles,
	} {
		if strings.TrimSpace(value) != "" {
			return fmt.Errorf("--exclude cannot be combined with %s", name)
		}
	}
	return nil
}

type outputOptions struct {
	pathsOnly        bool
	print0           bool
	quiet            bool
	failOnDuplicates bool
}

// scanDuplicates performs the scanning and hashing phases of duplicate detection.
func scanDuplicates(ctx context.Context, cfg *config.FindConfig, directories, files []string, explicitFiles bool, filterConfig *filter.Config, quiet bool) (*model.DuplicateReport, error) {
	if cfg.ShowFilters && !explicitFiles {
		if !quiet {
			filter.DisplayActiveFiltersTo(filterConfig, os.Stderr)
		}
		return &model.DuplicateReport{ScanDate: time.Now(), Stats: &model.Stats{StartTime: time.Now()}}, nil
	}

	var progressOut io.Writer = os.Stderr
	if quiet {
		progressOut = io.Discard
	}
	verbose := cfg.Verbose && !quiet
	sp := spinner.New(spinner.CharSets[35], 100*time.Millisecond, spinner.WithSuffix(" scanning...\n"), spinner.WithWriter(progressOut))
	_ = sp.Color("fgHiRed", "bold")

	if verbose {
		if !explicitFiles {
			_, _ = fmt.Fprintf(progressOut, "🔍 Scanning directories: %v\n", directories)
			filter.DisplayActiveFiltersTo(filterConfig, progressOut)
		}
		sp.UpdateCharSet(spinner.CharSets[7])
	}

	if !quiet {
		sp.Start()
	}
	s := &model.Stats{StartTime: time.Now()}

	// Phase 1: Group files by size
	var sizeGroups map[int64][]scanner.FileInfo
	var err error
	if explicitFiles {
		sizeGroups, err = scanner.GroupFilesBySizeFromFilesWithOptions(files, s, scanner.ScanOptions{
			IgnoreHardlinks: cfg.IgnoreHardlinks,
		})
	} else {
		scanOptions := scanner.ScanOptions{IgnoreHardlinks: cfg.IgnoreHardlinks}
		if cfg.MaxDepth != nil {
			scanOptions.MaxDepth = *cfg.MaxDepth
			scanOptions.MaxDepthSet = true
		}
		sizeGroups, err = scanner.GroupFilesBySizeWithOptions(ctx, directories, filterConfig, s, verbose, progressOut, scanner.ScanOptions{
			IgnoreHardlinks: scanOptions.IgnoreHardlinks,
			MaxDepth:        scanOptions.MaxDepth,
			MaxDepthSet:     scanOptions.MaxDepthSet,
		})
	}
	if !quiet {
		sp.Stop()
	}
	if err != nil {
		return nil, fmt.Errorf("error scanning files: %w", err)
	}

	if verbose {
		if s.TotalFiles > 0 {
			n := len(sizeGroups)
			_, _ = fmt.Fprintf(progressOut, "📊 Found %d file%s, %d size group%s.\n", s.TotalFiles, pluralize(s.TotalFiles), n, pluralize(n))
		} else {
			_, _ = fmt.Fprintln(progressOut, " Did not find any regular files.")
		}
	}

	report, err := finder.FindDuplicatesByHashWithOutput(ctx, sizeGroups, cfg.Workers, s, verbose, progressOut)
	s.Duration = time.Since(s.StartTime)
	if err != nil {
		return nil, fmt.Errorf("error finding duplicates: %w", err)
	}

	return report, nil
}

// findDuplicates performs the main logic of finding duplicate files.
func findDuplicates(ctx context.Context, cfg *config.FindConfig, directories, files []string, explicitFiles bool, filterConfig *filter.Config, outputOptions outputOptions) error {
	report, err := scanDuplicates(ctx, cfg, directories, files, explicitFiles, filterConfig, outputOptions.quiet)
	if err != nil {
		return err
	}
	var progressOut io.Writer = os.Stderr
	if outputOptions.quiet {
		progressOut = io.Discard
	}

	sortMode := strings.TrimSpace(strings.ToLower(cfg.Sort))
	if sortMode == "" {
		sortMode = "path"
	}
	if err := output.SortReport(report, sortMode, cfg.SortReverse); err != nil {
		return fmt.Errorf("invalid --sort value %q: %w", cfg.Sort, err)
	}

	var reg *output.FormatterRegistry
	if !outputOptions.pathsOnly {
		reg, err = output.InitFormatters()
		if err != nil {
			return fmt.Errorf("error initializing formatters: %w", err)
		}
	}

	outputFile := cfg.OutputFile
	var out io.Writer = os.Stdout

	if outputFile != "" && strings.ToLower(outputFile) != "stdout" {
		if strings.ToLower(outputFile) == "stderr" {
			out = os.Stderr
		} else {
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
	}

	var sp2 *spinner.Spinner
	isFsFile := out != os.Stdout && out != os.Stderr
	if isFsFile && !outputOptions.quiet {
		sp2 = spinner.New(spinner.CharSets[70], 100*time.Millisecond, spinner.WithSuffix("  writing the results...\n"), spinner.WithWriter(progressOut))
		_ = sp2.Color("fgHiMagenta", "bold")
		sp2.Start()
		defer sp2.Stop()
	}

	if outputOptions.pathsOnly {
		err = output.WritePaths(report, out, outputOptions.print0)
	} else {
		err = reg.Format(cfg.OutputFormat, report, out)
	}
	if err != nil {
		return fmt.Errorf("error formatting report: %w", err)
	}

	if isFsFile && !outputOptions.quiet {
		sp2.Stop()
		_, _ = fmt.Fprintf(progressOut, "\n✅ Results written to \"%s\"", outputFile)
		_, _ = fmt.Fprintln(progressOut)
	}

	if outputOptions.failOnDuplicates && len(report.Groups) > 0 {
		return ErrDuplicatesFound
	}

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
