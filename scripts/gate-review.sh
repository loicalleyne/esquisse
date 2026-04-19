#!/bin/bash
# =============================================================================
# gate-review.sh — Adversarial review enforcement Stop hook.
#
# Called by VS Code as an agent Stop hook. Scans .adversarial/*.json (one file
# per plan) and blocks if any active plan has a missing or FAILED verdict.
# Concurrent planning sessions are supported: each plan has its own state file.
#
# IMPORTANT: This script must run under Linux/WSL bash, not Windows cmd/PowerShell.
# Hook commands in hooks.json and agent frontmatter use "bash ./scripts/..." to
# ensure VS Code invokes WSL bash on Windows.
#
# Usage (automatic — called by VS Code Stop hook):
#   bash ./scripts/gate-review.sh [--strict]
#
# Options:
#   --strict    Block if no state files exist (use in EsquissePlan agent-scoped hook).
#               Default (no flag): fast-pass if no state files — used by the
#               global .github/hooks/hooks.json fallback so ordinary sessions
#               are not interrupted.
#
# Exit codes:
#   0   Session may exit normally.
#   2   Session blocked — adversarial review required or verdict is FAILED.
#
# State files: .adversarial/{plan-slug}.json  (one per plan)
# Canonical schema: SCHEMAS.md §8 (source of truth — read that before writing manually).
#   {
#     "plan_slug":        "<string>",          // matches filename without .json
#     "iteration":        <int>,
#     "last_model":       "<string>",
#     "last_verdict":     "PASSED|CONDITIONAL|FAILED",
#     "last_review_date": "YYYY-MM-DD"
#   }
# WARNING: field names are exact. "verdict" != "last_verdict". Any rename breaks this hook.
# =============================================================================

set -euo pipefail

# ── Platform guard ─────────────────────────────────────────────────────────────────────
if [[ "$(uname -s)" == MINGW* || "$(uname -s)" == CYGWIN* || "$(uname -s)" == MSYS* ]]; then
    echo "ERROR: gate-review.sh must run under Linux or WSL, not Windows Git Bash/MSYS." >&2
    echo "Ensure VS Code default terminal profile is set to WSL." >&2
    exit 1
fi

STRICT=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        --strict) STRICT=true; shift ;;
        *)        shift ;;
    esac
done

# Read the Stop hook input from stdin.
INPUT="$(cat)"

# ── Infinite-loop guard ───────────────────────────────────────────────────────
if echo "$INPUT" | grep -q '"stop_hook_active": *true'; then
    printf '{"continue": true}\n'
    exit 0
fi

ADVER_DIR=".adversarial"

# ── No state files at all ─────────────────────────────────────────────────────
# Collect JSON files at top level only (not reports/).
mapfile -t STATE_FILES < <(find "$ADVER_DIR" -maxdepth 1 -name "*.json" 2>/dev/null | sort)

if [[ ${#STATE_FILES[@]} -eq 0 ]]; then
    if [[ "$STRICT" == "true" ]]; then
        echo "Adversarial review required before this session can exit." >&2
        echo "Dispatch @Adversarial-r0 (or let EsquissePlan dispatch automatically via slot rotation)." >&2
        exit 2
    else
        # Global fallback mode: no planning review in progress — fast-pass.
        printf '{"continue": true}\n'
        exit 0
    fi
fi

# ── Parse verdicts from all state files ──────────────────────────────────────
# Collect any blocking plans; exit 0 only if ALL are PASSED or CONDITIONAL.
BLOCKING=()

for STATE_FILE in "${STATE_FILES[@]}"; do
    if command -v python3 &>/dev/null; then
        LAST_VERDICT="$(python3 -c "
import json
try:
    d = json.load(open('$STATE_FILE'))
    print(d.get('last_verdict', ''))
except Exception:
    print('')
" 2>/dev/null || echo "")"
        ITER="$(python3 -c "
import json
try:
    d = json.load(open('$STATE_FILE'))
    print(d.get('iteration', 0))
except Exception:
    print(0)
" 2>/dev/null || echo "0")"
    else
        LAST_VERDICT="$(grep -o '"last_verdict": *"[^"]*"' "$STATE_FILE" 2>/dev/null \
            | sed 's/.*"last_verdict": *"\([^"]*\)"/\1/' || echo "")"
        ITER="$(grep -o '"iteration": *[0-9]*' "$STATE_FILE" 2>/dev/null \
            | sed 's/.*"iteration": *\([0-9]*\)/\1/' || echo "0")"
    fi

    if [[ "$LAST_VERDICT" != "PASSED" && "$LAST_VERDICT" != "CONDITIONAL" ]]; then
        SLUG="$(basename "$STATE_FILE" .json)"
        SLOT=$(( ITER % 3 ))
        BLOCKING+=("  • ${SLUG}: verdict='${LAST_VERDICT:-absent}' — next reviewer: Adversarial-r${SLOT} (iteration ${ITER})")
    fi
done

# ── Evaluate ──────────────────────────────────────────────────────────────────
if [[ ${#BLOCKING[@]} -eq 0 ]]; then
    printf '{"continue": true}\n'
    exit 0
else
    echo "Adversarial review gate FAILED. The following plans require review:" >&2
    for line in "${BLOCKING[@]}"; do
        echo "$line" >&2
    done
    echo "Revise each blocked plan and resubmit to its next reviewer." >&2
    exit 2
fi
