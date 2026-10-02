//go:build darwin && (!cgo || ios)

package trash

// The macOS backend needs cgo (and trashItemAtURL does not exist on iOS).
func newBackend() backend { return unsupported{} }
