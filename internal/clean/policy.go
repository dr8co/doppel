package clean

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dr8co/doppel/internal/model"
)

const (
	// PolicyNewest keeps the most recently modified file.
	PolicyNewest = "newest"
	// PolicyOldest keeps the least recently modified file.
	PolicyOldest = "oldest"
	// PolicyShortestPath keeps the file with the shortest cleaned path.
	PolicyShortestPath = "shortest-path"
	// PolicyFirst keeps the file with the lexically first path.
	PolicyFirst = "first"

	// ModeDelete permanently deletes the selected duplicate.
	ModeDelete = "delete"
	// ModeTrash moves the selected duplicate to a recoverable trash location.
	ModeTrash = "trash"
	// ModeReplaceWithHardlink replaces the selected duplicate with a hard link
	// to the keeper.
	ModeReplaceWithHardlink = "replace-with-hardlink"
)

var (
	// ErrPolicyRequired indicates that a retention policy was not provided.
	ErrPolicyRequired = errors.New("a keep policy is required")
	// ErrInvalidPolicy indicates that a retention policy is not supported.
	ErrInvalidPolicy = errors.New("invalid keep policy")
	// ErrInvalidMode indicates that an action mode is not supported.
	ErrInvalidMode = errors.New("invalid action mode")
)

// File describes a regular file considered by the clean operation.
type File struct {
	// Path is the file path.
	Path string
	// Info contains the file's metadata.
	Info os.FileInfo
	// ModTime is the file's modification time.
	ModTime time.Time
}

// Target pairs the file to keep with the duplicate file to remove.
type Target struct {
	// Keeper is the file retained by the selected policy.
	Keeper File
	// Remove is the duplicate selected for action.
	Remove File
}

// ValidatePolicy reports whether policy is a supported retention policy.
func ValidatePolicy(policy string) error {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case PolicyNewest, PolicyOldest, PolicyShortestPath, PolicyFirst:
		return nil
	case "":
		return ErrPolicyRequired
	default:
		return fmt.Errorf("%w %q", ErrInvalidPolicy, policy)
	}
}

// ValidateMode reports whether mode is a supported cleanup action.
func ValidateMode(mode string) error {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case ModeDelete, ModeTrash, ModeReplaceWithHardlink:
		return nil
	default:
		return fmt.Errorf("%w %q", ErrInvalidMode, mode)
	}
}

// SelectTargets selects keeper/removal pairs from duplicate groups according
// to policy. It validates each path and requires regular files.
func SelectTargets(report *model.DuplicateReport, policy string) ([]Target, error) {
	policy = strings.ToLower(strings.TrimSpace(policy))
	if err := ValidatePolicy(policy); err != nil {
		return nil, err
	}

	var targets []Target
	for _, group := range report.Groups {
		files := make([]File, 0, len(group.Files))
		for _, path := range group.Files {
			info, err := os.Lstat(path)
			if err != nil {
				return nil, fmt.Errorf("stat %s: %w", path, err)
			}
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("clean target is not a regular file: %s", path)
			}
			files = append(files, File{Path: path, Info: info, ModTime: info.ModTime()})
		}
		if len(files) < 2 {
			continue
		}
		sort.Slice(files, func(i, j int) bool {
			return files[i].Path < files[j].Path
		})
		keeper := files[0]
		for _, file := range files[1:] {
			if preferred(file, keeper, policy) {
				keeper = file
			}
		}
		for _, file := range files {
			if file.Path != keeper.Path {
				targets = append(targets, Target{Keeper: keeper, Remove: file})
			}
		}
	}
	return targets, nil
}

func preferred(candidate, current File, policy string) bool {
	switch policy {
	case PolicyNewest:
		return candidate.ModTime.After(current.ModTime)
	case PolicyOldest:
		return candidate.ModTime.Before(current.ModTime)
	case PolicyShortestPath:
		candidateLen := len(filepath.Clean(candidate.Path))
		currentLen := len(filepath.Clean(current.Path))
		return candidateLen < currentLen
	case PolicyFirst:
		return candidate.Path < current.Path
	default:
		return false
	}
}
