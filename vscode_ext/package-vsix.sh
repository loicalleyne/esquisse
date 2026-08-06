#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

VSCE_ARGS=(--allow-missing-repository "$@")

if command -v vsce >/dev/null 2>&1; then
  echo "Using local vsce binary"
  vsce package "${VSCE_ARGS[@]}"
  exit 0
fi

echo "vsce not found on PATH; using npx @vscode/vsce"
npx --yes @vscode/vsce package "${VSCE_ARGS[@]}"