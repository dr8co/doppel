package display

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dr8co/doppel/internal/stats"
)

func TestShowResults(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "output_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Errorf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	// Create test files with specific sizes
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	file3 := filepath.Join(tempDir, "file3.txt")
	file4 := filepath.Join(tempDir, "file4.txt")

	// Create content of specific sizes
	content1KB := make([]byte, 1024) // 1KB
	content2KB := make([]byte, 2048) // 2KB

	// Write content to files
	if err := os.WriteFile(file1, content1KB, 0644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2, content1KB, 0644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}
	if err := os.WriteFile(file3, content2KB, 0644); err != nil {
		t.Fatalf("Failed to write file3: %v", err)
	}
	if err := os.WriteFile(file4, content2KB, 0644); err != nil {
		t.Fatalf("Failed to write file4: %v", err)
	}

	// Create duplicate groups
	duplicates := map[string][]string{
		"hash1": {file1, file2},
		"hash2": {file3, file4},
	}

	// Create stats object
	s := &stats.Stats{
		TotalFiles:      4,
		ProcessedFiles:  4,
		DuplicateGroups: 2,
		DuplicateFiles:  4,
		StartTime:       time.Now().Add(-1 * time.Second), // 1 second ago
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call the function being tested
	ShowResults(duplicates, s, true)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}
	output := buf.String()

	// Verify the output contains expected information
	expectedStrings := []string{
		"Duplicate group 1",
		"Duplicate group 2",
		"Size: 1.0 KB each",
		"Size: 2.0 KB each",
		"Duplicate files found: 4",
		"Total wasted space: 3.0 KB",
		"Total files scanned: 4",
		"Files processed for hashing: 4",
		"Processing time:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Output missing expected string: %s", expected)
		}
	}

	// Test with no duplicates
	r, w, _ = os.Pipe()
	os.Stdout = w

	ShowResults(map[string][]string{}, &stats.Stats{StartTime: time.Now()}, false)

	_ = w.Close()
	os.Stdout = oldStdout

	buf.Reset()
	_, err = io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}
	output = buf.String()

	if !strings.Contains(output, "No duplicate files found") {
		t.Errorf("Output should indicate no duplicates were found")
	}

	// Case: showStats = false, ErrorCount > 0
	s2 := &stats.Stats{
		ErrorCount: 2,
		StartTime:  time.Now().Add(-2 * time.Second),
	}

	oldStdout = os.Stdout
	r, w, _ = os.Pipe()
	os.Stdout = w

	ShowResults(map[string][]string{}, s2, false)

	_ = w.Close()
	os.Stdout = oldStdout

	buf.Reset()
	_, err = io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}
	output = buf.String()

	if !strings.Contains(output, "Files with errors: 2") {
		t.Errorf("Should show error count when showStats is false and errors exist")
	}

	// Case: showStats = true, with skipped dirs/files
	s3 := &stats.Stats{
		TotalFiles:     4,
		ProcessedFiles: 4,
		SkippedDirs:    1,
		SkippedFiles:   1,
		ErrorCount:     1,
		StartTime:      time.Now().Add(-1 * time.Second),
	}
	r, w, _ = os.Pipe()
	os.Stdout = w

	ShowResults(map[string][]string{"h": {file1, file1}}, s3, true)

	_ = w.Close()
	os.Stdout = oldStdout

	buf.Reset()
	_, err = io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}
	output = buf.String()

	for _, expected := range []string{
		"Directories skipped: 1",
		"Files skipped: 1",
		"Files with errors: 1",
		"Processing rate:",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("Output missing expected string: %s", expected)
		}

	}
}
