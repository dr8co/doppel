//go:build unix && !darwin && !linux

package trash

// Other Unix systems (the BSDs, Solaris/illumos, AIX, ...) are assumed to
// follow the freedesktop.org Trash specification.
func newBackend() backend { return freedesktop{} }
