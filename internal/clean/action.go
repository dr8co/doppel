// Package clean selects duplicate files for removal and applies cleanup actions.
package clean

import (
	"errors"
	"fmt"
	"os"
)

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
		return replaceWithHardlink(target.Keeper.Path, target.Remove.Path)
	default:
		return fmt.Errorf("%w %q", ErrInvalidMode, mode)
	}
}

func replaceWithHardlink(keeperPath, targetPath string) error {
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
