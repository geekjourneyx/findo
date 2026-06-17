# Spec 02: Architecture

## Goal

Define a small Go architecture for a source-aggregating search CLI.

The design follows the reference project's successful split between CLI, config, provider adapters, retrieval types, output, and errors. It removes researcher-specific concepts such as workspaces, reports, evidence ledgers, confidence scoring, and validation.

## Directory Layout

```text
findo/
├── cmd/findo/
├── internal/cli/
├── internal/config/
├── internal/search/
├── internal/source/
│   ├── bocha/
│   ├── volcengine/
│   └── zhihu/
├── internal/output/
├── internal/findoerr/
├── e2e/
├── scripts/
├── docs/
├── examples/
├── .github/workflows/
├── Makefile
├── CHANGELOG.md
├── LICENSE
└── README.md
```

## Package Responsibilities

### `cmd/findo`

Owns only process entry:

- pass `os.Args`
- pass version
- pass stdout/stderr
- call `internal/cli.Run`
- exit with returned code

### `internal/cli`

Owns command parsing and command dispatch:

- parse commands and flags
- load config
- build search requests
- select output renderer
- map top-level errors to exit codes

The reference project uses a small custom CLI layer and only depends on YAML. Findo should start the same way unless command parsing becomes materially painful.

### `internal/config`

Owns config loading:

- defaults
- YAML file loading
- env overrides
- secret redaction
- validation

It must not import source packages.

### `internal/search`

Owns the application core and product-level search contracts:

- `Query`
- `Result`
- `Citation`
- `Image`
- `Envelope`
- `SourceStatus`
- source capability interfaces
- orchestration

It must not know provider HTTP response shapes.

### `internal/source`

Owns source registry and shared adapter helpers.

Provider subpackages implement source-specific HTTP behavior:

- `internal/source/bocha`
- `internal/source/volcengine`
- `internal/source/zhihu`

Adapters depend on `internal/search` and `internal/findoerr`, not on `internal/cli`.

### `internal/output`

Owns output renderers:

- JSON
- Markdown
- Table
- Raw passthrough wrapper

Output renderers consume normalized `search.Envelope`.

Raw output is the exception: it consumes a redacted provider-shaped payload from the selected source adapter and must not route that payload through `search.Envelope`.

### `internal/findoerr`

Owns stable errors:

- error code constants
- retryability
- source name
- provider status/code
- exit code mapping

Provider raw errors are wrapped before reaching output.

## Core Types

### Capability

```go
type Capability string

const (
    CapabilityWebSearch Capability = "web_search"
    CapabilityAnswer    Capability = "answer"
    CapabilityHotlist   Capability = "hotlist"
)
```

### Source ID

```go
type SourceID string

const (
    SourceBochaWeb        SourceID = "bocha_web"
    SourceVolcengineAnswer SourceID = "volcengine_answer"
    SourceZhihuSearch     SourceID = "zhihu_search"
    SourceZhihuWeb        SourceID = "zhihu_web"
    SourceZhihuHot        SourceID = "zhihu_hot"
)
```

`SourceID` is the public JSON source identity. Provider package names such as `bocha`, `volcengine`, and `zhihu` are implementation groupings, not JSON source values.

### Result Type

```go
type ResultType string

const (
    ResultWeb    ResultType = "web"
    ResultAnswer ResultType = "answer"
    ResultHot    ResultType = "hot"
)
```

### Source Interfaces

```go
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
```

Do not use `map[string]any` as the main query extension point. The reference project used parameter maps to support a broader research engine. Findo should keep typed contracts until a real source proves a typed field is impossible.

### Query Types

```go
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
```

`AnswerQuery` exists because Volcengine has answer-specific controls. These controls should not leak into Bocha or Zhihu search queries.

### Result Types

```go
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
```

The first implementation does not need ranking. Results are grouped by source in request order.

### Envelope Types

```go
type Status string

const (
    StatusOK      Status = "ok"
    StatusPartial Status = "partial"
    StatusError   Status = "error"
)

type QueryMode string

const (
    QueryModeSearch  QueryMode = "search"
    QueryModeAnswer  QueryMode = "answer"
    QueryModeHotlist QueryMode = "hotlist"
    QueryModeMixed   QueryMode = "mixed"
)

type Query struct {
    Text    string     `json:"text"`
    Mode    QueryMode  `json:"mode"`
    Sources []SourceID `json:"sources"`
    Limit   int        `json:"limit"`
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
    Error          *Error            `json:"error"`
}

type Envelope struct {
    Version      string         `json:"version"`
    Query        Query          `json:"query"`
    Status       Status         `json:"status"`
    Results      []Result       `json:"results"`
    SourceStatus []SourceStatus `json:"source_status"`
    Errors       []Error        `json:"errors"`
}
```

## Orchestration

Generic search flow:

```text
parse CLI -> load config -> resolve sources -> create context -> run sources -> normalize results -> render envelope
```

Multi-source behavior:

- Preserve requested source order.
- Run sources concurrently when doing so does not make output nondeterministic.
- Use one context per command.
- Continue after individual source failure.
- Return `status=partial` when at least one source succeeds and another fails or is skipped.
- Return `status=error` when no requested source returns usable results.

The reference project's `internal/provider/multi` preserves source order with indexed result slices. Findo should use the same idea.

## Error Model

Provider errors are converted to stable Findo errors:

```go
type Error struct {
    Code           string         `json:"code"`
    Message        string         `json:"message"`
    Source         SourceID       `json:"source,omitempty"`
    ProviderStatus int            `json:"provider_status,omitempty"`
    ProviderCode   string         `json:"provider_code,omitempty"`
    Retryable      bool           `json:"retryable"`
    Details        map[string]string `json:"details,omitempty"`
}
```

Stable codes are defined in `internal/findoerr`.

Provider-specific guidance can be included in `Details` or verbose diagnostics, but the stable code must remain provider-neutral.

## Config Model

```go
type Config struct {
    Search     SearchConfig     `yaml:"search"`
    Bocha      BochaConfig      `yaml:"bocha"`
    Volcengine VolcengineConfig `yaml:"volcengine"`
    Zhihu      ZhihuConfig      `yaml:"zhihu"`
    Output     OutputConfig     `yaml:"output"`
}
```

Defaults:

- Bocha endpoint: `https://api.bocha.cn/v1/web-search`
- Volcengine endpoint: `https://ark.cn-beijing.volces.com/api/v3/responses`
- Volcengine model: `doubao-seed-2-0-lite-260215`
- timeout: `45s`
- limit: `10`
- output: `table`

The Volcengine model default comes from the reference project. It should be easy to override through config or `VOLCENGINE_MODEL`.

Config files do not expand `${ENV}` placeholders in v1.0.0. Environment variables override config values before validation.

## Dependency Policy

Start with:

- Go standard library
- `gopkg.in/yaml.v3`

Do not add a CLI framework until custom parsing becomes a real source of complexity.

Avoid dependencies for:

- ranking
- caching
- plugins
- scheduling
- markdown templating
- terminal styling

## Naming Rules

- Packages are lowercase and short.
- Provider packages use provider names.
- CLI commands use lowercase words.
- JSON uses snake case.
- Error codes use uppercase snake case.
- Go exported names explain product concepts, not provider internals.

## Anti-Overdesign Checks

Before adding a package, ask:

1. Does this package own a stable boundary?
2. Can it be tested independently?
3. Would a new source adapter need it?
4. Is it solving a current v1.0.0 problem?

If the answer is mostly no, keep the code in the existing package.
