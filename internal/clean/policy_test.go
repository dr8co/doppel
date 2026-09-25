package clean

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dr8co/doppel/internal/model"
)

func TestSelectTargetsPolicies(t *testing.T) {
	dir := t.TempDir()
	paths := []string{
		filepath.Join(dir, "long", "file.txt"),
		filepath.Join(dir, "short.txt"),
		filepath.Join(dir, "middle.txt"),
	}
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("same"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Unix(100, 0)
	new := time.Unix(200, 0)
	if err := os.Chtimes(paths[0], old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(paths[1], new, new); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(paths[2], old, old); err != nil {
		t.Fatal(err)
	}

	report := &model.DuplicateReport{Groups: []model.DuplicateGroup{{Files: paths}}}
	tests := []struct {
		policy string
		keeper string
	}{
		{PolicyNewest, paths[1]},
		{PolicyOldest, paths[0]},
		{PolicyShortestPath, paths[1]},
		{PolicyFirst, paths[0]},
	}
	for _, test := range tests {
		t.Run(test.policy, func(t *testing.T) {
			targets, err := SelectTargets(report, test.policy)
			if err != nil {
				t.Fatal(err)
			}
			if len(targets) != 2 {
				t.Fatalf("target count = %d, want 2", len(targets))
			}
			if targets[0].Keeper.Path != test.keeper {
				t.Fatalf("keeper = %q, want %q", targets[0].Keeper.Path, test.keeper)
			}
		})
	}
}

func TestSelectTargetsRequiresPolicy(t *testing.T) {
	if _, err := SelectTargets(&model.DuplicateReport{}, ""); err != ErrPolicyRequired {
		t.Fatalf("SelectTargets() error = %v, want %v", err, ErrPolicyRequired)
	}
}
