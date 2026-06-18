---
name: findo
description: >-
  Use Findo, an Agent Native Go CLI for Chinese internet research. Use when the user asks to search Chinese web sources, Zhihu, Zhihu hotlists, Bocha, Volcengine Ark, Chinese internet topics, source-backed briefs, contradiction checks, counterintuitive topic discovery, popular answers, live provider testing, or current Chinese web signals.
---

# Findo

Use `findo` to retrieve Chinese internet sources through provider APIs and return source-backed answers. Prefer `--json` for agent workflows.

## Before Running

Check availability:

```bash
findo version
findo config show --json
```

If `findo` is missing and the user asked for setup, install it:

```bash
npm install -g @geekjourneyx/findo
```

Never print `.env`, API keys, bearer tokens, raw config files, or CI secrets. Treat redacted config values as expected.

## Choose A Source

Use the narrowest source that matches the task:

| Need | Command |
| --- | --- |
| Broad web evidence | `findo bocha "<query>" --json --limit 5` |
| Web-grounded direct answer | `findo volc "<query>" --json --limit 1` |
| Zhihu opinions and discussions | `findo zhihu "<query>" --json --limit 5` |
| Zhihu-backed global web search | `findo zhihu web "<query>" --json --limit 5` |
| Current Zhihu hotlist | `findo hot zhihu --json` |
| Available source IDs | `findo sources --json` |

Use Zhihu for opinion-rich questions, Bocha for broad corroboration, Volcengine for synthesized direct answers, and hotlist for current attention.

For Zhihu global search only, filters are valid:

```bash
findo zhihu web "<query>" --filter 'host=="example.com"' --search-db realtime --json
```

## Research Workflow

1. Rewrite the user request into 2-5 concrete Chinese queries with aliases, product names, and time qualifiers when useful.
2. Start with `--limit 3` to `--limit 5`; increase only when coverage is weak.
3. Inspect `status`, `results`, `source_status`, and `errors` in JSON output.
4. Cross-check important claims across sources before presenting them as facts.
5. Separate source claims from your inference. Mention provider failures when they reduce confidence.

## Answer Shape

For research answers:

```markdown
**Conclusion**
One concise answer.

**Evidence**
- Source, title, URL, key point.
- Source, title, URL, key point.

**Assessment**
What is likely true, what is uncertain, and what to verify next.
```

For topic discovery:

```markdown
**Topic Candidates**
1. Topic: why it matters, source signal, suggested angle.
2. Topic: why it matters, source signal, suggested angle.

**Counterintuitive Point**
What contradicts common assumptions.

**Next Step**
Exact follow-up searches or validation steps.
```

## Failure Handling

| Signal | Action |
| --- | --- |
| `credential_missing` or exit `4` | Check redacted config; ask for the relevant provider key if needed. |
| Exit `2` | Remove incompatible flags and rerun the simplest valid command. |
| Exit `5` | Retry once with a smaller limit, then switch provider. |
| Exit `6` | Narrow the query and reduce `--limit`. |
| Exit `7` | Rewrite with alternate Chinese terms or broaden to Bocha/Zhihu global search. |

When one provider fails, continue with configured alternatives if the user asked for research rather than debugging.

## Guardrails

- Do not scrape browsers or websites when a Findo provider can answer the task.
- Do not use `--filter` or `--search-db` outside `findo zhihu web`.
- Do not treat `findo sources --json` as proof that credentials work; it is only source inventory.
- Do not present one provider's output as verified fact when sources conflict.
- Do not expand a narrow question into broad trend research unless the user asks for topic discovery.
