#!/usr/bin/env bash
# Local verification: unit tests then the acceptance pipeline.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
"$ROOT/scripts/test.sh"
"$ROOT/scripts/acceptance.sh"
