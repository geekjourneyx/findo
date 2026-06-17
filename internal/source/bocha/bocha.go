package bocha

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/geekjourneyx/findo/internal/findoerr"
	"github.com/geekjourneyx/findo/internal/search"
)

const (
	DefaultEndpoint = "https://api.bocha.cn/v1/web-search"
	sourceName      = string(search.SourceBochaWeb)
)

var (
	_ search.Source   = (*Client)(nil)
	_ search.Searcher = (*Client)(nil)
)

type Client struct {
	APIKey     string
	Endpoint   string
	HTTPClient *http.Client
}

func New(apiKey, endpoint string) *Client {
	return &Client{
		APIKey:   apiKey,
		Endpoint: endpoint,
	}
}

func (c *Client) ID() search.SourceID {
	return search.SourceBochaWeb
}

func (c *Client) ProviderFamily() string {
	return "bocha"
}

func (c *Client) Capabilities() []search.Capability {
	return []search.Capability{search.CapabilityWebSearch}
}

func (c *Client) Search(ctx context.Context, query search.SearchQuery) ([]search.Result, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, findoerr.Error{
			Code:    findoerr.CredentialMissing,
			Message: "bocha API key is required",
			Source:  sourceName,
		}
	}

	body, err := json.Marshal(map[string]any{
		"query":   query.Text,
		"summary": true,
	})
	if err != nil {
		return nil, findoerr.Error{
			Code:    findoerr.InternalError,
			Message: "failed to encode bocha request",
			Source:  sourceName,
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(body))
	if err != nil {
		return nil, findoerr.Error{
			Code:    findoerr.InvalidArgument,
			Message: "invalid bocha endpoint",
			Source:  sourceName,
		}
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, requestError(ctx, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, httpStatusError(resp)
	}

	var decoded apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, findoerr.Error{
			Code:           findoerr.SourceBadResponse,
			Message:        "bocha returned invalid JSON",
			Source:         sourceName,
			ProviderStatus: resp.StatusCode,
			Retryable:      true,
		}
	}
	if !decoded.success() {
		return nil, providerError(decoded)
	}

	results := make([]search.Result, 0, len(decoded.Data.WebPages.Value))
	for _, item := range decoded.Data.WebPages.Value {
		results = append(results, search.Result{
			Source:      search.SourceBochaWeb,
			Type:        search.ResultWeb,
			Title:       item.Name,
			URL:         item.URL,
			Snippet:     firstNonEmpty(item.Summary, item.Snippet),
			PublishedAt: item.DatePublished,
		})
	}
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}
	return results, nil
}

func (c *Client) endpoint() string {
	if strings.TrimSpace(c.Endpoint) != "" {
		return c.Endpoint
	}
	return DefaultEndpoint
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

type apiResponse struct {
	Code    json.RawMessage `json:"code"`
	Message string          `json:"message"`
	Msg     string          `json:"msg"`
	Data    struct {
		WebPages struct {
			Value []webPage `json:"value"`
		} `json:"webPages"`
	} `json:"data"`
}

type webPage struct {
	Name          string `json:"name"`
	URL           string `json:"url"`
	Summary       string `json:"summary"`
	Snippet       string `json:"snippet"`
	DatePublished string `json:"datePublished"`
}

func (r apiResponse) success() bool {
	if len(r.Code) == 0 || string(r.Code) == "null" {
		return true
	}

	var number int
	if err := json.Unmarshal(r.Code, &number); err == nil {
		return number == 0 || number == http.StatusOK
	}

	var text string
	if err := json.Unmarshal(r.Code, &text); err == nil {
		switch strings.ToLower(strings.TrimSpace(text)) {
		case "", "0", "200", "ok", "success":
			return true
		default:
			return false
		}
	}
	return false
}

func (r apiResponse) providerCode() string {
	if len(r.Code) == 0 || string(r.Code) == "null" {
		return ""
	}
	var text string
	if err := json.Unmarshal(r.Code, &text); err == nil {
		return text
	}
	var number int
	if err := json.Unmarshal(r.Code, &number); err == nil {
		return strconv.Itoa(number)
	}
	return string(r.Code)
}

func (r apiResponse) providerMessage() string {
	return firstNonEmpty(r.Message, r.Msg, "bocha returned an error")
}

func requestError(ctx context.Context, err error) error {
	if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return findoerr.Error{
			Code:      findoerr.SourceTimeout,
			Message:   "bocha request timed out",
			Source:    sourceName,
			Retryable: true,
		}
	}
	return findoerr.Error{
		Code:      findoerr.SourceTimeout,
		Message:   "bocha request failed",
		Source:    sourceName,
		Retryable: true,
		Details: map[string]string{
			"error": err.Error(),
		},
	}
}

func httpStatusError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = fmt.Sprintf("bocha returned HTTP %d", resp.StatusCode)
	}

	providerCode, providerMessage := parseErrorBody(body)
	if providerMessage != "" {
		message = providerMessage
	}

	code, retryable := mapProviderFailure(resp.StatusCode, providerCode, message)
	return findoerr.Error{
		Code:           code,
		Message:        message,
		Source:         sourceName,
		ProviderStatus: resp.StatusCode,
		ProviderCode:   providerCode,
		Retryable:      retryable,
	}
}

func providerError(resp apiResponse) error {
	providerCode := resp.providerCode()
	message := resp.providerMessage()
	status := providerCodeAsStatus(providerCode)
	code, retryable := mapProviderFailure(status, providerCode, message)
	return findoerr.Error{
		Code:         code,
		Message:      message,
		Source:       sourceName,
		ProviderCode: providerCode,
		Retryable:    retryable,
	}
}

func parseErrorBody(body []byte) (string, string) {
	var decoded struct {
		Code    json.RawMessage `json:"code"`
		Message string          `json:"message"`
		Msg     string          `json:"msg"`
		Error   string          `json:"error"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return "", ""
	}
	providerCode := apiResponse{Code: decoded.Code}.providerCode()
	return providerCode, firstNonEmpty(decoded.Message, decoded.Msg, decoded.Error)
}

func providerCodeAsStatus(providerCode string) int {
	if n, err := strconv.Atoi(providerCode); err == nil {
		return n
	}
	return 0
}

func mapProviderFailure(status int, providerCode, message string) (string, bool) {
	lower := strings.ToLower(message + " " + providerCode)
	if strings.Contains(lower, "api key") || strings.Contains(lower, "token") || strings.Contains(lower, "credential") {
		return findoerr.CredentialMissing, false
	}
	switch {
	case status == http.StatusBadRequest:
		return findoerr.InvalidArgument, false
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return findoerr.SourceUnauthorized, false
	case status == http.StatusTooManyRequests:
		return findoerr.SourceRateLimited, true
	case status >= 500:
		return findoerr.SourceUnavailable, true
	default:
		return findoerr.SourceBadResponse, true
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
