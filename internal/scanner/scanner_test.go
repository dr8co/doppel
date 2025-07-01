package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dr8co/doppel/internal/config"
	"github.com/dr8co/doppel/internal/stats"
)

func TestGroupFilesBySize(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "scanner_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Errorf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	// Create test directories
	dir1 := filepath.Join(tempDir, "dir1")
	dir2 := filepath.Join(tempDir, "dir2")
	skipDir := filepath.Join(tempDir, "skip_dir")

	for _, dir := range []string{dir1, dir2, skipDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create test files with different sizes
	testFiles := map[string]int{
		filepath.Join(dir1, "file1.txt"):       100,
		filepath.Join(dir1, "file2.txt"):       200,
		filepath.Join(dir2, "file3.txt"):       100, // Same size as file1.txt
		filepath.Join(dir2, "file4.txt"):       300,
		filepath.Join(skipDir, "skipfile.txt"): 400,
		filepath.Join(dir1, "skip.log"):        500, // Will be skipped by filter
	}

	for filePath, size := range testFiles {
		content := make([]byte, size)
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filePath, err)
		}
	}

	// Create a filter config that skips certain files and directories
	filterConfig := &config.FilterConfig{
		ExcludeDirs:  []string{"skip_dir"},
		ExcludeFiles: []string{"*.log"},
	}

	// Create stats object
	s := &stats.Stats{}

	// Test GroupFilesBySize
	sizeGroups, err := GroupFilesBySize([]string{tempDir}, filterConfig, s, false)
	if err != nil {
		t.Fatalf("GroupFilesBySize() error = %v", err)
	}

	// Verify the results
	// We should have 3 size groups: 100, 200, and 300 bytes
	if len(sizeGroups) != 3 {
		t.Errorf("GroupFilesBySize() returned %d size groups, want 3", len(sizeGroups))
	}

	// Check the 100-byte group (should contain 2 files)
	if files, ok := sizeGroups[100]; !ok || len(files) != 2 {
		t.Errorf("Size group 100 has %d files, want 2", len(files))
	}

	// Check the 200-byte group (should contain 1 file)
	if files, ok := sizeGroups[200]; !ok || len(files) != 1 {
		t.Errorf("Size group 200 has %d files, want 1", len(files))
	}

	// Check the 300-byte group (should contain 1 file)
	if files, ok := sizeGroups[300]; !ok || len(files) != 1 {
		t.Errorf("Size group 300 has %d files, want 1", len(files))
	}

	// Verify that skipped files are not included
	for _, files := range sizeGroups {
		for _, file := range files {
			if filepath.Base(file) == "skipfile.txt" || filepath.Base(file) == "skip.log" {
				t.Errorf("Skipped file %s was included in results", file)
			}
		}
	}

	// Verify stats
	if s.TotalFiles != 4 {
		t.Errorf("Stats.TotalFiles = %d, want 4", s.TotalFiles)
	}

	if s.SkippedDirs != 1 {
		t.Errorf("Stats.SkippedDirs = %d, want 1", s.SkippedDirs)
	}

	if s.SkippedFiles != 1 {
		t.Errorf("Stats.SkippedFiles = %d, want 1", s.SkippedFiles)
	}
}
