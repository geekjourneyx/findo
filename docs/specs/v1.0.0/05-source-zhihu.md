# Spec 05: Zhihu Sources

## Goal

Define Findo's Zhihu adapter contract using official Zhihu developer API details supplied for this project.

This document currently specifies Zhihu in-site content search, global web search, and hotlist. Zhida direct answer will be added when its official request and response fields are available in the same level of detail.

## Shared Zhihu Rules

### Auth

Every Zhihu API request uses:

```text
Authorization: Bearer <your_access_secret>
X-Request-Timestamp: <second-level Unix timestamp>
Content-Type: application/json
```

Resolve credentials using the global precedence `flags > env > config > defaults`.

Recognized credential inputs:

- `ZHIHU_ACCESS_SECRET`
- `ZHIHU_API_KEY`
- `zhihu.access_secret`

`ZHIHU_ACCESS_SECRET` is the preferred name because the official docs call the credential `access_secret`. `ZHIHU_API_KEY` remains a compatibility alias.

### Error Mapping

Provider response code mapping:

| Zhihu code | Meaning | Findo code | Retryable |
| --- | --- | --- | --- |
| `0` | success | none | false |
| `10001` | parameter error | `INVALID_ARGUMENT` | false |
| `20001` | auth failed | `SOURCE_UNAUTHORIZED` | false |
| `30001` | rate limited | `SOURCE_RATE_LIMITED` | true |
| `90001` | internal error | `SOURCE_UNAVAILABLE` | true |

HTTP-level mapping:

| HTTP condition | Findo code | Retryable |
| --- | --- | --- |
| missing local access secret | `CREDENTIAL_MISSING` | false |
| HTTP 400 | `INVALID_ARGUMENT` | false |
| HTTP 401/403 | `SOURCE_UNAUTHORIZED` | false |
| HTTP 429 | `SOURCE_RATE_LIMITED` | true |
| HTTP 5xx | `SOURCE_UNAVAILABLE` | true |
| empty 2xx body | `SOURCE_BAD_RESPONSE` | false |
| invalid JSON in 2xx body | `SOURCE_BAD_RESPONSE` | false |
| context deadline | `SOURCE_TIMEOUT` | true |

## Zhihu In-Site Search

### Capability

```go
CapabilityWebSearch
```

Zhihu in-site search is a content search source for questions, answers, and articles inside Zhihu. In Findo output it maps to normal `type=web` results with `source=zhihu_search`.

### Command

```bash
findo zhihu "RAG 评测方法"
findo zhihu "RAG 评测方法" --json
```

### Endpoint

```text
GET https://developer.zhihu.com/api/v1/content/zhihu_search
```

### Query Parameters

| Name | Type | Required | Findo source |
| --- | --- | --- | --- |
| `Query` | string | yes | search query text |
| `Count` | int32 | no | `--limit`, capped to 10 |

Rules:

- `Query` must not be empty.
- `Count` defaults to `10`.
- `Count <= 0` falls back to `10` server-side.
- `Count > 10` is truncated to `10` server-side.
- Findo should validate `Query` before sending the request.
- Findo should clamp `Count` to `1..10` before sending the request, so CLI behavior is explicit.

Example:

```bash
curl -G 'https://developer.zhihu.com/api/v1/content/zhihu_search' \
  --data-urlencode 'Query=怎么理解rave文化' \
  -d 'Count=5' \
  -H 'Authorization: Bearer <your_access_secret>' \
  -H "X-Request-Timestamp: $(date +%s)"
```

### Response Shape

Provider response:

```json
{
  "Code": 0,
  "Message": "success",
  "Data": {
    "HasMore": false,
    "SearchHashId": "1234567890",
    "Items": [
      {
        "Title": "RAG 评测方法综述",
        "ContentType": "Article",
        "ContentID": "123456789",
        "ContentText": "本文介绍了主流 RAG 评测框架，包括 RAGAS、TruLens ...",
        "Url": "https://zhuanlan.zhihu.com/p/123456789?utm_medium=openapi_platform&utm_source=6d23634e",
        "CommentCount": 15,
        "VoteUpCount": 128,
        "AuthorName": "张三",
        "AuthorAvatar": "https://picx.zhimg.com/example.jpg",
        "AuthorBadge": "",
        "AuthorBadgeText": "",
        "EditTime": 1710000000,
        "CommentInfoList": [],
        "AuthorityLevel": "2",
        "RankingScore": 0.98
      }
    ]
  }
}
```

### Response Mapping

Each `Data.Items[]` entry maps to:

```go
Result{
    Source:      "zhihu_search",
    Type:        "web",
    Title:       item.Title,
    URL:         item.Url,
    Snippet:     item.ContentText,
    PublishedAt: unixSecondsToRFC3339(item.EditTime),
}
```

Provider fields that are useful for `--raw`, diagnostics, or future metadata:

- `SearchHashId`
- `HasMore`
- `EmptyReason`
- `ContentType`
- `ContentID`
- `CommentCount`
- `VoteUpCount`
- `AuthorName`
- `AuthorAvatar`
- `AuthorBadge`
- `AuthorBadgeText`
- `CommentInfoList[].Content`
- `AuthorityLevel`
- `RankingScore`

### Content Type Handling

Official `ContentType` examples include `Article`. Findo should not hard-code only one value.

`ContentType` remains provider metadata in the first implementation. The stable Findo `type` remains `web` because all returned items are navigable content results.

### Time Handling

`EditTime` is an `Int32` Unix timestamp. Findo should convert it to RFC3339 UTC in `published_at`.

If `EditTime` is `0`, omit `published_at`.

### Empty Results

`Data.HasMore` is currently fixed to `false`.

If `Data.Items` is empty:

- source status is `ok`
- result count is `0`
- include `EmptyReason` in raw/debug details when present
- the whole command returns `NO_RESULTS` only if every requested source returns no usable result

### Success Criteria

A Zhihu search response is usable if:

- HTTP status is 2xx
- provider `Code == 0`
- `Data.Items` decodes successfully

Zero items is usable but empty.

## Zhihu Global Search

### Capability

```go
CapabilityWebSearch
```

Zhihu global search is a web search source for the broader internet. It can return Zhihu and non-Zhihu content depending on filter and index selection.

### Command

```bash
findo zhihu web "ChatGPT 电脑桌面版"
findo zhihu web "ChatGPT 电脑桌面版" --json
```

### Endpoint

```text
GET https://developer.zhihu.com/api/v1/content/global_search
```

### Query Parameters

| Name | Type | Required | Findo source |
| --- | --- | --- | --- |
| `Query` | string | yes | search query text |
| `Count` | int32 | no | `--limit`, capped to 20 |
| `Filter` | string | no | raw `--filter` advanced filter |
| `SearchDB` | string | no | `all`, `realtime`, or `static` |

Rules:

- `Query` must not be empty.
- `Count` defaults to `10`.
- `Count` max is `20`.
- `SearchDB` defaults to `all`.
- `SearchDB` must be one of `all`, `realtime`, `static`.
- `Filter` must be URL encoded as a query parameter.
- Prefer SDK/query parameter encoding or `url.Values`; do not concatenate encoded query strings manually.

### SearchDB Values

| Value | Meaning |
| --- | --- |
| `all` | all indexes, default |
| `realtime` | realtime index only |
| `static` | static index only |

### Filter Syntax

Supported fields:

| Field | Type | Example |
| --- | --- | --- |
| `host` | string | `host=="example.com"` |
| `publish_time` | int64 Unix timestamp | `publish_time>=1778494631` |

Operators:

- `host` supports `==` and `!=`.
- `publish_time` supports `==`, `!=`, `>`, `>=`, `<`, and `<=`.
- string values must use double quotes.
- numeric values must not use quotes.
- logical operators `AND` and `OR` must be uppercase.
- `AND` has higher priority than `OR`.
- parentheses may be used to control precedence.
- `host=="zhihu.com"` and Zhihu subdomains are not supported. Use `zhihu_search` through `findo zhihu <query>` for Zhihu-only content.

Examples:

```text
host=="example.com"
host=="example.com" AND publish_time>=1778494631
(host=="example.com" OR host=="news.example.com") AND publish_time>1778494631
```

### Findo Filter Policy

For v1.0.0, Findo exposes:

- `--filter <expr>`: raw advanced filter expression, passed through after validating it is non-empty.
- `--search-db <all|realtime|static>`: explicit index selection.

Typed flags such as `--host`, `--published-after`, and `--published-before` are deferred. The official filter syntax is already expressive; v1.0.0 should avoid building a second partial query language.

### Request Example

```bash
curl -G 'https://developer.zhihu.com/api/v1/content/global_search' \
  --data-urlencode 'Query=怎么理解rave文化' \
  --data-urlencode 'Filter=host=="example.com" AND publish_time>=1778494631' \
  --data-urlencode 'SearchDB=all' \
  -d 'Count=5' \
  -H 'Authorization: Bearer <your_access_secret>' \
  -H "X-Request-Timestamp: $(date +%s)"
```

### Response Shape

Provider response:

```json
{
  "Code": 0,
  "Message": "success",
  "Data": {
    "HasMore": false,
    "Items": [
      {
        "Title": "ChatGPT现在还值得开会员吗？",
        "ContentType": "Answer",
        "ContentID": "1903044959663284716",
        "ContentText": "首先要澄清一个常见误解...",
        "Url": "https://www.zhihu.com/answer/1903044959663284716?utm_medium=openapi_platform&utm_source=6d23634e",
        "CommentCount": 22,
        "VoteUpCount": 18,
        "AuthorName": "时光纪",
        "AuthorAvatar": "https://picx.zhimg.com/50/example_l.jpg",
        "AuthorBadge": "",
        "AuthorBadgeText": "",
        "EditTime": 1748355858,
        "CommentInfoList": [
          {
            "Content": "免费版现在也可以用gpt4o啊，只不过有限制，用的不多也够用"
          }
        ],
        "AuthorityLevel": "2"
      }
    ]
  }
}
```

### Response Mapping

Each `Data.Items[]` entry maps to:

```go
Result{
    Source:      "zhihu_web",
    Type:        "web",
    Title:       item.Title,
    URL:         item.Url,
    Snippet:     stripOrPreserveHighlight(item.ContentText),
    PublishedAt: unixSecondsToRFC3339(item.EditTime),
}
```

`ContentText` may contain `<em>` tags for highlighted text. For JSON output, Findo should preserve the text safely as a string. Table and Markdown renderers may strip or convert `<em>` tags to avoid rendering raw HTML unexpectedly.

Provider fields that are useful for `--raw`, diagnostics, or future metadata:

- `HasMore`
- `ContentType`
- `ContentID`
- `CommentCount`
- `VoteUpCount`
- `AuthorName`
- `AuthorAvatar`
- `AuthorBadge`
- `AuthorBadgeText`
- `CommentInfoList[].Content`
- `AuthorityLevel`
- `SearchDB`
- `Filter`

### Time Handling

`EditTime` is an `Int64` Unix timestamp. Findo should convert it to RFC3339 UTC in `published_at`.

If `EditTime` is `0`, omit `published_at`.

### Empty Results

If `Data.Items` is empty:

- source status is `ok`
- result count is `0`
- the whole command returns `NO_RESULTS` only if every requested source returns no usable result

### Success Criteria

A Zhihu global search response is usable if:

- HTTP status is 2xx
- provider `Code == 0`
- `Data.Items` decodes successfully

Zero items is usable but empty.

## Zhihu Hotlist

### Capability

```go
CapabilityHotlist
```

Zhihu hotlist returns the current structured Zhihu hot topics. The official API currently returns question and article hotlist items.

### Command

```bash
findo hot zhihu
findo hot zhihu --json
findo hot zhihu --limit 10 --markdown
```

### Endpoint

```text
GET https://developer.zhihu.com/api/v1/content/hot_list
```

### Query Parameters

| Name | Type | Required | Findo source |
| --- | --- | --- | --- |
| `Limit` | int32 | no | `--limit`, capped to 30 |

Rules:

- `Limit` defaults to `30`.
- `Limit` max is `30`.
- When `Limit <= 0` or `Limit > 30`, the server falls back to `30`.
- Findo should clamp `Limit` to `1..30` before sending the request, so CLI behavior is explicit.

Request example:

```bash
curl 'https://developer.zhihu.com/api/v1/content/hot_list?Limit=10' \
  -H 'Authorization: Bearer <your_access_secret>' \
  -H "X-Request-Timestamp: $(date +%s)"
```

### Response Shape

Provider response:

```json
{
  "Code": 0,
  "Message": "success",
  "Data": {
    "Total": 2,
    "Items": [
      {
        "Title": "如何评价某个热点问题？",
        "Url": "https://www.zhihu.com/question/123456789",
        "ThumbnailUrl": "https://pic1.zhimg.com/v2-d4b0f8158e064dbcc71eb6ce970230a9.jpg",
        "Summary": "这是该问题的内容摘要"
      },
      {
        "Title": "一篇正在热榜上的文章标题",
        "Url": "https://zhuanlan.zhihu.com/p/987654321",
        "ThumbnailUrl": "",
        "Summary": ""
      }
    ]
  }
}
```

### Response Mapping

Each `Data.Items[]` entry maps to:

```go
Result{
    Source:  "zhihu_hot",
    Type:    "hot",
    Title:   item.Title,
    URL:     item.Url,
    Snippet: item.Summary,
    Images:  imageFromThumbnail(item.ThumbnailUrl),
}
```

`ThumbnailUrl` and `Summary` are always returned by the provider. If either is an empty string, omit the corresponding optional Findo field.

Provider fields that are useful for `--raw`, diagnostics, or future metadata:

- `Total`
- `ThumbnailUrl`
- item index/rank

Findo should preserve hotlist order. The first provider item is rank 1.

### Empty Results

If `Data.Items` is empty:

- source status is `ok`
- result count is `0`
- the whole command returns `NO_RESULTS`

Hotlist is a single-source command, so there is no partial success case unless future versions combine hot sources.

### Success Criteria

A Zhihu hotlist response is usable if:

- HTTP status is 2xx
- provider `Code == 0`
- `Data.Items` decodes successfully
- `Data.Total` decodes successfully

Zero items is usable but empty.

### Error Mapping

Provider response code mapping:

| Zhihu code | Meaning | Findo code | Retryable |
| --- | --- | --- | --- |
| `0` | success | none | false |
| `20001` | auth failed | `SOURCE_UNAUTHORIZED` | false |
| `30001` | rate limited | `SOURCE_RATE_LIMITED` | true |
| `90001` | internal error | `SOURCE_UNAVAILABLE` | true |

## Deferred Zhihu APIs

The following commands are not part of the v1.0.0 public contract and need official field-level specs before implementation:

```bash
findo zhihu answer <query>
```

Expected API keys from the official docs URLs:

- `zhida`

Do not implement these commands from guesses. Add exact endpoint, method, headers, query/body parameters, response schema, success criteria, and error mapping first.
