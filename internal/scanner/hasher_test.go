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

func TestQuickHashFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "quick_hasher_test")
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
			name:    "small file (less than quick hash size)",
			content: "Hello, world!",
			wantErr: false,
		},
		{
			name:    "medium file (exactly quick hash size)",
			content: string(make([]byte, quickHashSize)), // 8KB of zero bytes
			wantErr: false,
		},
		{
			name:    "large file (more than quick hash size)",
			content: string(make([]byte, quickHashSize*3)), // 24KB of zero bytes
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

			// Call the function being tested
			gotHash, err := QuickHashFile(filePath)

			// Check for errors
			if (err != nil) != tt.wantErr {
				t.Errorf("QuickHashFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we expect an error, don't check the hash
			if tt.wantErr {
				return
			}

			// Verify the hash is not empty
			if gotHash == "" {
				t.Errorf("QuickHashFile() returned empty hash")
			}

			// For small files, quick hash should equal full hash
			if len(tt.content) <= quickHashSize {
				fullHash, err := HashFile(filePath)
				if err != nil {
					t.Fatalf("Failed to compute full hash for comparison: %v", err)
				}
				if gotHash != fullHash {
					t.Errorf("QuickHashFile() = %v, want %v (should equal full hash for small files)", gotHash, fullHash)
				}
			}
		})
	}

	// Test with a non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		_, err := QuickHashFile(filepath.Join(tempDir, "non-existent-file"))
		if err == nil {
			t.Errorf("QuickHashFile() expected error for non-existent file, got nil")
		}
	})
}

func TestQuickHashConsistency(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "quick_hash_consistency_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Errorf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	// Create two files with identical content
	content := make([]byte, quickHashSize*3) // 24KB - enough to have a middle section
	for i := range content {
		content[i] = byte(i % 256)
	}

	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")

	err = os.WriteFile(file1, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	err = os.WriteFile(file2, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Compute quick hashes
	hash1, err := QuickHashFile(file1)
	if err != nil {
		t.Fatalf("Failed to compute quick hash for file1: %v", err)
	}

	hash2, err := QuickHashFile(file2)
	if err != nil {
		t.Fatalf("Failed to compute quick hash for file2: %v", err)
	}

	// Hashes should be identical for identical files
	if hash1 != hash2 {
		t.Errorf("QuickHashFile() produced different hashes for identical files: %v vs %v", hash1, hash2)
	}

	// Create a file with different content (same size, different middle)
	// For a 24KB file: the first 8 KB (0-8191) and last 8KB (16384-24575) are hashed
	// So we can safely change content in the middle section (8192-16383)
	differentContent := make([]byte, quickHashSize*3)
	copy(differentContent, content)
	// Change content in the middle section that won't be hashed by quick hash
	differentContent[quickHashSize+1000] = 255 // Position 9216, which is in the middle section

	file3 := filepath.Join(tempDir, "file3.txt")
	err = os.WriteFile(file3, differentContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create file3: %v", err)
	}

	hash3, err := QuickHashFile(file3)
	if err != nil {
		t.Fatalf("Failed to compute quick hash for file3: %v", err)
	}

	// Quick hashes should be the same (since only beginning and end are hashed)
	if hash1 != hash3 {
		t.Errorf("QuickHashFile() produced different hashes for files with same beginning/end: %v vs %v", hash1, hash3)
	}

	// But full hashes should be different
	fullHash1, err := HashFile(file1)
	if err != nil {
		t.Fatalf("Failed to compute full hash for file1: %v", err)
	}

	fullHash3, err := HashFile(file3)
	if err != nil {
		t.Fatalf("Failed to compute full hash for file3: %v", err)
	}

	if fullHash1 == fullHash3 {
		t.Errorf("HashFile() produced same hashes for different files: %v", fullHash1)
	}
}
