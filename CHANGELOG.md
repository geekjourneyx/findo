# Changelog

## v1.2.3

- Keep `skills/findo` as pure Agent Skill content and remove Go code from `skills/`.
- Load bundled skill content through `internal/skillcontent` using the repository or npm package `skills/` directory.
- Ship `skills/findo` in npm packages and GitHub Release archives, and smoke-test `findo skills read` from npm installs.
- Simplify `skills/findo/SKILL.md` into a concise English Agent SOP.

## v1.2.2

- Remove the duplicated internal Agent skill copy.
- Move toward a single root Agent skill source after removing the mirrored internal copy.
- Update release checks to guard against mirrored skill files.

## v1.2.1

- Move embedded skill implementation under `internal/skillcontent` instead of exposing it from the module root.
- Add release checks and tests to prevent drift between the root Agent skill and the embedded copy.

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
