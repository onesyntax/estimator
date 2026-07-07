#!/usr/bin/env bash
# Soft Gherkin acceptance mutation: mutate each feature's example values with the
# APS gherkin-mutator and prove the acceptance tests detect the changes. Uses the
# project runner adapter (cmd/acceptance-mutation-runner) as the mutator's
# persistent worker. Kept separate from unit, acceptance, and language-mutation
# runs.
#
# Level is soft by default (reuse clean scenarios across implementation-hash
# changes); pass a different level as the first argument.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
. "$ROOT/scripts/goenv.sh"
cd "$ROOT"

LEVEL="${1:-soft}"
APS_DIR="$ROOT/.aps"
APS_REPO="https://github.com/unclebob/Acceptance-Pipeline-Specification.git"
RUNNER="$ROOT/build/acceptance-mutation-runner"
WORKERS="${MUTATION_WORKERS:-8}"
STATUS_INTERVAL="${MUTATION_STATUS_INTERVAL:-10s}"

# Procure / refresh the APS tooling (Babashka gherkin-mutator).
if [ ! -d "$APS_DIR/.git" ]; then
  echo "[acceptance-mutation] cloning APS tooling into .aps"
  git clone --depth 1 "$APS_REPO" "$APS_DIR"
else
  echo "[acceptance-mutation] refreshing APS tooling in .aps"
  git -C "$APS_DIR" pull --ff-only --depth 1 >/dev/null 2>&1 || \
    echo "[acceptance-mutation] offline; using existing .aps checkout"
fi

# Build the persistent runner adapter once.
echo "[acceptance-mutation] building runner adapter"
mkdir -p "$ROOT/build"
go build -o "$RUNNER" ./cmd/acceptance-mutation-runner

shopt -s nullglob
features=("$ROOT"/features/*.feature)
if [ ${#features[@]} -eq 0 ]; then
  echo "[acceptance-mutation] no feature files found under features/"
  exit 1
fi

status=0
for feature in "${features[@]}"; do
  rel="features/$(basename "$feature")"
  base="$(basename "${feature%.feature}")"
  echo "[acceptance-mutation] mutating $rel (level=$LEVEL)"
  if ! bb --config "$APS_DIR/bb.edn" gherkin-mutator \
      --feature "$rel" \
      --level "$LEVEL" \
      --runner-worker "$RUNNER" \
      --workers "$WORKERS" \
      --status-interval "$STATUS_INTERVAL" \
      --work-dir "build/acceptance-mutation/$base"; then
    status=1
    echo "[acceptance-mutation] survivors or errors in $rel"
  fi
done

exit "$status"
