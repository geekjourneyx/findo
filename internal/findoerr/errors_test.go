package findoerr

import "testing"

func TestExitCodeForCode(t *testing.T) {
	tests := []struct {
		code string
		want int
	}{
		{InvalidArgument, 2},
		{ConfigNotFound, 3},
		{ConfigInvalid, 3},
		{CredentialMissing, 4},
		{SourceUnavailable, 5},
		{SourceTimeout, 6},
		{NoResults, 7},
		{InternalError, 9},
	}
	for _, tt := range tests {
		if got := ExitCodeForCode(tt.code); got != tt.want {
			t.Fatalf("ExitCodeForCode(%q) = %d, want %d", tt.code, got, tt.want)
		}
	}
}

func TestErrorDetailsAreStrings(t *testing.T) {
	err := Error{
		Code:    SourceTimeout,
		Message: "bocha request timed out",
		Details: map[string]string{
			"timeout": "12s",
		},
	}
	if err.Details["timeout"] != "12s" {
		t.Fatalf("details not preserved")
	}
}
