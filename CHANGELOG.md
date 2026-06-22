# Changelog

## v2.0.2

- Add Unix-style `--help`, `-h`, command-specific help, and `--version`.
- Restore generic `tanso <query>` and `tanso all <query>` retrieval commands.
- Accept `--timeout`, `--source`, `--no-color`, and `--verbose` according to the public CLI contract.
- Make human table/Markdown failure paths print diagnostics to `stderr`.
- Fix source-specific flag validation when global flags appear before `zhihu web`.
- Make `tanso sources --json` report local credential presence after config/env resolution.
- Clarify that `configured` is inventory metadata, not a live provider authentication check.
- Return `NO_RESULTS` when all successful sources return zero results.
- Add regression tests for agent-friendly command discovery, generic retrieval, source credential metadata, and config errors.

## v2.0.1

- Use a stable default config path across platforms: `~/.config/tanso/config.yaml`.
- Support `TANSO_CONFIG` as the config file path override for `config init`, `config path`, `config show`, and retrieval commands.
- Document macOS/Linux and Windows config paths explicitly.

## v2.0.0

- Rename the project, Go module, npm package, CLI command, release artifacts, and Agent skill to Tanso.
- Update documentation and visuals around the Tanso positioning: exploring Chinese internet signals through an AI Search CLI.
- Add `tanso zhihu hot` as the primary Zhihu hotlist command while keeping the old command order compatible.
- Rename the internal error package to `tansoerr` and update release, npm, and skill packaging paths.

## v1.2.3

- Keep `skills/tanso` as pure Agent Skill content and remove Go code from `skills/`.
- Load bundled skill content through `internal/skillcontent` using the repository or npm package `skills/` directory.
- Ship `skills/tanso` in npm packages and GitHub Release archives, and smoke-test `tanso skills read` from npm installs.
- Simplify `skills/tanso/SKILL.md` into a concise English Agent SOP.

## v1.2.2

- Remove the duplicated internal Agent skill copy.
- Move toward a single root Agent skill source after removing the mirrored internal copy.
- Update release checks to guard against mirrored skill files.

## v1.2.1

- Move embedded skill implementation under `internal/skillcontent` instead of exposing it from the module root.
- Add release checks and tests to prevent drift between the root Agent skill and the embedded copy.

## v1.2.0

- Embed the Tanso Agent skill in the release binary.
- Add `tanso skills list/read` so agents can discover and read the current-version SOP without relying on stale external docs.
- Extend release checks to guard embedded skill documentation and version consistency.

## v1.1.0

- Add default config initialization, path discovery, and redacted config inspection.

## v1.0.0

- Initial stable CLI contract.
- Add Bocha web search, Volcengine web-grounded answer, and Zhihu search/hotlist adapters.
- Add JSON envelope, source status, stable error codes, and release gates.
- Add GitHub Actions CI and tag-triggered cross-platform release builds.
- Add npm global installer package for the matching GitHub Release binary.
