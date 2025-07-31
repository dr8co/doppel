package scanner

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/dr8co/doppel/internal/config"
	"github.com/dr8co/doppel/internal/model"
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

	// Test a directory tree without any files
	sizeGroups, err := GroupFilesBySize([]string{tempDir}, &config.FilterConfig{}, &model.Stats{}, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(sizeGroups) != 0 {
		t.Errorf("Expected 0 size groups for empty directory tree")
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
	s := &model.Stats{}

	// Test GroupFilesBySize
	sizeGroups, err = GroupFilesBySize([]string{tempDir}, filterConfig, s, false)
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

	// All files skipped due to size
	filterConfig2 := &config.FilterConfig{MinSize: 1000}
	sizeGroups, err = GroupFilesBySize([]string{tempDir}, filterConfig2, &model.Stats{}, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(sizeGroups) != 0 {
		t.Errorf("Expected 0 size groups when all files skipped by size")
	}
}
func TestProcessDirectories_EmptyInput(t *testing.T) {
	// Should return the current directory as an absolute path
	dirs, err := processDirectories([]string{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(dirs) != 1 {
		t.Fatalf("Expected 1 directory, got %d", len(dirs))
	}
	absDot, _ := filepath.Abs(".")
	if dirs[0] != absDot {
		t.Errorf("Expected %s, got %s", absDot, dirs[0])
	}
}

func TestProcessDirectories_NonExistentDir(t *testing.T) {
	nonExistent := filepath.Join(os.TempDir(), "definitely-does-not-exist-12345")
	_, err := processDirectories([]string{nonExistent})
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected error for non-existent directory, got: %v", err)
	}
}

func TestProcessDirectories_NotADirectory(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "notadir")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func(name string) {
		_ = os.Remove(name)
	}(tmpFile.Name())

	_, err = processDirectories([]string{tmpFile.Name()})
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("Expected error for file input, got: %v", err)
	}
}

func TestProcessDirectories_Deduplication(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dedupe")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpDir)

	rel, _ := filepath.Rel(".", tmpDir)
	dirs, err := processDirectories([]string{tmpDir, tmpDir, rel})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(dirs) != 2 {
		t.Errorf("Expected 2 unique directories, got %d: %v", len(dirs), dirs)
	}
}

func TestProcessDirectories_SubdirectoryElimination(t *testing.T) {
	parent, err := os.MkdirTemp("", "parent")
	if err != nil {
		t.Fatalf("Failed to create parent dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(parent)

	sub := filepath.Join(parent, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	other, err := os.MkdirTemp("", "other")
	if err != nil {
		t.Fatalf("Failed to create other dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(other)

	dirs, err := processDirectories([]string{parent, sub, other})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Only parent and other should remain
	expected := map[string]bool{
		filepath.Clean(parent): true,
		filepath.Clean(other):  true,
	}

	if len(dirs) != 2 {
		t.Errorf("Expected 2 directories, got %d: %v", len(dirs), dirs)
	}

	for _, d := range dirs {
		if !expected[d] {
			t.Errorf("Unexpected directory in result: %s", d)
		}
	}
}

func TestProcessDirectories_MultipleSubdirectoryLevels(t *testing.T) {
	root, err := os.MkdirTemp("", "root")
	if err != nil {
		t.Fatalf("Failed to create root dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(root)

	sub1 := filepath.Join(root, "sub1")
	sub2 := filepath.Join(sub1, "sub2")
	if err := os.MkdirAll(sub2, 0755); err != nil {
		t.Fatalf("Failed to create nested subdirs: %v", err)
	}

	dirs, err := processDirectories([]string{root, sub1, sub2})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(dirs) != 1 || dirs[0] != filepath.Clean(root) {
		t.Errorf("Expected only root dir, got: %v", dirs)
	}
}

func TestProcessDirectories_SiblingDirs(t *testing.T) {
	root, err := os.MkdirTemp("", "root")
	if err != nil {
		t.Fatalf("Failed to create root dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(root)

	sub1 := filepath.Join(root, "sub1")
	sub2 := filepath.Join(root, "sub2")
	if err := os.MkdirAll(sub1, 0755); err != nil {
		t.Fatalf("Failed to create sub1: %v", err)
	}

	if err := os.MkdirAll(sub2, 0755); err != nil {
		t.Fatalf("Failed to create sub2: %v", err)
	}

	dirs, err := processDirectories([]string{sub1, sub2})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := map[string]bool{
		filepath.Clean(sub1): true,
		filepath.Clean(sub2): true,
	}

	if len(dirs) != 2 {
		t.Errorf("Expected 2 directories, got %d: %v", len(dirs), dirs)
	}

	for _, d := range dirs {
		if !expected[d] {
			t.Errorf("Unexpected directory in result: %s", d)
		}
	}
}
func TestRemoveSubdirectories_EmptyAndSingle(t *testing.T) {
	// Empty input
	result := removeSubdirectories([]string{})
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %v", result)
	}
	// Single directory
	result = removeSubdirectories([]string{"/foo"})
	if len(result) != 1 || result[0] != "/foo" {
		t.Errorf("Expected [/foo], got %v", result)
	}
}

func TestRemoveSubdirectories_NoSubdirectories(t *testing.T) {
	dirs := []string{"/a", "/b", "/c"}
	result := removeSubdirectories(dirs)
	expected := []string{"/a", "/b", "/c"}
	if len(result) != 3 {
		t.Errorf("Expected 3 directories, got %d: %v", len(result), result)
	}
	for _, dir := range expected {
		if !slices.Contains(result, dir) {
			t.Errorf("Expected directory %s in result", dir)
		}
	}
}

func TestRemoveSubdirectories_RemovesSubdirs(t *testing.T) {
	dirs := []string{"/foo", "/foo/bar", "/foo/bar/baz", "/baz"}
	result := removeSubdirectories(dirs)
	expected := []string{"/baz", "/foo"}
	if len(result) != 2 {
		t.Errorf("Expected 2 directories, got %d: %v", len(result), result)
	}
	for _, dir := range expected {
		if !slices.Contains(result, dir) {
			t.Errorf("Expected directory %s in result", dir)
		}
	}
}

func TestRemoveSubdirectories_SiblingSubdirs(t *testing.T) {
	dirs := []string{"/foo", "/foo/bar", "/foo/baz", "/foo/bar/qux"}
	result := removeSubdirectories(dirs)

	if len(result) != 1 || result[0] != "/foo" {
		t.Errorf("Expected only /foo, got %v", result)
	}
}

func TestRemoveSubdirectories_MixedOrder(t *testing.T) {
	dirs := []string{"/foo/bar", "/foo", "/baz", "/foo/bar/baz"}
	result := removeSubdirectories(dirs)

	expected := []string{"/baz", "/foo"}
	if len(result) != 2 {
		t.Errorf("Expected 2 directories, got %d: %v", len(result), result)
	}

	for _, dir := range expected {
		if !slices.Contains(result, dir) {
			t.Errorf("Expected directory %s in result", dir)
		}
	}
}

func TestRemoveSubdirectories_DuplicatePaths(t *testing.T) {
	dirs := []string{"/foo", "/foo", "/foo/bar", "/foo/bar"}
	result := removeSubdirectories(dirs)

	if len(result) != 2 && result[0] != "/foo" && result[1] != "/foo" {
		t.Errorf("Expected only [/foo /foo], got %v", result)
	}
}

func TestRemoveSubdirectories_ParentIsSubdirOfChild(t *testing.T) {
	// Should not remove parent if child comes first
	dirs := []string{"/foo/bar", "/foo"}
	result := removeSubdirectories(dirs)

	if len(result) != 1 || result[0] != "/foo" {
		t.Errorf("Expected only /foo, got %v", result)
	}
}

func TestRemoveSubdirectories_MultipleSubdirectories(t *testing.T) {
	dirs := []string{
		"/foo",
		"/foo/bar",
		"/foo", // duplicate
		"/usr/local/bin",
		"/foo/bar/baz",
		"/foo/bar/baz/qux",
		"/tmp",
		"/usr/local/bin", // duplicate
		"/usr/local/share",
		"/usr/share",
		"/home",
		"/usr/local",
		"/tmp/subdir",
	}

	expected := []string{
		"/foo",
		"/foo", // removeSubdirectories doesn't deduplicate
		"/home",
		"/tmp",
		"/usr/local",
		"/usr/share",
	}

	result := removeSubdirectories(dirs)
	if len(result) != len(expected) {
		t.Errorf("Expected %d, got %d: %v", len(expected), len(result), result)
	}
	for _, dir := range expected {
		if !slices.Contains(result, dir) {
			t.Errorf("Expected directory %s in result", dir)
		}
	}

}

func TestIsSubdirectory_BasicCases(t *testing.T) {
	// child is direct subdirectory of parent
	if !isSubdirectory("/foo/bar", "/foo") {
		t.Errorf("Expected /foo/bar to be subdirectory of /foo")
	}

	// child is nested subdirectory of parent
	if !isSubdirectory("/foo/bar/baz", "/foo") {
		t.Errorf("Expected /foo/bar/baz to be subdirectory of /foo")
	}

	// the child is not a subdirectory of the parent
	if isSubdirectory("/foo", "/foo/bar") {
		t.Errorf("Did not expect /foo to be subdirectory of /foo/bar")
	}

	// child and parent are the same
	if isSubdirectory("/foo/bar", "/foo/bar") {
		t.Errorf("Did not expect /foo/bar to be subdirectory of itself")
	}

	// child is not a subdirectory, just a prefix
	if isSubdirectory("/foobar", "/foo") {
		t.Errorf("Did not expect /foobar to be subdirectory of /foo")
	}
}

func TestIsSubdirectory_WithTrailingSlashes(t *testing.T) {
	// parent with trailing slash
	if !isSubdirectory("/foo/bar", "/foo/") {
		t.Errorf("Expected /foo/bar to be subdirectory of /foo/")
	}

	// child with trailing slash
	if !isSubdirectory("/foo/bar/", "/foo") {
		t.Errorf("Expected /foo/bar/ to be subdirectory of /foo")
	}

	// both with trailing slashes
	if !isSubdirectory("/foo/bar/", "/foo/") {
		t.Errorf("Expected /foo/bar/ to be subdirectory of /foo/")
	}
}

func TestIsSubdirectory_RelativePaths(t *testing.T) {
	// relative child and parent
	if !isSubdirectory("foo/bar", "foo") {
		t.Errorf("Expected foo/bar to be subdirectory of foo")
	}

	// relative, not a subdirectory
	if isSubdirectory("foo", "foo/bar") {
		t.Errorf("Did not expect foo to be subdirectory of foo/bar")
	}

	// relative, same path
	if isSubdirectory("foo/bar", "foo/bar") {
		t.Errorf("Did not expect foo/bar to be subdirectory of itself")
	}
}

func TestIsSubdirectory_DotAndDotDot(t *testing.T) {
	// "." and ".." should be handled correctly
	if isSubdirectory(".", ".") {
		t.Errorf("Did not expect . to be subdirectory of itself")
	}

	if isSubdirectory("..", ".") {
		t.Errorf("Did not expect .. to be subdirectory of .")
	}

	// "foo/../bar" is not a subdirectory of "foo"
	if isSubdirectory("foo/../bar", "foo") {
		t.Errorf("Did not expect foo/../bar to be subdirectory of foo")
	}
}

func TestIsSubdirectory_WindowsPaths(t *testing.T) {
	// Only run on Windows
	if os.PathSeparator != '\\' {
		t.Skip("Skipping Windows path test on non-Windows OS")
	}

	// Windows-style paths
	if !isSubdirectory(`C:\foo\bar`, `C:\foo`) {
		t.Errorf("Expected C:\\foo\\bar to be subdirectory of C:\\foo")
	}

	if isSubdirectory(`C:\foo`, `C:\foo\bar`) {
		t.Errorf("Did not expect C:\\foo to be subdirectory of C:\\foo\\bar")
	}

	if isSubdirectory(`C:\foo`, `C:\foo`) {
		t.Errorf("Did not expect C:\\foo to be subdirectory of itself")
	}
}

func TestIsSubdirectory_UnicodeAndSpecialChars(t *testing.T) {
	// Unicode and special characters
	if !isSubdirectory("/föö/bär", "/föö") {
		t.Errorf("Expected /föö/bär to be subdirectory of /föö")
	}

	if isSubdirectory("/föö", "/föö/bär") {
		t.Errorf("Did not expect /föö to be subdirectory of /föö/bär")
	}
}

func TestIsSubdirectory_EmptyStrings(t *testing.T) {
	if isSubdirectory("", "") {
		t.Errorf("Did not expect empty string to be subdirectory of itself")
	}

	if isSubdirectory("foo", "") {
		t.Errorf("Did not expect foo to be subdirectory of empty string")
	}

	if isSubdirectory("", "foo") {
		t.Errorf("Did not expect empty string to be subdirectory of foo")
	}

	fooWithSeparator := string(filepath.Separator) + "foo"
	if isSubdirectory(fooWithSeparator, "") {
		t.Errorf("Did not expect %s to be subdirectory of empty string", fooWithSeparator)
	}
	if isSubdirectory("", fooWithSeparator) {
		t.Errorf("Did not expect an empty string to be subdirectory of %s", fooWithSeparator)
	}

	if isSubdirectory("", ".foo") {
		t.Errorf("Did not expect an empty string to be subdirectory of .foo")
	}
	if isSubdirectory(".foo", "") {
		t.Errorf("Did not expect .foo to be subdirectory of an empty string")
	}
}

func TestIsSubdirectory_CommonPrefix(t *testing.T) {
	if isSubdirectory("foobar", "foobars") {
		t.Errorf("Did not expect /foobar to be subdirectory of /foobars")
	}

	if isSubdirectory("foobars", "foobar") {
		t.Errorf("Did not expect /foobars to be subdirectory of /foobar")
	}

	// similar names with separators
	if isSubdirectory("foo/bar", "foo/bars") {
		t.Errorf("Did not expect foo/bar to be subdirectory of foo/bars")
	}

	if isSubdirectory("foo/bars", "foo/bar") {
		t.Errorf("Did not expect foo/bars to be subdirectory of foo/bar")
	}

	// parent is a prefix of child, but not a directory boundary
	if isSubdirectory("foo/barbaz", "foo/bar") {
		t.Errorf("Did not expect foo/barbaz to be subdirectory of foo/bar")
	}

	// child is a prefix of parent, but not a directory boundary
	if isSubdirectory("foo/bar", "foo/barbaz") {
		t.Errorf("Did not expect foo/bar to be subdirectory of foo/barbaz")
	}

	// Edge case: parent is root
	if !isSubdirectory("/foo", "/") {
		t.Errorf("Expected /foo to be subdirectory of /")
	}
	if isSubdirectory("/", "/") {
		t.Errorf("Did not expect / to be subdirectory of itself")
	}

	// Windows drive letters (if running on Windows)
	if os.PathSeparator == '\\' {
		if isSubdirectory(`C:\foobar`, `C:\foobars`) {
			t.Errorf("Did not expect C:\\foobar to be subdirectory of C:\\foobars")
		}
		if isSubdirectory(`C:\foobars`, `C:\foobar`) {
			t.Errorf("Did not expect C:\\foobars to be subdirectory of C:\\foobar")
		}
	}
}
