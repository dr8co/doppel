//go:build !unix && !windows

package trash

// No trash support on this platform: Move returns errors.ErrUnsupported.
func newBackend() backend { return unsupported{} }
