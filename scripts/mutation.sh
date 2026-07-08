#!/usr/bin/env bash
# Language mutation hardening for the testable Go source. Runs the hardening
# test suite (behind the `hardening` build tag, kept out of the unit suite,
# coverage, CRAP, and DRY), then differential mutation against the embedded
# per-file manifests. Fails if any mutation survives or any covered site is
# uncovered.
#
# Mutation runs one file at a time with 8 workers. mutate4go exits 0 even with
# survivors, so this script parses each report and gates on it.
#
# Pass --mutate-all as the first argument to re-verify every mutation instead of
# only functions changed since the last manifest.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
. "$ROOT/scripts/goenv.sh"
cd "$ROOT"

EXTRA=()
if [ "${1:-}" = "--mutate-all" ]; then
  EXTRA+=(--mutate-all)
fi

if ! command -v mutate4go >/dev/null 2>&1; then
  echo "[mutation] mutate4go not found on PATH; install github.com/unclebob/mutate4go/cmd/mutate4go" >&2
  exit 1
fi

# Covered source files paired with the package test command that runs their
# unit and hardening tests. steps, generated, and the cmd shells are the
# acceptance boundary (hardened by soft Gherkin acceptance mutation instead).
files_internal="internal/wbs/wbs.go internal/wbs/service.go internal/wbs/provider.go internal/wbs/document.go internal/wbs/errors.go internal/wbs/risk.go internal/wbs/estimate.go internal/wbs/metrics.go internal/wbs/pricing.go internal/wbs/proposal.go"
files_aiprovider="internal/aiprovider/anthropic.go internal/aiprovider/anthropic_risk.go internal/aiprovider/anthropic_estimate.go"
files_httpapi="internal/httpapi/server.go"
files_runtime="acceptance/runtime/ir.go acceptance/runtime/run.go"
files_generator="acceptance/generator/generator.go"
files_runner="acceptance/mutationrunner/runner.go"

run_pkg() {
  local pkg="$1"; shift
  local tc="go test -tags hardening ./$pkg/"
  echo "[mutation] hardening tests: $pkg"
  go test -tags hardening "./$pkg/" >/dev/null
  for f in "$@"; do
    echo "[mutation] mutate $f"
    local out
    out="$(mutate4go "$f" --test-command "$tc" --max-workers 8 ${EXTRA[@]+"${EXTRA[@]}"})"
    local survived uncovered
    survived="$(printf '%s\n' "$out" | awk -F': ' '/^Survived:/{print $2}')"
    uncovered="$(printf '%s\n' "$out" | awk -F': ' '/^Uncovered:/{print $2}')"
    if [ "${survived:-0}" != "0" ] || [ "${uncovered:-0}" != "0" ]; then
      printf '%s\n' "$out" | sed -n '/Mutation Report/,$p'
      echo "[mutation] FAIL $f (survived=${survived:-?} uncovered=${uncovered:-?})" >&2
      exit 1
    fi
  done
}

run_pkg internal/wbs $files_internal
run_pkg internal/aiprovider $files_aiprovider
run_pkg internal/httpapi $files_httpapi
run_pkg acceptance/runtime $files_runtime
run_pkg acceptance/generator $files_generator
run_pkg acceptance/mutationrunner $files_runner

echo "[mutation] all covered files: no survivors, no uncovered sites"
