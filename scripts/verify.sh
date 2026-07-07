#!/usr/bin/env bash
# Local verification: unit tests, property tests, then the acceptance pipeline.
# Property tests run as their own explicit command (behind the `property` build
# tag) so they stay out of unit coverage, mutation, and CRAP.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
"$ROOT/scripts/test.sh"
"$ROOT/scripts/property.sh"
"$ROOT/scripts/acceptance.sh"
