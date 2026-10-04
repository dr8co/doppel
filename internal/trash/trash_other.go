//go:build !unix && !windows

package trash

const trashSupported bool = false

// No trash support on this platform: Move returns errors.ErrUnsupported.
func newBackend() backend { return unsupported{} }
