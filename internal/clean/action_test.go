package clean

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dr8co/doppel/internal/model"
)

func TestApplyDeleteAndDryRun(t *testing.T) {
	dir := t.TempDir()
	keeperPath := filepath.Join(dir, "keeper")
	removePath := filepath.Join(dir, "remove")
	for _, path := range []string{keeperPath, removePath} {
		if err := os.WriteFile(path, []byte("same"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	report := &model.DuplicateReport{Groups: []model.DuplicateGroup{{Files: []string{keeperPath, removePath}}}}
	targets, err := SelectTargets(report, PolicyFirst)
	if err != nil {
		t.Fatal(err)
	}
	results := Apply(targets, ModeDelete, true)
	if results[0].Err != nil {
		t.Fatal(results[0].Err)
	}
	if _, err := os.Stat(removePath); err != nil {
		t.Fatalf("dry-run removed target: %v", err)
	}

	results = Apply(targets, ModeDelete, false)
	if results[0].Err != nil {
		t.Fatal(results[0].Err)
	}
	if _, err := os.Stat(removePath); !os.IsNotExist(err) {
		t.Fatalf("target still exists, stat error = %v", err)
	}
}

func TestApplySkipsSameInode(t *testing.T) {
	dir := t.TempDir()
	keeperPath := filepath.Join(dir, "keeper")
	aliasPath := filepath.Join(dir, "alias")
	if err := os.WriteFile(keeperPath, []byte("same"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(keeperPath, aliasPath); err != nil {
		t.Fatal(err)
	}
	keeper, err := os.Lstat(keeperPath)
	if err != nil {
		t.Fatal(err)
	}
	alias, err := os.Lstat(aliasPath)
	if err != nil {
		t.Fatal(err)
	}
	results := Apply([]Target{{Keeper: File{Path: keeperPath, Info: keeper}, Remove: File{Path: aliasPath, Info: alias}}}, ModeDelete, false)
	if results[0].Err != nil {
		t.Fatal(results[0].Err)
	}
	if _, err := os.Stat(aliasPath); err != nil {
		t.Fatalf("same-inode target was mutated: %v", err)
	}
}

func TestApplyReplaceWithHardlink(t *testing.T) {
	dir := t.TempDir()
	keeperPath := filepath.Join(dir, "keeper")
	removePath := filepath.Join(dir, "remove")
	for _, path := range []string{keeperPath, removePath} {
		if err := os.WriteFile(path, []byte("same"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	keeper, err := os.Lstat(keeperPath)
	if err != nil {
		t.Fatal(err)
	}
	remove, err := os.Lstat(removePath)
	if err != nil {
		t.Fatal(err)
	}

	results := Apply([]Target{{Keeper: File{Path: keeperPath, Info: keeper}, Remove: File{Path: removePath, Info: remove}}}, ModeReplaceWithHardlink, false)
	if results[0].Err != nil {
		t.Fatalf("hardlink replacement failed: %v", results[0].Err)
	}
	keeper, err = os.Stat(keeperPath)
	if err != nil {
		t.Fatal(err)
	}
	remove, err = os.Stat(removePath)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(keeper, remove) {
		t.Fatal("replacement path does not share the keeper's file")
	}
}
