//go:build windows && !cgo

package trash

const trashSupported bool = false

// The Windows backend needs cgo.
func newBackend() backend { return unsupported{} }
