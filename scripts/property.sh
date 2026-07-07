#!/usr/bin/env bash
# Run the property tests. These live behind the `property` build tag so they
# stay out of the normal unit suite, coverage, mutation, and CRAP runs; this is
# the explicit command that exercises them.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
. "$ROOT/scripts/goenv.sh"
cd "$ROOT"
go test -tags property -run 'TestProp' ./...
