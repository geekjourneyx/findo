# Findo v1.0.0 Specs

This directory turns `prd.md` into implementation-facing specs.

The reference project is `/root/go/src/industry-research/researcher`. It already implements Bocha direct search, Volcengine Ark Responses API with web search, config loading, provider error mapping, multi-provider partial failure, and release scripts. Findo should reuse the proven ideas, not the researcher product semantics.

## Spec Index

- `01-cli-contract.md`: command names, flags, env vars, output modes, exit codes.
- `02-architecture.md`: Go package layout, source interfaces, orchestration, error model.
- `03-source-bocha.md`: Bocha provider spec based on the reference implementation.
- `04-source-volcengine.md`: Volcengine provider spec based on the reference implementation.
- `05-source-zhihu.md`: Zhihu official API source specs.
- `06-testing-release.md`: unit, contract, e2e, smoke, CI, release gates.

## Scope of This First Batch

This first batch covers the foundation, the two source families already proven in the reference project, and the official Zhihu API details supplied for Findo:

- Bocha web search
- Volcengine Ark web-enhanced answer
- Zhihu in-site content search
- Zhihu global web search
- Zhihu hotlist
- CLI/output/error/config contracts needed by supported sources
- Test and release closure

Zhida direct answer and Bocha image search are deferred; they are not part of the v1.0.0 public command contract.

## Locked Product Decisions

1. Public version starts at `v1.0.0`.
2. Default human output is `table`.
3. Automation output is `--json`.
4. Markdown is for research notes.
5. Partial success is a first-class state, not a hidden warning.
6. Source adapters are compiled into the binary; there is no plugin runtime.
7. Provider-specific metadata stays internal unless it is useful in `--raw` or diagnostics.
8. The first implementation should keep dependencies small. The reference project only needs `gopkg.in/yaml.v3`.

## Source Notes

The source-specific specs are canonical. These notes only explain why the v1.0.0 contracts preserve a few behaviors from the reference project and official Zhihu API docs.

Bocha:

- Bocha uses `POST https://api.bocha.cn/v1/web-search`.
- Auth uses `Authorization: Bearer <BOCHA_API_KEY>`.
- Provider code may be string, number, or absent; normalize before mapping errors.
- `dateLastCrawled` ending in `Z` was normalized to `+08:00` in the reference implementation.
- HTTP 429 maps to retryable rate limit.
- HTTP 401 maps to auth failure.
- HTTP 403 can mean quota or billing failure.

Volcengine:

- Volcengine uses Ark Responses API: `POST https://ark.cn-beijing.volces.com/api/v3/responses`.
- Auth uses `Authorization: Bearer <ARK_API_KEY>`.
- The default model in the reference project is `doubao-seed-2-0-lite-260215`.
- A successful HTTP response is not enough; the response must contain a `web_search_call`.
- URL citations are parsed from `output[].content[].annotations[]`.
- HTTP 429 maps to retryable rate limit.
- HTTP 401/403 maps to auth or missing web search permission.

Multi-source:

- Multi-source calls should preserve source order in output.
- One provider failure should not erase successful provider results.
- All-provider failure is a different state from partial failure.

Zhihu:

- Zhihu in-site search uses `GET https://developer.zhihu.com/api/v1/content/zhihu_search`.
- Zhihu global search uses `GET https://developer.zhihu.com/api/v1/content/global_search`.
- Zhihu hotlist uses `GET https://developer.zhihu.com/api/v1/content/hot_list`.
- Auth uses `Authorization: Bearer <access_secret>`.
- Requests must include `X-Request-Timestamp` with a second-level Unix timestamp.
- `Count` defaults to 10, is capped at 10, and values `<= 0` fall back to 10 server-side.
- Global search `Count` is capped at 20 and supports `Filter` plus `SearchDB`.
- Hotlist `Limit` defaults to 30 and is capped at 30.
- Provider error codes include `10001`, `20001`, `30001`, and `90001`.

## Non-Goals for Specs

These specs should not describe:

- MCP
- scheduled jobs
- cache
- reranking
- plugin runtime
- browser scraping
- report generation
- researcher workspaces

Those belong after the search contract is stable.
