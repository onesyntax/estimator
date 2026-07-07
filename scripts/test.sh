#!/usr/bin/env bash
# Run the Go unit tests (excludes generated acceptance tests, which have no
# _test.go files and are exercised via scripts/acceptance.sh).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
. "$ROOT/scripts/goenv.sh"
cd "$ROOT"
go test ./...
