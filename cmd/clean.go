package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/internal/clean"
	"github.com/dr8co/doppel/internal/config"
	"github.com/dr8co/doppel/internal/filter"
	"github.com/dr8co/doppel/internal/scanner"
)

// CleanCommand returns the CLI command that scans for duplicate files and
// applies the configured cleanup action.
func CleanCommand(cfg *config.CleanConfig, loadConfig ConfigLoader) *cli.Command {
	return &cli.Command{
		Name:      "clean",
		Usage:     "Remove duplicate files according to an explicit retention policy",
		ArgsUsage: "[directories...] or --files [files...] or --files-from <file>",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "workers", Aliases: []string{"w"}, Value: runtime.NumCPU(), Usage: "Number of worker goroutines for hashing"},
			&cli.BoolFlag{Name: "verbose", Aliases: []string{"v"}, Usage: "Enable verbose scan output"},
			&cli.BoolFlag{Name: "quiet", Usage: "Suppress scan progress"},
			&cli.BoolFlag{Name: "files", Usage: "Treat positional arguments as explicit regular files"},
			&cli.StringFlag{Name: "files-from", Usage: "Read regular file paths from a file, or '-' for stdin"},
			&cli.BoolFlag{Name: "null", Usage: "Read NUL-delimited paths from --files-from"},
			&cli.BoolFlag{Name: "ignore-empty-paths", Usage: "Ignore empty paths from --files-from"},
			&cli.BoolFlag{Name: "ignore-hardlinks", Usage: "Treat hard-linked paths as one underlying file"},
			&cli.StringFlag{Name: "exclude", Usage: "Comma-separated glob patterns to exclude files and directories"},
			&cli.IntFlag{Name: "max-depth", Usage: "Maximum containing-directory depth to scan (0 = root level)", DefaultText: "unlimited"},
			&cli.StringFlag{Name: "exclude-dirs", Usage: "Comma-separated directory glob patterns to exclude"},
			&cli.StringFlag{Name: "exclude-files", Usage: "Comma-separated file glob patterns to exclude"},
			&cli.StringFlag{Name: "min-size", Usage: "Minimum file size"},
			&cli.StringFlag{Name: "max-size", Usage: "Maximum file size"},
			&cli.StringFlag{Name: "mode", Value: "delete", Usage: "Action: delete, trash, or replace-with-hardlink"},
			&cli.StringFlag{Name: "keep", Usage: "Required retention policy: newest, oldest, shortest-path, or first"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Show selected actions without changing files"},
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip interactive confirmation"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			loaded, err := loadConfig(ctx, c)
			if err != nil {
				return err
			}
			*cfg = loaded.Clean
			return cleanDuplicatesCmd(ctx, c, loaded)
		},
	}
}

//nolint:gocyclo
func cleanDuplicatesCmd(ctx context.Context, c *cli.Command, loaded *config.Config) error {
	cleanCfg := loaded.Clean
	findCfg := loaded.Find
	if cleanCfg.Workers != 0 {
		findCfg.Workers = cleanCfg.Workers
	}
	if cleanCfg.Verbose {
		findCfg.Verbose = true
	}
	if cleanCfg.Quiet {
		findCfg.Quiet = true
	}
	if cleanCfg.Exclude != "" {
		findCfg.Exclude = cleanCfg.Exclude
	}
	if cleanCfg.MaxDepth != nil {
		maxDepth := *cleanCfg.MaxDepth
		findCfg.MaxDepth = &maxDepth
	}
	if cleanCfg.ExcludeDirs != "" {
		findCfg.ExcludeDirs = cleanCfg.ExcludeDirs
	}
	if cleanCfg.ExcludeFiles != "" {
		findCfg.ExcludeFiles = cleanCfg.ExcludeFiles
	}
	if cleanCfg.MinSize != "" {
		findCfg.MinSize = cleanCfg.MinSize
	}
	if cleanCfg.MaxSize != "" {
		findCfg.MaxSize = cleanCfg.MaxSize
	}
	if cleanCfg.IgnoreHardlinks {
		findCfg.IgnoreHardlinks = true
	}
	dryRun := cleanCfg.DryRun
	if c.IsSet("dry-run") {
		dryRun = c.Bool("dry-run")
	}
	if c.IsSet("mode") {
		cleanCfg.Mode = c.String("mode")
	}
	if c.IsSet("keep") {
		cleanCfg.Keep = c.String("keep")
	}
	if err := clean.ValidatePolicy(cleanCfg.Keep); err != nil {
		return err
	}
	if err := clean.ValidateMode(cleanCfg.Mode); err != nil {
		return err
	}
	if c.IsSet("workers") {
		findCfg.Workers = c.Int("workers")
	}
	if c.IsSet("verbose") {
		findCfg.Verbose = c.Bool("verbose")
	}
	if c.IsSet("quiet") {
		findCfg.Quiet = c.Bool("quiet")
	}
	if c.IsSet("exclude") {
		findCfg.Exclude = c.String("exclude")
	}
	if c.IsSet("max-depth") {
		maxDepth := c.Int("max-depth")
		findCfg.MaxDepth = &maxDepth
	}
	if c.IsSet("exclude-dirs") {
		findCfg.ExcludeDirs = c.String("exclude-dirs")
	}
	if c.IsSet("exclude-files") {
		findCfg.ExcludeFiles = c.String("exclude-files")
	}
	if c.IsSet("min-size") {
		findCfg.MinSize = c.String("min-size")
	}
	if c.IsSet("max-size") {
		findCfg.MaxSize = c.String("max-size")
	}
	if c.IsSet("ignore-hardlinks") {
		findCfg.IgnoreHardlinks = c.Bool("ignore-hardlinks")
	}
	if findCfg.Quiet && findCfg.Verbose {
		return errors.New("--quiet cannot be combined with --verbose")
	}
	if err := validateExclusionMode(&findCfg); err != nil {
		return err
	}
	if findCfg.MaxDepth != nil && *findCfg.MaxDepth < 0 {
		return fmt.Errorf("invalid --max-depth: %d (must be zero or greater)", *findCfg.MaxDepth)
	}

	directories, files, explicitFiles, filterConfig, err := cleanInputs(c, &findCfg)
	if err != nil {
		return err
	}
	report, err := scanDuplicates(ctx, &findCfg, directories, files, explicitFiles, filterConfig, findCfg.Quiet)
	if err != nil {
		return err
	}
	targets, err := clean.SelectTargets(report, cleanCfg.Keep)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No duplicate files require action.")
		return nil
	}

	if !dryRun && !c.Bool("yes") {
		if !isTerminal(os.Stdin) {
			return errors.New("clean requires --yes when standard input is not a terminal")
		}
		_, _ = fmt.Fprintf(os.Stderr, "About to %s %d duplicate file(s), keeping one per group. Continue? [y/N] ", cleanCfg.Mode, len(targets))
		answer, readErr := bufio.NewReader(os.Stdin).ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return fmt.Errorf("read confirmation: %w", readErr)
		}
		if !strings.EqualFold(strings.TrimSpace(answer), "y") && !strings.EqualFold(strings.TrimSpace(answer), "yes") {
			_, _ = fmt.Fprintln(os.Stderr, "No changes made.")
			return nil
		}
	}

	results := clean.Apply(targets, cleanCfg.Mode, dryRun)
	failed := 0
	for _, result := range results {
		if result.Err != nil {
			failed++
			_, _ = fmt.Fprintf(os.Stdout, "error: %s -> %s: %v\n", result.Target.Remove.Path, result.Target.Keeper.Path, result.Err)
			continue
		}
		_, _ = fmt.Fprintf(os.Stdout, "%s: %s -> keep %s\n", cleanCfg.Mode, result.Target.Remove.Path, result.Target.Keeper.Path)
	}
	_, _ = fmt.Fprintf(os.Stdout, "Processed %d target(s), %d failed.\n", len(results), failed)
	if failed > 0 {
		return fmt.Errorf("%d clean action(s) failed", failed)
	}
	return nil
}

func cleanInputs(c *cli.Command, cfg *config.FindConfig) ([]string, []string, bool, *filter.Config, error) {
	filesFromSet := c.IsSet("files-from")
	if c.Bool("null") && !filesFromSet {
		return nil, nil, false, nil, errors.New("--null requires --files-from")
	}
	if filesFromSet && c.Bool("files") {
		return nil, nil, false, nil, errors.New("--files-from cannot be combined with --files")
	}
	if filesFromSet && c.Args().Len() > 0 {
		return nil, nil, false, nil, errors.New("--files-from cannot be combined with positional arguments")
	}

	if filesFromSet {
		path := c.String("files-from")
		if path == "" {
			return nil, nil, false, nil, errors.New("--files-from requires a file path or '-' for stdin")
		}
		var input io.Reader = os.Stdin
		var file *os.File
		var err error
		if path != "-" {
			//nolint:gosec
			file, err = os.Open(path)
			if err != nil {
				return nil, nil, false, nil, fmt.Errorf("error opening file list %s: %w", path, err)
			}
			defer func() {
				_ = file.Close()
			}()
			input = file
		}
		files, err := scanner.GetFilesFromReader(input, c.Bool("null"), c.Bool("ignore-empty-paths"))
		return nil, files, true, &filter.Config{}, err
	}
	if c.Bool("files") {
		files, err := scanner.GetFilesFromArgs(c)
		return nil, files, true, &filter.Config{}, err
	}

	directories, err := scanner.GetDirectoriesFromArgs(c)
	if err != nil {
		return nil, nil, false, nil, err
	}
	var minSize, maxSize int64
	if cfg.MinSize != "" {
		minSize, err = filter.ParseFileSize(cfg.MinSize)
		if err != nil {
			return nil, nil, false, nil, fmt.Errorf("invalid min-size: %w", err)
		}
	}
	if cfg.MaxSize != "" {
		maxSize, err = filter.ParseFileSize(cfg.MaxSize)
		if err != nil {
			return nil, nil, false, nil, fmt.Errorf("invalid max-size: %w", err)
		}
	}
	filterConfig, err := filter.BuildConfig(cfg.ExcludeDirs, cfg.ExcludeFiles, minSize, maxSize, cfg.Exclude)
	return directories, nil, false, filterConfig, err
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
