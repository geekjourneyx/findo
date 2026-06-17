package search

import (
	"context"

	"github.com/geekjourneyx/findo/internal/findoerr"
)

type Capability string

const (
	CapabilityWebSearch Capability = "web_search"
	CapabilityAnswer    Capability = "answer"
	CapabilityHotlist   Capability = "hotlist"
)

type SourceID string

const (
	SourceBochaWeb         SourceID = "bocha_web"
	SourceVolcengineAnswer SourceID = "volcengine_answer"
	SourceZhihuSearch      SourceID = "zhihu_search"
	SourceZhihuWeb         SourceID = "zhihu_web"
	SourceZhihuHot         SourceID = "zhihu_hot"
)

type ResultType string

const (
	ResultWeb    ResultType = "web"
	ResultAnswer ResultType = "answer"
	ResultHot    ResultType = "hot"
)

type QueryMode string

const (
	QueryModeSearch  QueryMode = "search"
	QueryModeAnswer  QueryMode = "answer"
	QueryModeHotlist QueryMode = "hotlist"
	QueryModeMixed   QueryMode = "mixed"
)

type Status string

const (
	StatusOK      Status = "ok"
	StatusPartial Status = "partial"
	StatusError   Status = "error"
)

type Source interface {
	ID() SourceID
	ProviderFamily() string
	Capabilities() []Capability
}

type Searcher interface {
	Search(ctx context.Context, query SearchQuery) ([]Result, error)
}

type Answerer interface {
	Answer(ctx context.Context, query AnswerQuery) ([]Result, error)
}

type Hotlister interface {
	Hotlist(ctx context.Context, query HotlistQuery) ([]Result, error)
}

type SearchQuery struct {
	Text     string
	Limit    int
	Language string
	Filter   string
	SearchDB string
}

type AnswerQuery struct {
	Text     string
	Limit    int
	Language string
	Model    string
}

type HotlistQuery struct {
	Limit    int
	Language string
}

type Query struct {
	Text    string     `json:"text"`
	Mode    QueryMode  `json:"mode"`
	Sources []SourceID `json:"sources"`
	Limit   int        `json:"limit"`
}

type Result struct {
	Source      SourceID   `json:"source"`
	Type        ResultType `json:"type"`
	Title       string     `json:"title"`
	URL         string     `json:"url,omitempty"`
	Snippet     string     `json:"snippet,omitempty"`
	Content     string     `json:"content,omitempty"`
	Images      []Image    `json:"images,omitempty"`
	Citations   []Citation `json:"citations,omitempty"`
	PublishedAt string     `json:"published_at,omitempty"`
}

type Image struct {
	Title string `json:"title,omitempty"`
	URL   string `json:"url"`
	Page  string `json:"page,omitempty"`
}

type Citation struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type SourceStatusValue string

const (
	SourceStatusOK           SourceStatusValue = "ok"
	SourceStatusSkipped      SourceStatusValue = "skipped"
	SourceStatusError        SourceStatusValue = "error"
	SourceStatusTimeout      SourceStatusValue = "timeout"
	SourceStatusUnauthorized SourceStatusValue = "unauthorized"
	SourceStatusRateLimited  SourceStatusValue = "rate_limited"
)

type SourceStatus struct {
	Source         SourceID          `json:"source"`
	Status         SourceStatusValue `json:"status"`
	Results        int               `json:"results"`
	EffectiveLimit int               `json:"effective_limit"`
	DurationMS     int64             `json:"duration_ms"`
	Error          *findoerr.Error   `json:"error"`
}

type Envelope struct {
	Version      string           `json:"version"`
	Query        Query            `json:"query"`
	Status       Status           `json:"status"`
	Results      []Result         `json:"results"`
	SourceStatus []SourceStatus   `json:"source_status"`
	Errors       []findoerr.Error `json:"errors"`
}
