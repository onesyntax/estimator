#!/usr/bin/env bash
# End-to-end QA suite (qa/wbs_qa_suite.md), executed through the estimationd
# HTTP user interface only. Launches estimationd in deterministic mock mode,
# runs the HTTP QA runner against it, and tears the server down. The runner
# speaks HTTP only and imports no project package, so QA never uses a private
# API into the project.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
. "$ROOT/scripts/goenv.sh"
cd "$ROOT"

HOST=127.0.0.1
PORT="${QA_PORT:-8137}"
BASE="http://$HOST:$PORT"
BIN="$ROOT/build/estimationd"

mkdir -p "$ROOT/build"
echo "[qa] building estimationd"
go build -o "$BIN" ./cmd/estimationd

echo "[qa] starting estimationd on $BASE (mock mode)"
"$BIN" --ai-provider=mock --addr="$HOST:$PORT" >"$ROOT/build/estimationd.qa.log" 2>&1 &
SERVER_PID=$!
cleanup() {
  kill "$SERVER_PID" 2>/dev/null || true
  wait "$SERVER_PID" 2>/dev/null || true
}
trap cleanup EXIT

echo "[qa] running HTTP QA suite"
set +e
QA_BASE_URL="$BASE" go run ./qa/runner
status=$?
set -e

if [ "$status" -ne 0 ]; then
  echo "[qa] server log:"
  sed 's/^/[qa]   /' "$ROOT/build/estimationd.qa.log" || true
fi
exit "$status"
