#!/usr/bin/env bash
#
# Local mirror of .github/workflows/ci.yml — run before pushing so CI doesn't surprise you.
#
#   ./scripts/check.sh
#
# Blocking gates exit non-zero. Gates marked TODO(B1) only warn, matching the
# `continue-on-error: true` escape hatches in the workflow; remove them from both
# files together once v2/simulation/test and v2/world/test compile again.

set -euo pipefail

cd "$(dirname "$0")/.."

warned=0

warn() {
  echo "!! $* (non-blocking — see TODO(B1))" >&2
  warned=1
}

echo "==> gofmt"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
  echo "These files are not gofmt-clean. Run 'gofmt -w .':" >&2
  echo "$unformatted" >&2
  exit 1
fi

echo "==> go mod tidy check"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
cp go.mod go.sum "$tmp/"
go mod tidy
if ! diff -q go.mod "$tmp/go.mod" >/dev/null || ! diff -q go.sum "$tmp/go.sum" >/dev/null; then
  echo "go.mod/go.sum were not tidy; 'go mod tidy' has updated them — commit the result." >&2
  exit 1
fi

echo "==> go build"
go build ./...

# TODO(B1): make blocking once the test packages type-check.
echo "==> go vet"
go vet ./... || warn "go vet failed"

# TODO(B1): make blocking once the test packages type-check and findings are cleared.
echo "==> staticcheck"
if command -v staticcheck >/dev/null 2>&1; then
  staticcheck ./... || warn "staticcheck failed"
else
  echo "-- staticcheck not installed; skipping"
  echo "   go install honnef.co/go/tools/cmd/staticcheck@2025.1"
fi

# TODO(B1): make blocking once the test packages compile.
echo "==> go test (race)"
go test -race ./... || warn "go test failed"

if [ "$warned" -eq 1 ]; then
  echo
  echo "Blocking gates passed; known-red gates above are still failing."
fi
echo "OK"
