package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v3"
)

const (
	version = "1.0.0"
)

type FileInfo struct {
	Path string
	Size int64
	Hash string
}

type FilterConfig struct {
	ExcludeDirs      []string
	ExcludeFiles     []string
	ExcludeDirRegex  []*regexp.Regexp
	ExcludeFileRegex []*regexp.Regexp
	MinSize          int64
	MaxSize          int64
}

type Stats struct {
	TotalFiles      int
	ProcessedFiles  int
	SkippedDirs     int
	SkippedFiles    int
	ErrorCount      int
	DuplicateGroups int
	DuplicateFiles  int
	StartTime       time.Time
}

func main() {
	app := &cli.Command{
		Name:    "doppel",
		Usage:   "Find duplicate files across directories",
		Version: version,
		Authors: []any{
			"Ian Duncan",
		},
		Copyright: "(c) 2025 Ian Duncan",
		Description: `A fast, concurrent duplicate file finder with advanced filtering capabilities.
		
This tool scans directories for duplicate files by comparing file sizes first, 
then computing SHA-256 hashes for files of the same size. It supports parallel 
processing and extensive filtering options to skip unwanted files and directories.`,
		Commands: []*cli.Command{
			{
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
						Value:   4,
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
			},
			{
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
			},
		},
		DefaultCommand: "find",
		Suggest:        true,
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func findDuplicates(_ context.Context, c *cli.Command) error {
	directories := c.Args().Slice()
	if len(directories) == 0 {
		directories = []string{"."}
	}

	// Build filter configuration
	filterConfig, err := buildFilterConfig(
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
		displayFilterConfig(filterConfig)
		return nil
	}

	verbose := c.Bool("verbose")
	showStats := c.Bool("stats")
	workers := c.Int("workers")

	stats := &Stats{StartTime: time.Now()}

	if verbose {
		fmt.Printf("🔍 Scanning directories: %v\n", directories)
		displayFilterConfig(filterConfig)
	}

	// Phase 1: Group files by size
	sizeGroups, err := groupFilesBySize(directories, filterConfig, stats, verbose)
	if err != nil {
		return fmt.Errorf("error scanning files: %w", err)
	}

	if verbose {
		fmt.Printf("📊 Found %d files, %d size groups\n", stats.TotalFiles, len(sizeGroups))
	}

	// Phase 2: Hash files that have potential duplicates
	duplicates, err := findDuplicatesByHash(sizeGroups, workers, stats, verbose)
	if err != nil {
		return fmt.Errorf("error finding duplicates: %w", err)
	}

	// Phase 3: Display results
	displayResults(duplicates, stats, showStats || verbose)

	return nil
}

func findDuplicatesWithPreset(_ context.Context, c *cli.Command, preset string) error {
	filterConfig := getPresetConfig(preset)

	directories := c.Args().Slice()
	if len(directories) == 0 {
		directories = []string{"."}
	}

	verbose := c.Bool("verbose")
	showStats := c.Bool("stats")
	workers := c.Int("workers")

	stats := &Stats{StartTime: time.Now()}

	if verbose {
		fmt.Printf("🔍 Using preset '%s' to scan directories: %v\n", preset, directories)
		displayFilterConfig(filterConfig)
	}

	// Phase 1: Group files by size
	sizeGroups, err := groupFilesBySize(directories, filterConfig, stats, verbose)
	if err != nil {
		return fmt.Errorf("error scanning files: %w", err)
	}

	if verbose {
		fmt.Printf("📊 Found %d files, %d size groups\n", stats.TotalFiles, len(sizeGroups))
	}

	// Phase 2: Hash files that have potential duplicates
	duplicates, err := findDuplicatesByHash(sizeGroups, workers, stats, verbose)
	if err != nil {
		return fmt.Errorf("error finding duplicates: %w", err)
	}

	// Phase 3: Display results
	displayResults(duplicates, stats, showStats || verbose)

	return nil
}

func getPresetConfig(preset string) *FilterConfig {
	switch preset {
	case "dev":
		return &FilterConfig{
			ExcludeDirs:  []string{"node_modules", ".git", "build", "dist", "target", "__pycache__", ".vscode", ".idea", "vendor"},
			ExcludeFiles: []string{"*.tmp", "*.log", "*.swp", "*.swo", "*~", ".DS_Store", "Thumbs.db", "*.pyc", "*.pyo"},
			MinSize:      100, // Skip very small files
		}
	case "media":
		return &FilterConfig{
			ExcludeDirs: []string{".git", "__pycache__", "node_modules"},
			MinSize:     10240, // 10KB minimum for media files
		}
	case "docs":
		return &FilterConfig{
			ExcludeDirs:  []string{".git", "__pycache__", "node_modules", "build", "dist"},
			ExcludeFiles: []string{"*.tmp", "*.log", "*.swp", "*~"},
			MinSize:      1024, // 1KB minimum
		}
	case "clean":
		return &FilterConfig{
			ExcludeDirs:  []string{".git", "__pycache__", "node_modules", ".cache", "tmp", "temp"},
			ExcludeFiles: []string{"*.tmp", "*.log", "*.cache", "*.swp", "*~", ".DS_Store", "Thumbs.db"},
		}
	default:
		return &FilterConfig{}
	}
}

// buildFilterConfig creates a FilterConfig from command line arguments
func buildFilterConfig(excludeDirs, excludeFiles, excludeDirRegex, excludeFileRegex string, minSize, maxSize int64) (*FilterConfig, error) {
	config := &FilterConfig{
		MinSize: minSize,
		MaxSize: maxSize,
	}

	// Parse exclude directory patterns
	if excludeDirs != "" {
		config.ExcludeDirs = parseCommaSeparated(excludeDirs)
	}

	// Parse exclude file patterns
	if excludeFiles != "" {
		config.ExcludeFiles = parseCommaSeparated(excludeFiles)
	}

	// Parse exclude directory regex patterns
	if excludeDirRegex != "" {
		patterns := parseCommaSeparated(excludeDirRegex)
		for _, pattern := range patterns {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid directory regex pattern '%s': %w", pattern, err)
			}
			config.ExcludeDirRegex = append(config.ExcludeDirRegex, regex)
		}
	}

	// Parse exclude file regex patterns
	if excludeFileRegex != "" {
		patterns := parseCommaSeparated(excludeFileRegex)
		for _, pattern := range patterns {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid file regex pattern '%s': %w", pattern, err)
			}
			config.ExcludeFileRegex = append(config.ExcludeFileRegex, regex)
		}
	}

	return config, nil
}

// parseCommaSeparated splits a comma-separated string and trims whitespace
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// shouldExcludeDir checks if a directory should be excluded based on filters
func (fc *FilterConfig) shouldExcludeDir(dirPath string) bool {
	dirName := filepath.Base(dirPath)

	// Check exact matches
	for _, pattern := range fc.ExcludeDirs {
		if matched, _ := filepath.Match(pattern, dirName); matched {
			return true
		}
		// Also check if the pattern matches the full path
		if matched, _ := filepath.Match(pattern, dirPath); matched {
			return true
		}
	}

	// Check regex patterns
	for _, regex := range fc.ExcludeDirRegex {
		if regex.MatchString(dirName) || regex.MatchString(dirPath) {
			return true
		}
	}

	return false
}

// shouldExcludeFile checks if a file should be excluded based on filters
func (fc *FilterConfig) shouldExcludeFile(filePath string, size int64) bool {
	fileName := filepath.Base(filePath)

	// Check size limits
	if fc.MinSize > 0 && size < fc.MinSize {
		return true
	}
	if fc.MaxSize > 0 && size > fc.MaxSize {
		return true
	}

	// Check exact matches
	for _, pattern := range fc.ExcludeFiles {
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return true
		}
		// Also check if the pattern matches the full path
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
	}

	// Check regex patterns
	for _, regex := range fc.ExcludeFileRegex {
		if regex.MatchString(fileName) || regex.MatchString(filePath) {
			return true
		}
	}

	return false
}

// displayFilterConfig shows the current filter configuration
func displayFilterConfig(config *FilterConfig) {
	fmt.Println("🔧 Active filters:")
	if len(config.ExcludeDirs) > 0 {
		fmt.Printf("  📁 Exclude directories: %s\n", strings.Join(config.ExcludeDirs, ", "))
	}
	if len(config.ExcludeFiles) > 0 {
		fmt.Printf("  📄 Exclude files: %s\n", strings.Join(config.ExcludeFiles, ", "))
	}
	if len(config.ExcludeDirRegex) > 0 {
		patterns := make([]string, len(config.ExcludeDirRegex))
		for i, regex := range config.ExcludeDirRegex {
			patterns[i] = regex.String()
		}
		fmt.Printf("  📁 Exclude directory regex: %s\n", strings.Join(patterns, ", "))
	}
	if len(config.ExcludeFileRegex) > 0 {
		patterns := make([]string, len(config.ExcludeFileRegex))
		for i, regex := range config.ExcludeFileRegex {
			patterns[i] = regex.String()
		}
		fmt.Printf("  📄 Exclude file regex: %s\n", strings.Join(patterns, ", "))
	}
	if config.MinSize > 0 {
		fmt.Printf("  📏 Minimum file size: %s\n", formatBytes(config.MinSize))
	}
	if config.MaxSize > 0 {
		fmt.Printf("  📏 Maximum file size: %s\n", formatBytes(config.MaxSize))
	}
	if len(config.ExcludeDirs) == 0 && len(config.ExcludeFiles) == 0 &&
		len(config.ExcludeDirRegex) == 0 && len(config.ExcludeFileRegex) == 0 &&
		config.MinSize == 0 && config.MaxSize == 0 {
		fmt.Println("  ✅ No filters active")
	}
	fmt.Println()
}

func groupFilesBySize(directories []string, filterConfig *FilterConfig, stats *Stats, verbose bool) (map[int64][]string, error) {
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
			if dirEnt.IsDir() && filterConfig.shouldExcludeDir(path) {
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
				if filterConfig.shouldExcludeFile(path, size) {
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

// hashFile computes SHA-256 hash of the entire file
func hashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	hasher := sha256.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// findDuplicatesByHash processes files with same sizes and finds actual duplicates
func findDuplicatesByHash(sizeGroups map[int64][]string, numWorkers int, stats *Stats, verbose bool) (map[string][]string, error) {
	var candidateFiles []string
	for _, files := range sizeGroups {
		if len(files) > 1 {
			candidateFiles = append(candidateFiles, files...)
		}
	}

	if len(candidateFiles) == 0 {
		return make(map[string][]string), nil
	}

	if verbose {
		fmt.Printf("🔐 Hashing %d candidate files with %d workers\n", len(candidateFiles), numWorkers)
	}

	// Create the work channel and the result channel
	workChan := make(chan string, len(candidateFiles))
	resultChan := make(chan FileInfo, len(candidateFiles))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range workChan {
				hash, err := hashFile(filePath)
				if err != nil {
					if verbose {
						log.Printf("❌ Error hashing %s: %v", filePath, err)
					}
					stats.ErrorCount++
					continue
				}

				// Get file size for the result
				info, err := os.Stat(filePath)
				if err != nil {
					if verbose {
						log.Printf("❌ Error stating %s: %v", filePath, err)
					}
					stats.ErrorCount++
					continue
				}

				resultChan <- FileInfo{
					Path: filePath,
					Size: info.Size(),
					Hash: hash,
				}
			}
		}()
	}

	// Send work
	for _, file := range candidateFiles {
		workChan <- file
	}
	close(workChan)

	// Wait for workers to finish and close the result channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results and group by hash
	hashGroups := make(map[string][]string)
	for result := range resultChan {
		hashGroups[result.Hash] = append(hashGroups[result.Hash], result.Path)
		stats.ProcessedFiles++
	}

	duplicates := make(map[string][]string)
	for hash, files := range hashGroups {
		if len(files) > 1 {
			duplicates[hash] = files
			stats.DuplicateGroups++
			stats.DuplicateFiles += len(files)
		}
	}

	return duplicates, nil
}

func displayResults(duplicates map[string][]string, stats *Stats, showStats bool) {
	groupCount := 0
	var totalSize int64

	for _, files := range duplicates {
		groupCount++
		fmt.Printf("\n🔗 Duplicate group %d (%d files):\n", groupCount, len(files))

		// Get file size for the group
		if len(files) > 0 {
			if info, err := os.Stat(files[0]); err == nil {
				groupSize := info.Size()
				wastedSpace := groupSize * int64(len(files)-1)
				totalSize += wastedSpace
				fmt.Printf("   Size: %s each, %s wasted space\n", formatBytes(groupSize), formatBytes(wastedSpace))
			}
		}

		for _, file := range files {
			fmt.Printf("   📄 %s\n", file)
		}
	}

	fmt.Printf("\n📊 Summary:\n")
	if stats.DuplicateFiles > 0 {
		fmt.Printf("   🔗 Duplicate files found: %d (in %d groups)\n", stats.DuplicateFiles, stats.DuplicateGroups)
		fmt.Printf("   💾 Total wasted space: %s\n", formatBytes(totalSize))
	} else {
		fmt.Printf("   ✅ No duplicate files found\n")
	}

	if showStats {
		duration := time.Since(stats.StartTime)
		fmt.Printf("\n📈 Detailed Statistics:\n")
		fmt.Printf("   📁 Total files scanned: %d\n", stats.TotalFiles)
		fmt.Printf("   🔐 Files processed for hashing: %d\n", stats.ProcessedFiles)
		fmt.Printf("   ⏭️  Directories skipped: %d\n", stats.SkippedDirs)
		fmt.Printf("   ⏭️  Files skipped: %d\n", stats.SkippedFiles)
		fmt.Printf("   ❌ Files with errors: %d\n", stats.ErrorCount)
		fmt.Printf("   ⏱️  Processing time: %v\n", duration.Round(time.Millisecond))
		if stats.ProcessedFiles > 0 && duration > 0 {
			rate := float64(stats.ProcessedFiles) / duration.Seconds()
			fmt.Printf("   🚀 Processing rate: %.1f files/second\n", rate)
		}
	} else if stats.ErrorCount > 0 {
		fmt.Printf("   ❌ Files with errors: %d\n", stats.ErrorCount)
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
