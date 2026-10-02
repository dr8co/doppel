//go:build linux

package trash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeTool installs an executable called name into dir that records its
// arguments (one per line) in dir/<name>.args and deletes the file named by
// its last argument, like a successful trash helper would.
func fakeTool(t *testing.T, dir, name string) {
	t.Helper()
	script := `#!/bin/sh
for a in "$@"; do printf '%s\n' "$a"; done > "$0.args"
if [ "$1" = move ]; then /bin/rm -rf -- "${2#file://}"; else /bin/rm -rf -- "$2"; fi
`
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func readArgs(t *testing.T, dir, name string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name+".args"))
	if err != nil {
		t.Fatalf("%s was not invoked: %v", name, err)
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

func TestToolPriority(t *testing.T) {
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	work := t.TempDir()

	// Only gio available.
	fakeTool(t, bin, "gio")
	p := filepath.Join(work, "ab.txt")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := New().Move(p); err != nil {
		t.Fatal(err)
	}
	if got := readArgs(t, bin, "gio"); len(got) != 2 || got[0] != "trash" || got[1] != p {
		t.Errorf("gio args = %q", got)
	}

	// kioclient is preferred over gio ...
	fakeTool(t, bin, "kioclient")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := New().Move(p); err != nil {
		t.Fatal(err)
	}
	got := readArgs(t, bin, "kioclient")
	if len(got) != 3 || got[0] != "move" || got[1] != "file://"+work+"/ab.txt" || got[2] != "trash:/" {
		t.Errorf("kioclient args = %q", got)
	}

	// ... and kioclient5 is preferred over both.
	fakeTool(t, bin, "kioclient5")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := New().Move(p); err != nil {
		t.Fatal(err)
	}
	readArgs(t, bin, "kioclient5")
}

func TestKioURLEscaping(t *testing.T) {
	got := kioArgs("/tmp/a b#c?d%e.txt")
	if want := "file:///tmp/a%20b%23c%3Fd%25e.txt"; got[1] != want {
		t.Errorf("got %s, want %s", got[1], want)
	}
}

func TestFallbackWithoutTools(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty: no helper available
	if _, ok := newBackend().(freedesktop); !ok {
		t.Errorf("expected the freedesktop fallback, got %T", newBackend())
	}
}

func TestToolFailureIsReported(t *testing.T) {
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	script := "#!/bin/sh\necho 'no trash here' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "gio"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	err := New().Move(p)
	if err == nil || !strings.Contains(err.Error(), "no trash here") {
		t.Errorf("got %v", err)
	}
	if _, statErr := os.Lstat(p); statErr != nil {
		t.Errorf("file must be untouched after failure: %v", statErr)
	}
}
