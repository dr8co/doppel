package clean

import (
	"errors"
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
