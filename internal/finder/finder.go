package finder

import (
	"context"
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
		fmt.Printf("\n🔐 Multi-stage hashing %d candidate files with %d workers\n\n", len(candidateFiles), numWorkers)
	}

	ctx := context.TODO()

	// Stage 1: Quick hashing
	if verbose {
		fmt.Printf("Stage 1: Quick hashing...\n")
	}

	quickHashGroups := make(map[string][]string)
	quickWorkChan := make(chan string, len(candidateFiles))
	quickResultChan := make(chan struct {
		path      string
		quickHash string
		size      int64
	}, len(candidateFiles))

	// Start workers for quick hashing
	var quickWg sync.WaitGroup
	for range numWorkers {
		quickWg.Add(1)
		go func() {
			defer quickWg.Done()
			for filePath := range quickWorkChan {
				quickHash, err := scanner.QuickHashFile(filePath)
				if err != nil {
					logError(ctx, err, "quick hash", filePath)
					stats.IncrementErrorCount()
					continue
				}

				// Get file size for the result
				info, err := os.Stat(filePath)
				if err != nil {
					logError(ctx, err, "stat", filePath)
					stats.IncrementErrorCount()
					continue
				}

				quickResultChan <- struct {
					path      string
					quickHash string
					size      int64
				}{filePath, quickHash, info.Size()}
			}
		}()
	}

	// Send work for quick hashing
	for _, file := range candidateFiles {
		quickWorkChan <- file
	}
	close(quickWorkChan)

	// Wait for quick hashing workers to finish
	go func() {
		quickWg.Wait()
		close(quickResultChan)
	}()

	// Collect quick hash results and group by quick hash
	for result := range quickResultChan {
		quickHashGroups[result.quickHash] = append(quickHashGroups[result.quickHash], result.path)
		stats.ProcessedFiles++
	}

	// Stage 2: Full hashing only for files with matching quick hashes
	var fullHashCandidates []string
	for _, files := range quickHashGroups {
		if len(files) > 1 {
			fullHashCandidates = append(fullHashCandidates, files...)
		}
	}

	if verbose {
		fmt.Printf("Stage 2: Full hashing %d files with potential duplicates...\n", len(fullHashCandidates))
	}

	// If no candidates for full hashing, return early
	if len(fullHashCandidates) == 0 {
		return &model.DuplicateReport{
			ScanDate:         time.Now(),
			Stats:            stats,
			TotalWastedSpace: 0,
			Groups:           nil,
		}, nil
	}

	// Create channels for full hashing
	fullWorkChan := make(chan string, len(fullHashCandidates))
	fullResultChan := make(chan scanner.FileInfo, len(fullHashCandidates))

	// Start workers for full hashing
	var fullWg sync.WaitGroup
	for range numWorkers {
		fullWg.Add(1)
		go func() {
			defer fullWg.Done()
			for filePath := range fullWorkChan {
				hash, err := scanner.HashFile(filePath)
				if err != nil {
					logError(ctx, err, "full hash", filePath)
					stats.IncrementErrorCount()
					continue
				}

				// Get file size for the result
				info, err := os.Stat(filePath)
				if err != nil {
					logError(ctx, err, "stat", filePath)
					stats.IncrementErrorCount()
					continue
				}

				fullResultChan <- scanner.FileInfo{
					Path: filePath,
					Size: info.Size(),
					Hash: hash,
				}
			}
		}()
	}

	// Send work for full hashing
	for _, file := range fullHashCandidates {
		fullWorkChan <- file
	}
	close(fullWorkChan)

	// Wait for full hashing workers to finish
	go func() {
		fullWg.Wait()
		close(fullResultChan)
	}()

	// Collect results and group by full hash
	hashGroups := make(map[string][]scanner.FileInfo)
	for result := range fullResultChan {
		hashGroups[result.Hash] = append(hashGroups[result.Hash], result)
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

// logError logs errors encountered during file processing
func logError(ctx context.Context, err error, action, filePath string) {
	if errors.Is(err, os.ErrNotExist) {
		logger.ErrorAttrs(ctx, "file removed after the scan but before hashing", slog.String("path", filePath), slog.String("err", err.Error()))
		return
	}
	var filepathErr *os.PathError

	if errors.As(err, &filepathErr) {
		logger.ErrorAttrs(ctx, "failed to "+action+" a file", slog.String("path", filepathErr.Path), slog.String("op", filepathErr.Op), slog.String("err", filepathErr.Err.Error()))
	} else {
		logger.ErrorAttrs(ctx, "failed to "+action+" a file", slog.String("path", filePath), slog.String("err", err.Error()))
	}
}
