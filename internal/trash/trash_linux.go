//go:build linux

package trash

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// tools lists the supported helpers in order of priority.
var tools = []struct {
	name string
	args func(abs string) []string
}{
	{"kioclient5", kioArgs},
	{"kioclient", kioArgs},
	{"gio", gioArgs},
}

// kio takes URLs; a properly escaped file:// URL avoids any ambiguity with
// characters such as '#', '?' or '%' in the path.
func kioArgs(abs string) []string {
	u := url.URL{Scheme: "file", Path: abs}
	return []string{"move", u.String(), "trash:/"}
}

// abs always starts with '/', so it can never be mistaken for an option.
func gioArgs(abs string) []string { return []string{"trash", abs} }

// newBackend picks the first available helper, or falls back to a direct
// implementation of the freedesktop.org Trash specification.
func newBackend() backend {
	for _, t := range tools {
		if exe, err := exec.LookPath(t.name); err == nil {
			return execBackend{exe: exe, args: t.args}
		}
	}
	return freedesktop{}
}

type execBackend struct {
	exe  string
	args func(abs string) []string
}

func (b execBackend) move(abs string, _ fs.FileInfo) error {
	name := filepath.Base(b.exe)
	out, err := exec.Command(b.exe, b.args(abs)...).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("%s: %w: %s", name, err, msg)
		}
		return fmt.Errorf("%s: %w", name, err)
	}
	// Don't trust the exit status alone: the item must really be gone.
	if _, err := os.Lstat(abs); err == nil {
		return fmt.Errorf("%s reported success but %s still exists", name, abs)
	}
	return nil
}
