//go:build unix && !darwin

package trash

// Implementation of the freedesktop.org Trash specification, version 1.0:
// https://specifications.freedesktop.org/trash/latest/
//
// Layout of a trash directory ($trash):
//
//	$trash/files/<name>            the trashed item
//	$trash/info/<name>.trashinfo   original path + deletion date
//	$trash/directorysizes          optional cache of trashed directory sizes

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

const (
	infoSuffix  = ".trashinfo"
	maxNameLen  = 255 // NAME_MAX on practically every filesystem
	maxAttempts = 10000
)

type freedesktop struct{}

func (freedesktop) move(abs string, fi fs.FileInfo) error {
	// Resolve symlinks in the parent directory only. The last component must
	// stay as is so that a symlink is trashed itself, not its target.
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return err
	}
	abs = filepath.Join(parent, filepath.Base(abs))

	dev, ok := deviceOf(fi)
	if !ok {
		return errors.New("cannot determine the device of the path")
	}
	pfi, err := os.Stat(parent)
	if err != nil {
		return err
	}
	if pdev, _ := deviceOf(pfi); pdev != dev {
		return fmt.Errorf("%w: %s is a mount point", ErrNotTrashable, abs)
	}

	// Same filesystem as the home trash: use the home trash.
	home, err := homeTrash()
	if err != nil {
		return err
	}
	hfi, err := os.Stat(home)
	if err != nil {
		return err
	}
	if hdev, _ := deviceOf(hfi); hdev == dev {
		return trashInto(home, "", abs, fi)
	}

	// Different filesystem: use a trash directory in its top directory.
	// We refuse rather than copy the data into the home trash.
	top, err := topDir(parent, dev)
	if err != nil {
		return err
	}
	dir, err := topdirTrash(top)
	if err != nil {
		return err
	}
	return trashInto(dir, top, abs, fi)
}

func deviceOf(fi fs.FileInfo) (uint64, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	//nolint:unconvert
	return uint64(st.Dev), true
}

// homeTrash returns $XDG_DATA_HOME/Trash, creating it if needed. Per the XDG
// Base Directory spec, an empty or relative $XDG_DATA_HOME is ignored and
// $HOME/.local/share is used instead.
func homeTrash() (string, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" || !filepath.IsAbs(dataHome) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(home) {
			return "", fmt.Errorf("home directory %q is not absolute", home)
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(dataHome, "Trash")
	if err := ensureTrashDir(dir, false); err != nil {
		return "", err
	}
	return dir, nil
}

// topDir returns the mount point of the filesystem (device dev) containing
// the directory parent, by walking up while the device does not change.
func topDir(parent string, dev uint64) (string, error) {
	cur := parent
	for {
		up := filepath.Dir(cur)
		if up == cur {
			return cur, nil
		}
		fi, err := os.Stat(up)
		if err != nil {
			return "", err
		}
		if d, _ := deviceOf(fi); d != dev {
			return cur, nil
		}
		cur = up
	}
}

// topdirTrash returns (creating if needed) the per-user trash directory for
// the filesystem mounted at top: first $top/.Trash/$uid when $top/.Trash is an
// administrator-created sticky directory (method 1), else $top/.Trash-$uid
// (method 2).
func topdirTrash(top string) (string, error) {
	uid := strconv.Itoa(os.Getuid())

	shared := filepath.Join(top, ".Trash")
	// Lstat: a symlink must not be used, and the sticky bit is mandatory.
	if fi, err := os.Lstat(shared); err == nil && fi.IsDir() && fi.Mode()&fs.ModeSticky != 0 {
		dir := filepath.Join(shared, uid)
		if err := ensureTrashDir(dir, true); err == nil {
			return dir, nil
		}
	}

	dir := filepath.Join(top, ".Trash-"+uid)
	if err := ensureTrashDir(dir, true); err != nil {
		return "", fmt.Errorf("%w: no usable trash on %s: %w", ErrNotTrashable, top, err)
	}
	return dir, nil
}

// ensureTrashDir creates dir with its files/ and info/ subdirectories (mode
// 0700). With strict set, dir must be a real directory (not a symlink) owned
// by the current user, as it may live in a world-writable location.
func ensureTrashDir(dir string, strict bool) error {
	if strict {
		//nolint:gosec
		if err := os.Mkdir(dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
		//nolint:gosec
		fi, err := os.Lstat(dir)
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			return fmt.Errorf("%s is not a directory", dir)
		}
		if st, ok := fi.Sys().(*syscall.Stat_t); !ok || int(st.Uid) != os.Getuid() {
			return fmt.Errorf("%s is not owned by the current user", dir)
		}
	} else {
		//nolint:gosec
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	for _, sub := range []string{"files", "info"} {
		//nolint:gosec
		if err := os.Mkdir(filepath.Join(dir, sub), 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
	}
	return nil
}

// trashInto moves abs into the trash directory trashDir. For the home trash
// top is "" and the info file records the absolute path; for a top-directory
// trash it records the path relative to top.
func trashInto(trashDir, top, abs string, fi fs.FileInfo) error {
	if realDir, err := filepath.EvalSymlinks(trashDir); err == nil {
		trashDir = realDir
	}
	if isWithin(abs, trashDir) || isWithin(trashDir, abs) {
		return fmt.Errorf("%w: %s overlaps the trash directory", ErrNotTrashable, abs)
	}

	origPath := abs
	if top != "" {
		rel, err := filepath.Rel(top, abs)
		if err != nil {
			return err
		}
		origPath = rel
	}
	info := "[Trash Info]\nPath=" + encodePath(origPath) +
		"\nDeletionDate=" + time.Now().Format("2006-01-02T15:04:05") + "\n"

	filesDir := filepath.Join(trashDir, "files")
	infoDir := filepath.Join(trashDir, "info")
	base := filepath.Base(abs)

	for n := 1; n <= maxAttempts; n++ {
		name := uniqueName(base, n)
		infoPath := filepath.Join(infoDir, name+infoSuffix)
		dst := filepath.Join(filesDir, name)

		// The spec requires creating the info file first, atomically
		// (O_EXCL), so concurrent trashers never pick the same name.
		//nolint:gosec
		f, err := os.OpenFile(infoPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return err
		}

		// A stray entry without info file must not be overwritten either.
		if _, err := os.Lstat(dst); err == nil {
			_ = f.Close()
			_ = os.Remove(infoPath)
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			_ = f.Close()
			_ = os.Remove(infoPath)
			return err
		}

		_, werr := io.WriteString(f, info)
		cerr := f.Close()
		if err := errors.Join(werr, cerr); err != nil {
			_ = os.Remove(infoPath)
			return err
		}

		if err := os.Rename(abs, dst); err != nil {
			_ = os.Remove(infoPath)
			return err
		}
		if fi.IsDir() {
			// Best effort: the cache is optional and readers recompute
			// missing entries.
			_ = updateDirectorySizes(trashDir, name, infoPath, dst)
		}
		return nil
	}
	return errors.New("could not find a unique name in the trash")
}

// uniqueName returns base for n == 1 and base.n otherwise, truncated so that
// the name plus ".trashinfo" fits into a single path component.
func uniqueName(base string, n int) string {
	suffix := ""
	if n > 1 {
		suffix = "." + strconv.Itoa(n)
	}
	limit := maxNameLen - len(infoSuffix) - len(suffix)
	if len(base) > limit {
		for limit > 0 && !utf8.RuneStart(base[limit]) {
			limit--
		}
		base = base[:limit]
	}
	return base + suffix
}

// isWithin reports whether path equals dir or lies inside it.
func isWithin(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// encodePath percent-encodes p as in URLs (RFC 2396): everything except
// unreserved characters and '/' becomes %XX, byte by byte.
func encodePath(p string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := range len(p) {
		c := p[i]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9',
			c == '-', c == '_', c == '.', c == '~', c == '/':
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&15])
		}
	}
	return b.String()
}

// updateDirectorySizes adds "<size> <mtime> <encoded name>" for the trashed
// directory to $trash/directorysizes. The size is the disk usage as computed
// by `du -B1`; mtime is that of the .trashinfo file. The file is replaced via
// a temporary file and an atomic rename, as the spec requires.
func updateDirectorySizes(trashDir, name, infoPath, dst string) error {
	size, err := diskUsage(dst)
	if err != nil {
		return err
	}
	ifi, err := os.Stat(infoPath)
	if err != nil {
		return err
	}
	encName := encodePath(name)
	entry := fmt.Sprintf("%d %d %s\n", size, ifi.ModTime().Unix(), encName)

	cachePath := filepath.Join(trashDir, "directorysizes")
	var kept strings.Builder
	//nolint:gosec
	if old, err := os.ReadFile(cachePath); err == nil {
		for line := range strings.SplitSeq(string(old), "\n") {
			if line == "" {
				continue
			}
			if parts := strings.SplitN(line, " ", 3); len(parts) == 3 && parts[2] == encName {
				continue // stale entry for a previous item of the same name
			}
			kept.WriteString(line)
			kept.WriteString("\n")
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	kept.WriteString(entry)

	tmp, err := os.CreateTemp(trashDir, "directorysizes.*.tmp")
	if err != nil {
		return err
	}
	if _, err := tmp.WriteString(kept.String()); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), cachePath); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return nil
}

// diskUsage mimics `du -B1 -s`: the sum of st_blocks*512 over the directory
// and everything below it, counting hard-linked files once and never
// following symlinks.
func diskUsage(root string) (int64, error) {
	type inode struct{ dev, ino uint64 }
	seen := make(map[inode]struct{})
	var total int64
	err := filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		st, ok := fi.Sys().(*syscall.Stat_t)
		if !ok {
			return errors.New("no stat information available")
		}
		if st.Nlink > 1 && !d.IsDir() {
			//nolint:unconvert
			key := inode{uint64(st.Dev), uint64(st.Ino)}
			if _, dup := seen[key]; dup {
				return nil
			}
			seen[key] = struct{}{}
		}
		//nolint:unconvert
		total += int64(st.Blocks) * 512
		return nil
	})
	return total, err
}
