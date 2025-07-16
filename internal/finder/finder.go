package finder

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/dr8co/doppel/internal/logger"
	"github.com/dr8co/doppel/internal/model"
	"github.com/dr8co/doppel/internal/scanner"
)

// FindDuplicatesByHash processes files with same sizes and returns a DuplicateReport directly
func FindDuplicatesByHash(sizeGroups map[int64][]string, numWorkers int, stats *model.Stats, verbose bool) (*model.DuplicateReport, error) {
	var candidateFiles []string
	for _, files := range sizeGroups {
		if len(files) > 1 {
			candidateFiles = append(candidateFiles, files...)
		}
	}

	if len(candidateFiles) < 2 {
		return &model.DuplicateReport{
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
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range workChan {
				hash, err := scanner.HashFile(filePath)
				if err != nil {
					if verbose {
						var filepathErr *os.PathError
						if errors.As(err, &filepathErr) {
							logger.ErrorAttrs("error hashing file", slog.String("path", filepathErr.Path), slog.String("op", filepathErr.Op), slog.String("err", filepathErr.Err.Error()))
						} else {
							logger.ErrorAttrs("error hashing file", slog.String("path", filePath), slog.String("err", err.Error()))
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
							logger.ErrorAttrs("error stating file", slog.String("path", filepathErr.Path), slog.String("op", filepathErr.Op), slog.String("err", filepathErr.Err.Error()))
						} else {
							logger.ErrorAttrs("error stating file", slog.String("path", filePath), slog.String("err", err.Error()))
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

	var groups []model.DuplicateGroup
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

			groups = append(groups, model.DuplicateGroup{
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

	return &model.DuplicateReport{
		ScanDate:         time.Now(),
		Stats:            stats,
		TotalWastedSpace: totalWasted,
		Groups:           groups,
	}, nil
}
