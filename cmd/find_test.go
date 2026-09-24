package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/dr8co/doppel/internal/config"
	"github.com/dr8co/doppel/internal/model"
)

func newTestFindCommand() *cli.Command {
	cfg := config.DefaultConfig()
	return FindCommand(&cfg.Find, func(context.Context, *cli.Command) (*config.Config, error) {
		return config.DefaultConfig(), nil
	})
}

// TestFindCommandRejectsConflictingProgressFlags ensures that the find command returns an error when both --quiet and --verbose are specified.
func TestFindCommandRejectsConflictingProgressFlags(t *testing.T) {
	err := newTestFindCommand().Run(context.Background(), []string{"find", "--quiet", "--verbose"})
	if err == nil || err.Error() != "--quiet cannot be combined with --verbose" {
		t.Fatalf("Run() error = %v, want quiet/verbose conflict", err)
	}
}

func TestFindCommandFailOnDuplicatesWritesOutput(t *testing.T) {
	tempDir := t.TempDir()
	first := filepath.Join(tempDir, "first.txt")
	second := filepath.Join(tempDir, "second.txt")
	outputFile := filepath.Join(tempDir, "report.txt")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("duplicate"), 0o600); err != nil {
			t.Fatalf("write test file: %v", err)
		}
	}

	err := newTestFindCommand().Run(context.Background(), []string{
		"find", "--files", "--quiet", "--fail-on-duplicates", "--output-file", outputFile, first, second,
	})
	if !errors.Is(err, ErrDuplicatesFound) {
		t.Fatalf("Run() error = %v, want ErrDuplicatesFound", err)
	}

	contents, readErr := os.ReadFile(outputFile)
	if readErr != nil {
		t.Fatalf("read report: %v", readErr)
	}
	if len(contents) == 0 {
		t.Fatal("report is empty after duplicate failure")
	}
}

func TestFindCommandFilesFromAndFilterBypass(t *testing.T) {
	tempDir := t.TempDir()
	first := filepath.Join(tempDir, "first.log")
	second := filepath.Join(tempDir, "second.log")
	listFile := filepath.Join(tempDir, "paths.txt")
	fromReport := filepath.Join(tempDir, "from.json")
	filesReport := filepath.Join(tempDir, "files.json")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("duplicate"), 0o600); err != nil {
			t.Fatalf("write test file: %v", err)
		}
	}
	if err := os.WriteFile(listFile, []byte(first+"\n"+second+"\n"), 0o600); err != nil {
		t.Fatalf("write file list: %v", err)
	}

	if err := newTestFindCommand().Run(context.Background(), []string{
		"find", "--files-from", listFile, "--quiet", "--output-format", "json", "--output-file", fromReport,
	}); err != nil {
		t.Fatalf("files-from run: %v", err)
	}
	if report := readTestReport(t, fromReport); len(report.Groups) != 1 {
		t.Fatalf("files-from groups = %d, want 1", len(report.Groups))
	}

	if err := newTestFindCommand().Run(context.Background(), []string{
		"find", "--files", "--exclude-files", "*.log", "--quiet", "--output-format", "json", "--output-file", filesReport, first, second,
	}); err != nil {
		t.Fatalf("explicit-files run: %v", err)
	}
	if report := readTestReport(t, filesReport); len(report.Groups) != 1 {
		t.Fatalf("explicit-files groups = %d, want 1 despite file filter", len(report.Groups))
	}
}

func TestFindCommandFilesFromRejectsInvalidPath(t *testing.T) {
	tempDir := t.TempDir()
	listFile := filepath.Join(tempDir, "paths.txt")
	if err := os.WriteFile(listFile, []byte(filepath.Join(tempDir, "missing")), 0o600); err != nil {
		t.Fatalf("write file list: %v", err)
	}

	err := newTestFindCommand().Run(context.Background(), []string{"find", "--files-from", listFile, "--quiet"})
	if err == nil || !strings.Contains(err.Error(), "path does not exist") {
		t.Fatalf("Run() error = %v, want missing-path error", err)
	}
}

func TestFindCommandRejectsInvalidSortMode(t *testing.T) {
	err := newTestFindCommand().Run(context.Background(), []string{"find", "--sort", "bad", "--quiet"})
	if err == nil || !strings.Contains(err.Error(), "invalid --sort") {
		t.Fatalf("Run() error = %v, want invalid --sort error", err)
	}
}

func readTestReport(t *testing.T, path string) model.DuplicateReport {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report model.DuplicateReport
	if err := json.Unmarshal(contents, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	return report
}
