package zhihu

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/geekjourneyx/findo/internal/findoerr"
	"github.com/geekjourneyx/findo/internal/search"
)

var (
	_ search.Source    = Client{}
	_ search.Searcher  = Client{}
	_ search.Hotlister = Client{}
)

func TestSearchAddsHeadersClampsCountAndMapsItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/zhihu_search" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content-type = %q", r.Header.Get("Content-Type"))
		}
		ts, err := strconv.ParseInt(r.Header.Get("X-Request-Timestamp"), 10, 64)
		if err != nil {
			t.Fatalf("timestamp = %q", r.Header.Get("X-Request-Timestamp"))
		}
		if diff := time.Now().Unix() - ts; diff < -1 || diff > 1 {
			t.Fatalf("timestamp diff = %d", diff)
		}
		if r.URL.Query().Get("Query") != "RAG 评测" {
			t.Fatalf("query = %q", r.URL.Query().Get("Query"))
		}
		if r.URL.Query().Get("Count") != "10" {
			t.Fatalf("count = %q", r.URL.Query().Get("Count"))
		}
		_, _ = w.Write([]byte(`{"Code":0,"Message":"success","Data":{"Items":[{"Title":"T","Url":"https://example.com","ContentText":"S","EditTime":1710000000}]}}`))
	}))
	defer server.Close()

	client := Client{EndpointBase: server.URL, AccessSecret: "secret", HTTPClient: server.Client()}
	results, err := client.Search(context.Background(), search.SearchQuery{Text: "RAG 评测", Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("len = %d", len(results))
	}
	got := results[0]
	if got.Source != search.SourceZhihuSearch || got.Type != search.ResultWeb || got.Title != "T" || got.URL != "https://example.com" || got.Snippet != "S" {
		t.Fatalf("result = %#v", got)
	}
	if got.PublishedAt != "2024-03-09T16:00:00Z" {
		t.Fatalf("published_at = %q", got.PublishedAt)
	}
}

func TestGlobalSearchEncodesFilterSearchDBAndClampsCount(t *testing.T) {
	rawFilter := `host=="example.com" AND publish_time>=1778494631`
	var rawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/global_search" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		rawQuery = r.URL.RawQuery
		values, err := url.ParseQuery(rawQuery)
		if err != nil {
			t.Fatal(err)
		}
		if values.Get("Filter") != rawFilter {
			t.Fatalf("filter = %q", values.Get("Filter"))
		}
		if values.Get("SearchDB") != "realtime" {
			t.Fatalf("searchdb = %q", values.Get("SearchDB"))
		}
		if values.Get("Count") != "20" {
			t.Fatalf("count = %q", values.Get("Count"))
		}
		_, _ = w.Write([]byte(`{"Code":0,"Data":{"Items":[{"Title":"G","Url":"https://g.example.com","ContentText":"GS","EditTime":1748355858}]}}`))
	}))
	defer server.Close()

	client := Client{EndpointBase: server.URL, AccessSecret: "secret", HTTPClient: server.Client()}
	results, err := client.GlobalSearch(context.Background(), search.SearchQuery{
		Text:     "ChatGPT 桌面版",
		Limit:    99,
		Filter:   rawFilter,
		SearchDB: "realtime",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Source != search.SourceZhihuWeb || results[0].PublishedAt != "2025-05-27T14:24:18Z" {
		t.Fatalf("results = %#v", results)
	}
	if rawQuery == "" || rawQuery == `Filter=`+rawFilter {
		t.Fatalf("raw query was not encoded: %q", rawQuery)
	}
}

func TestHotlistClampsLimitAndMapsThumbnail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hot_list" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("Limit") != "30" {
			t.Fatalf("limit = %q", r.URL.Query().Get("Limit"))
		}
		_, _ = w.Write([]byte(`{"Code":0,"Message":"success","Data":{"Total":1,"Items":[{"Title":"Hot","Url":"https://example.com/hot","ThumbnailUrl":"https://img.example.com/a.jpg","Summary":"Summary"}]}}`))
	}))
	defer server.Close()

	client := Client{EndpointBase: server.URL, AccessSecret: "secret", HTTPClient: server.Client()}
	results, err := client.Hotlist(context.Background(), search.HotlistQuery{Limit: 99})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("len = %d", len(results))
	}
	got := results[0]
	if got.Source != search.SourceZhihuHot || got.Type != search.ResultHot || got.Title != "Hot" || got.URL != "https://example.com/hot" || got.Snippet != "Summary" {
		t.Fatalf("result = %#v", got)
	}
	if len(got.Images) != 1 || got.Images[0].URL != "https://img.example.com/a.jpg" {
		t.Fatalf("images = %#v", got.Images)
	}
}

func TestInvalidSearchDBReturnsInvalidArgumentWithoutRequest(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
	}))
	defer server.Close()

	client := Client{EndpointBase: server.URL, AccessSecret: "secret", HTTPClient: server.Client()}
	_, err := client.GlobalSearch(context.Background(), search.SearchQuery{Text: "query", SearchDB: "archive"})
	assertFindoError(t, err, findoerr.InvalidArgument, search.SourceZhihuWeb, false, 0, "")
	if requests != 0 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestEmptyQueryReturnsInvalidArgumentWithoutRequest(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
	}))
	defer server.Close()

	client := Client{EndpointBase: server.URL, AccessSecret: "secret", HTTPClient: server.Client()}
	_, err := client.Search(context.Background(), search.SearchQuery{Text: "   ", Limit: 5})
	assertFindoError(t, err, findoerr.InvalidArgument, search.SourceZhihuSearch, false, 0, "")
	if requests != 0 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestMissingAccessSecretReturnsCredentialMissingWithSource(t *testing.T) {
	client := Client{EndpointBase: "https://example.invalid"}
	_, err := client.Hotlist(context.Background(), search.HotlistQuery{Limit: 5})
	assertFindoError(t, err, findoerr.CredentialMissing, search.SourceZhihuHot, false, 0, "")
}

func TestProviderAuthAndRateLimitMapping(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantCode   string
		wantSource search.SourceID
		retryable  bool
		provider   string
	}{
		{
			name:       "auth",
			body:       `{"Code":20001,"Message":"auth failed"}`,
			wantCode:   findoerr.SourceUnauthorized,
			wantSource: search.SourceZhihuSearch,
			retryable:  false,
			provider:   "20001",
		},
		{
			name:       "rate limit",
			body:       `{"Code":30001,"Message":"rate limited"}`,
			wantCode:   findoerr.SourceRateLimited,
			wantSource: search.SourceZhihuWeb,
			retryable:  true,
			provider:   "30001",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := Client{EndpointBase: server.URL, AccessSecret: "secret", HTTPClient: server.Client()}
			var err error
			if tt.wantSource == search.SourceZhihuSearch {
				_, err = client.Search(context.Background(), search.SearchQuery{Text: "query", Limit: 5})
			} else {
				_, err = client.GlobalSearch(context.Background(), search.SearchQuery{Text: "query", Limit: 5})
			}
			assertFindoError(t, err, tt.wantCode, tt.wantSource, tt.retryable, http.StatusOK, tt.provider)
		})
	}
}

func TestHTTPAndDecodeErrorMapping(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantCode  string
		retryable bool
	}{
		{name: "bad request", status: http.StatusBadRequest, wantCode: findoerr.InvalidArgument, retryable: false},
		{name: "unauthorized", status: http.StatusUnauthorized, wantCode: findoerr.SourceUnauthorized, retryable: false},
		{name: "forbidden", status: http.StatusForbidden, wantCode: findoerr.SourceUnauthorized, retryable: false},
		{name: "rate limited", status: http.StatusTooManyRequests, wantCode: findoerr.SourceRateLimited, retryable: true},
		{name: "server error", status: http.StatusInternalServerError, wantCode: findoerr.SourceUnavailable, retryable: true},
		{name: "invalid json", status: http.StatusOK, body: `{`, wantCode: findoerr.SourceBadResponse, retryable: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := Client{
				EndpointBase: "https://developer.zhihu.test/api/v1/content",
				AccessSecret: "secret",
				HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.status,
						Body:       io.NopCloser(strings.NewReader(tt.body)),
						Header:     make(http.Header),
						Request:    req,
					}, nil
				})},
			}
			_, err := client.Search(context.Background(), search.SearchQuery{Text: "query", Limit: 5})
			assertFindoError(t, err, tt.wantCode, search.SourceZhihuSearch, tt.retryable, tt.status, "")
		})
	}
}

func TestTimeoutErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "context deadline", err: context.DeadlineExceeded},
		{name: "net timeout", err: timeoutError{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := Client{
				EndpointBase: "https://developer.zhihu.test/api/v1/content",
				AccessSecret: "secret",
				HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					return nil, tt.err
				})},
			}
			_, err := client.GlobalSearch(context.Background(), search.SearchQuery{Text: "query", Limit: 5})
			assertFindoError(t, err, findoerr.SourceTimeout, search.SourceZhihuWeb, true, 0, "")
		})
	}
}

func assertFindoError(t *testing.T, err error, code string, source search.SourceID, retryable bool, providerStatus int, providerCode string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	got, ok := err.(findoerr.Error)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if got.Code != code || got.Source != string(source) || got.Retryable != retryable || got.ProviderStatus != providerStatus || got.ProviderCode != providerCode {
		t.Fatalf("error = %#v", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }
