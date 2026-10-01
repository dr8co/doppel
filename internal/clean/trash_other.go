//go:build !linux
package clean

func (osTrash) Move(path string) error {
	return ErrUnsupportedTrash
}