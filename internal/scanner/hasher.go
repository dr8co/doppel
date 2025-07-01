package scanner

import (
	"fmt"
	"io"
	"lukechampine.com/blake3"
	"os"
)

// FileInfo represents a file with its path, size, and hash
type FileInfo struct {
	Path string
	Size int64
	Hash string
}

// HashFile computes Blake3 hash of the entire file
func HashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	hasher := blake3.New(32, nil)
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
