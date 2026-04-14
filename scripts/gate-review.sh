#!/bin/bash
# =============================================================================
# gate-review.sh — Adversarial review enforcement Stop hook.
#
# Called by VS Code as an agent Stop hook. Reads .adversarial/state.json to
# check the last adversarial review verdict before allowing the session to exit.
#
# IMPORTANT: This script must run under Linux/WSL bash, not Windows cmd/PowerShell.
# Hook commands in hooks.json and agent frontmatter use "bash ./scripts/..." to
# ensure VS Code invokes WSL bash on Windows.
#
# Usage (automatic — called by VS Code Stop hook):
#   bash ./scripts/gate-review.sh [--strict]
#
# Options:
#   --strict    Block if no state file exists (use in EsquissePlan agent-scoped hook).
#               Default (no flag): fast-pass if no state file — used by the
#               global .github/hooks/hooks.json fallback so ordinary sessions
#               are not interrupted.
#
# Exit codes:
#   0   Session may exit normally.
#   2   Session blocked — adversarial review required or verdict is FAILED.
#
# State file: .adversarial/state.json
# Canonical schema: SCHEMAS.md §8 (source of truth — read that before writing manually).
#   {
#     "iteration":        <int>,
#     "last_model":       "<string>",
#     "last_verdict":     "PASSED|CONDITIONAL|FAILED",
#     "last_review_date": "YYYY-MM-DD"
#   }
# WARNING: field names are exact. "verdict" != "last_verdict". Any rename breaks this hook.
# =============================================================================

set -euo pipefail

# ── Platform guard ─────────────────────────────────────────────────────────────────────
# Hook scripts require Linux (native or WSL). Reject Windows-native execution.
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
# VS Code sets stop_hook_active:true when a previous hook already triggered
# continuation in this turn. Pass through immediately to prevent infinite loops.
if echo "$INPUT" | grep -q '"stop_hook_active": *true'; then
    printf '{"continue": true}\n'
    exit 0
fi

STATE_FILE=".adversarial/state.json"

# ── No state file ─────────────────────────────────────────────────────────────
if [[ ! -f "$STATE_FILE" ]]; then
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

# ── Parse verdict from state file ────────────────────────────────────────────
# Use python3 for reliable JSON parsing; fall back to grep if python3 absent.
if command -v python3 &>/dev/null; then
    LAST_VERDICT="$(python3 -c "
import json, sys
try:
    d = json.load(open('$STATE_FILE'))
    print(d.get('last_verdict', ''))
except Exception:
    print('')
" 2>/dev/null || echo "")"
    ITER="$(python3 -c "
import json, sys
try:
    d = json.load(open('$STATE_FILE'))
    print(d.get('iteration', 0))
except Exception:
    print(0)
" 2>/dev/null || echo "0")"
else
    # Fallback: grep for the last_verdict value.
    LAST_VERDICT="$(grep -o '"last_verdict": *"[^"]*"' "$STATE_FILE" 2>/dev/null \
        | sed 's/.*"last_verdict": *"\([^"]*\)"/\1/' || echo "")"
    ITER="$(grep -o '"iteration": *[0-9]*' "$STATE_FILE" 2>/dev/null \
        | sed 's/.*"iteration": *\([0-9]*\)/\1/' || echo "0")"
fi

# ── Evaluate verdict ──────────────────────────────────────────────────────────
if [[ "$LAST_VERDICT" == "PASSED" || "$LAST_VERDICT" == "CONDITIONAL" ]]; then
    printf '{"continue": true}\n'
    exit 0
else
    SLOT=$(( ITER % 3 ))
    echo "Adversarial review verdict is '${LAST_VERDICT:-absent}'." >&2
    echo "Revise the plan and resubmit. Next reviewer: Adversarial-r${SLOT} (iteration ${ITER})." >&2
    exit 2
fi
