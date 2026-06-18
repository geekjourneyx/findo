#!/usr/bin/env bash
set -euo pipefail

bash scripts/check-version.sh
test -f package.json
test -f skills/findo/SKILL.md
test -f internal/skillcontent/skills/findo/SKILL.md
test -f docs/specs/v1.2.0/01-embedded-skills.md
test -f scripts/install.js
test -f scripts/run.js
test "$(node -p "require('./package.json').name")" = "@geekjourneyx/findo"
grep -q "findo" README.md
grep -q "BOCHA_API_KEY" README.md
grep -q "npm install -g @geekjourneyx/findo" README.md
grep -q "findo skills list --json" README.md
grep -q "findo skills read findo --json" README.md
grep -q "findo skills read findo --json" skills/findo/SKILL.md
cmp -s skills/findo/SKILL.md internal/skillcontent/skills/findo/SKILL.md
cmp -s skills/findo/test-prompts.json internal/skillcontent/skills/findo/test-prompts.json
grep -q "findo skills read findo --json" docs/specs/v1.2.0/01-embedded-skills.md
grep -q "FINDO_RELEASE_BASE_URL" scripts/install.js
grep -q "npm publish --access public" .github/workflows/release.yml
grep -q "NPM_TOKEN" .github/workflows/release.yml
test -z "$(gofmt -l .)"
npm_config_cache="${TMPDIR:-/tmp}/findo-npm-cache" npm pack --json --dry-run >/dev/null
CGO_ENABLED=0 GOFLAGS="-buildvcs=false" go test -count=1 ./...
GOFLAGS="-buildvcs=false" go vet ./...
GOFLAGS="-buildvcs=false" go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0 run
