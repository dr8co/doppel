package output

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dr8co/doppel/internal/model"
)

// PrettyFormatter formats duplicate reports in a human-readable way
type PrettyFormatter struct{}

// NewPrettyFormatter creates a new PrettyFormatter instance
func NewPrettyFormatter() *PrettyFormatter {
	return &PrettyFormatter{}
}

// Format formats the duplicate report in a human-readable way and writes it to the provided writer
func (f *PrettyFormatter) Format(report *model.DuplicateReport, w io.Writer) error {
	renderer := lipgloss.NewRenderer(w)

	// Define the styles using True Color (24-bit hex codes)
	groupHeaderStyle := renderer.NewStyle().Foreground(lipgloss.Color("#FF4DDC")).Bold(true)   // magenta pink
	sizeStyle := renderer.NewStyle().Foreground(lipgloss.Color("#00CFFF"))                     // cyan blue
	wastedStyle := renderer.NewStyle().Foreground(lipgloss.Color("#FF7F50"))                   // coral orange
	fileStyle := renderer.NewStyle().Foreground(lipgloss.Color("#00BFFF"))                     // deep sky blue
	summaryHeaderStyle := renderer.NewStyle().Foreground(lipgloss.Color("#B266FF")).Bold(true) // purple
	statLabelStyle := renderer.NewStyle().Foreground(lipgloss.Color("#A0A0A0"))                // gray
	statValueStyle := renderer.NewStyle().Foreground(lipgloss.Color("#00FF99")).Bold(true)     // spring green
	okStyle := renderer.NewStyle().Foreground(lipgloss.Color("#00FF66"))                       // green
	errorStyle := renderer.NewStyle().Foreground(lipgloss.Color("#FF3333"))                    // red
	rateStyle := renderer.NewStyle().Foreground(lipgloss.Color("#FFD700"))                     // gold

	for _, group := range report.Groups {
		// Print group header
		header := groupHeaderStyle.Render(fmt.Sprintf("\n🔗 Duplicate group %d (%d files):", group.Id, group.Count))
		if _, err := fmt.Fprintln(w, header); err != nil {
			return err
		}

		// Print size and wasted space
		sizeStr := sizeStyle.Render(fmt.Sprintf("Size: %s each", FormatBytes(group.Size)))
		wastedStr := wastedStyle.Render(fmt.Sprintf("%s wasted space", FormatBytes(int64(group.WastedSpace))))
		if _, err := fmt.Fprintf(w, "   %s, %s\n", sizeStr, wastedStr); err != nil {
			return err
		}

		// Print files
		for _, file := range group.Files {
			fileLine := fileStyle.Render(fmt.Sprintf("📄 \"%s\"", file))
			if _, err := fmt.Fprintf(w, "   %s\n", fileLine); err != nil {
				return err
			}
		}
	}

	// Summary
	if _, err := fmt.Fprintln(w, summaryHeaderStyle.Render("\n📊 Summary:")); err != nil {
		return err
	}

	if report.Stats.DuplicateFiles > 0 {
		found := statLabelStyle.Render("🔗 Duplicate files found:") + " " + statValueStyle.Render(fmt.Sprintf("%d", report.Stats.DuplicateFiles)) +
			statLabelStyle.Render(" (in ") + statValueStyle.Render(fmt.Sprintf("%d", report.Stats.DuplicateGroups)) + statLabelStyle.Render(" groups)")
		if _, err := fmt.Fprintf(w, "   %s\n", found); err != nil {
			return err
		}
		wasted := wastedStyle.Render("💾 Total wasted space:") + " " + statValueStyle.Render(FormatBytes(int64(report.TotalWastedSpace)))
		if _, err := fmt.Fprintf(w, "   %s\n", wasted); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "   %s\n", okStyle.Render("✅ No duplicate files found")); err != nil {
			return err
		}
	}

	// Detailed stats
	if _, err := fmt.Fprintln(w, summaryHeaderStyle.Render("\n📈 Detailed Statistics:")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "   %s %s\n", statLabelStyle.Render("📁 Total files scanned:"), statValueStyle.Render(fmt.Sprintf("%d", report.Stats.TotalFiles))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "   %s %s\n", statLabelStyle.Render("🔐 Files processed for hashing:"), statValueStyle.Render(fmt.Sprintf("%d", report.Stats.ProcessedFiles))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "   %s %s\n", statLabelStyle.Render("⏭️ Directories skipped:"), statValueStyle.Render(fmt.Sprintf("%d", report.Stats.SkippedDirs))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "   %s %s\n", statLabelStyle.Render("⏭️ Files skipped:"), statValueStyle.Render(fmt.Sprintf("%d", report.Stats.SkippedFiles))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "   %s %s\n", statLabelStyle.Render("❌ Files with errors:"), errorStyle.Render(fmt.Sprintf("%d", report.Stats.ErrorCount))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "   %s %s\n", statLabelStyle.Render("⏱️ Processing time:"), statValueStyle.Render(report.Stats.Duration.Round(time.Millisecond).String())); err != nil {
		return err
	}

	// Show processing rate if applicable
	if report.Stats.ProcessedFiles > 0 && report.Stats.Duration > 0 {
		rate := float64(report.Stats.ProcessedFiles) / report.Stats.Duration.Seconds()
		if _, err := fmt.Fprintf(w, "   %s %.1f files/second\n", rateStyle.Render("🚀 Processing rate:"), rate); err != nil {
			return err
		}
	}

	return nil
}

// Name returns the name of the formatter
func (f *PrettyFormatter) Name() string {
	return "pretty"
}
