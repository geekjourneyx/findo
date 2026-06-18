---
name: findo
description: >-
  Use Findo, an Agent Native Go CLI for Chinese internet research. Use this skill whenever the user asks to research Chinese web sources, search Zhihu, Bocha, Volcengine Ark, Chinese internet topics, hotlists, contradiction checks, source-backed briefs, or to configure/test the findo CLI. Also use it for Chinese prompts such as 搜一下知乎, 全网搜索, 查热榜, 火山引擎直答, 找热门回答, 反直觉话题, or 真实流程测试. This skill is appropriate when an agent needs automation-safe JSON retrieval from Chinese sources instead of browser scraping.
---

# Findo

Use this skill to run reliable Chinese internet research through the `findo` CLI and turn the results into source-backed answers. Findo is designed for agents: commands are explicit, stdout is reserved for results, stderr is reserved for diagnostics, and `--json` returns a stable envelope for automation.

## First Checks

Before running research, establish the local state:

```bash
findo version
findo config path
findo config show --json
findo skills read findo --json
```

Treat redacted values in `config show --json` as expected. Do not print secrets from `.env`, shell history, config files, or CI settings.

`findo skills read findo --json` reads the SOP embedded in the current CLI binary. Use it when an external skill install, README, or local repository checkout may be stale relative to the executable on `PATH`.

If `findo` is missing, install the npm package:

```bash
npm install -g @geekjourneyx/findo
findo version
```

🔴 CHECKPOINT: Before installing packages, writing config outside the current repository, or running real provider calls, confirm that the user asked for setup, installation, or live testing. Dry research can proceed with already installed tools and already available credentials.

## Configuration

Initialize the default config only when the user asks for setup or the config file is missing:

```bash
findo config init
findo config path
```

Findo reads credentials from environment variables or the config file. Use environment variables for one-off agent runs:

```text
BOCHA_API_KEY
ARK_API_KEY
VOLCENGINE_API_KEY
VOLCENGINE_MODEL
ZHIHU_ACCESS_SECRET
ZHIHU_API_KEY
```

When writing config from an `.env` file, preserve secret hygiene:

- Read only the variable names needed for the task.
- Never echo raw keys in chat or logs.
- Write the config file with `0600` permissions.
- Verify with `findo config show --json`, which redacts secrets.

## Source Selection

Choose the narrowest provider that answers the task:

| Need | Command |
| --- | --- |
| General web search via Bocha | `findo bocha "<query>" --json` |
| Web-grounded direct answer via Volcengine Ark | `findo volc "<query>" --json` |
| Zhihu in-site content search | `findo zhihu "<query>" --json` |
| Zhihu-backed global web search | `findo zhihu web "<query>" --json` |
| Current Zhihu hotlist | `findo hot zhihu --json` |
| Inspect available source IDs | `findo sources --json` |

Use `--json` for agent workflows. Use table or Markdown for direct human display.

Use `--limit` to constrain cost and noise:

```bash
findo zhihu "AI 搜索" --json --limit 5
findo bocha "AI Agent 商业化" --json --limit 5
findo volc "AI 搜索是什么" --json --limit 1
```

Zhihu global search supports provider-specific filters:

```bash
findo zhihu web "ChatGPT 桌面版" \
  --filter 'host=="example.com"' \
  --search-db realtime \
  --json
```

`--filter` and `--search-db` are only valid for `findo zhihu web`.

## Research Workflow

1. Convert the user's question into 2-5 concrete search queries. Include Chinese terms, product names, aliases, and time qualifiers when useful.
2. Start narrow. Use Zhihu for opinion-rich Chinese discussions, Bocha for broad web evidence, Volcengine for a synthesized answer, and Zhihu hotlist for current attention.
3. Keep result count small first: use `--limit 3` to `--limit 5`. Increase only when the first pass lacks coverage or confidence.
4. Inspect the JSON envelope:
   - `status`: overall command status.
   - `results`: normalized result list.
   - `source_status`: provider-level status, result count, effective limit, duration, and error.
   - `errors`: structured failures.
5. Cross-check claims across sources before presenting them as facts. Separate facts, source claims, and agent inference.
6. Report useful URLs, titles, source names, and timestamps when available. Do not overquote provider content.

## Output Format For Answers

For research answers, use this compact structure:

```markdown
**结论**
一句话回答用户问题。

**证据**
- Source: title, URL, key point.
- Source: title, URL, key point.

**判断**
What is likely true, what is uncertain, and what should be checked next.
```

For topic discovery, use:

```markdown
**候选话题**
1. Topic: why it is interesting, source signal, suggested angle.
2. Topic: why it is interesting, source signal, suggested angle.

**反直觉点**
What contradicts common assumptions.

**下一步**
The exact follow-up searches or validation steps.
```

## Failure Handling

Map failures to action with explicit branches:

| If this happens | First action | If still unresolved |
| --- | --- | --- |
| Exit `4` or `credential_missing` | Check `findo config show --json` for redacted provider fields. | Ask for or configure the relevant key; do not dump config. |
| Exit `2` | Remove incompatible flags and check source-specific usage. | Re-run the simplest valid command for that provider. |
| Exit `5` | Retry once with `--limit 1` to rule out payload or provider instability. | Switch provider and report the failed provider as a confidence limit. |
| Exit `6` | Re-run with a smaller limit and a narrower query. | Use another provider or ask whether to spend more time on a live retry. |
| Exit `7` | Rewrite the query with alternate Chinese terms and aliases. | Broaden to Bocha or Zhihu global search. |
| `sources --json` shows static config state | Treat it as source inventory, not readiness validation. | Run one real provider smoke call with `--limit 1`. |

When one provider fails, continue with other configured providers if the user asked for research rather than provider debugging. Preserve the failure in the final answer when it affects confidence.

## Do Not Do

These anti-patterns create noise, leak secrets, or break automation:

- Do not print `.env`, API keys, bearer tokens, raw config files, or CI secrets.
- Do not scrape browsers or websites when a configured Findo provider can answer the task.
- Do not send `--filter` or `--search-db` to commands other than `findo zhihu web`.
- Do not treat `findo sources --json` as proof that credentials are configured.
- Do not present a single provider's summary as verified fact when other sources contradict it.
- Do not hide provider failures if they reduce confidence in the answer.
- Do not invent source IDs, output fields, exit codes, or commands that the installed `findo version` does not support.
- Do not expand a narrow user question into broad trend research unless the user asks for topic discovery.

## Development Work In This Repository

When modifying Findo itself, use the repository contract instead of inventing new behavior:

```bash
make test
make release-check
make version-check
```

Before changing CLI behavior, read the relevant files:

- `README.md` for user-facing contract.
- `docs/specs/v1.0.0/` for design intent.
- `docs/specs/v1.2.0/` for embedded Agent skill commands.
- `internal/cli/cli.go` for actual command parsing and exit codes.
- `internal/config/config.go` for configuration precedence and secret handling.

🛑 STOP: Before changing CLI behavior, update or confirm the relevant spec and tests. The CLI is the product API; accidental command drift creates downstream agent failures.

Keep changes narrow and Agent Native:

- Results go to stdout; diagnostics go to stderr.
- JSON schemas, source IDs, and exit codes are compatibility commitments.
- Do not add browser scraping, hidden global state, caches, background daemons, or broad plugin systems unless the product spec explicitly adds them.
- Add tests for command contracts, config precedence, provider adapters, output envelopes, and release/version checks.
