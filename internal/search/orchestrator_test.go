package search

import (
	"testing"

	"github.com/geekjourneyx/findo/internal/findoerr"
)

func TestDecideAllOK(t *testing.T) {
	status, code, exit := Decide([]SourceStatus{
		{Status: SourceStatusOK, Results: 0},
		{Status: SourceStatusOK, Results: 3},
	})

	if status != StatusOK || code != "" || exit != 0 {
		t.Fatalf("got %s %q %d", status, code, exit)
	}
}

func TestDecidePartialUsesFirstFailureCode(t *testing.T) {
	timeoutErr := findoerr.Error{Code: findoerr.SourceTimeout, Message: "timeout", Retryable: true}
	rateLimitErr := findoerr.Error{Code: findoerr.SourceRateLimited, Message: "rate limited", Retryable: true}

	status, code, exit := Decide([]SourceStatus{
		{Status: SourceStatusOK, Results: 1},
		{Status: SourceStatusTimeout, Error: &timeoutErr},
		{Status: SourceStatusRateLimited, Error: &rateLimitErr},
	})

	if status != StatusPartial || code != findoerr.SourceTimeout || exit != 1 {
		t.Fatalf("got %s %q %d", status, code, exit)
	}
}

func TestDecideAllTimeoutOrErrorUsesFirstFailureExitCode(t *testing.T) {
	timeoutErr := findoerr.Error{Code: findoerr.SourceTimeout, Message: "timeout", Retryable: true}
	badResponseErr := findoerr.Error{Code: findoerr.SourceBadResponse, Message: "bad response", Retryable: true}

	status, code, exit := Decide([]SourceStatus{
		{Status: SourceStatusTimeout, Error: &timeoutErr},
		{Status: SourceStatusError, Error: &badResponseErr},
	})

	if status != StatusError || code != findoerr.SourceTimeout || exit != findoerr.ExitCodeForCode(findoerr.SourceTimeout) {
		t.Fatalf("got %s %q %d", status, code, exit)
	}
}

func TestDecideEmptyOrNoErrorFallback(t *testing.T) {
	tests := []struct {
		name     string
		statuses []SourceStatus
	}{
		{name: "empty"},
		{name: "non ok without error", statuses: []SourceStatus{{Status: SourceStatusSkipped}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, code, exit := Decide(tt.statuses)
			if status != StatusError || code != findoerr.NoResults || exit != findoerr.ExitCodeForCode(findoerr.NoResults) {
				t.Fatalf("got %s %q %d", status, code, exit)
			}
		})
	}
}
