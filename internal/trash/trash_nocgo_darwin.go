//go:build darwin && (!cgo || ios)

package trash

const trashSupported bool = false

// The macOS backend needs cgo (and trashItemAtURL does not exist on iOS).
func newBackend() backend { return unsupported{} }
