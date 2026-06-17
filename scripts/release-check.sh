#!/usr/bin/env bash
set -euo pipefail

test "${VERSION:-1.0.0}" = "1.0.0"
grep -q "v1.0.0" CHANGELOG.md
grep -q "findo" README.md
grep -q "BOCHA_API_KEY" README.md
test -z "$(gofmt -l .)"
CGO_ENABLED=0 GOFLAGS="-buildvcs=false" go test -count=1 ./...
GOFLAGS="-buildvcs=false" go vet ./...
GOFLAGS="-buildvcs=false" go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0 run
