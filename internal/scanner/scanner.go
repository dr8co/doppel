package scanner

import (
	"fmt"
	"io/fs"
	"log"
	"path/filepath"

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
					log.Printf("❌ Error accessing %s: %v", path, err)
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
						log.Printf("❌ Error getting info for %s: %v", path, err)
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
		fmt.Printf("⏭️  Skipped %d directories and %d files due to filters\n", stats.SkippedDirs, stats.SkippedFiles)
	}

	return sizeGroups, nil
}

// GetDirectoriesFromArgs returns the directories to scan from command arguments
func GetDirectoriesFromArgs(c *cli.Command) []string {
	directories := c.Args().Slice()
	if len(directories) == 0 {
		directories = []string{"."}
	}
	return directories
}
