package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/geekjourneyx/findo/internal/findoerr"
	"github.com/geekjourneyx/findo/internal/search"
)

func TestWriteJSONEnvelopeIncludesRequiredArraysAndEffectiveLimit(t *testing.T) {
	env := search.Envelope{
		Version: "1.0.0",
		Query: search.Query{
			Text:    "AI 搜索",
			Mode:    search.QueryModeMixed,
			Sources: []search.SourceID{search.SourceBochaWeb},
			Limit:   10,
		},
		Status:  search.StatusOK,
		Results: []search.Result{},
		SourceStatus: []search.SourceStatus{{
			Source:         search.SourceBochaWeb,
			Status:         search.SourceStatusOK,
			Results:        0,
			EffectiveLimit: 10,
			DurationMS:     1,
			Error:          nil,
		}},
		Errors: []findoerr.Error{},
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, env); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if decoded["results"] == nil || decoded["errors"] == nil {
		t.Fatalf("arrays must be present: %s", buf.String())
	}
	status := decoded["source_status"].([]any)[0].(map[string]any)
	if status["effective_limit"] != float64(10) {
		t.Fatalf("effective_limit missing: %s", buf.String())
	}
	if _, ok := status["error"]; !ok {
		t.Fatalf("error key must be present even when null: %s", buf.String())
	}
}

func TestWriteJSONEnvelopeNormalizesNilSlicesToArrays(t *testing.T) {
	env := search.Envelope{
		Version: "1.0.0",
		Query: search.Query{
			Text:  "AI 搜索",
			Mode:  search.QueryModeSearch,
			Limit: 10,
		},
		Status: search.StatusOK,
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, env); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	for _, key := range []string{"results", "source_status", "errors"} {
		array, ok := decoded[key].([]any)
		if !ok {
			t.Fatalf("%s must be an array, got %T in %s", key, decoded[key], buf.String())
		}
		if len(array) != 0 {
			t.Fatalf("%s = %v, want empty array", key, array)
		}
	}
}
