package zhihu

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/geekjourneyx/findo/internal/findoerr"
	"github.com/geekjourneyx/findo/internal/search"
)

const DefaultEndpointBase = "https://developer.zhihu.com/api/v1/content"

var (
	_ search.Source    = Client{}
	_ search.Searcher  = Client{}
	_ search.Hotlister = Client{}
)

type Client struct {
	EndpointBase string
	AccessSecret string
	HTTPClient   *http.Client
}

func (c Client) ID() search.SourceID {
	return search.SourceZhihuSearch
}

func (c Client) ProviderFamily() string {
	return "zhihu"
}

func (c Client) Capabilities() []search.Capability {
	return []search.Capability{search.CapabilityWebSearch, search.CapabilityHotlist}
}

func (c Client) Search(ctx context.Context, query search.SearchQuery) ([]search.Result, error) {
	return c.search(ctx, "/zhihu_search", search.SourceZhihuSearch, clampDefault(query.Limit, 10, 1, 10), query)
}

func (c Client) GlobalSearch(ctx context.Context, query search.SearchQuery) ([]search.Result, error) {
	if err := validateSearchDB(query.SearchDB); err != nil {
		return nil, err
	}
	return c.search(ctx, "/global_search", search.SourceZhihuWeb, clampDefault(query.Limit, 10, 1, 20), query)
}

func (c Client) Hotlist(ctx context.Context, query search.HotlistQuery) ([]search.Result, error) {
	source := search.SourceZhihuHot
	if err := c.validateSecret(source); err != nil {
		return nil, err
	}

	u, err := c.url("/hot_list")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("Limit", strconv.Itoa(clampDefault(query.Limit, 30, 1, 30)))
	u.RawQuery = q.Encode()

	var decoded hotResponse
	if err := c.get(ctx, u.String(), source, &decoded); err != nil {
		return nil, err
	}
	if decoded.Code != 0 {
		return nil, providerCodeError(decoded.Code, decoded.Message, source, http.StatusOK)
	}

	results := make([]search.Result, 0, len(decoded.Data.Items))
	for _, item := range decoded.Data.Items {
		result := search.Result{
			Source:  source,
			Type:    search.ResultHot,
			Title:   item.Title,
			URL:     item.URL,
			Snippet: item.Summary,
		}
		if item.ThumbnailURL != "" {
			result.Images = []search.Image{{URL: item.ThumbnailURL}}
		}
		results = append(results, result)
	}
	return results, nil
}

func (c Client) search(ctx context.Context, path string, source search.SourceID, limit int, query search.SearchQuery) ([]search.Result, error) {
	if strings.TrimSpace(query.Text) == "" {
		return nil, findoerr.Error{Code: findoerr.InvalidArgument, Message: "zhihu query is required", Source: string(source)}
	}
	if err := c.validateSecret(source); err != nil {
		return nil, err
	}

	u, err := c.url(path)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("Query", query.Text)
	q.Set("Count", strconv.Itoa(limit))
	if source == search.SourceZhihuWeb {
		if query.Filter != "" {
			q.Set("Filter", query.Filter)
		}
		if query.SearchDB != "" {
			q.Set("SearchDB", query.SearchDB)
		}
	}
	u.RawQuery = q.Encode()

	var decoded searchResponse
	if err := c.get(ctx, u.String(), source, &decoded); err != nil {
		return nil, err
	}
	if decoded.Code != 0 {
		return nil, providerCodeError(decoded.Code, decoded.Message, source, http.StatusOK)
	}

	results := make([]search.Result, 0, len(decoded.Data.Items))
	for _, item := range decoded.Data.Items {
		results = append(results, search.Result{
			Source:      source,
			Type:        search.ResultWeb,
			Title:       item.Title,
			URL:         item.URL,
			Snippet:     item.ContentText,
			PublishedAt: unixToRFC3339(item.EditTime),
		})
	}
	return results, nil
}

func (c Client) validateSecret(source search.SourceID) error {
	if strings.TrimSpace(c.AccessSecret) != "" {
		return nil
	}
	return findoerr.Error{
		Code:    findoerr.CredentialMissing,
		Message: "zhihu access secret is required",
		Source:  string(source),
	}
}

func (c Client) get(ctx context.Context, endpoint string, source search.SourceID, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return findoerr.Error{Code: findoerr.InvalidArgument, Message: "invalid zhihu endpoint", Source: string(source)}
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessSecret)
	req.Header.Set("X-Request-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return requestError(ctx, err, source)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return httpStatusError(resp.StatusCode, source)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return findoerr.Error{
			Code:           findoerr.SourceBadResponse,
			Message:        "zhihu returned invalid JSON",
			Source:         string(source),
			ProviderStatus: resp.StatusCode,
		}
	}
	return nil
}

func (c Client) url(path string) (*url.URL, error) {
	base := strings.TrimRight(c.EndpointBase, "/")
	if base == "" {
		base = DefaultEndpointBase
	}
	return url.Parse(base + path)
}

func (c Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

type searchResponse struct {
	Code    int    `json:"Code"`
	Message string `json:"Message"`
	Data    struct {
		Items []struct {
			Title       string `json:"Title"`
			URL         string `json:"Url"`
			ContentText string `json:"ContentText"`
			EditTime    int64  `json:"EditTime"`
		} `json:"Items"`
	} `json:"Data"`
}

type hotResponse struct {
	Code    int    `json:"Code"`
	Message string `json:"Message"`
	Data    struct {
		Total int64 `json:"Total"`
		Items []struct {
			Title        string `json:"Title"`
			URL          string `json:"Url"`
			ThumbnailURL string `json:"ThumbnailUrl"`
			Summary      string `json:"Summary"`
		} `json:"Items"`
	} `json:"Data"`
}

func validateSearchDB(value string) error {
	switch value {
	case "", "all", "realtime", "static":
		return nil
	default:
		return findoerr.Error{Code: findoerr.InvalidArgument, Message: "invalid zhihu SearchDB", Source: string(search.SourceZhihuWeb)}
	}
}

func requestError(ctx context.Context, err error, source search.SourceID) error {
	if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) || isTimeout(err) {
		return findoerr.Error{
			Code:      findoerr.SourceTimeout,
			Message:   "zhihu request timed out",
			Source:    string(source),
			Retryable: true,
		}
	}
	return findoerr.Error{
		Code:      findoerr.SourceUnavailable,
		Message:   "zhihu request failed",
		Source:    string(source),
		Retryable: true,
		Details: map[string]string{
			"error": err.Error(),
		},
	}
}

func httpStatusError(status int, source search.SourceID) error {
	code, retryable := mapHTTPStatus(status)
	return findoerr.Error{
		Code:           code,
		Message:        http.StatusText(status),
		Source:         string(source),
		ProviderStatus: status,
		Retryable:      retryable,
	}
}

func providerCodeError(code int, message string, source search.SourceID, status int) error {
	findoCode, retryable := mapProviderCode(code)
	if message == "" {
		message = "zhihu provider returned an error"
	}
	return findoerr.Error{
		Code:           findoCode,
		Message:        message,
		Source:         string(source),
		ProviderStatus: status,
		ProviderCode:   strconv.Itoa(code),
		Retryable:      retryable,
	}
}

func mapHTTPStatus(status int) (string, bool) {
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
		return findoerr.SourceBadResponse, false
	}
}

func mapProviderCode(code int) (string, bool) {
	switch code {
	case 10001:
		return findoerr.InvalidArgument, false
	case 20001:
		return findoerr.SourceUnauthorized, false
	case 30001:
		return findoerr.SourceRateLimited, true
	case 90001:
		return findoerr.SourceUnavailable, true
	default:
		return findoerr.SourceBadResponse, false
	}
}

func clampDefault(value, fallback, min, max int) int {
	if value <= 0 {
		return fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func unixToRFC3339(ts int64) string {
	if ts == 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func isTimeout(err error) bool {
	type timeout interface {
		Timeout() bool
	}
	var timeoutErr timeout
	return errors.As(err, &timeoutErr) && timeoutErr.Timeout()
}
