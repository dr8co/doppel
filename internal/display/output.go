package display

import (
	"fmt"
	"os"
	"time"

	"github.com/dr8co/doppel/internal/stats"
)

// ShowResults shows the duplicate files found and optionally displays statistics
func ShowResults(duplicates map[string][]string, s *stats.Stats, showStats bool) {
	groupCount := 0
	var totalSize int64

	for _, files := range duplicates {
		groupCount++
		fmt.Printf("\n🔗 Duplicate group %d (%d files):\n", groupCount, len(files))

		// Get file size for the group
		if len(files) > 0 {
			if info, err := os.Stat(files[0]); err == nil {
				groupSize := info.Size()
				wastedSpace := groupSize * int64(len(files)-1)
				totalSize += wastedSpace
				fmt.Printf("   Size: %s each, %s wasted space\n", stats.FormatBytes(groupSize), stats.FormatBytes(wastedSpace))
			}
		}

		for _, file := range files {
			fmt.Printf("   📄 %s\n", file)
		}
	}

	fmt.Printf("\n📊 Summary:\n")
	if s.DuplicateFiles > 0 {
		fmt.Printf("   🔗 Duplicate files found: %d (in %d groups)\n", s.DuplicateFiles, s.DuplicateGroups)
		fmt.Printf("   💾 Total wasted space: %s\n", stats.FormatBytes(totalSize))
	} else {
		fmt.Printf("   ✅ No duplicate files found\n")
	}

	if showStats {
		duration := time.Since(s.StartTime)
		fmt.Printf("\n📈 Detailed Statistics:\n")
		fmt.Printf("   📁 Total files scanned: %d\n", s.TotalFiles)
		fmt.Printf("   🔐 Files processed for hashing: %d\n", s.ProcessedFiles)
		fmt.Printf("   ⏭️  Directories skipped: %d\n", s.SkippedDirs)
		fmt.Printf("   ⏭️  Files skipped: %d\n", s.SkippedFiles)
		fmt.Printf("   ❌ Files with errors: %d\n", s.ErrorCount)
		fmt.Printf("   ⏱️  Processing time: %v\n", duration.Round(time.Millisecond))
		if s.ProcessedFiles > 0 && duration > 0 {
			rate := float64(s.ProcessedFiles) / duration.Seconds()
			fmt.Printf("   🚀 Processing rate: %.1f files/second\n", rate)
		}
	} else if s.ErrorCount > 0 {
		fmt.Printf("   ❌ Files with errors: %d\n", s.ErrorCount)
	}
}
