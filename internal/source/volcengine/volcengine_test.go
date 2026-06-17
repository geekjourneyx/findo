package volcengine

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/geekjourneyx/findo/internal/findoerr"
	"github.com/geekjourneyx/findo/internal/search"
)

func TestClientAnswerSendsWebSearchRequestAndMapsCitations(t *testing.T) {
	var gotAuth string
	var gotContentType string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output": [
				{"type": "web_search_call"},
				{
					"type": "message",
					"content": [
						{
							"type": "output_text",
							"text": "first paragraph",
							"annotations": [
								{"type": "url_citation", "title": "One", "url": "https://example.com/one"},
								{"type": "other", "title": "Ignored", "url": "https://example.com/ignored"}
							]
						},
						{
							"type": "output_text",
							"text": "second paragraph",
							"annotations": [
								{"type": "url_citation", "title": "Two", "url": "https://example.com/two"}
							]
						}
					]
				}
			]
		}`))
	}))
	defer server.Close()

	client := Client{
		Endpoint: server.URL,
		APIKey:   "test-key",
		Model:    "client-model",
	}

	results, err := client.Answer(context.Background(), search.AnswerQuery{
		Text:  "hello",
		Limit: 7,
		Model: "query-model",
	})
	if err != nil {
		t.Fatalf("Answer returned error: %v", err)
	}

	if gotAuth != "Bearer test-key" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q", gotContentType)
	}
	if gotBody["model"] != "query-model" {
		t.Fatalf("model = %v", gotBody["model"])
	}
	tools := gotBody["tools"].([]any)
	tool := tools[0].(map[string]any)
	if tool["type"] != "web_search" || tool["limit"] != float64(7) {
		t.Fatalf("tools = %#v", gotBody["tools"])
	}
	input := gotBody["input"].([]any)
	message := input[0].(map[string]any)
	content := message["content"].([]any)
	item := content[0].(map[string]any)
	if message["role"] != "user" || item["type"] != "input_text" || item["text"] != "hello" {
		t.Fatalf("input = %#v", gotBody["input"])
	}

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	result := results[0]
	if result.Source != search.SourceVolcengineAnswer {
		t.Fatalf("Source = %q", result.Source)
	}
	if result.Type != search.ResultAnswer {
		t.Fatalf("Type = %q", result.Type)
	}
	if result.Title != "Volcengine Answer" {
		t.Fatalf("Title = %q", result.Title)
	}
	if result.Content != "first paragraph\nsecond paragraph" {
		t.Fatalf("Content = %q", result.Content)
	}
	wantCitations := []search.Citation{
		{Title: "One", URL: "https://example.com/one"},
		{Title: "Two", URL: "https://example.com/two"},
	}
	if len(result.Citations) != len(wantCitations) {
		t.Fatalf("citations = %#v", result.Citations)
	}
	for i := range wantCitations {
		if result.Citations[i] != wantCitations[i] {
			t.Fatalf("citation %d = %#v, want %#v", i, result.Citations[i], wantCitations[i])
		}
	}
}

func TestClientAnswerFallsBackToClientModel(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_, _ = w.Write([]byte(`{"output":[{"type":"web_search_call"},{"content":[{"type":"output_text","text":"answer"}]}]}`))
	}))
	defer server.Close()

	client := Client{Endpoint: server.URL, APIKey: "test-key", Model: "client-model"}
	_, err := client.Answer(context.Background(), search.AnswerQuery{Text: "hello", Limit: 3})
	if err != nil {
		t.Fatalf("Answer returned error: %v", err)
	}
	if gotBody["model"] != "client-model" {
		t.Fatalf("model = %v", gotBody["model"])
	}
}

func TestClientAnswerMissingAPIKey(t *testing.T) {
	client := Client{Endpoint: "https://example.invalid", Model: "model"}

	_, err := client.Answer(context.Background(), search.AnswerQuery{Text: "hello", Limit: 1})
	assertFindoError(t, err, findoerr.CredentialMissing, 0, false)
}

func TestClientAnswerRequiresWebSearchCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"output":[{"content":[{"type":"output_text","text":"ungrounded"}]}]}`))
	}))
	defer server.Close()

	client := Client{Endpoint: server.URL, APIKey: "test-key", Model: "model"}
	_, err := client.Answer(context.Background(), search.AnswerQuery{Text: "hello", Limit: 1})
	assertFindoError(t, err, findoerr.NoRetrievalTriggered, 0, false)
}

func TestClientAnswerMapsProviderErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantCode   string
		retryable  bool
	}{
		{name: "401", statusCode: http.StatusUnauthorized, wantCode: findoerr.SourceUnauthorized},
		{name: "429", statusCode: http.StatusTooManyRequests, wantCode: findoerr.SourceRateLimited, retryable: true},
		{name: "5xx", statusCode: http.StatusBadGateway, wantCode: findoerr.SourceUnavailable, retryable: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "provider error", tt.statusCode)
			}))
			defer server.Close()

			client := Client{Endpoint: server.URL, APIKey: "test-key", Model: "model"}
			_, err := client.Answer(context.Background(), search.AnswerQuery{Text: "hello", Limit: 1})
			assertFindoError(t, err, tt.wantCode, tt.statusCode, tt.retryable)
		})
	}
}

func TestClientAnswerInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := Client{Endpoint: server.URL, APIKey: "test-key", Model: "model"}
	_, err := client.Answer(context.Background(), search.AnswerQuery{Text: "hello", Limit: 1})
	assertFindoError(t, err, findoerr.SourceBadResponse, 0, false)
}

func TestClientAnswerContextDeadlineMapsToTimeout(t *testing.T) {
	client := Client{
		Endpoint: "http://10.255.255.1",
		APIKey:   "test-key",
		Model:    "model",
		HTTPClient: doFunc(func(*http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		}),
	}

	_, err := client.Answer(context.Background(), search.AnswerQuery{Text: "hello", Limit: 1})
	assertFindoError(t, err, findoerr.SourceTimeout, 0, true)
}

func TestClientImplementsInterfaces(t *testing.T) {
	var _ search.Source = Client{}
	var _ search.Answerer = Client{}
}

func assertFindoError(t *testing.T, err error, wantCode string, wantProviderStatus int, wantRetryable bool) {
	t.Helper()
	var ferr findoerr.Error
	if !errors.As(err, &ferr) {
		t.Fatalf("error = %T %v, want findoerr.Error", err, err)
	}
	if ferr.Code != wantCode {
		t.Fatalf("code = %q, want %q", ferr.Code, wantCode)
	}
	if ferr.Source != string(search.SourceVolcengineAnswer) {
		t.Fatalf("source = %q", ferr.Source)
	}
	if ferr.ProviderStatus != wantProviderStatus {
		t.Fatalf("provider status = %d, want %d", ferr.ProviderStatus, wantProviderStatus)
	}
	if ferr.Retryable != wantRetryable {
		t.Fatalf("retryable = %v, want %v", ferr.Retryable, wantRetryable)
	}
}

type doFunc func(*http.Request) (*http.Response, error)

func (fn doFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestClientSourceMetadata(t *testing.T) {
	client := Client{}
	if client.ID() != search.SourceVolcengineAnswer {
		t.Fatalf("ID = %q", client.ID())
	}
	if client.ProviderFamily() != "volcengine" {
		t.Fatalf("ProviderFamily = %q", client.ProviderFamily())
	}
	if strings.Join(capabilitiesToStrings(client.Capabilities()), ",") != "answer" {
		t.Fatalf("Capabilities = %#v", client.Capabilities())
	}
}

func capabilitiesToStrings(capabilities []search.Capability) []string {
	values := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		values = append(values, string(capability))
	}
	return values
}
