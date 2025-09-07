package model

import (
	"sync/atomic"
	"time"
)

// DuplicateGroup represents a group of duplicate files with their metadata.
type DuplicateGroup struct {
	// ID is a unique identifier for the group.
	ID int `json:"ID" yaml:"ID"`

	// Count is the number of files in this group.
	Count int `json:"count" yaml:"count"`

	// Size is the size of each file in the group.
	Size int64 `json:"size" yaml:"size"`

	// WastedSpace is the total wasted space due to duplicates in this group.
	WastedSpace uint64 `json:"wasted_space" yaml:"wasted_space"`

	// Files contains the paths of the files in this group.
	Files []string `json:"files" yaml:"files"`
}

// DuplicateReport represents the report of duplicate files found during a scan.
type DuplicateReport struct {
	// ScanDate is the date and time when the scan was performed.
	ScanDate time.Time `json:"scan_date" yaml:"scan_date"`

	// Stats contain various statistics about the scan.
	Stats *Stats `json:"stats" yaml:"stats"`

	// TotalWastedSpace is the total wasted space due to duplicates across all groups.
	TotalWastedSpace uint64 `json:"total_wasted_space" yaml:"total_wasted_space"`

	// Groups contain the list of duplicate file groups found.
	Groups []DuplicateGroup `json:"groups" yaml:"groups"`
}

// Stats track various statistics during the duplicate file finding process.
type Stats struct {
	// TotalFiles is the total number of files scanned.
	TotalFiles uint64 `json:"total_files" yaml:"total_files"`

	// ProcessedFiles is the number of files processed (not skipped).
	ProcessedFiles uint64 `json:"processed_files" yaml:"processed_files"`

	// SkippedDirs is the number of directories skipped due to filters.
	SkippedDirs uint64 `json:"skipped_dirs" yaml:"skipped_dirs"`

	// SkippedFiles is the number of files skipped due to filters.
	SkippedFiles uint64 `json:"skipped_files" yaml:"skipped_files"`

	// ErrorCount is the number of errors encountered during processing.
	ErrorCount uint64 `json:"error_count" yaml:"error_count"`

	// DuplicateGroups is the number of duplicate file groups found.
	DuplicateGroups uint64 `json:"duplicate_groups" yaml:"duplicate_groups"`

	// DuplicateFiles is the total number of files that are part of duplicate groups.
	DuplicateFiles uint64 `json:"duplicate_files" yaml:"duplicate_files"`

	// StartTime is the time when the scan started.
	StartTime time.Time `json:"start_time" yaml:"start_time"`

	// Duration is the total duration of the scan.
	Duration time.Duration `json:"duration" yaml:"duration"`
}

// IncrementErrorCount atomically increments the error count.
func (s *Stats) IncrementErrorCount() {
	atomic.AddUint64(&s.ErrorCount, 1)
}

// IncrementProcessedFiles atomically increments the processed files count.
func (s *Stats) IncrementProcessedFiles() {
	atomic.AddUint64(&s.ProcessedFiles, 1)
}

// IncrementDuplicateGroups atomically increments the duplicate groups count.
func (s *Stats) IncrementDuplicateGroups() {
	atomic.AddUint64(&s.DuplicateGroups, 1)
}

// AddDuplicateFiles atomically adds to the duplicate files count.
func (s *Stats) AddDuplicateFiles(count uint64) {
	atomic.AddUint64(&s.DuplicateFiles, count)
}

// GetErrorCount atomically retrieves the error count.
func (s *Stats) GetErrorCount() uint64 {
	return atomic.LoadUint64(&s.ErrorCount)
}

// GetProcessedFiles atomically retrieves the processed files count.
func (s *Stats) GetProcessedFiles() uint64 {
	return atomic.LoadUint64(&s.ProcessedFiles)
}

// GetDuplicateGroups atomically retrieves the duplicate groups count.
func (s *Stats) GetDuplicateGroups() uint64 {
	return atomic.LoadUint64(&s.DuplicateGroups)
}

// GetDuplicateFiles atomically retrieves the duplicate files count.
func (s *Stats) GetDuplicateFiles() uint64 {
	return atomic.LoadUint64(&s.DuplicateFiles)
}
