package display

import (
	"os"
	"time"

	"github.com/dr8co/doppel/internal/stats"
)

// DuplicateGroup represents a group of duplicate files with their metadata
type DuplicateGroup struct {
	Id          int      `json:"id"`
	Count       int      `json:"count"`
	Size        int64    `json:"size"`
	WastedSpace uint64   `json:"wasted_space"`
	Files       []string `json:"files"`
}

// DuplicateReport represents the report of duplicate files found during a scan
type DuplicateReport struct {
	ScanDate         time.Time        `json:"scan_date"`
	Stats            *stats.Stats     `json:"stats"`
	TotalWastedSpace uint64           `json:"total_wasted_space"`
	Groups           []DuplicateGroup `json:"groups"`
}

// ConvertToReport converts a map of duplicate files to a DuplicateReport
func ConvertToReport(duplicates map[string][]string, s *stats.Stats) *DuplicateReport {
	report := &DuplicateReport{
		ScanDate:         time.Now(),
		Stats:            s,
		TotalWastedSpace: 0,
		Groups:           make([]DuplicateGroup, 0, len(duplicates)),
	}

	id := 0

	for _, files := range duplicates {
		if len(files) < 2 {
			continue
		}
		id++
		group := DuplicateGroup{
			Id:    id,
			Count: len(files),
			Files: files,
		}

		if info, err := os.Stat(files[0]); err == nil {
			group.Size = info.Size()
			group.WastedSpace = uint64(group.Size) * uint64(len(files)-1)
			report.TotalWastedSpace += group.WastedSpace
		}

		report.Groups = append(report.Groups, group)
	}
	return report
}
