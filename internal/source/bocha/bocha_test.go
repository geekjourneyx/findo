package bocha

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geekjourneyx/findo/internal/findoerr"
	"github.com/geekjourneyx/findo/internal/search"
)

func TestClientImplementsInterfaces(t *testing.T) {
	var _ search.Source = (*Client)(nil)
	var _ search.Searcher = (*Client)(nil)
}

func TestSearchSuccessMappingAndRequest(t *testing.T) {
	var gotRequest struct {
		Method        string
		Authorization string
		ContentType   string
		Body          map[string]any
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequest.Method = r.Method
		gotRequest.Authorization = r.Header.Get("Authorization")
		gotRequest.ContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotRequest.Body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 200,
			"data": {
				"webPages": {
					"value": [
						{"name":"first","url":"https://example.com/1","summary":"summary text","snippet":"snippet text","datePublished":"2026-06-01"},
						{"name":"second","url":"https://example.com/2","snippet":"fallback snippet","datePublished":"2026-06-02"}
					]
				}
			}
		}`))
	}))
	defer server.Close()

	client := New("secret", server.URL)
	results, err := client.Search(context.Background(), search.SearchQuery{Text: "findo"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if gotRequest.Method != http.MethodPost {
		t.Fatalf("method = %q, want %q", gotRequest.Method, http.MethodPost)
	}
	if gotRequest.Authorization != "Bearer secret" {
		t.Fatalf("authorization = %q", gotRequest.Authorization)
	}
	if gotRequest.ContentType != "application/json" {
		t.Fatalf("content-type = %q", gotRequest.ContentType)
	}
	if gotRequest.Body["query"] != "findo" {
		t.Fatalf("query body = %#v", gotRequest.Body["query"])
	}
	if gotRequest.Body["summary"] != true {
		t.Fatalf("summary body = %#v", gotRequest.Body["summary"])
	}

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	wantFirst := search.Result{
		Source:      search.SourceBochaWeb,
		Type:        search.ResultWeb,
		Title:       "first",
		URL:         "https://example.com/1",
		Snippet:     "summary text",
		PublishedAt: "2026-06-01",
	}
	if results[0].Source != wantFirst.Source ||
		results[0].Type != wantFirst.Type ||
		results[0].Title != wantFirst.Title ||
		results[0].URL != wantFirst.URL ||
		results[0].Snippet != wantFirst.Snippet ||
		results[0].PublishedAt != wantFirst.PublishedAt {
		t.Fatalf("first result = %#v, want %#v", results[0], wantFirst)
	}
	if results[1].Snippet != "fallback snippet" {
		t.Fatalf("second snippet = %q", results[1].Snippet)
	}
}

func TestSearchMissingAPIKey(t *testing.T) {
	client := New("", "http://example.invalid")
	_, err := client.Search(context.Background(), search.SearchQuery{Text: "findo"})
	assertFindoError(t, err, findoerr.CredentialMissing)
}

func TestSearchLimitTruncatesLocally(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"code": 200,
			"data": {
				"webPages": {
					"value": [
						{"name":"first","url":"https://example.com/1"},
						{"name":"second","url":"https://example.com/2"},
						{"name":"third","url":"https://example.com/3"}
					]
				}
			}
		}`))
	}))
	defer server.Close()

	client := New("secret", server.URL)
	results, err := client.Search(context.Background(), search.SearchQuery{Text: "findo", Limit: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[1].Title != "second" {
		t.Fatalf("last result = %q, want second", results[1].Title)
	}
}

func TestSearchHTTPStatusErrors(t *testing.T) {
	tests := []struct {
		name string
		code int
		want string
	}{
		{name: "bad request", code: http.StatusBadRequest, want: findoerr.InvalidArgument},
		{name: "unauthorized", code: http.StatusUnauthorized, want: findoerr.SourceUnauthorized},
		{name: "rate limited", code: http.StatusTooManyRequests, want: findoerr.SourceRateLimited},
		{name: "server error", code: http.StatusInternalServerError, want: findoerr.SourceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "provider failed", tt.code)
			}))
			defer server.Close()

			client := New("secret", server.URL)
			_, err := client.Search(context.Background(), search.SearchQuery{Text: "findo"})
			assertFindoError(t, err, tt.want)
		})
	}
}

func TestSearchInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	client := New("secret", server.URL)
	_, err := client.Search(context.Background(), search.SearchQuery{Text: "findo"})
	assertFindoError(t, err, findoerr.SourceBadResponse)
}

func TestSearchProviderNonSuccessCode(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "provider bad request", body: `{"code":400,"msg":"bad query"}`, want: findoerr.InvalidArgument},
		{name: "provider token", body: `{"code":401,"msg":"invalid token"}`, want: findoerr.CredentialMissing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := New("secret", server.URL)
			_, err := client.Search(context.Background(), search.SearchQuery{Text: "findo"})
			assertFindoError(t, err, tt.want)
		})
	}
}

func assertFindoError(t *testing.T, err error, wantCode string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error")
	}
	var ferr findoerr.Error
	if !errors.As(err, &ferr) {
		t.Fatalf("error = %T %[1]v, want findoerr.Error", err)
	}
	if ferr.Code != wantCode {
		t.Fatalf("code = %q, want %q; error = %#v", ferr.Code, wantCode, ferr)
	}
	if ferr.Source != string(search.SourceBochaWeb) {
		t.Fatalf("source = %q, want %q", ferr.Source, search.SourceBochaWeb)
	}
}
