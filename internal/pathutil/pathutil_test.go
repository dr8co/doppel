package pathutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestValidateRegularFile tests the [ValidateRegularFile] function.
func TestValidateRegularFile(t *testing.T) {
	// Create a temp dir and file for testing
	tmpDir, err := os.MkdirTemp("", "pathutil_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatal(err)
		}
	}(tmpDir)

	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create symlink
	symlink := filepath.Join(tmpDir, "symlink.txt")
	if err := os.Symlink(tmpFile, symlink); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{
			name: "valid regular file",
			path: tmpFile,
		},
		{
			name: "valid symlink to regular file",
			path: symlink,
		},
		{
			name:    "directory",
			path:    tmpDir,
			wantErr: ErrIsDirectory,
		},
		{
			name:    "non-existent",
			path:    filepath.Join(tmpDir, "nonexistent"),
			wantErr: ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateRegularFile(tt.path)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateRegularFile() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("ValidateRegularFile() unexpected error: %v", err)
			}
			if got == "" {
				t.Error("ValidateRegularFile() returned empty path")
			}
		})
	}
}

// TestValidateDirectory tests the [ValidateDirectory] function.
func TestValidateDirectory(t *testing.T) {
	// Create temp dir for testing
	tmpDir, err := os.MkdirTemp("", "pathutil_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatal(err)
		}
	}(tmpDir)

	// Create a regular file
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create symlink to directory
	symlink := filepath.Join(tmpDir, "symlink")
	if err := os.Symlink(tmpDir, symlink); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{
			name: "valid directory",
			path: tmpDir,
		},
		{
			name: "valid symlink to directory",
			path: symlink,
		},
		{
			name:    "regular file",
			path:    tmpFile,
			wantErr: ErrNotDirectory,
		},
		{
			name:    "non-existent",
			path:    filepath.Join(tmpDir, "nonexistent"),
			wantErr: ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateDirectory(tt.path)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateDirectory() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("ValidateDirectory() unexpected error: %v", err)
			}
			if got == "" {
				t.Error("ValidateDirectory() returned empty path")
			}
		})
	}
}
