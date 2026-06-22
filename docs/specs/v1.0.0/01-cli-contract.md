# Spec 01: CLI Contract

## Goal

Define the stable public command contract for `tanso v1.0.0`.

The CLI is the product API. Command names, output shape, exit codes, and flag semantics must be treated as compatibility commitments after `v1.0.0`.

## Command Shape

```bash
tanso <query>
tanso all <query>

tanso bocha <query>

tanso volc <query>
tanso volc answer <query>

tanso zhihu <query>
tanso zhihu web <query>
tanso zhihu hot

tanso sources
tanso config init
tanso config path
tanso config show --json
tanso init
tanso version
tanso --version
tanso help
tanso --help
```

### v1.0.0 Command Matrix

| Command | Source ID | Provider family | Capability | Output contract |
| --- | --- | --- | --- | --- |
| `tanso <query>` | default query source IDs | mixed | `web_search`, `answer` | retrieval envelope, `query.mode=mixed` |
| `tanso all <query>` | all enabled query source IDs | mixed | `web_search`, `answer` | retrieval envelope, `query.mode=mixed` |
| `tanso bocha <query>` | `bocha_web` | `bocha` | `web_search` | stable envelope |
| `tanso volc <query>` | `volcengine_answer` | `volcengine` | `answer` | stable envelope |
| `tanso volc answer <query>` | `volcengine_answer` | `volcengine` | `answer` | stable envelope |
| `tanso zhihu <query>` | `zhihu_search` | `zhihu` | `web_search` | stable envelope |
| `tanso zhihu web <query>` | `zhihu_web` | `zhihu` | `web_search` | stable envelope |
| `tanso zhihu hot` | `zhihu_hot` | `zhihu` | `hotlist` | stable envelope |
| `tanso sources` | n/a | mixed | inspection | source inventory; JSON when `--json` |
| `tanso config init` | n/a | mixed | config creation | human output |
| `tanso config path` | n/a | mixed | inspection | config path text |
| `tanso config show --json` | n/a | mixed | inspection | redacted JSON config |
| `tanso init` | n/a | mixed | config creation | alias for `tanso config init` |
| `tanso version` | n/a | n/a | inspection | stable text; JSON when `--json` |
| `tanso --version` | n/a | n/a | inspection | stable text |
| `tanso help`, `tanso --help`, `tanso <command> --help` | n/a | n/a | inspection | human output |

Deferred commands are not part of the v1.0.0 public contract until their source specs are complete: `tanso bocha image <query>` and `tanso zhihu answer <query>`.

## Output Rules

- Results go to `stdout`.
- Diagnostics go to `stderr`.
- Human default is `--table`.
- Machine workflows use `--json`.
- Research workflows use `--markdown`.
- Provider debugging uses `--raw`.
- `--json`, `--markdown`, `--table`, and `--raw` are mutually exclusive.
- No progress text is printed to `stdout`.
- v1.0.0 does not read query text from `stdin`. Shell composition is output-first: pipe `stdout` to downstream tools.

## Global Flags

```bash
--json
--markdown
--table
--raw
--limit int
--timeout duration
--config path
--source source_id
--no-color
--verbose
```

Flag behavior:

- `--limit` is the requested per-source limit. Valid CLI range for v1.0.0 is `1..50`.
- Providers with lower limits clamp to an effective limit before sending the request. JSON `query.limit` records the requested limit; `source_status.effective_limit` records the per-source value used for each selected source.
- `--timeout` is a total command timeout. Provider calls receive derived contexts.
- `--source` accepts source IDs. It may be repeated or comma-separated for generic commands.
- Family aliases may expand deterministically in generic commands: `bocha` -> `bocha_web`, `volcengine` -> `volcengine_answer`, `zhihu` -> `zhihu_search`.
- `--raw` currently prints the same JSON retrieval envelope as `--json`; provider-shaped raw output is deferred.
- `--no-color` and `--verbose` are accepted for script compatibility; current output has no color and verbose diagnostics are deferred.

Source-specific flags:

```bash
--filter string                  # only valid for tanso zhihu web
--search-db all|realtime|static  # only valid for tanso zhihu web
```

Passing a source-specific flag to an incompatible command returns `INVALID_ARGUMENT` with exit code `2`.

## Environment Variables

Tanso-specific variables:

```text
TANSO_CONFIG
BOCHA_API_KEY
VOLCENGINE_API_KEY
VOLCENGINE_MODEL
ARK_API_KEY
ZHIHU_ACCESS_SECRET
ZHIHU_API_KEY
```

Volcengine credential resolution:

1. `VOLCENGINE_API_KEY`
2. `ARK_API_KEY`
3. config field `volcengine.api_key`

`ARK_API_KEY` is supported because the reference implementation and Volcengine Ark examples use it. `VOLCENGINE_API_KEY` remains the Tanso-native name.

Zhihu credential resolution:

1. `ZHIHU_ACCESS_SECRET`
2. `ZHIHU_API_KEY`
3. config field `zhihu.access_secret`

`ZHIHU_ACCESS_SECRET` matches Zhihu's `access_secret` terminology. `ZHIHU_API_KEY` is kept as a compatibility alias for earlier Tanso docs.

## Config Resolution

Value precedence:

1. CLI flags
2. Environment variables
3. Config file values
4. Built-in defaults

Config file discovery:

1. Explicit path from `--config`
2. `TANSO_CONFIG`
3. `$XDG_CONFIG_HOME/tanso/config.yaml`, when `XDG_CONFIG_HOME` is set
4. `~/.config/tanso/config.yaml`

The fallback default is intentionally the same Unix-style user path across supported platforms. On macOS this means `~/.config/tanso/config.yaml`, not `~/Library/Application Support/tanso/config.yaml`; on Windows this means `%USERPROFILE%\.config\tanso\config.yaml`, not `%AppData%\tanso\config.yaml`.

If an explicit config path is provided and cannot be loaded, the command fails. If default paths do not exist, built-in defaults are used.

## Source IDs

`source` in stable JSON is always a `source_id`, not a provider package name.

| Provider family | Source ID | Capability | Command |
| --- | --- | --- | --- |
| `bocha` | `bocha_web` | `web_search` | `tanso bocha <query>` |
| `volcengine` | `volcengine_answer` | `answer` | `tanso volc <query>` |
| `zhihu` | `zhihu_search` | `web_search` | `tanso zhihu <query>` |
| `zhihu` | `zhihu_web` | `web_search` | `tanso zhihu web <query>` |
| `zhihu` | `zhihu_hot` | `hotlist` | `tanso zhihu hot` |

## Query Modes and Source Selection

Allowed retrieval `query.mode` values:

- `search`: one or more web-search sources.
- `answer`: one answer source.
- `hotlist`: one hotlist source.
- `mixed`: a generic query that may include both `web_search` and `answer` sources.

Selection rules:

- `tanso <query>` selects `search.default_source_ids` in config order. The default v1 set is `bocha_web`, `volcengine_answer`, and `zhihu_search`, so its mode is `mixed`.
- `tanso all <query>` selects every enabled source whose capability is `web_search` or `answer`, in source registry order. It never selects `hotlist`.
- `tanso bocha <query>` selects only `bocha_web` and uses `query.mode=search`.
- `tanso volc <query>` and `tanso volc answer <query>` select only `volcengine_answer` and use `query.mode=answer`.
- `tanso zhihu <query>` selects only `zhihu_search` and uses `query.mode=search`.
- `tanso zhihu web <query>` selects only `zhihu_web` and uses `query.mode=search`.
- `tanso zhihu hot` selects only `zhihu_hot` and uses `query.mode=hotlist`.

`--source` accepts source IDs for generic commands. Family aliases may expand deterministically in generic commands: `bocha` -> `bocha_web`, `volcengine` -> `volcengine_answer`, `zhihu` -> `zhihu_search`.

## Retrieval JSON Envelope

All retrieval commands use this stable envelope:

```json
{
  "version": "1.0.0",
  "query": {
    "text": "AI Agent 商业化",
    "mode": "search",
    "sources": ["bocha_web", "volcengine_answer"],
    "limit": 10
  },
  "status": "ok",
  "results": [],
  "source_status": [],
  "errors": []
}
```

Allowed `status` values:

- `ok`
- `partial`
- `error`

Required top-level fields:

- `version`
- `query`
- `status`
- `results`
- `source_status`
- `errors`

Arrays are emitted as empty arrays, not `null`.

## Result Object

```json
{
  "source": "bocha_web",
  "type": "web",
  "title": "AI Agent 市场趋势报告",
  "url": "https://example.com/report",
  "snippet": "摘要内容"
}
```

Required fields:

- `source`
- `type`
- `title`

Optional fields:

- `url`
- `snippet`
- `content`
- `images`
- `citations`
- `published_at`

Empty optional fields are omitted.

## Source Status Object

```json
{
  "source": "bocha_web",
  "status": "ok",
  "results": 10,
  "effective_limit": 10,
  "duration_ms": 842,
  "error": null
}
```

Allowed source statuses:

- `ok`
- `skipped`
- `error`
- `timeout`
- `unauthorized`
- `rate_limited`

Required fields:

- `source`
- `status`
- `results`
- `effective_limit`
- `duration_ms`
- `error`

`error` is `null` for successful source calls and an error object otherwise.

## Error Object

```json
{
  "code": "SOURCE_TIMEOUT",
  "message": "bocha request timed out after 45s",
  "source": "bocha_web",
  "retryable": true,
  "details": {
    "timeout": "45s"
  }
}
```

Stable error codes:

- `INVALID_ARGUMENT`
- `CONFIG_NOT_FOUND`
- `CONFIG_INVALID`
- `CREDENTIAL_MISSING`
- `SOURCE_UNAVAILABLE`
- `SOURCE_UNAUTHORIZED`
- `SOURCE_RATE_LIMITED`
- `SOURCE_TIMEOUT`
- `SOURCE_BAD_RESPONSE`
- `NO_RESULTS`
- `NO_RETRIEVAL_TRIGGERED`
- `INTERNAL_ERROR`

Error object rules:

- `code`, `message`, and `retryable` are always present.
- `source` is present for source-specific errors and omitted for command/config/internal errors.
- `provider_status`, `provider_code`, and `details` are optional diagnostics.
- `details` values are strings only.

## Exit Codes

```text
0  success
1  partial success
2  invalid arguments
3  config error
4  credential error
5  source/API error
6  timeout
7  no results
9  internal error
```

Exit code `1` is reserved for partial success with usable output.

Automation note: shell scripts that can tolerate degraded output should explicitly accept `0` and `1`, for example:

```bash
tanso all "AI 搜索" --json > out.json
case $? in
  0|1) jq '.results[]?.url' out.json ;;
  *) exit 1 ;;
esac
```

## Status and Exit Decision Table

| Condition | Envelope status | Primary error code | Exit |
| --- | --- | --- | --- |
| invalid command, flag, or query | no retrieval envelope required | `INVALID_ARGUMENT` | `2` |
| explicit config path missing | no retrieval envelope required | `CONFIG_NOT_FOUND` | `3` |
| config parse or validation error | no retrieval envelope required | `CONFIG_INVALID` | `3` |
| explicit source command missing credentials | `error` | `CREDENTIAL_MISSING` | `4` |
| generic command has one or more skipped unconfigured sources and at least one usable result | `partial` | `CREDENTIAL_MISSING` or skip reason | `1` |
| generic command has all selected sources skipped for missing credentials | `error` | `CREDENTIAL_MISSING` | `4` |
| all attempted sources succeed and at least one result exists | `ok` | none | `0` |
| all attempted sources succeed but all return zero results | `error` | `NO_RESULTS` | `7` |
| at least one source has results and at least one source fails, times out, or is skipped | `partial` | source-specific code | `1` |
| no source has results and at least one source times out | `error` | `SOURCE_TIMEOUT` | `6` |
| no source has results and at least one source has a non-timeout source/API failure | `error` | source-specific code | `5` |
| unexpected internal failure | no retrieval envelope required | `INTERNAL_ERROR` | `9` |

If multiple error classes apply and no usable result exists, exit-code precedence is: invalid argument, config, credential, timeout, source/API, no results, internal.

## Human Output

Table output must fit a normal terminal and should include:

- source
- rank or index
- type
- title
- URL when available

It must not hide source failures. If a source fails, a concise warning is printed to `stderr`.

## Markdown Output

Markdown output must preserve:

- query
- source sections
- answer text when present
- URLs
- citations
- source status

Markdown style may evolve without a major version bump, but content semantics must not.

## Raw Output

`--raw` is for provider debugging:

- It may expose provider response shape.
- It must not expose local credentials.
- It is not covered by SemVer schema compatibility.
- It does not use the stable retrieval envelope.
- It is written to `stdout` after credential redaction. Diagnostics still go to `stderr`.

Provider raw types must not enter `search.Envelope`; raw rendering is a CLI/output path around a single selected source call.

## Inspection JSON

Inspection commands do not use the retrieval envelope.

`tanso version --json`:

```json
{
  "version": "1.0.0"
}
```

`tanso sources --json`:

```json
{
  "version": "1.0.0",
  "sources": [
    {
      "source": "bocha_web",
      "provider_family": "bocha",
      "capabilities": ["web_search"],
      "enabled": true,
      "configured": false
    }
  ]
}
```

`configured` reports whether local credential material is present after normal
config/env resolution. It does not perform a live provider authentication check;
downstream agents must use real query status to verify credential validity.

`tanso config show --json` returns redacted resolved config. Secret values must be either omitted or rendered as `"***"`.

## Compatibility Rules

Allowed in minor versions:

- Add optional JSON fields.
- Add new commands.
- Add new sources.
- Add new source status details.

Not allowed without major version bump:

- Remove public commands.
- Rename public JSON fields.
- Change existing field meaning.
- Change exit code class semantics.
- Make config mandatory for simple usage.
