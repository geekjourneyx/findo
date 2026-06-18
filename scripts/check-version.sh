#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "version check failed: $*" >&2
  exit 1
}

normalize_version() {
  local value="$1"
  value="${value#v}"
  value="${value#[}"
  value="${value%]}"
  printf '%s' "$value"
}

package_version="$(node -p "require('./package.json').version")"

expected="${1:-}"
if [[ -z "$expected" && -n "${GITHUB_REF_NAME:-}" ]]; then
  expected="$GITHUB_REF_NAME"
fi
if [[ -z "$expected" && -n "${VERSION:-}" ]]; then
  expected="$VERSION"
fi
if [[ -z "$expected" ]]; then
  expected="$package_version"
fi
expected="$(normalize_version "$expected")"

[[ "$expected" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]] || fail "invalid semver: $expected"

[[ "$package_version" == "$expected" ]] || fail "package.json version is $package_version, expected $expected"

makefile_version="$(awk '/^VERSION[[:space:]]*\?=/{print $3; exit}' Makefile)"
[[ -n "$makefile_version" ]] || fail "Makefile VERSION is missing"
[[ "$makefile_version" == "$expected" ]] || fail "Makefile VERSION is $makefile_version, expected $expected"

main_version="$(awk -F\" '/var version = /{print $2; exit}' cmd/findo/main.go)"
[[ -n "$main_version" ]] || fail "cmd/findo/main.go version is missing"
[[ "$main_version" == "$expected" ]] || fail "cmd/findo/main.go version is $main_version, expected $expected"

changelog_version="$(awk '/^## /{print $2; exit}' CHANGELOG.md)"
[[ -n "$changelog_version" ]] || fail "CHANGELOG.md has no version heading"
changelog_version="$(normalize_version "$changelog_version")"
[[ "$changelog_version" == "$expected" ]] || fail "top CHANGELOG version is $changelog_version, expected $expected"

if [[ -n "${GITHUB_REF_NAME:-}" ]]; then
  [[ "$GITHUB_REF_NAME" == "v$expected" ]] || fail "git ref is $GITHUB_REF_NAME, expected v$expected"
fi

echo "version check ok: $expected"
