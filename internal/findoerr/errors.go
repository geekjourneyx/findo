package findoerr

const (
	InvalidArgument      = "INVALID_ARGUMENT"
	ConfigNotFound       = "CONFIG_NOT_FOUND"
	ConfigInvalid        = "CONFIG_INVALID"
	CredentialMissing    = "CREDENTIAL_MISSING"
	SourceUnavailable    = "SOURCE_UNAVAILABLE"
	SourceUnauthorized   = "SOURCE_UNAUTHORIZED"
	SourceRateLimited    = "SOURCE_RATE_LIMITED"
	SourceTimeout        = "SOURCE_TIMEOUT"
	SourceBadResponse    = "SOURCE_BAD_RESPONSE"
	NoResults            = "NO_RESULTS"
	NoRetrievalTriggered = "NO_RETRIEVAL_TRIGGERED"
	InternalError        = "INTERNAL_ERROR"
)

type Error struct {
	Code           string            `json:"code"`
	Message        string            `json:"message"`
	Source         string            `json:"source,omitempty"`
	ProviderStatus int               `json:"provider_status,omitempty"`
	ProviderCode   string            `json:"provider_code,omitempty"`
	Retryable      bool              `json:"retryable"`
	Details        map[string]string `json:"details,omitempty"`
}

func (e Error) Error() string {
	return e.Message
}

func ExitCodeForCode(code string) int {
	switch code {
	case InvalidArgument:
		return 2
	case ConfigNotFound, ConfigInvalid:
		return 3
	case CredentialMissing:
		return 4
	case SourceTimeout:
		return 6
	case NoResults:
		return 7
	case InternalError:
		return 9
	default:
		return 5
	}
}
