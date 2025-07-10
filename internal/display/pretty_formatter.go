package display

import (
	"fmt"
	"io"
	"time"

	"github.com/dr8co/doppel/internal/stats"
)

// PrettyFormatter formats duplicate reports in a human-readable way
type PrettyFormatter struct{}

// NewPrettyFormatter creates a new PrettyFormatter instance
func NewPrettyFormatter() *PrettyFormatter {
	return &PrettyFormatter{}
}

// Format formats the duplicate report in a human-readable way and writes it to the provided writer
func (f *PrettyFormatter) Format(report *DuplicateReport, w io.Writer) error {
	for _, group := range report.Groups {
		// Print group header
		if _, err := fmt.Fprintf(w, "\n🔗 Duplicate group %d (%d files):\n", group.Id, group.Count); err != nil {
			return err
		}

		// Print size and wasted space
		if _, err := fmt.Fprintf(w, "   Size: %s each, %s wasted space\n", stats.FormatBytes(group.Size), stats.FormatBytes(int64(group.WastedSpace))); err != nil {
			return err
		}

		// Print files
		for _, file := range group.Files {
			if _, err := fmt.Fprintf(w, "   📄 %s\n", file); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintf(w, "\n📊 Summary:\n"); err != nil {
		return err
	}

	if report.Stats.DuplicateFiles > 0 {
		if _, err := fmt.Fprintf(w, "   🔗 Duplicate files found: %d (in %d groups)\n", report.Stats.DuplicateFiles, report.Stats.DuplicateGroups); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "   💾 Total wasted space: %s\n", stats.FormatBytes(int64(report.TotalWastedSpace))); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "   ✅ No duplicate files found\n"); err != nil {
			return err
		}
	}

	// Show detailed stats if showStats is true in ShowResults, but here always print detailed stats
	duration := time.Since(report.Stats.StartTime)
	if _, err := fmt.Fprintf(w, "\n📈 Detailed Statistics:\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "   📁 Total files scanned: %d\n", report.Stats.TotalFiles); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "   🔐 Files processed for hashing: %d\n", report.Stats.ProcessedFiles); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "   ⏭️ Directories skipped: %d\n", report.Stats.SkippedDirs); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "   ⏭️ Files skipped: %d\n", report.Stats.SkippedFiles); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "   ❌ Files with errors: %d\n", report.Stats.ErrorCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "   ⏱️ Processing time: %v\n", duration.Round(time.Millisecond)); err != nil {
		return err
	}
	if report.Stats.ProcessedFiles > 0 && duration > 0 {
		rate := float64(report.Stats.ProcessedFiles) / duration.Seconds()
		if _, err := fmt.Fprintf(w, "   🚀 Processing rate: %.1f files/second\n", rate); err != nil {
			return err
		}
	}

	return nil
}
