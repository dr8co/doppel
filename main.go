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
	"runtime"
	"sync"
)

type FileInfo struct {
	Path string
	Size int64
	Hash string
}

func main() {
	var workers = flag.Int("workers", runtime.NumCPU(), "Number of worker goroutines for hashing")
	var verbose = flag.Bool("v", false, "Verbose output")

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

	if *verbose {
		fmt.Printf("Scanning directories: %v\n", directories)
	}

	// Phase 1: Group files by size
	sizeGroups, totalFiles, errors := groupFilesBySize(directories, *verbose)

	if *verbose {
		fmt.Printf("Found %d files, %d size groups\n", totalFiles, len(sizeGroups))
	}

	// Phase 2: Hash files that have potential duplicates (same size)
	duplicates, hashErrors := findDuplicatesByHash(sizeGroups, *workers, *verbose)

	// Phase 3: Display results
	displayResults(duplicates, totalFiles, errors+hashErrors)
}

// groupFilesBySize walks directories and groups files by size
func groupFilesBySize(directories []string, verbose bool) (map[int64][]string, int, int) {
	sizeGroups := make(map[int64][]string)
	totalFiles := 0
	errors := 0

	for _, dir := range directories {
		err := filepath.WalkDir(dir, func(path string, dirEnt fs.DirEntry, err error) error {
			if err != nil {
				if verbose {
					log.Printf("Error accessing %s: %v", path, err)
				}
				errors++
				return nil // Continue walking despite errors
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

	return sizeGroups, totalFiles, errors
}

// hashFile computes SHA-256 hash of the entire file
func hashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Error closing file %s: %v", filePath, err)
		}
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
		fmt.Printf("Hashing %d candidate files with %d workers\n", len(candidateFiles), numWorkers)
	}

	// Create a work channel and a result channel
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
