package scanner

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// FileInfo represents a file with its path, size, and hash
type FileInfo struct {
	Path string
	Size int64
	Hash string
}

// HashFile computes SHA-256 hash of the entire file
func HashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	hasher := sha256.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
