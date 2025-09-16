package finder

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"

	"github.com/dr8co/doppel/internal/model"
	"github.com/dr8co/doppel/internal/scanner"
)

// TestFindDuplicatesByHash tests the functionality of finding duplicate files by hashing their contents.
// It verifies correct grouping of duplicates, statistics gathering,
// and handling of edge cases like no duplicates or empty input.
func TestFindDuplicatesByHash(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		// Create a temporary directory for test files
		tempDir, err := os.MkdirTemp("", "finder_test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer func(path string) {
			err := os.RemoveAll(path)
			if err != nil {
				t.Errorf("Failed to remove temp directory: %v", err)
			}
		}(tempDir)

		// Create test files with the same content but different names
		content1 := []byte("This is test content for duplicate files")
		content2 := []byte("This is different content")

		// Create files with content1 (duplicates)
		file1 := filepath.Join(tempDir, "file1.txt")
		file2 := filepath.Join(tempDir, "file2.txt")
		file3 := filepath.Join(tempDir, "file3.txt")

		// Create files with content2 (duplicates)
		file4 := filepath.Join(tempDir, "file4.txt")
		file5 := filepath.Join(tempDir, "file5.txt")

		// Create a unique file
		file6 := filepath.Join(tempDir, "file6.txt")

		// Write content to files
		if err := os.WriteFile(file1, content1, 0644); err != nil {
			t.Fatalf("Failed to write file1: %v", err)
		}
		if err := os.WriteFile(file2, content1, 0644); err != nil {
			t.Fatalf("Failed to write file2: %v", err)
		}
		if err := os.WriteFile(file3, content1, 0644); err != nil {
			t.Fatalf("Failed to write file3: %v", err)
		}
		if err := os.WriteFile(file4, content2, 0644); err != nil {
			t.Fatalf("Failed to write file4: %v", err)
		}
		if err := os.WriteFile(file5, content2, 0644); err != nil {
			t.Fatalf("Failed to write file5: %v", err)
		}
		if err := os.WriteFile(file6, []byte("Unique content"), 0644); err != nil {
			t.Fatalf("Failed to write file6: %v", err)
		}

		// Create size groups
		sizeGroups := map[int64][]scanner.FileInfo{
			int64(len(content1)):         {scanner.FileInfo{Path: file1}, scanner.FileInfo{Path: file2}, scanner.FileInfo{Path: file3}},
			int64(len(content2)):         {scanner.FileInfo{Path: file4}, scanner.FileInfo{Path: file5}},
			int64(len("Unique content")): {scanner.FileInfo{Path: file6}},
		}

		// Create stats object
		s := &model.Stats{}
		ctx := context.Background()

		// Call the function being tested
		report, err := FindDuplicatesByHash(ctx, sizeGroups, 2, s, false)
		if err != nil {
			t.Fatalf("FindDuplicatesByHash() error = %v", err)
		}

		// Verify the results
		if len(report.Groups) != 2 {
			t.Errorf("FindDuplicatesByHash() returned %d duplicate groups, want 2", len(report.Groups))
		}

		// Check if the duplicate groups contain the expected files
		foundGroup1 := false
		foundGroup2 := false

		for _, group := range report.Groups {
			switch group.Count {
			case 3:
				// This should be the content1 group
				foundGroup1 = true
				if !containsAll(group.Files, []string{file1, file2, file3}) {
					t.Errorf("Duplicate group missing expected files: %v", group.Files)
				}
			case 2:
				// This should be the content2 group
				foundGroup2 = true
				if !containsAll(group.Files, []string{file4, file5}) {
					t.Errorf("Duplicate group missing expected files: %v", group.Files)
				}
			}
		}

		if !foundGroup1 {
			t.Errorf("Missing duplicate group for content1")
		}
		if !foundGroup2 {
			t.Errorf("Missing duplicate group for content2")
		}

		// Verify stats
		// Only files in size groups with more than one file are processed,
		// So file6 (the unique file) is not processed
		if s.ProcessedFiles != 5 {
			t.Errorf("Stats.ProcessedFiles = %d, want 5", s.ProcessedFiles)
		}

		if s.DuplicateGroups != 2 {
			t.Errorf("Stats.DuplicateGroups = %d, want 2", s.DuplicateGroups)
		}

		if s.DuplicateFiles != 5 {
			t.Errorf("Stats.DuplicateFiles = %d, want 5", s.DuplicateFiles)
		}

		// Test with no duplicates
		sizeGroups = map[int64][]scanner.FileInfo{
			int64(len(content1)): {scanner.FileInfo{Path: file1}},
			int64(len(content2)): {scanner.FileInfo{Path: file4}},
		}

		report, err = FindDuplicatesByHash(ctx, sizeGroups, 2, s, false)
		if err != nil {
			t.Fatalf("FindDuplicatesByHash() error = %v", err)
		}
		if len(report.Groups) != 0 {
			t.Errorf("FindDuplicatesByHash() returned %d duplicate groups, want 0", len(report.Groups))
		}

		// Test with all files duplicate
		sizeGroups = map[int64][]scanner.FileInfo{
			int64(len(content1)): {scanner.FileInfo{Path: file1}, scanner.FileInfo{Path: file2}, scanner.FileInfo{Path: file3}},
		}

		report, err = FindDuplicatesByHash(ctx, sizeGroups, 2, s, false)
		if err != nil {
			t.Fatalf("FindDuplicatesByHash() error = %v", err)
		}
		if len(report.Groups) != 1 {
			t.Errorf("FindDuplicatesByHash() returned %d duplicate groups, want 1", len(report.Groups))
		}

		// Test with only one file in input
		sizeGroups = map[int64][]scanner.FileInfo{
			int64(len(content1)): {scanner.FileInfo{Path: file1}},
		}
		report, err = FindDuplicatesByHash(ctx, sizeGroups, 2, s, false)
		if err != nil {
			t.Fatalf("FindDuplicatesByHash() error = %v", err)
		}
		if len(report.Groups) != 0 {
			t.Errorf("FindDuplicatesByHash() returned %d duplicate groups, want 0", len(report.Groups))
		}

		// Test with empty input
		report, err = FindDuplicatesByHash(ctx, map[int64][]scanner.FileInfo{}, 2, s, false)
		if err != nil {
			t.Fatalf("FindDuplicatesByHash() error = %v", err)
		}
		if len(report.Groups) != 0 {
			t.Errorf("FindDuplicatesByHash() returned %d duplicate groups, want 0", len(report.Groups))
		}
		synctest.Wait()
	})
}

// A helper function to check if a slice contains all expected elements.
func containsAll(slice, expected []string) bool {
	if len(slice) != len(expected) {
		return false
	}

	// Create a map for O(1) lookups
	elementMap := make(map[string]bool)
	for _, s := range slice {
		elementMap[s] = true
	}

	// Check if all expected elements are in the map
	for _, e := range expected {
		if !elementMap[e] {
			return false
		}
	}

	return true
}
