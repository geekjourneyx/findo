package output

import (
	"fmt"
	"io"

	"github.com/geekjourneyx/findo/internal/search"
)

func WriteMarkdown(w io.Writer, env search.Envelope) error {
	for _, result := range env.Results {
		if _, err := fmt.Fprintf(w, "## %s\n\n", result.Title); err != nil {
			return err
		}
		if result.URL != "" {
			if _, err := fmt.Fprintf(w, "%s\n\n", result.URL); err != nil {
				return err
			}
		}
		text := firstNonEmpty(result.Content, result.Snippet)
		if text != "" {
			if _, err := fmt.Fprintf(w, "%s\n\n", text); err != nil {
				return err
			}
		}
		for _, citation := range result.Citations {
			if _, err := fmt.Fprintf(w, "- [%s](%s)\n", citation.Title, citation.URL); err != nil {
				return err
			}
		}
		if len(result.Citations) > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
