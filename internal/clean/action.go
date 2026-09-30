// Package clean selects duplicate files for removal and applies cleanup actions.
package clean

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// ErrUnsupportedTrash indicates that the trash action is unavailable on the
// current platform or that no trash backend was provided.
var ErrUnsupportedTrash = errors.New("trash is not supported on this platform")

// Trash moves a file to a recoverable trash location.
type Trash interface {
	// Move moves path to the trash location.
	Move(path string) error
}

// TrashFunc adapts a function into a Trash implementation.
type TrashFunc func(string) error

// Move calls f with path.
func (f TrashFunc) Move(path string) error {
	return f(path)
}

type osTrash struct{}

// NewOSTrash returns a Trash implementation that uses the operating system's
// user trash location.
func NewOSTrash() Trash {
	return osTrash{}
}

func (osTrash) Move(path string) error {
	if runtime.GOOS != "linux" {
		return ErrUnsupportedTrash
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("find home directory: %w", err)
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	trashDir := filepath.Join(dataHome, "Trash")
	filesDir := filepath.Join(trashDir, "files")
	infoDir := filepath.Join(trashDir, "info")
	//nolint:gosec
	if err := os.MkdirAll(filesDir, 0o700); err != nil {
		return fmt.Errorf("create trash directory: %w", err)
	}
	//nolint:gosec
	if err := os.MkdirAll(infoDir, 0o700); err != nil {
		return fmt.Errorf("create trash metadata directory: %w", err)
	}

	name := filepath.Base(path)
	if name == "." || name == string(filepath.Separator) || strings.TrimSpace(name) == "" {
		return fmt.Errorf("invalid trash name for %s", path)
	}
	for index := 0; ; index++ {
		candidate := name
		if index > 0 {
			candidate = fmt.Sprintf("%s.%d", name, index)
		}
		trashedPath := filepath.Join(filesDir, candidate)
		infoPath := filepath.Join(infoDir, candidate+".trashinfo")
		//nolint:gosec
		if _, err := os.Lstat(trashedPath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("check trash path: %w", err)
		}
		metadata := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", url.PathEscape(path), time.Now().UTC().Format("20060102T150405"))
		//nolint:gosec
		if err := os.WriteFile(infoPath, []byte(metadata), 0o600); err != nil {
			return fmt.Errorf("write trash metadata: %w", err)
		}
		//nolint:gosec
		if err := os.Rename(path, trashedPath); err != nil {
			_ = os.Remove(infoPath)
			return fmt.Errorf("move %s to trash: %w", path, err)
		}
		return nil
	}
}

// Result describes the outcome of applying a cleanup action to a target.
type Result struct {
	// Target is the keeper/removal pair considered by the action.
	Target Target
	// Mode is the action mode used for the result.
	Mode string
	// DryRun reports whether the action was only previewed.
	DryRun bool
	// Skipped reports whether the target was skipped.
	Skipped bool
	// Err contains the action error, if any.
	Err error
}

// Apply applies mode to each target and returns one result per target. When
// dryRun is true, targets are reported without being changed.
func Apply(targets []Target, mode string, dryRun bool, trash Trash) []Result {
	results := make([]Result, 0, len(targets))
	for _, target := range targets {
		result := Result{Target: target, Mode: mode, DryRun: dryRun}
		if dryRun {
			results = append(results, result)
			continue
		}
		result.Err = applyTarget(target, mode, trash)
		results = append(results, result)
	}
	return results
}

func applyTarget(target Target, mode string, trash Trash) error {
	keeperInfo, err := os.Lstat(target.Keeper.Path)
	if err != nil {
		return fmt.Errorf("stat keeper %s: %w", target.Keeper.Path, err)
	}
	removeInfo, err := os.Lstat(target.Remove.Path)
	if err != nil {
		return fmt.Errorf("stat target %s: %w", target.Remove.Path, err)
	}
	if !keeperInfo.Mode().IsRegular() || !removeInfo.Mode().IsRegular() {
		return errors.New("clean targets must be regular files")
	}
	if os.SameFile(keeperInfo, removeInfo) {
		return nil
	}

	switch mode {
	case ModeDelete:
		return os.Remove(target.Remove.Path)
	case ModeTrash:
		if trash == nil {
			return ErrUnsupportedTrash
		}
		return trash.Move(target.Remove.Path)
	case ModeReplaceWithHardlink:
		return replaceWithHardlink(target.Keeper.Path, target.Remove.Path, keeperInfo, removeInfo)
	default:
		return fmt.Errorf("%w %q", ErrInvalidMode, mode)
	}
}

func replaceWithHardlink(keeperPath, targetPath string, keeperInfo, targetInfo os.FileInfo) error {
	if keeperInfo.Sys() == nil || targetInfo.Sys() == nil {
		return errors.New("cannot compare filesystems for hardlink replacement")
	}
	keeperDevice, ok := deviceID(keeperInfo)
	if !ok {
		return errors.New("cannot determine keeper filesystem")
	}
	targetDevice, ok := deviceID(targetInfo)
	if !ok || keeperDevice != targetDevice {
		return errors.New("keeper and target are on different filesystems")
	}

	tempPath := targetPath + ".doppel-hardlink-tmp"
	for attempt := range 100 {
		candidate := tempPath
		if attempt > 0 {
			candidate = fmt.Sprintf("%s.%d", tempPath, attempt)
		}
		if _, err := os.Lstat(candidate); err != nil && errors.Is(err, os.ErrNotExist) {
			tempPath = candidate
			break
		}
	}
	if err := os.Link(keeperPath, tempPath); err != nil {
		return fmt.Errorf("create hardlink for %s: %w", targetPath, err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace %s with hardlink: %w", targetPath, err)
	}
	return nil
}

func deviceID(info os.FileInfo) (uint64, bool) {
	return deviceIDFromSys(info.Sys())
}

func deviceIDFromSys(value any) (uint64, bool) {
	switch stat := value.(type) {
	case *syscall.Stat_t:
		//nolint:unconvert // stat.Dev is not uint64 on all platforms, e.g. darwin.
		return uint64(stat.Dev), true
	default:
		return 0, false
	}
}
