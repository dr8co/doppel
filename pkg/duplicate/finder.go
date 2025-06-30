package duplicate

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/dr8co/doppel/internal/scanner"
	"github.com/dr8co/doppel/internal/stats"
)

// FindDuplicatesByHash processes files with same sizes and finds actual duplicates
func FindDuplicatesByHash(sizeGroups map[int64][]string, numWorkers int, stats *stats.Stats, verbose bool) (map[string][]string, error) {
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
	resultChan := make(chan scanner.FileInfo, len(candidateFiles))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range workChan {
				hash, err := scanner.HashFile(filePath)
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

				resultChan <- scanner.FileInfo{
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
