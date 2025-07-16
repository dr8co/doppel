package model

import (
	"sync/atomic"
	"time"
)

// DuplicateGroup represents a group of duplicate files with their metadata
type DuplicateGroup struct {
	Id          int      `json:"id" yaml:"id"`
	Count       int      `json:"count" yaml:"count"`
	Size        int64    `json:"size" yaml:"size"`
	WastedSpace uint64   `json:"wasted_space" yaml:"wasted_space"`
	Files       []string `json:"files" yaml:"files"`
}

// DuplicateReport represents the report of duplicate files found during a scan
type DuplicateReport struct {
	ScanDate         time.Time        `json:"scan_date" yaml:"scan_date"`
	Stats            *Stats           `json:"stats" yaml:"stats"`
	TotalWastedSpace uint64           `json:"total_wasted_space" yaml:"total_wasted_space"`
	Groups           []DuplicateGroup `json:"groups" yaml:"groups"`
}

// Stats tracks various statistics during the duplicate file finding process
type Stats struct {
	TotalFiles      uint64        `json:"total_files" yaml:"total_files"`
	ProcessedFiles  uint64        `json:"processed_files" yaml:"processed_files"`
	SkippedDirs     uint64        `json:"skipped_dirs" yaml:"skipped_dirs"`
	SkippedFiles    uint64        `json:"skipped_files" yaml:"skipped_files"`
	ErrorCount      uint64        `json:"error_count" yaml:"error_count"`
	DuplicateGroups uint64        `json:"duplicate_groups" yaml:"duplicate_groups"`
	DuplicateFiles  uint64        `json:"duplicate_files" yaml:"duplicate_files"`
	StartTime       time.Time     `json:"start_time" yaml:"start_time"`
	Duration        time.Duration `json:"duration" yaml:"duration"`
}

// IncrementErrorCount atomically increments the error count
func (s *Stats) IncrementErrorCount() {
	atomic.AddUint64(&s.ErrorCount, 1)
}

// IncrementProcessedFiles atomically increments the processed files count
func (s *Stats) IncrementProcessedFiles() {
	atomic.AddUint64(&s.ProcessedFiles, 1)
}

// IncrementDuplicateGroups atomically increments the duplicate groups count
func (s *Stats) IncrementDuplicateGroups() {
	atomic.AddUint64(&s.DuplicateGroups, 1)
}

// AddDuplicateFiles atomically adds to the duplicate files count
func (s *Stats) AddDuplicateFiles(count uint64) {
	atomic.AddUint64(&s.DuplicateFiles, count)
}

// GetErrorCount atomically retrieves the error count
func (s *Stats) GetErrorCount() uint64 {
	return atomic.LoadUint64(&s.ErrorCount)
}

// GetProcessedFiles atomically retrieves the processed files count
func (s *Stats) GetProcessedFiles() uint64 {
	return atomic.LoadUint64(&s.ProcessedFiles)
}

// GetDuplicateGroups atomically retrieves the duplicate groups count
func (s *Stats) GetDuplicateGroups() uint64 {
	return atomic.LoadUint64(&s.DuplicateGroups)
}

// GetDuplicateFiles atomically retrieves the duplicate files count
func (s *Stats) GetDuplicateFiles() uint64 {
	return atomic.LoadUint64(&s.DuplicateFiles)
}
