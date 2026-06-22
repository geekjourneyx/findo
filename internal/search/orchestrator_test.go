package search

import (
	"testing"

	"github.com/geekjourneyx/tanso/internal/tansoerr"
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

func TestDecideAllOKButNoResults(t *testing.T) {
	status, code, exit := Decide([]SourceStatus{
		{Status: SourceStatusOK, Results: 0},
		{Status: SourceStatusOK, Results: 0},
	})

	if status != StatusError || code != tansoerr.NoResults || exit != tansoerr.ExitCodeForCode(tansoerr.NoResults) {
		t.Fatalf("got %s %q %d", status, code, exit)
	}
}

func TestDecidePartialUsesFirstFailureCode(t *testing.T) {
	timeoutErr := tansoerr.Error{Code: tansoerr.SourceTimeout, Message: "timeout", Retryable: true}
	rateLimitErr := tansoerr.Error{Code: tansoerr.SourceRateLimited, Message: "rate limited", Retryable: true}

	status, code, exit := Decide([]SourceStatus{
		{Status: SourceStatusOK, Results: 1},
		{Status: SourceStatusTimeout, Error: &timeoutErr},
		{Status: SourceStatusRateLimited, Error: &rateLimitErr},
	})

	if status != StatusPartial || code != tansoerr.SourceTimeout || exit != 1 {
		t.Fatalf("got %s %q %d", status, code, exit)
	}
}

func TestDecideAllTimeoutOrErrorUsesFirstFailureExitCode(t *testing.T) {
	timeoutErr := tansoerr.Error{Code: tansoerr.SourceTimeout, Message: "timeout", Retryable: true}
	badResponseErr := tansoerr.Error{Code: tansoerr.SourceBadResponse, Message: "bad response", Retryable: true}

	status, code, exit := Decide([]SourceStatus{
		{Status: SourceStatusTimeout, Error: &timeoutErr},
		{Status: SourceStatusError, Error: &badResponseErr},
	})

	if status != StatusError || code != tansoerr.SourceTimeout || exit != tansoerr.ExitCodeForCode(tansoerr.SourceTimeout) {
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
			if status != StatusError || code != tansoerr.NoResults || exit != tansoerr.ExitCodeForCode(tansoerr.NoResults) {
				t.Fatalf("got %s %q %d", status, code, exit)
			}
		})
	}
}
