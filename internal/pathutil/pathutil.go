// Package pathutil provides small helpers for validating and resolving
// filesystem paths used throughout the project.
package pathutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Custom error types
var (
	ErrNotExist     = errors.New("path does not exist")
	ErrNotDirectory = errors.New("not a directory")
	ErrIsDirectory  = errors.New("is a directory")
	ErrNotRegular   = errors.New("not a regular file")
)

// cleanAndResolve cleans, makes absolute, and resolves symlinks.
func cleanAndResolve(path string) (string, error) {
	cleaned := filepath.Clean(path)

	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("cannot make absolute: %w", err)
	}

	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("cannot resolve symlinks: %w", err)
	}

	return resolved, nil
}

// ValidateRegularFile ensures the path exists and is a regular file.
// It resolves symlinks.
func ValidateRegularFile(path string) (string, error) {
	resolved, err := cleanAndResolve(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrNotExist, resolved)
		}
		return "", err
	}

	// Check if the file exists
	info, err := os.Stat(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrNotExist, resolved)
		}
		return "", err
	}

	if info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrIsDirectory, resolved)
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: %s", ErrNotRegular, resolved)
	}

	return resolved, nil
}

// ValidateDirectory ensures the path exists and is a directory.
func ValidateDirectory(path string) (string, error) {
	resolved, err := cleanAndResolve(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrNotExist, resolved)
		}
		return "", err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrNotExist, resolved)
		}
		return "", err
	}

	if !info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrNotDirectory, resolved)
	}

	return resolved, nil
}
