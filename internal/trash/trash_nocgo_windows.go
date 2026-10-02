//go:build windows && !cgo

package trash

// The Windows backend needs cgo.
func newBackend() backend { return unsupported{} }
