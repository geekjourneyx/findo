<div align="center">

# Tanso

**Search Chinese web sources from one Go CLI.**

<img src="assets/banner.webp" alt="Tanso - Chinese web sources from one Go CLI" width="100%">

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8.svg)](./go.mod)

</div>

Tanso queries Bocha, Volcengine Ark, and Zhihu through provider APIs, then returns normalized terminal output or automation-safe JSON. The name comes from the feel of "探索 / 探索する": Tanso explores Chinese internet signals for developers, AI agents, and research workflows without scraping, browser sessions, or hidden side effects.

## Install

Recommended:

```bash
npm install -g @geekjourneyx/tanso
tanso version
```

The npm package installs the matching GitHub Release binary for your platform and verifies it against `SHA256SUMS`.

Alternative Go install:

```bash
go install github.com/geekjourneyx/tanso/cmd/tanso@v2.0.2
```

Prebuilt binaries and checksums are available on the [GitHub Releases](https://github.com/geekjourneyx/tanso/releases) page.

From a local checkout:

```bash
make build
./tanso version
```

## Configure

Initialize the default config file:

```bash
tanso config init
tanso config path
```

This creates a config file at Tanso's stable user config path:

| Platform | Default path |
| --- | --- |
| macOS/Linux | `~/.config/tanso/config.yaml` |
| Windows | `%USERPROFILE%\.config\tanso\config.yaml` |

If `XDG_CONFIG_HOME` is set, Tanso uses `$XDG_CONFIG_HOME/tanso/config.yaml`.

Run `tanso config path` for the exact path on your machine.

Edit that file and set the credentials for the providers you want to use:

| Provider | Environment variables |
| --- | --- |
| Bocha | `BOCHA_API_KEY` |
| Volcengine Ark | `ARK_API_KEY` or `VOLCENGINE_API_KEY` |
| Zhihu | `ZHIHU_ACCESS_SECRET` or `ZHIHU_API_KEY` |

The config file uses the same provider fields:

```yaml
bocha:
  api_key: ""

volcengine:
  api_key: ""
  model: doubao-seed-2-0-lite-260215

zhihu:
  access_secret: ""
```

Configuration path resolution is:

1. `--config`
2. `TANSO_CONFIG`
3. `$XDG_CONFIG_HOME/tanso/config.yaml`, when `XDG_CONFIG_HOME` is set
4. `~/.config/tanso/config.yaml`

Configuration value precedence is:

1. Provider environment variables
2. Config file values
3. Built-in defaults

Inspect the merged config without leaking secrets:

```bash
tanso config show --json
```

`tanso sources --json` is an inventory endpoint. Its `configured` field only means
local credential material is present after config/env resolution; it is not a live
authentication check. Use a real source query to verify whether credentials work.

Before publishing a release, verify version alignment across `package.json`, `Makefile`, `CHANGELOG.md`, and the release tag:

```bash
make version-check
make release-check
```

## Quick Start

```bash
tanso version
tanso --help
tanso sources --json
tanso skills list --json
tanso skills read tanso --json

tanso "AI Agent 商业化" --json
tanso all "AI Agent 商业化" --json
tanso "AI Agent 商业化" --source bocha_web --source zhihu_search --json
BOCHA_API_KEY=... tanso bocha "AI Agent 商业化" --json
BOCHA_API_KEY=... tanso bocha "微信 AI Agent" --json
ARK_API_KEY=... tanso volc "瑞幸咖啡 2026 门店数是否可信" --json
ZHIHU_ACCESS_SECRET=... tanso zhihu "AI 搜索" --json
ZHIHU_ACCESS_SECRET=... tanso zhihu web "ChatGPT 桌面版" --json
ZHIHU_ACCESS_SECRET=... tanso zhihu hot --json
```

Human output defaults to a table. Use `--json` for scripts, agents, CI, and downstream tools. Use `tanso --help` or `tanso <command> --help` to inspect the installed CLI surface.

## Agent Skill

Recommended Agent Skills install:

```bash
npx skills add geekjourneyx/tanso
```

Tanso also ships the same Agent SOP with the CLI package so agents can read instructions that match the executable on `PATH`:

```bash
tanso skills list --json
tanso skills read tanso
tanso skills read tanso --json
```

`skills read` defaults to raw Markdown for direct agent context. With `--json`, the same content is returned in the `content` field with version metadata. Use this when you need to verify the SOP bundled with the installed CLI.

## Output

Retrieval commands return a stable envelope. A successful JSON response looks like this:

```json
{
  "version": "2.0.2",
  "query": {
    "text": "AI Agent 商业化",
    "mode": "search",
    "sources": ["bocha_web"],
    "limit": 10
  },
  "status": "ok",
  "results": [
    {
      "source": "bocha_web",
      "title": "Example result title",
      "url": "https://example.com/article",
      "snippet": "A normalized summary from the provider response."
    }
  ],
  "source_status": [
    {
      "source": "bocha_web",
      "status": "ok",
      "results": 1,
      "effective_limit": 10,
      "duration_ms": 842,
      "error": null
    }
  ],
  "errors": []
}
```

Exit codes are part of the public contract:

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Partial success |
| `2` | Invalid argument |
| `3` | Config error |
| `4` | Credential missing |
| `5` | Source error |
| `6` | Timeout |
| `7` | No results |
| `9` | Internal error |

## Sources

| Source ID | Command | Provider | Capability |
| --- | --- | --- | --- |
| `bocha_web` | `tanso bocha` | Bocha | Web search |
| `volcengine_answer` | `tanso volc` | Volcengine Ark | Web-grounded answer |
| `zhihu_search` | `tanso zhihu` | Zhihu | In-site search |
| `zhihu_web` | `tanso zhihu web` | Zhihu | Global web search |
| `zhihu_hot` | `tanso zhihu hot` | Zhihu | Hotlist |

Zhihu global search supports provider-specific filters:

```bash
tanso zhihu web "ChatGPT 桌面版" \
  --filter 'host=="example.com"' \
  --search-db realtime \
  --json
```

`--filter` and `--search-db` are only valid for `tanso zhihu web`.

## Automation Contract

Tanso keeps the automation contract narrow and predictable:

- stdout is reserved for results.
- stderr is reserved for diagnostics.
- JSON output keeps source IDs, source status, error codes, and exit codes stable.
- Provider behavior stays behind typed Go adapters.
- Built-in timeout defaults to `45s`; override it with `--timeout` when a workflow needs a different budget.

## Development

```bash
make build
make test
make lint
make release-check
```

The normal test suite does not require real provider credentials. Real API smoke checks are separate:

```bash
make smoke-bocha
make smoke-volcengine
make smoke-zhihu
```

## Non-Goals

Tanso v1.0.0 intentionally does not implement browser scraping, cache, reranking, plugin runtime, MCP, stdin query input, Bocha image search, or Zhihu direct answer. Those boundaries keep the CLI small, testable, and stable.

## License

[MIT](./LICENSE)

## Author

- Website: [jieni.ai](https://jieni.ai)
- GitHub: [@geekjourneyx](https://github.com/geekjourneyx)
- X: [@seekjourney](https://x.com/seekjourney)
- WeChat Official Account: 极客杰尼
