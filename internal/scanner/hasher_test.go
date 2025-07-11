package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"lukechampine.com/blake3"
)

func TestHashFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "hasher_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Errorf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "empty file",
			content: "",
			wantErr: false,
		},
		{
			name:    "small file",
			content: "Hello, world!",
			wantErr: false,
		},
		{
			name:    "larger file",
			content: string(make([]byte, 1024)), // 1KB of zero bytes
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test file with the specified content
			filePath := filepath.Join(tempDir, tt.name)
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Compute the expected hash
			hasher := blake3.New(32, nil)
			_, err = hasher.Write([]byte(tt.content))
			if err != nil {
				t.Fatalf("Failed to compute expected hash: %v", err)
			}
			expectedHash := fmt.Sprintf("%x", hasher.Sum(nil))

			// Call the function being tested
			gotHash, err := HashFile(filePath)

			// Check for errors
			if (err != nil) != tt.wantErr {
				t.Errorf("HashFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we expect an error, don't check the hash
			if tt.wantErr {
				return
			}

			// Verify the hash matches what we expect
			if bytesToHex(gotHash) != expectedHash {
				t.Errorf("HashFile() = %v, want %v", gotHash, expectedHash)
			}
		})
	}

	// Test with a non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		_, err := HashFile(filepath.Join(tempDir, "non-existent-file"))
		if err == nil {
			t.Errorf("HashFile() expected error for non-existent file, got nil")
		}
	})
}

func TestFileInfo(t *testing.T) {
	// Test the FileInfo struct
	fileInfo := FileInfo{
		Path: "/path/to/file",
		Size: 1024,
		Hash: "abcdef1234567890",
	}

	if fileInfo.Path != "/path/to/file" {
		t.Errorf("FileInfo.Path = %v, want %v", fileInfo.Path, "/path/to/file")
	}

	if fileInfo.Size != 1024 {
		t.Errorf("FileInfo.Size = %v, want %v", fileInfo.Size, 1024)
	}

	if fileInfo.Hash != "abcdef1234567890" {
		t.Errorf("FileInfo.Hash = %v, want %v", fileInfo.Hash, "abcdef1234567890")
	}
}

func bytesToHex(b string) string {
	return fmt.Sprintf("%x", b)
}
