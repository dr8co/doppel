package output

import (
	"fmt"
	"io"

	"github.com/dr8co/doppel/internal/model"
)

// WritePaths writes every path in every duplicate group using the requested delimiter.
func WritePaths(report *model.DuplicateReport, w io.Writer, nullDelimited bool) error {
	delimiter := "\n"
	if nullDelimited {
		delimiter = "\x00"
	}

	for _, group := range report.Groups {
		for _, path := range group.Files {
			if _, err := fmt.Fprint(w, path, delimiter); err != nil {
				return err
			}
		}
	}

	return nil
}
