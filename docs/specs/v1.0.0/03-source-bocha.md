# Spec 03: Bocha Source

## Goal

Specify the Bocha web search adapter using the working reference implementation in `/root/go/src/industry-research/researcher`.

Findo should preserve Bocha behavior that affects correctness, but normalize responses into Findo's result and error contracts.

## Bocha Web Search

### Capability

```go
CapabilityWebSearch
```

### Source Identity

| Field | Value |
| --- | --- |
| Provider family | `bocha` |
| Source ID | `bocha_web` |
| Result type | `web` |

### Command

```bash
findo bocha "AI Agent 商业化"
findo bocha "AI Agent 商业化" --json
```

### Endpoint

```text
POST https://api.bocha.cn/v1/web-search
```

Config override:

```yaml
bocha:
  endpoint: https://api.bocha.cn/v1/web-search
```

### Auth

Header:

```text
Authorization: Bearer <BOCHA_API_KEY>
Content-Type: application/json
```

Resolve credentials using the global precedence `flags > env > config > defaults`.

Recognized credential inputs:

- `BOCHA_API_KEY`
- `bocha.api_key`

### Request

Minimum request body:

```json
{
  "query": "AI Agent 商业化",
  "summary": true
}
```

Optional Findo-controlled fields:

- `--limit` is applied by Findo after decoding provider results.
- provider-specific fields may be added only after they are represented by typed config or typed query fields.

The reference project sends `summary: true` by default. Findo should keep this default because summaries improve terminal and Markdown usefulness.

Bocha v1.0.0 does not rely on provider-side count support. `source_status.effective_limit` equals the requested CLI/config limit for `bocha_web`.

### Response Mapping

Bocha path:

```text
data.webPages.value[]
```

Map each page to:

```go
Result{
    Source:      "bocha_web",
    Type:        "web",
    Title:       page.Name,
    URL:         page.URL,
    Snippet:     firstNonEmpty(page.Summary, page.Snippet),
    PublishedAt: page.DatePublished,
}
```

Provider fields that may be preserved in raw/debug mode:

- `id`
- `displayUrl`
- `siteName`
- `siteIcon`
- `cachedPageUrl`
- `dateLastCrawled`
- `language`
- `isFamilyFriendly`
- `isNavigational`
- `totalEstimatedMatches`
- `someResultsRemoved`
- `log_id`

### Time Handling

The reference implementation normalizes `dateLastCrawled` values ending in `Z` to `+08:00`.

Findo should not put `last_crawled_at` in the stable v1.0.0 result schema. If exposed later, it must be normalized consistently.

### Success Criteria

A Bocha result is usable if:

- provider HTTP status is 2xx
- provider code normalizes to `200`
- pages can be decoded without parse failure

Zero results is not a provider error by itself. It becomes `NO_RESULTS` only if all requested sources produce no usable result.

### Error Mapping

Provider code may be string, number, or absent. Normalize before comparison.

| Provider condition | Findo code | Retryable |
| --- | --- | --- |
| missing local API key | `CREDENTIAL_MISSING` | false |
| HTTP 400 or provider `400` | `INVALID_ARGUMENT` | false |
| 400 message mentions API key/auth/token | `CREDENTIAL_MISSING` | false |
| HTTP 401 or provider `401` | `SOURCE_UNAUTHORIZED` | false |
| HTTP 403 or provider `403` | `SOURCE_UNAUTHORIZED` | false |
| HTTP 429 or provider `429` | `SOURCE_RATE_LIMITED` | true |
| HTTP 5xx | `SOURCE_UNAVAILABLE` | true |
| empty 2xx body | `SOURCE_BAD_RESPONSE` | false |
| invalid JSON in 2xx body | `SOURCE_BAD_RESPONSE` | false |
| context deadline | `SOURCE_TIMEOUT` | true |

The reference project maps Bocha 403 to quota exhaustion. Findo v1.0.0 does not expose a separate quota code; put provider status and message in error details.

### Smoke Command

```bash
BOCHA_API_KEY=... findo bocha "瑞幸咖啡 2026 门店数" --json
```

Smoke assertions:

- exit code is documented
- stdout parses as JSON
- `source_status` contains `bocha_web`
- result count or structured error is present
- secrets are not printed
