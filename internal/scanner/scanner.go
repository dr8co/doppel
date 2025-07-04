package scanner

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/internal/config"
	"github.com/dr8co/doppel/internal/stats"
)

// GroupFilesBySize scans directories and groups files by their size
func GroupFilesBySize(directories []string, filterConfig *config.FilterConfig, stats *stats.Stats, verbose bool) (map[int64][]string, error) {
	sizeGroups := make(map[int64][]string)

	for _, dir := range directories {
		err := filepath.WalkDir(dir, func(path string, dirEnt fs.DirEntry, err error) error {
			if err != nil {
				if verbose {
					var filepathErr *os.PathError
					if errors.As(err, &filepathErr) {
						log.Printf("❌ Error accessing: %v", filepathErr)
					} else {
						log.Printf("❌ Error accessing %s: %v", path, err)
					}
				}
				stats.ErrorCount++
				return nil
			}

			// Check if we should skip this directory
			if dirEnt.IsDir() && filterConfig.ShouldExcludeDir(path) {
				if verbose {
					log.Printf("⏭️  Skipping directory: %s", path)
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
							log.Printf("❌ Error getting info: %v", filepathErr)
						} else {
							log.Printf("❌ Error getting info for %s: %v", path, err)
						}
					}
					stats.ErrorCount++
					return nil
				}

				size := info.Size()

				// Check if we should skip this file
				if filterConfig.ShouldExcludeFile(path, size) {
					if verbose {
						log.Printf("⏭️  Skipping file: %s", path)
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
		fmt.Printf("\n⏭️  Skipped %d directories and %d files due to filters\n", stats.SkippedDirs, stats.SkippedFiles)
	}

	return sizeGroups, nil
}

// GetDirectoriesFromArgs returns the directories to scan from command arguments
func GetDirectoriesFromArgs(c *cli.Command) ([]string, error) {
	directories := c.Args().Slice()
	if len(directories) == 0 {
		absDot, err := filepath.Abs(".")
		return []string{absDot}, err
	}

	// Use a map to track unique absolute paths for deduplication
	uniqueDirs := make(map[string]bool)
	var absDirs []string

	for _, dir := range directories {
		// Make the path absolute
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("error converting to absolute path %s: %w", dir, err)
		}

		// Check if the directory exists and is valid
		info, err := os.Stat(absDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("path does not exist: %s", dir)
			} else {
				return nil, fmt.Errorf("error accessing directory %s: %w", dir, err)
			}
		} else if !info.IsDir() {
			return nil, fmt.Errorf("not a directory: %s", dir)
		}

		// Add to the result only if not already present (deduplication)
		if !uniqueDirs[absDir] {
			uniqueDirs[absDir] = true
			absDirs = append(absDirs, absDir)
		}
	}

	// Remove directories that are subdirectories of other directories
	var result []string
	for i, dir := range absDirs {
		isSubdir := false
		for j, otherDir := range absDirs {
			if i != j && isSubdirectory(dir, otherDir) {
				isSubdir = true
				break
			}
		}
		if !isSubdir {
			result = append(result, dir)
		}
	}

	return result, nil
}

// isSubdirectory checks if child is a subdirectory of parent
func isSubdirectory(child, parent string) bool {
	// A directory cannot be a subdirectory of itself
	if child == parent {
		return false
	}

	// Check if the child path starts with the parent path followed by the separator
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}

	// If the relative path doesn't start with ".." then child is under parent
	return !filepath.IsAbs(rel) && !strings.HasPrefix(rel, "..")
}
