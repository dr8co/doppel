package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/internal/config"
)

func newTestCleanCommand() *cli.Command {
	cfg := config.DefaultConfig()
	return CleanCommand(&cfg.Clean, func(context.Context, *cli.Command) (*config.Config, error) {
		return config.DefaultConfig(), nil
	})
}

func makeDuplicateFiles(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	first := filepath.Join(dir, "first.txt")
	second := filepath.Join(dir, "second.txt")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("duplicate"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return first, second
}

func TestCleanRequiresKeepPolicy(t *testing.T) {
	first, second := makeDuplicateFiles(t)
	err := newTestCleanCommand().Run(context.Background(), []string{"clean", "--files", "--yes", first, second})
	if err == nil || !strings.Contains(err.Error(), "keep policy is required") {
		t.Fatalf("Run() error = %v, want required-policy error", err)
	}
}

func TestCleanDryRunDoesNotMutate(t *testing.T) {
	first, second := makeDuplicateFiles(t)
	err := newTestCleanCommand().Run(context.Background(), []string{"clean", "--files", "--keep", "first", "--dry-run", first, second})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(first); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(second); err != nil {
		t.Fatal(err)
	}
}

func TestCleanDeleteWithYes(t *testing.T) {
	first, second := makeDuplicateFiles(t)
	err := newTestCleanCommand().Run(context.Background(), []string{"clean", "--files", "--keep", "first", "--yes", second, first})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(second); !os.IsNotExist(err) {
		t.Fatalf("second file still exists, stat error = %v", err)
	}
	if _, err := os.Stat(first); err != nil {
		t.Fatalf("keeper was removed: %v", err)
	}
}
