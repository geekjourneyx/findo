package output

import (
	"encoding/json"
	"io"

	"github.com/geekjourneyx/findo/internal/findoerr"
	"github.com/geekjourneyx/findo/internal/search"
)

func WriteJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	value = normalizeValue(value)
	return enc.Encode(value)
}

func normalizeValue(value any) any {
	switch v := value.(type) {
	case search.Envelope:
		return normalizeEnvelope(v)
	case *search.Envelope:
		if v == nil {
			return v
		}
		normalized := normalizeEnvelope(*v)
		return &normalized
	default:
		return value
	}
}

func normalizeEnvelope(env search.Envelope) search.Envelope {
	if env.Results == nil {
		env.Results = []search.Result{}
	}
	if env.SourceStatus == nil {
		env.SourceStatus = []search.SourceStatus{}
	}
	if env.Errors == nil {
		env.Errors = []findoerr.Error{}
	}
	return env
}
