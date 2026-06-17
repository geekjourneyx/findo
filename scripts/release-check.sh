#!/usr/bin/env bash
set -euo pipefail

test "${VERSION:-1.0.0}" = "1.0.0"
test -f package.json
test -f scripts/install.js
test -f scripts/run.js
test "$(node -p "require('./package.json').version")" = "${VERSION:-1.0.0}"
test "$(node -p "require('./package.json').name")" = "@geekjourneyx/findo"
grep -q "v1.0.0" CHANGELOG.md
grep -q "findo" README.md
grep -q "BOCHA_API_KEY" README.md
grep -q "npm install -g @geekjourneyx/findo" README.md
grep -q "FINDO_RELEASE_BASE_URL" scripts/install.js
grep -q "npm publish --access public" .github/workflows/release.yml
grep -q "NPM_TOKEN" .github/workflows/release.yml
test -z "$(gofmt -l .)"
npm_config_cache="${TMPDIR:-/tmp}/findo-npm-cache" npm pack --json --dry-run >/dev/null
CGO_ENABLED=0 GOFLAGS="-buildvcs=false" go test -count=1 ./...
GOFLAGS="-buildvcs=false" go vet ./...
GOFLAGS="-buildvcs=false" go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0 run
