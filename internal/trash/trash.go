// Package trash moves files and directories to the operating system's
// trash / recycle bin instead of deleting them permanently.
//
// Platform behaviour:
//
//   - Linux: uses kioclient5, kioclient or gio (in that order of priority) if
//     one is found in PATH. Otherwise it implements the freedesktop.org Trash
//     specification 1.0 directly ($XDG_DATA_HOME/Trash, falling back to
//     $HOME/.local/share/Trash, plus per-mount-point trash directories for
//     files that live on another filesystem).
//   - macOS: NSFileManager's trashItemAtURL:resultingItemURL:error: (cgo).
//   - Windows: IFileOperation::DeleteItem with recycle semantics (cgo). An
//     item that cannot be recycled is never deleted permanently; Move fails
//     with ErrNotTrashable instead.
//   - Other Unix systems: the freedesktop.org Trash specification.
//   - Everything else (and darwin/windows builds without cgo): Move returns
//     an error wrapping errors.ErrUnsupported.
//
// Building on macOS needs the Xcode command line tools; building on Windows
// needs a MinGW-w64 C compiler (e.g. gcc) on PATH, as for any cgo program.
package trash

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ErrNotTrashable is returned (wrapped) when the item cannot be moved to a
// trash on its volume, e.g. a filesystem root, or a volume with no usable
// trash location. The item is left untouched.
var ErrNotTrashable = errors.New("item cannot be moved to the trash")

// Trash moves filesystem items to the trash.
type Trash interface {
	// Move moves the file, directory or symbolic link at path to the trash.
	// Symbolic links are trashed themselves, never their targets. Relative
	// paths are resolved against the current working directory.
	//
	// Errors are *fs.PathError values with Op "trash"; use errors.Is with
	// fs.ErrNotExist, fs.ErrPermission, ErrNotTrashable or
	// errors.ErrUnsupported to classify them.
	Move(path string) error
}

// backend is the platform-specific part. abs is a cleaned absolute path and
// fi is the result of os.Lstat(abs).
type backend interface {
	move(abs string, fi fs.FileInfo) error
}

type trasher struct{ b backend }

// New returns a Trash for the current platform. On Linux, the external tool
// (if any) is detected once, at construction time.
func New() Trash { return trasher{newBackend()} }

var defaultTrash = sync.OnceValue(New)

// Move moves path to the trash using a lazily created default Trash.
func Move(path string) error { return defaultTrash().Move(path) }

func (t trasher) Move(path string) error {
	abs, fi, err := resolve(path)
	if err != nil {
		return &fs.PathError{Op: "trash", Path: path, Err: err}
	}
	if err := t.b.move(abs, fi); err != nil {
		return &fs.PathError{Op: "trash", Path: path, Err: err}
	}
	return nil
}

// resolve validates path and returns its absolute form and Lstat info.
func resolve(path string) (string, fs.FileInfo, error) {
	if path == "" || strings.IndexByte(path, 0) >= 0 {
		return "", nil, fs.ErrInvalid
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}
	if filepath.Dir(abs) == abs { // filesystem root / drive root
		return "", nil, ErrNotTrashable
	}
	fi, err := os.Lstat(abs)
	if err != nil {
		if pe, ok := errors.AsType[*fs.PathError](err); ok {
			err = pe.Err
		}
		return "", nil, err
	}
	return abs, fi, nil
}

// unsupported is the backend for platforms without trash support.
type unsupported struct{}

func (unsupported) move(string, fs.FileInfo) error { return errors.ErrUnsupported }
