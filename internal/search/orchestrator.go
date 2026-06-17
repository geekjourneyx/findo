package search

import "github.com/geekjourneyx/findo/internal/findoerr"

func Decide(statuses []SourceStatus) (Status, string, int) {
	hasOK := false
	hasNonOK := false

	for _, st := range statuses {
		if st.Status == SourceStatusOK {
			hasOK = true
			continue
		}
		hasNonOK = true
	}

	if hasOK && !hasNonOK {
		return StatusOK, "", 0
	}

	code := firstFailureCode(statuses)
	if hasOK {
		return StatusPartial, code, 1
	}

	return StatusError, code, findoerr.ExitCodeForCode(code)
}

func firstFailureCode(statuses []SourceStatus) string {
	for _, st := range statuses {
		if st.Status == SourceStatusOK {
			continue
		}
		if st.Error != nil && st.Error.Code != "" {
			return st.Error.Code
		}
		return findoerr.NoResults
	}
	return findoerr.NoResults
}
