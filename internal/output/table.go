package output

import (
	"fmt"
	"io"

	"github.com/geekjourneyx/findo/internal/search"
)

func WriteTable(w io.Writer, env search.Envelope) error {
	for _, result := range env.Results {
		if result.URL != "" {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", result.Source, result.Title, result.URL); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\n", result.Source, result.Title); err != nil {
			return err
		}
	}
	return nil
}
