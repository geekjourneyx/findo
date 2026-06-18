# Changelog

## v1.2.0

- Embed the Findo Agent skill in the release binary.
- Add `findo skills list/read` so agents can discover and read the current-version SOP without relying on stale external docs.
- Extend release checks to guard embedded skill documentation and version consistency.

## v1.1.0

- Add default config initialization, path discovery, and redacted config inspection.

## v1.0.0

- Initial stable CLI contract.
- Add Bocha web search, Volcengine web-grounded answer, and Zhihu search/hotlist adapters.
- Add JSON envelope, source status, stable error codes, and release gates.
- Add GitHub Actions CI and tag-triggered cross-platform release builds.
- Add npm global installer package for the matching GitHub Release binary.
