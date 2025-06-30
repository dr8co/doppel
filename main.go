package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
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

func main() {
	var workers = flag.Int("workers", runtime.NumCPU(), "Number of worker goroutines for hashing")
	var verbose = flag.Bool("v", false, "Verbose output")
	var excludeDirs = flag.String("exclude-dirs", "", "Comma-separated list of directory patterns to exclude")
	var excludeFiles = flag.String("exclude-files", "", "Comma-separated list of file patterns to exclude")
	var excludeDirRegex = flag.String("exclude-dir-regex", "", "Comma-separated list of regex patterns for directories to exclude")
	var excludeFileRegex = flag.String("exclude-file-regex", "", "Comma-separated list of regex patterns for files to exclude")
	var minSize = flag.Int64("min-size", 0, "Minimum file size in bytes (0 = no limit)")
	var maxSize = flag.Int64("max-size", 0, "Maximum file size in bytes (0 = no limit)")
	var showFilters = flag.Bool("show-filters", false, "Show active filters and exit")

	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] [directory1 directory2 ...]\n\n", os.Args[0])
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Options:")
		flag.PrintDefaults()
	}
	flag.Parse()

	directories := flag.Args()
	if len(directories) == 0 {
		directories = []string{"."}
	}

	// Build filter configuration
	filterConfig, err := buildFilterConfig(*excludeDirs, *excludeFiles, *excludeDirRegex, *excludeFileRegex, *minSize, *maxSize)
	if err != nil {
		log.Fatalf("Error building filter configuration: %v", err)
	}

	if *showFilters {
		displayFilterConfig(filterConfig)
		return
	}

	if *verbose {
		fmt.Printf("Scanning directories: %v\n\n", directories)
		displayFilterConfig(filterConfig)
	}

	// Phase 1: Group files by size
	sizeGroups, totalFiles, errors := groupFilesBySize(directories, filterConfig, *verbose)

	if *verbose {
		fmt.Printf("\nFound %d files, %d size groups\n\n", totalFiles, len(sizeGroups))
	}

	// Phase 2: Hash files that have potential duplicates (same size)
	duplicates, hashErrors := findDuplicatesByHash(sizeGroups, *workers, *verbose)

	// Phase 3: Display results
	displayResults(duplicates, totalFiles, errors+hashErrors)
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
				return nil, fmt.Errorf("invalid directory regex pattern '%s': %v", pattern, err)
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
				return nil, fmt.Errorf("invalid file regex pattern '%s': %v", pattern, err)
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
	fmt.Println("Active filters:")
	if len(config.ExcludeDirs) > 0 {
		fmt.Printf("  Exclude directories: %s\n", strings.Join(config.ExcludeDirs, ", "))
	}
	if len(config.ExcludeFiles) > 0 {
		fmt.Printf("  Exclude files: %s\n", strings.Join(config.ExcludeFiles, ", "))
	}
	if len(config.ExcludeDirRegex) > 0 {
		patterns := make([]string, len(config.ExcludeDirRegex))
		for i, regex := range config.ExcludeDirRegex {
			patterns[i] = regex.String()
		}
		fmt.Printf("  Exclude directory regex: %s\n", strings.Join(patterns, ", "))
	}
	if len(config.ExcludeFileRegex) > 0 {
		patterns := make([]string, len(config.ExcludeFileRegex))
		for i, regex := range config.ExcludeFileRegex {
			patterns[i] = regex.String()
		}
		fmt.Printf("  Exclude file regex: %s\n", strings.Join(patterns, ", "))
	}
	if config.MinSize > 0 {
		fmt.Printf("  Minimum file size: %d bytes\n", config.MinSize)
	}
	if config.MaxSize > 0 {
		fmt.Printf("  Maximum file size: %d bytes\n", config.MaxSize)
	}
	if len(config.ExcludeDirs) == 0 && len(config.ExcludeFiles) == 0 &&
		len(config.ExcludeDirRegex) == 0 && len(config.ExcludeFileRegex) == 0 &&
		config.MinSize == 0 && config.MaxSize == 0 {
		fmt.Println("  No filters active")
	}
	fmt.Println()
}

// groupFilesBySize walks directories and groups files by size
func groupFilesBySize(directories []string, filterConfig *FilterConfig, verbose bool) (map[int64][]string, int, int) {
	sizeGroups := make(map[int64][]string)
	totalFiles := 0
	errors := 0
	skippedDirs := 0
	skippedFiles := 0

	for _, dir := range directories {
		err := filepath.WalkDir(dir, func(path string, dirEnt fs.DirEntry, err error) error {
			if err != nil {
				if verbose {
					log.Printf("Error accessing %s: %v", path, err)
				}
				errors++
				return nil // Continue walking despite errors
			}

			// Check if we should skip this directory
			if dirEnt.IsDir() && filterConfig.shouldExcludeDir(path) {
				if verbose {
					log.Printf("Skipping directory: %s", path)
				}
				skippedDirs++
				return filepath.SkipDir
			}

			if dirEnt.Type().IsRegular() {
				info, err := dirEnt.Info()
				if err != nil {
					if verbose {
						log.Printf("Error getting info for %s: %v", path, err)
					}
					errors++
					return nil
				}

				size := info.Size()

				// Check if we should skip this file
				if filterConfig.shouldExcludeFile(path, size) {
					if verbose {
						log.Printf("Skipping file: %s", path)
					}
					skippedFiles++
					return nil
				}

				sizeGroups[size] = append(sizeGroups[size], path)
				totalFiles++
			}
			return nil
		})

		if err != nil {
			log.Printf("Error walking directory %s: %v", dir, err)
			errors++
		}
	}

	if verbose && (skippedDirs > 0 || skippedFiles > 0) {
		fmt.Printf("Skipped %d directories and %d files due to filters\n", skippedDirs, skippedFiles)
	}

	return sizeGroups, totalFiles, errors
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
func findDuplicatesByHash(sizeGroups map[int64][]string, numWorkers int, verbose bool) (map[string][]string, int) {
	// Only process size groups with multiple files
	var candidateFiles []string
	for _, files := range sizeGroups {
		if len(files) > 1 {
			candidateFiles = append(candidateFiles, files...)
		}
	}

	if len(candidateFiles) == 0 {
		return make(map[string][]string), 0
	}

	if verbose {
		fmt.Printf("Hashing %d candidate files with %d workers\n\n", len(candidateFiles), numWorkers)
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
						log.Printf("Error hashing %s: %v", filePath, err)
					}
					continue
				}

				// Get file size for the result
				info, err := os.Stat(filePath)
				if err != nil {
					if verbose {
						log.Printf("Error stating %s: %v", filePath, err)
					}
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
	processedCount := 0
	errorCount := len(candidateFiles) // Start with the total, subtract successful ones

	for result := range resultChan {
		hashGroups[result.Hash] = append(hashGroups[result.Hash], result.Path)
		processedCount++
	}

	errorCount -= processedCount

	// Filter to only return actual duplicates (hash groups with >1 file)
	duplicates := make(map[string][]string)
	for hash, files := range hashGroups {
		if len(files) > 1 {
			duplicates[hash] = files
		}
	}

	return duplicates, errorCount
}

// displayResults shows the duplicate files found
func displayResults(duplicates map[string][]string, totalFiles, errors int) {
	duplicateCount := 0
	groupCount := 0

	for _, files := range duplicates {
		groupCount++
		fmt.Printf("\nDuplicate group %d (%d files):\n", groupCount, len(files))
		for _, file := range files {
			fmt.Printf("  %s\n", file)
			duplicateCount++
		}
	}

	fmt.Printf("\nSummary:\n")
	if duplicateCount > 0 {
		fmt.Printf("  Duplicate files found: %d (in %d groups)\n", duplicateCount, groupCount)
	} else {
		fmt.Printf("  No duplicate files found\n")
	}
	fmt.Printf("  Total files scanned: %d\n", totalFiles)
	if errors > 0 {
		fmt.Printf("  Files with errors: %d\n", errors)
	}
}
