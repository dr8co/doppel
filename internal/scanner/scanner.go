package scanner

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/internal/filter"
	"github.com/dr8co/doppel/internal/logger"
	"github.com/dr8co/doppel/internal/model"
)

// GroupFilesBySize scans directories and groups files by their size.
func GroupFilesBySize(directories []string, filterConfig *filter.Config, stats *model.Stats, verbose bool) (map[int64][]string, error) {
	sizeGroups := make(map[int64][]string)
	ctx := context.TODO()

	for _, dir := range directories {
		err := filepath.WalkDir(dir, func(path string, dirEnt fs.DirEntry, err error) error {
			if err != nil {
				if verbose {
					var filepathErr *os.PathError
					if errors.As(err, &filepathErr) {
						logger.ErrorAttrs(ctx, "error accessing file",
							slog.String("path", filepathErr.Path), slog.String("op", filepathErr.Op),
							slog.String("err", filepathErr.Err.Error()))
					} else {
						logger.ErrorAttrs(ctx, "error accessing file", slog.String("path", path),
							slog.String("err", err.Error()))
					}
				}
				stats.ErrorCount++
				return nil
			}

			// Check if we should skip this directory
			if dirEnt.IsDir() && filterConfig.ShouldExcludeDir(path) {
				if verbose {
					logger.InfoAttrs(ctx, "skipping directory", slog.String("path", path))
				}
				stats.SkippedDirs++
				return filepath.SkipDir
			}

			if dirEnt.Type().IsRegular() {
				info, err := dirEnt.Info()
				if err != nil {
					if verbose {
						var filepathErr *os.PathError
						if errors.As(err, &filepathErr) {
							logger.ErrorAttrs(ctx, "error getting file info",
								slog.String("path", filepathErr.Path), slog.String("op", filepathErr.Op),
								slog.String("err", filepathErr.Err.Error()))
						} else {
							logger.ErrorAttrs(ctx, "error getting file info", slog.String("path", path),
								slog.String("err", err.Error()))
						}
					}
					stats.ErrorCount++
					return nil
				}

				size := info.Size()

				// Check if we should skip this file
				if filterConfig.ShouldExcludeFile(path, size) {
					if verbose {
						logger.InfoAttrs(ctx, "skipping file", slog.String("path", path),
							slog.Int64("size", size))
					}
					stats.SkippedFiles++
					return nil
				}

				sizeGroups[size] = append(sizeGroups[size], path)
				stats.TotalFiles++
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking directory %s: %w", dir, err)
		}
	}

	if verbose && (stats.SkippedDirs > 0 || stats.SkippedFiles > 0) {
		fmt.Printf("\n⏭️  Skipped %d directories and %d files due to filters\n",
			stats.SkippedDirs, stats.SkippedFiles)
	}

	return sizeGroups, nil
}

// GetDirectoriesFromArgs returns the directories to scan from command arguments.
func GetDirectoriesFromArgs(c *cli.Command) ([]string, error) {
	return processDirectories(c.Args().Slice())
}

// processDirectories receives a list of directories, resolves absolute paths, validates them, and returns unique paths.
// Handles subdirectory elimination and ensures consistent output by sorting the final result.
// Returns an error if any directory is invalid or inaccessible.
func processDirectories(directories []string) ([]string, error) {
	if len(directories) == 0 {
		absDot, err := filepath.Abs(".")
		return []string{absDot}, err
	}

	// Use a map to track unique absolute paths for deduplication
	uniqueDirs := make(map[string]bool, len(directories))
	absDirs := make([]string, 0, len(directories))

	for _, dir := range directories {
		// Make the path absolute
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("error converting to absolute path %s: %w", dir, err)
		}

		// skip if we've already processed this path
		if uniqueDirs[absDir] {
			continue
		}

		// Check if the directory exists and is valid
		info, err := os.Stat(absDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("path does not exist: %s", absDir)
			} else {
				return nil, fmt.Errorf("error accessing directory %s: %w", absDir, err)
			}
		} else if !info.IsDir() {
			return nil, fmt.Errorf("not a directory: %s", absDir)
		}

		// Add to the result only if not already present
		if !uniqueDirs[absDir] {
			uniqueDirs[absDir] = true
			absDirs = append(absDirs, absDir)
		}
	}

	return removeSubdirectories(absDirs), nil
}

// removeSubdirectories removes paths that are subdirectories of other paths.
// The paths are expected to be absolute.
func removeSubdirectories(dirs []string) []string {
	if len(dirs) <= 1 {
		return dirs
	}

	// Sort paths lexicographically - this ensures parents come before their children
	sort.Strings(dirs)

	result := make([]string, 0, len(dirs))
	result = append(result, dirs[0])
	for _, dir := range dirs[1:] {
		if !isSubdirectory(dir, result[len(result)-1]) {
			result = append(result, dir)
		}
	}

	return result
}

// isSubdirectory checks if child is a subdirectory of parent
// This function assumes both paths are absolute.
func isSubdirectory(child, parent string) bool {
	// Empty paths are not valid
	if child == "" || parent == "" {
		return false
	}

	// Clean both paths for a consistent comparison
	child = filepath.Clean(child)
	parent = filepath.Clean(parent)

	if len(parent) >= len(child) {
		return false
	}

	if !strings.HasPrefix(child, parent) {
		return false
	}

	// Special case: parent is root
	if parent == string(filepath.Separator) {
		return child[0] == filepath.Separator
	}

	return child[len(parent)] == filepath.Separator
}
