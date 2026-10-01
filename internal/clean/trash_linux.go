package clean

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (osTrash) Move(path string) error {
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
