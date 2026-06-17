package volcengine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/geekjourneyx/findo/internal/findoerr"
	"github.com/geekjourneyx/findo/internal/search"
)

const (
	DefaultEndpoint = "https://ark.cn-beijing.volces.com/api/v3/responses"
	sourceName      = string(search.SourceVolcengineAnswer)
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

var (
	_ search.Source   = Client{}
	_ search.Answerer = Client{}
)

type Client struct {
	Endpoint   string
	APIKey     string
	Model      string
	HTTPClient httpDoer
}

func (c Client) ID() search.SourceID {
	return search.SourceVolcengineAnswer
}

func (c Client) ProviderFamily() string {
	return "volcengine"
}

func (c Client) Capabilities() []search.Capability {
	return []search.Capability{search.CapabilityAnswer}
}

func (c Client) Answer(ctx context.Context, query search.AnswerQuery) ([]search.Result, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, findoerr.Error{
			Code:    findoerr.CredentialMissing,
			Message: "volcengine API key is required",
			Source:  sourceName,
		}
	}

	body, err := json.Marshal(map[string]any{
		"model": firstNonEmpty(query.Model, c.Model),
		"tools": []map[string]any{{
			"type":  "web_search",
			"limit": query.Limit,
		}},
		"input": []map[string]any{{
			"role": "user",
			"content": []map[string]string{{
				"type": "input_text",
				"text": query.Text,
			}},
		}},
	})
	if err != nil {
		return nil, findoerr.Error{Code: findoerr.InternalError, Message: "failed to encode volcengine request", Source: sourceName}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(body))
	if err != nil {
		return nil, findoerr.Error{Code: findoerr.InvalidArgument, Message: "invalid volcengine endpoint", Source: sourceName}
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
			Code:    findoerr.SourceBadResponse,
			Message: "volcengine returned invalid JSON",
			Source:  sourceName,
		}
	}
	if !decoded.hasWebSearchCall() {
		return nil, findoerr.Error{
			Code:    findoerr.NoRetrievalTriggered,
			Message: "volcengine response did not include web_search_call",
			Source:  sourceName,
		}
	}

	answer, citations := decoded.answerAndCitations()
	return []search.Result{{
		Source:    search.SourceVolcengineAnswer,
		Type:      search.ResultAnswer,
		Title:     "Volcengine Answer",
		Content:   answer,
		Citations: citations,
	}}, nil
}

func (c Client) endpoint() string {
	if strings.TrimSpace(c.Endpoint) != "" {
		return c.Endpoint
	}
	return DefaultEndpoint
}

func (c Client) httpClient() httpDoer {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

type apiResponse struct {
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			Annotations []struct {
				Type  string `json:"type"`
				Title string `json:"title"`
				URL   string `json:"url"`
			} `json:"annotations"`
		} `json:"content"`
	} `json:"output"`
}

func (r apiResponse) hasWebSearchCall() bool {
	for _, item := range r.Output {
		if item.Type == "web_search_call" {
			return true
		}
	}
	return false
}

func (r apiResponse) answerAndCitations() (string, []search.Citation) {
	var parts []string
	var citations []search.Citation
	for _, item := range r.Output {
		for _, content := range item.Content {
			if content.Type == "output_text" && content.Text != "" {
				parts = append(parts, content.Text)
			}
			for _, annotation := range content.Annotations {
				if annotation.Type == "url_citation" && annotation.URL != "" {
					citations = append(citations, search.Citation{Title: annotation.Title, URL: annotation.URL})
				}
			}
		}
	}
	return strings.Join(parts, "\n"), citations
}

func requestError(ctx context.Context, err error) error {
	if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return findoerr.Error{
			Code:      findoerr.SourceTimeout,
			Message:   "volcengine request timed out",
			Source:    sourceName,
			Retryable: true,
		}
	}
	return findoerr.Error{
		Code:      findoerr.SourceUnavailable,
		Message:   "volcengine request failed",
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
		message = http.StatusText(resp.StatusCode)
	}

	code, retryable := mapHTTPStatus(resp.StatusCode)
	return findoerr.Error{
		Code:           code,
		Message:        message,
		Source:         sourceName,
		ProviderStatus: resp.StatusCode,
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
