# Spec 06: Testing and Release

## Goal

Define the quality gates needed to ship `findo v1.0.0` as a reliable Go CLI.

This spec follows `go-cli-ship`: stable CLI contract, testable output, release artifacts, checksums, and README quality.

## Test Layers

### Unit Tests

Required packages:

- `internal/config`
- `internal/findoerr`
- `internal/output`
- `internal/search`
- `internal/source/bocha`
- `internal/source/volcengine`
- `internal/source/zhihu`

Required coverage:

- config defaults
- config path resolution
- env override precedence
- secret redaction
- invalid flag handling
- JSON rendering
- Markdown rendering
- table rendering
- provider error mapping
- provider response normalization
- timeout handling

### Contract Tests

Golden files should cover:

- successful Bocha JSON envelope
- successful Volcengine JSON envelope
- successful Zhihu search JSON envelope
- successful Zhihu global search JSON envelope
- partial success envelope
- all-source failure envelope
- Markdown output with citations
- source status with one failed provider

Contract tests must assert:

- stable field names
- stable source IDs, not provider family aliases
- omitted empty optional fields
- source order
- status value
- error code
- exit code class
- `query.limit` preserves the requested limit
- `source_status.effective_limit` is present for every selected retrieval source and records provider clamping when it differs
- provider ranking scores do not appear in stable `results`
- status and exit behavior matches the decision table in `01-cli-contract.md`

### CLI Tests

Use a testable `cli.Run(args, version, stdout, stderr)` function, following the reference project pattern.

Required cases:

```text
findo help
findo version
findo sources --json
findo config
findo bocha --json
findo bocha "query" --json with missing key
findo volc "query" --json with missing key
findo zhihu "query" --json with missing key
findo zhihu web "query" --json with missing key
findo hot zhihu --json with missing key
findo all "query" --json with one provider failure
unknown command
missing query
invalid --limit
invalid --timeout
invalid --filter
invalid --search-db
mutually exclusive output flags
```

### E2E Tests

E2E tests build and execute the binary:

```bash
go build -trimpath -o /tmp/findo-e2e ./cmd/findo
/tmp/findo-e2e version
/tmp/findo-e2e sources --json
```

E2E tests must not require real credentials by default.

## Provider Test Fixtures

Bocha fixtures:

- 200 with `code=200` and pages
- 200 empty body
- 200 invalid JSON
- non-2xx JSON provider error
- 429 rate limit
- 401 unauthorized
- 403 quota or permission error
- 500 unavailable
- `code` as string
- `code` as number

Volcengine fixtures:

- 200 with `web_search_call`
- 200 with answer text and URL citations
- 200 without `web_search_call`
- 200 empty body
- 200 invalid JSON
- 400 invalid request
- 401 unauthorized
- 403 missing permission
- 429 rate limit
- 500 unavailable

Zhihu search fixtures:

- 200 with `Code=0` and items
- 200 with `Code=0` and empty items
- 200 with `Code=10001` parameter error
- 200 with `Code=20001` auth failure
- 200 with `Code=30001` rate limit
- 200 with `Code=90001` internal error
- 200 empty body
- 200 invalid JSON
- HTTP 401 unauthorized
- HTTP 429 rate limit
- HTTP 500 unavailable
- `Count > 10` clamped before request
- `source_status.effective_limit == 10` when requested limit is above the Zhihu search cap
- missing `X-Request-Timestamp` must not happen in generated requests

Zhihu global search fixtures:

- 200 with `Code=0` and items
- 200 with `Code=0` and empty items
- 200 with `Code=10001` parameter error
- 200 with `Code=20001` auth failure
- 200 with `Code=30001` rate limit
- 200 with `Code=90001` internal error
- 200 empty body
- 200 invalid JSON
- HTTP 401 unauthorized
- HTTP 429 rate limit
- HTTP 500 unavailable
- `Count > 20` clamped before request
- `source_status.effective_limit == 20` when requested limit is above the Zhihu global cap
- `SearchDB` accepts only `all`, `realtime`, and `static`
- `Filter` is encoded through query parameter encoding
- missing `X-Request-Timestamp` must not happen in generated requests

Zhihu hotlist fixtures:

- 200 with `Code=0` and items
- 200 with `Code=0` and empty items
- 200 with `Code=20001` auth failure
- 200 with `Code=30001` rate limit
- 200 with `Code=90001` internal error
- 200 empty body
- 200 invalid JSON
- HTTP 401 unauthorized
- HTTP 429 rate limit
- HTTP 500 unavailable
- `Limit > 30` clamped before request
- `source_status.effective_limit == 30` when requested limit is above the Zhihu hotlist cap
- `Limit <= 0` not sent from valid CLI paths
- empty `ThumbnailUrl` omits image fields
- empty `Summary` omits snippet
- missing `X-Request-Timestamp` must not happen in generated requests

## Real API Smoke Tests

Smoke tests are separate from normal CI and gated by credentials.

Bocha:

```bash
BOCHA_API_KEY=... make smoke-bocha
```

Volcengine:

```bash
ARK_API_KEY=... make smoke-volcengine
```

Zhihu:

```bash
ZHIHU_ACCESS_SECRET=... make smoke-zhihu
```

Expected smoke assertions:

- command exits `0`, `1`, or a documented source error code
- stdout is valid JSON when `--json` is used
- stderr does not contain secrets
- `source_status` contains the source ID
- successful responses include at least one result
- failed responses include stable error code and retryable flag

Smoke tests may skip when credentials are absent. A release candidate cannot be marked verified until all supported source families have a recorded smoke pass.

## Make Targets

Required:

```bash
make build
make test
make lint
make release-check
make smoke-bocha
make smoke-volcengine
make smoke-zhihu
```

Suggested implementation:

```makefile
VERSION ?= 1.0.0
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o findo ./cmd/findo

test:
	CGO_ENABLED=0 go test -count=1 ./...

lint:
	gofmt -l .
	go vet ./...
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0 run

release-check:
	bash scripts/release-check.sh
```

## Release Check

`scripts/release-check.sh` must verify:

- `VERSION` is `1.0.0` for first release
- `CHANGELOG.md` has `v1.0.0`
- `README.md` contains install and quick start commands
- `README.md` documents JSON output
- `README.md` documents credential env vars
- `go test ./...` passes
- `gofmt -l .` is empty
- lint uses pinned `golangci-lint`
- release tag, when present, matches `v1.0.0`

## GitHub Actions

CI workflow:

- checkout
- setup Go
- run `go test ./...`
- run `go vet ./...`
- run `gofmt` check
- run pinned `golangci-lint`

Release workflow:

- trigger on `v*` tags
- build matrix:
  - linux amd64
  - linux arm64
  - darwin amd64
  - darwin arm64
- inject version via `-ldflags`
- upload artifacts
- generate `SHA256SUMS`

## Release Artifacts

Artifact naming:

```text
findo_1.0.0_linux_amd64.tar.gz
findo_1.0.0_linux_arm64.tar.gz
findo_1.0.0_darwin_amd64.tar.gz
findo_1.0.0_darwin_arm64.tar.gz
SHA256SUMS
```

Each archive should include:

- `findo` binary
- `README.md`
- `LICENSE`

## README Gate

README must answer in the first screen:

1. What is Findo?
2. What command do I run first?
3. What sources work today?
4. How do I get JSON?

Required sections:

- install
- quick start
- config
- env vars
- source status and partial success
- JSON contract
- Markdown workflow
- security
- contributing a source

## Definition of Done for First Specs Implementation

The first implementation pass is done when:

1. CLI foundation works.
2. Config loading works.
3. Bocha web search works.
4. Volcengine answer works.
5. Zhihu in-site search works.
6. Zhihu global search works.
7. Zhihu hotlist works.
8. JSON envelope is contract-tested.
9. Partial success is contract-tested.
10. Missing credentials are handled without panic.
11. Provider fixtures cover error mapping.
12. Provider limit clamping is visible through `effective_limit`.
13. `make release-check` exists.
14. README can onboard a new user to one real source.
