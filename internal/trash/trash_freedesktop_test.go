//go:build unix && !darwin

package trash

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fdTrash returns a Trash forced onto the freedesktop backend, a scratch
// directory to create files in, and the home trash directory in use.
func fdTrash(t *testing.T) (Trash, string, string) {
	t.Helper()
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	work, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return trasher{freedesktop{}}, work, filepath.Join(dataHome, "Trash")
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readInfo(t *testing.T, trash, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(trash, "info", name+".trashinfo"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestFileAndMetadata(t *testing.T) {
	tr, work, trash := fdTrash(t)
	p := filepath.Join(work, "hello world#1.txt")
	write(t, p, "data")

	if err := tr.Move(p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(p); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("original still exists: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(trash, "files", "hello world#1.txt"))
	if err != nil || string(b) != "data" {
		t.Fatalf("trashed content = %q, %v", b, err)
	}
	info := readInfo(t, trash, "hello world#1.txt")
	lines := strings.Split(strings.TrimSpace(info), "\n")
	if len(lines) != 3 || lines[0] != "[Trash Info]" {
		t.Fatalf("bad info file:\n%s", info)
	}
	if want := "Path=" + encodePath(p); lines[1] != want {
		t.Errorf("got %q, want %q", lines[1], want)
	}
	if !strings.HasPrefix(lines[2], "DeletionDate=") || len(lines[2]) != len("DeletionDate=2006-01-02T15:04:05") {
		t.Errorf("bad deletion date line %q", lines[2])
	}
}

func TestNameCollision(t *testing.T) {
	tr, work, trash := fdTrash(t)
	p := filepath.Join(work, "dup")
	for i, content := range []string{"one", "two", "three"} {
		write(t, p, content)
		if err := tr.Move(p); err != nil {
			t.Fatalf("move %d: %v", i, err)
		}
	}
	for name, want := range map[string]string{"dup": "one", "dup.2": "two", "dup.3": "three"} {
		b, err := os.ReadFile(filepath.Join(trash, "files", name))
		if err != nil || string(b) != want {
			t.Errorf("%s = %q, %v; want %q", name, b, err, want)
		}
		readInfo(t, trash, name) // must exist
	}
}

func TestDirectoryAndSizeCache(t *testing.T) {
	tr, work, trash := fdTrash(t)
	for _, name := range []string{"d1", "d2"} {
		d := filepath.Join(work, name)
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(d, "f"), strings.Repeat("x", 10000))
		if err := tr.Move(d); err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile(filepath.Join(trash, "directorysizes"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 cache lines, got:\n%s", b)
	}
	for i, name := range []string{"d1", "d2"} {
		f := strings.Fields(lines[i])
		if len(f) != 3 || f[2] != name || f[0] == "0" {
			t.Errorf("bad cache line %q", lines[i])
		}
	}
	if _, err := os.Stat(filepath.Join(trash, "files", "d1", "f")); err != nil {
		t.Errorf("directory contents not preserved: %v", err)
	}
}

func TestSymlinkIsTrashedNotTarget(t *testing.T) {
	tr, work, trash := fdTrash(t)
	target := filepath.Join(work, "target")
	link := filepath.Join(work, "link")
	write(t, target, "keep me")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := tr.Move(link); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("target was affected: %v", err)
	}
	if fi, err := os.Lstat(filepath.Join(trash, "files", "link")); err != nil || fi.Mode()&fs.ModeSymlink == 0 {
		t.Errorf("symlink not found in trash: %v", err)
	}
}

func TestErrors(t *testing.T) {
	tr, work, trash := fdTrash(t)

	if err := tr.Move(filepath.Join(work, "missing")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: got %v", err)
	}
	if err := tr.Move(""); !errors.Is(err, fs.ErrInvalid) {
		t.Errorf("empty path: got %v", err)
	}
	if err := tr.Move("/"); !errors.Is(err, ErrNotTrashable) {
		t.Errorf("root: got %v", err)
	}

	// The trash must never be moved into itself.
	write(t, filepath.Join(work, "x"), "x")
	if err := tr.Move(filepath.Join(work, "x")); err != nil {
		t.Fatal(err)
	}
	if err := tr.Move(trash); !errors.Is(err, ErrNotTrashable) {
		t.Errorf("trashing the trash: got %v", err)
	}
	if err := tr.Move(filepath.Join(trash, "files", "x")); !errors.Is(err, ErrNotTrashable) {
		t.Errorf("trashing inside the trash: got %v", err)
	}
}

func TestHomeFallbackWhenXDGDataHomeEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", home)
	got, err := homeTrash()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".local", "share", "Trash"); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestUniqueNameFitsNameMax(t *testing.T) {
	long := strings.Repeat("é", 200) // 400 bytes
	for _, n := range []int{1, 2, 123} {
		name := uniqueName(long, n)
		if len(name)+len(infoSuffix) > maxNameLen {
			t.Errorf("n=%d: %d bytes is too long", n, len(name)+len(infoSuffix))
		}
		if !utf8Valid(name) {
			t.Errorf("n=%d: truncation split a rune", n)
		}
	}
}

func utf8Valid(s string) bool { return strings.ToValidUTF8(s, "\uFFFD") == s }

func TestEncodePath(t *testing.T) {
	got := encodePath("/a b/ü#%.txt")
	if want := "/a%20b/%C3%BC%23%25.txt"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
