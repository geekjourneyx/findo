package search

import "github.com/geekjourneyx/tanso/internal/tansoerr"

func Decide(statuses []SourceStatus) (Status, string, int) {
	hasOK := false
	hasNonOK := false
	resultCount := 0

	for _, st := range statuses {
		if st.Status == SourceStatusOK {
			hasOK = true
			resultCount += st.Results
			continue
		}
		hasNonOK = true
	}

	if hasOK && !hasNonOK {
		if resultCount == 0 {
			return StatusError, tansoerr.NoResults, tansoerr.ExitCodeForCode(tansoerr.NoResults)
		}
		return StatusOK, "", 0
	}

	code := firstFailureCode(statuses)
	if hasOK {
		return StatusPartial, code, 1
	}

	return StatusError, code, tansoerr.ExitCodeForCode(code)
}

func firstFailureCode(statuses []SourceStatus) string {
	for _, st := range statuses {
		if st.Status == SourceStatusOK {
			continue
		}
		if st.Error != nil && st.Error.Code != "" {
			return st.Error.Code
		}
		return tansoerr.NoResults
	}
	return tansoerr.NoResults
}
