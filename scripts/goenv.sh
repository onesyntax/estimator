#!/usr/bin/env bash
# Shared Go environment for this project. Keeps all caches inside the worktree
# so build/test commands never write outside the project tree.
# Source this file: `. scripts/goenv.sh`
set -a
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export GOCACHE="$ROOT/.gocache"
export GOMODCACHE="$ROOT/.gomodcache"
export GOFLAGS="-mod=mod"
set +a
