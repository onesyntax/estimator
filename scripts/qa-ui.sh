#!/usr/bin/env bash
# End-to-end UI QA suite (qa/ui_pipeline_qa_suite.md), executed through the
# estimationd two-stage user interface only. Launches estimationd in
# deterministic mock mode, drives the HTML screens with the UI QA runner (which
# submits the on-screen forms and asserts on the rendered screen), and tears the
# server down. The runner speaks the user interface only — it never calls the
# JSON API and imports no project package — so QA never uses a private API into
# the project.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
. "$ROOT/scripts/goenv.sh"
cd "$ROOT"

HOST=127.0.0.1
PORT="${QA_UI_PORT:-8138}"
BASE="http://$HOST:$PORT"
BIN="$ROOT/build/estimationd"

mkdir -p "$ROOT/build"
echo "[qa-ui] building estimationd"
go build -o "$BIN" ./cmd/estimationd

echo "[qa-ui] starting estimationd on $BASE (mock mode)"
"$BIN" --ai-provider=mock --addr="$HOST:$PORT" >"$ROOT/build/estimationd.qa-ui.log" 2>&1 &
SERVER_PID=$!
cleanup() {
  kill "$SERVER_PID" 2>/dev/null || true
  wait "$SERVER_PID" 2>/dev/null || true
}
trap cleanup EXIT

echo "[qa-ui] running UI QA suite"
set +e
QA_BASE_URL="$BASE" go run ./qa/uirunner
status=$?
set -e

if [ "$status" -ne 0 ]; then
  echo "[qa-ui] server log:"
  sed 's/^/[qa-ui]   /' "$ROOT/build/estimationd.qa-ui.log" || true
fi
exit "$status"
