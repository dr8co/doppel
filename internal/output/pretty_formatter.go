package output

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/dr8co/doppel/internal/model"
)

// PrettyFormatter formats duplicate reports in a human-readable way.
type PrettyFormatter struct{}

// NewPrettyFormatter creates a new [PrettyFormatter] instance.
func NewPrettyFormatter() *PrettyFormatter {
	return &PrettyFormatter{}
}

// Format formats the duplicate report in a human-readable way and writes it to the provided writer.
// Inspired by the Catppuccin theme.
func (f *PrettyFormatter) Format(report *model.DuplicateReport, w io.Writer) error {
	hasDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	lightDark := lipgloss.LightDark(hasDark)
	// groupHeaderStyle: Header for file groups. Mauve is distinct and pleasant.
	groupHeaderStyle := lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#8839ef"), lipgloss.Color("#cba6f7"))).
		Bold(true)

	// sizeStyle: File size. Teal provides good readability without being too loud.
	sizeStyle := lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#179299"), lipgloss.Color("#94e2d5")))

	// wastedStyle: Wasted space. Peach has a slight "warning" feel, perfect for this metric.
	wastedStyle := lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#fe640b"), lipgloss.Color("#fab387")))

	// fileStyle: The path of a duplicate file. Blue is a classic choice for file paths or links.
	fileStyle := lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#1e66f5"), lipgloss.Color("#89b4fa")))

	// summaryHeaderStyle: Header for the final statistics summary. Green feels positive and conclusive.
	summaryHeaderStyle := lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#40a02b"), lipgloss.Color("#a6e3a1"))).
		Bold(true)

	// statLabelStyle: The label for a statistic. A subdued color for contrast with the value.
	statLabelStyle := lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#6c6f85"), lipgloss.Color("#a6adc8")))

	// statValueStyle: The actual statistic value. The primary text color in bold makes it pop.
	statValueStyle := lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#4c4f69"), lipgloss.Color("#cdd6f4"))).
		Bold(true)

	// okStyle: For success messages. A clear and standard green.
	okStyle := lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#40a02b"), lipgloss.Color("#a6e3a1")))

	// errorStyle: For error messages. A clear and standard red.
	errorStyle := lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#d20f39"), lipgloss.Color("#f38ba8")))

	// rateStyle: For displaying rates. Yellow is great for dynamic or important numbers.
	rateStyle := lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#df8e1d"), lipgloss.Color("#f9e2af")))

	for _, group := range report.Groups {
		// Print group header
		header := groupHeaderStyle.Render(fmt.Sprintf("\n🔗 Duplicate group %d (%d files):", group.ID, group.Count))
		if _, err := lipgloss.Fprintln(w, header); err != nil {
			return err
		}

		// Print size and wasted space
		sizeStr := sizeStyle.Render(fmt.Sprintf("Size: %s each", FormatBytes(group.Size)))
		//nolint:gosec
		wastedStr := wastedStyle.Render(FormatBytes(int64(group.WastedSpace)) + " wasted space")
		if _, err := lipgloss.Fprintf(w, "   %s, %s\n", sizeStr, wastedStr); err != nil {
			return err
		}

		// Print files
		for _, file := range group.Files {
			fileLine := fileStyle.Render(fmt.Sprintf("📄 \"%s\"", file))
			if _, err := lipgloss.Fprintf(w, "   %s\n", fileLine); err != nil {
				return err
			}
		}
	}

	// Summary
	if _, err := lipgloss.Fprintln(w, summaryHeaderStyle.Render("\n📊 Summary:")); err != nil {
		return err
	}

	if report.Stats.DuplicateFiles > 0 {
		found := statLabelStyle.Render("🔗 Duplicate files found:") + " " + statValueStyle.Render(strconv.FormatUint(report.Stats.DuplicateFiles, 10)) +
			statLabelStyle.Render(" (in ") + statValueStyle.Render(strconv.FormatUint(report.Stats.DuplicateGroups, 10)) +
			statLabelStyle.Render(" group"+pluralize(report.Stats.DuplicateGroups)+")")
		if _, err := lipgloss.Fprintf(w, "   %s\n", found); err != nil {
			return err
		}
		//nolint:gosec
		wasted := wastedStyle.Render("💾 Total wasted space:") + " " + statValueStyle.Render(FormatBytes(int64(report.TotalWastedSpace)))
		if _, err := lipgloss.Fprintf(w, "   %s\n", wasted); err != nil {
			return err
		}
	} else {
		if _, err := lipgloss.Fprintf(w, "   %s\n", okStyle.Render("✅ No duplicate files found.")); err != nil {
			return err
		}
	}

	// Detailed stats
	if _, err := lipgloss.Fprintln(w, summaryHeaderStyle.Render("\n📈 Detailed Statistics:")); err != nil {
		return err
	}
	if _, err := lipgloss.Fprintf(w, "   %s %s\n", statLabelStyle.Render("📁 Total files scanned:"), statValueStyle.Render(strconv.FormatUint(report.Stats.TotalFiles, 10))); err != nil {
		return err
	}
	if _, err := lipgloss.Fprintf(w, "   %s %s\n", statLabelStyle.Render("🔐 Files processed for hashing:"), statValueStyle.Render(strconv.FormatUint(report.Stats.ProcessedFiles, 10))); err != nil {
		return err
	}
	if _, err := lipgloss.Fprintf(w, "   %s %s\n", statLabelStyle.Render("⏭️ Directories skipped:"), statValueStyle.Render(strconv.FormatUint(report.Stats.SkippedDirs, 10))); err != nil {
		return err
	}
	if _, err := lipgloss.Fprintf(w, "   %s %s\n", statLabelStyle.Render("⏭️ Files skipped:"), statValueStyle.Render(strconv.FormatUint(report.Stats.SkippedFiles, 10))); err != nil {
		return err
	}
	if _, err := lipgloss.Fprintf(w, "   %s %s\n", statLabelStyle.Render("❌ Files with errors:"), errorStyle.Render(strconv.FormatUint(report.Stats.ErrorCount, 10))); err != nil {
		return err
	}
	if _, err := lipgloss.Fprintf(w, "   %s %s\n", statLabelStyle.Render("⏱️ Processing time:"), statValueStyle.Render(report.Stats.Duration.Round(time.Millisecond).String())); err != nil {
		return err
	}

	// Show processing rate if applicable
	if report.Stats.ProcessedFiles > 0 && report.Stats.Duration > 0 {
		rate := float64(report.Stats.ProcessedFiles) / report.Stats.Duration.Seconds()
		if _, err := lipgloss.Fprintf(w, "   %s %.1f files/second\n", rateStyle.Render("🚀 Processing rate:"), rate); err != nil {
			return err
		}
	}

	return nil
}

// Name returns the name of the formatter.
func (f *PrettyFormatter) Name() string {
	return "pretty"
}

func pluralize(num uint64) string {
	if num < 2 {
		return ""
	}
	return "s"
}
