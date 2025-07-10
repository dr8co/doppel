package duplicate

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/dr8co/doppel/internal/scanner"
	"github.com/dr8co/doppel/internal/stats"
)

// DuplicateGroup represents a group of duplicate files with their metadata
type DuplicateGroup struct {
	Id          int      `json:"id"`
	Count       int      `json:"count"`
	Size        int64    `json:"size"`
	WastedSpace uint64   `json:"wasted_space"`
	Files       []string `json:"files"`
}

// DuplicateReport represents the report of duplicate files found during a scan
type DuplicateReport struct {
	ScanDate         time.Time        `json:"scan_date"`
	Stats            *stats.Stats     `json:"stats"`
	TotalWastedSpace uint64           `json:"total_wasted_space"`
	Groups           []DuplicateGroup `json:"groups"`
}

// FindDuplicatesByHash processes files with same sizes and returns a DuplicateReport directly
func FindDuplicatesByHash(sizeGroups map[int64][]string, numWorkers int, stats *stats.Stats, verbose bool) (*DuplicateReport, error) {
	var candidateFiles []string
	for _, files := range sizeGroups {
		if len(files) > 1 {
			candidateFiles = append(candidateFiles, files...)
		}
	}

	if len(candidateFiles) < 2 {
		return &DuplicateReport{
			ScanDate:         time.Now(),
			Stats:            stats,
			TotalWastedSpace: 0,
			Groups:           nil,
		}, nil
	}

	if verbose {
		fmt.Printf("\n🔐 Hashing %d candidate files with %d workers\n\n", len(candidateFiles), numWorkers)
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
						var filepathErr *os.PathError
						if errors.As(err, &filepathErr) {
							log.Printf("❌ Error hashing: %v", filepathErr)
						} else {
							log.Printf("❌ Error hashing %s: %v", filePath, err)
						}
					}
					stats.IncrementErrorCount()
					continue
				}

				// Get file size for the result
				info, err := os.Stat(filePath)
				if err != nil {
					if verbose {
						var filepathErr *os.PathError
						if errors.As(err, &filepathErr) {
							log.Printf("❌ Error stating: %v", filepathErr)
						} else {
							log.Printf("❌ Error stating %s: %v", filePath, err)
						}
					}
					stats.IncrementErrorCount()
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
	hashGroups := make(map[string][]scanner.FileInfo)
	for result := range resultChan {
		hashGroups[result.Hash] = append(hashGroups[result.Hash], result)
		stats.ProcessedFiles++
	}

	var groups []DuplicateGroup
	totalWasted := uint64(0)
	groupId := 0
	for _, files := range hashGroups {
		if len(files) > 1 {
			groupId++
			filePaths := make([]string, len(files))
			for i, fi := range files {
				filePaths[i] = fi.Path
			}
			size := files[0].Size
			wasted := uint64(size) * uint64(len(files)-1)
			totalWasted += wasted
			groups = append(groups, DuplicateGroup{
				Id:          groupId,
				Count:       len(files),
				Size:        size,
				WastedSpace: wasted,
				Files:       filePaths,
			})
			stats.DuplicateGroups++
			stats.DuplicateFiles += uint64(len(files))
		}
	}

	return &DuplicateReport{
		ScanDate:         time.Now(),
		Stats:            stats,
		TotalWastedSpace: totalWasted,
		Groups:           groups,
	}, nil
}
