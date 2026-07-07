#!/usr/bin/env bash
# Run the acceptance pipeline: parse each feature to JSON IR with the
# APS-supplied gherkin-parser, generate acceptance entry points, then run the
# generated executable tests. Steps run sequentially (parse, generate, run).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
. "$ROOT/scripts/goenv.sh"

APS_DIR="$ROOT/.aps"
APS_REPO="https://github.com/unclebob/Acceptance-Pipeline-Specification.git"
BUILD="$ROOT/build/acceptance"
GEN="$ROOT/acceptance/generated"

# Procure / refresh the APS gherkin-parser (Babashka task).
if [ ! -d "$APS_DIR/.git" ]; then
  echo "[acceptance] cloning APS tooling into .aps"
  git clone --depth 1 "$APS_REPO" "$APS_DIR"
else
  echo "[acceptance] refreshing APS tooling in .aps"
  git -C "$APS_DIR" pull --ff-only --depth 1 >/dev/null 2>&1 || \
    echo "[acceptance] offline; using existing .aps checkout"
fi

# Clean previous generation.
mkdir -p "$BUILD"
rm -f "$BUILD"/*.json
rm -f "$GEN"/*_gen.go
rm -rf "$GEN/metadata"

shopt -s nullglob
features=("$ROOT"/features/*.feature)
if [ ${#features[@]} -eq 0 ]; then
  echo "[acceptance] no feature files found under features/"
  exit 1
fi

for feature in "${features[@]}"; do
  base="$(basename "${feature%.feature}")"
  ir="$BUILD/$base.json"
  rel="features/$(basename "$feature")"
  echo "[acceptance] parsing $rel"
  ( cd "$APS_DIR" && bb gherkin-parser "$feature" "$ir" )
  echo "[acceptance] generating entry point for $rel"
  ACCEPTANCE_FEATURE_PATH="$rel" \
    go run ./cmd/acceptance-entrypoint-generator "$ir" "$GEN"
done

echo "[acceptance] running generated acceptance tests"
( cd "$ROOT" && go run ./acceptance/generated )
