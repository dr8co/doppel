package scanner

import (
	"errors"
	"hash"
	"io"
	"os"

	"github.com/zeebo/xxh3"
)

const quickHashSize = 8 * 1024 // 8 KB for quick hash

// FileInfo represents a file with its path, size, and hash.
type FileInfo struct {
	Path string `json:"path" yaml:"path"`
	Size int64  `json:"size" yaml:"size"`
	Hash string `json:"hash" yaml:"hash"`
}

// HashFile computes the hash of an entire file.
func HashFile(filePath string, buf []byte, hasher hash.Hash) (string, error) {
	if hasher == nil {
		return "", errors.New("hasher is nil")
	}
	//nolint:gosec
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	hasher.Reset()

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
func QuickHashFile(filePath string, size int64, buf []byte, hasher *xxh3.Hasher) (uint64, error) {
	if size <= 0 {
		return 0, nil
	}
	if hasher == nil {
		return 0, errors.New("hasher is nil")
	}

	//nolint:gosec
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	hasher.Reset()

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
