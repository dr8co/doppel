package scanner

import (
	"io"
	"os"

	"lukechampine.com/blake3"
)

const (
	chunkSize     = 64 * 1024 // 64 KB
	quickHashSize = 8 * 1024  // 8 KB for quick hash
)

// FileInfo represents a file with its path, size, and hash
type FileInfo struct {
	Path string `json:"path" yaml:"path"`
	Size int64  `json:"size" yaml:"size"`
	Hash string `json:"hash" yaml:"hash"`
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

// QuickHashFile computes a Blake3 hash of the first and the last portions of a file
// This is used as a quick preliminary check before computing the full hash
func QuickHashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	// Get file size
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	fileSize := info.Size()

	hasher := blake3.New(32, nil)

	// For small files, hash the entire content
	if fileSize <= quickHashSize {
		buf := make([]byte, quickHashSize)
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
	} else {
		// Hash first quickHashSize bytes
		buf := make([]byte, quickHashSize)
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return "", err
		}
		if n > 0 {
			if _, err2 := hasher.Write(buf[:n]); err2 != nil {
				return "", err2
			}
		}

		// Hash last quickHashSize bytes
		if fileSize > quickHashSize*2 {
			_, err = file.Seek(-quickHashSize, io.SeekEnd)
			if err != nil {
				return "", err
			}

			n, err = file.Read(buf)
			if err != nil && err != io.EOF {
				return "", err
			}
			if n > 0 {
				if _, err2 := hasher.Write(buf[:n]); err2 != nil {
					return "", err2
				}
			}
		}
	}

	return string(hasher.Sum(nil)), nil
}
