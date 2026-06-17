# Spec 04: Volcengine Source

## Goal

Specify the Volcengine Ark answer adapter using the working reference implementation in `/root/go/src/industry-research/researcher`.

Findo should expose Volcengine as a web-grounded answer source, not as a general chat interface.

## Volcengine Ark Answer

### Capability

```go
CapabilityAnswer
```

### Source Identity

| Field | Value |
| --- | --- |
| Provider family | `volcengine` |
| Source ID | `volcengine_answer` |
| Result type | `answer` |

### Commands

```bash
findo volc "2026 AI 搜索趋势"
findo volc answer "2026 AI 搜索趋势" --json
```

`findo volc` is an alias for `findo volc answer`.

### Endpoint

```text
POST https://ark.cn-beijing.volces.com/api/v3/responses
```

Config override:

```yaml
volcengine:
  endpoint: https://ark.cn-beijing.volces.com/api/v3/responses
```

### Auth

Header:

```text
Authorization: Bearer <api key>
Content-Type: application/json
```

Resolve credentials using the global precedence `flags > env > config > defaults`.

Recognized credential inputs:

- `VOLCENGINE_API_KEY`
- `ARK_API_KEY`
- `volcengine.api_key`

### Default Model

```text
doubao-seed-2-0-lite-260215
```

Override order:

1. `VOLCENGINE_MODEL`
2. `volcengine.model`
3. built-in default

### Request

Minimum request body:

```json
{
  "model": "doubao-seed-2-0-lite-260215",
  "tools": [
    {
      "type": "web_search",
      "limit": 10
    }
  ],
  "input": [
    {
      "role": "user",
      "content": [
        {
          "type": "input_text",
          "text": "2026 AI 搜索趋势"
        }
      ]
    }
  ]
}
```

Typed request controls:

- `limit`
- `model`

Only `limit` needs CLI flag support in the first pass. `model` is supported through env/config. Do not add Ark-specific controls to v1 public query structs until a command exposes them.

### Response Validation

A 2xx HTTP response is not enough.

The response must include at least one item:

```json
{
  "type": "web_search_call"
}
```

If no `web_search_call` exists, return:

```text
NO_RETRIEVAL_TRIGGERED
```

This behavior comes from the reference project and is important. Without it, Findo could present a model-only answer as if it were web-grounded.

### Response Mapping

Answer text comes from:

```text
output[].content[] where type == "output_text"
```

URL citations come from:

```text
output[].content[].annotations[] where type == "url_citation"
```

Map answer to a primary result:

```go
Result{
    Source:    "volcengine_answer",
    Type:      "answer",
    Title:     "Volcengine Answer",
    Content:   answerText,
    Citations: citations,
}
```

Map citations to `Citation`:

```go
Citation{
    Title: annotation.Title,
    URL:   annotation.URL,
}
```

Findo v1.0.0 must keep citations inside the answer result. It must not emit separate citation results in the stable JSON envelope.

### Usage Metadata

Volcengine usage data may be preserved in raw/debug output.

It does not enter the stable v1.0.0 result schema unless a specific user-facing use case appears.

### Error Mapping

| Provider condition | Findo code | Retryable |
| --- | --- | --- |
| missing local API key | `CREDENTIAL_MISSING` | false |
| HTTP 400 | `INVALID_ARGUMENT` | false |
| HTTP 401/403 | `SOURCE_UNAUTHORIZED` | false |
| HTTP 429 | `SOURCE_RATE_LIMITED` | true |
| HTTP 5xx | `SOURCE_UNAVAILABLE` | true |
| empty 2xx body | `SOURCE_BAD_RESPONSE` | false |
| invalid JSON in 2xx body | `SOURCE_BAD_RESPONSE` | false |
| no `web_search_call` | `NO_RETRIEVAL_TRIGGERED` | false |
| context deadline | `SOURCE_TIMEOUT` | true |

For 401/403, the error message should mention both credential validity and web search permission, because the reference implementation found both are plausible causes.

### Smoke Command

```bash
ARK_API_KEY=... findo volc "瑞幸咖啡 2026 门店数是否可信" --json
```

Smoke assertions:

- exit code is documented
- stdout parses as JSON
- `source_status` contains `volcengine_answer`
- result count or structured error is present
- secrets are not printed
