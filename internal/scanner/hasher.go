package scanner

import (
	"io"
	"os"

	"lukechampine.com/blake3"
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
	const chunkSize = 64 * 1024 // 64 KB
	buf := make([]byte, chunkSize)

	for {
		n, err := file.Read(buf)
		if n > 0 {
			if _, err2 := hasher.Write(buf[:n]); err2 != nil {
				return "", err2
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return string(hasher.Sum(nil)), nil
}
