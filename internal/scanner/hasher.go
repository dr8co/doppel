package scanner

import (
	"io"
	"os"

	"github.com/zeebo/xxh3"
	"lukechampine.com/blake3"
)

const (
	chunkSize     = 64 * 1024 // 64 KB
	quickHashSize = 8 * 1024  // 8 KB for quick hash
)

// FileInfo represents a file with its path, size, and hash.
type FileInfo struct {
	Path string `json:"path" yaml:"path"`
	Size int64  `json:"size" yaml:"size"`
	Hash string `json:"hash" yaml:"hash"`
}

// HashFile computes Blake3 hash of the entire file.
func HashFile(filePath string) (string, error) {
	//nolint:gosec
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	hasher := blake3.New(32, nil)
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

// QuickHashFile computes a XXH3 hash of the first and the last portions of a file
// This is used as a quick preliminary check before computing the full hash.
func QuickHashFile(filePath string, size int64) (uint64, error) {
	if size <= 0 {
		return 0, nil
	}

	//nolint:gosec
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	buf := make([]byte, quickHashSize)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return 0, err
	}

	if size < quickHashSize*2 {
		// For small files, hash the entire content
		if n > 0 {
			return xxh3.Hash(buf[:n]), nil
		}
	} else {
		// Hash first quickHashSize bytes
		hasher := xxh3.New()
		if n > 0 {
			_, _ = hasher.Write(buf[:n])
		}

		// Hash last quickHashSize bytes
		_, err = file.Seek(-quickHashSize, io.SeekEnd)
		if err != nil {
			return 0, err
		}

		n, err = file.Read(buf)
		if err != nil && err != io.EOF {
			return 0, err
		}
		if n > 0 {
			_, _ = hasher.Write(buf[:n])
		}

		return hasher.Sum64(), nil
	}

	return 0, nil // unreachable!
}
