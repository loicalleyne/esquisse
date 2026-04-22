# P2-012: Eliminate Bare python3 Calls from Bash Scripts

## Status
Status: Done
Depends on: none
Blocks: none

## Summary

`scripts/gate-review.sh` and `scripts/upgrade.sh` invoke bare `python3 -c` to
parse JSON state files. The project convention mandates that any Python execution
use `uv` in a virtual environment. Rather than wrapping the JSON stdlib calls in
a `uv run` harness (which requires venv bootstrapping inside bash scripts), this
task replaces them with `jq` — the idiomatic JSON processor for shell scripts —
with a grep/sed fallback for environments where `jq` is also absent. The result
is zero Python in the scripts, satisfying the constraint by elimination.

## Problem

Both scripts call `python3 -c "import json; …"` to read fields from
`.adversarial/*.json` state files. The calls use only Python stdlib (`json`);
no third-party packages are required. But bare `python3` is banned by the
project convention — all Python must run under `uv` in a virtual environment.
Wrapping stdlib-only one-liners in `uv venv` + `uv run` adds significant
bash complexity (venv path management, activation, cleanup) for zero functional
gain, since `jq` performs the same operation natively in shell without a
runtime or venv.

## Solution

Replace every `python3 -c` invocation in `gate-review.sh` and `upgrade.sh`
with `jq -r` expressions. Add a `jq` availability check at the top of each
affected script. Preserve the existing grep/sed fallback path (already present
in `gate-review.sh`) as the second fallback when `jq` is also unavailable.
After the replacement, zero Python remains in either script; the uv constraint
is satisfied by elimination.

## Scope

### In Scope
- `scripts/gate-review.sh` — replace two `python3 -c` blocks (`.last_verdict`,
  `.iteration`) with `jq -r` equivalents; add `jq` availability check.
- `scripts/upgrade.sh` — replace four `python3 -c` blocks (`.last_verdict`,
  `.plan`, JSON write for migration, JSON write for deprecation stub) with
  `jq` equivalents; add `jq` availability check.
- `scripts/upgrade.sh` — the two JSON-write blocks (migrating state.json and
  writing the deprecation stub) cannot use `jq` for output; replace them with
  `printf` heredocs producing the JSON directly in bash — no Python needed.
- Update the comment on line 193 of `upgrade.sh` that says
  `# Read fields using python3 (required by gate-review.sh anyway)` since
  neither script will require python3 after this task.
- Update `upgrade.sh` header comment: remove the `python3` availability
  warning text; replace with `jq` availability warning.

### Out of Scope
- `scripts/gate-check.sh` — does not call python3; no change.
- `scripts/rebuild-ast.sh` — does not call python3; no change.
- `esquisse-mcp/` — Go binary; no Python.
- Adding `jq` as a hard dependency. Availability check must degrade gracefully:
  if `jq` absent, fall through to the existing grep/sed path.
- Installing `uv` or creating a venv anywhere.
- Any Python code outside `scripts/`.

## Prerequisites
- [ ] `jq` is available in the test environment (WSL/Ubuntu: `sudo apt-get install jq`)
- [ ] Both scripts pass their current functional behavior before the change
  (verify by running `bash scripts/gate-review.sh` in a project with a
  valid `.adversarial/*.json` file)

## Specification

### gate-review.sh changes

**Current pattern (lines 87–108):**
```bash
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
```

**Replacement pattern:**
```bash
if command -v jq &>/dev/null; then
    LAST_VERDICT="$(jq -r '.last_verdict // ""' "$STATE_FILE" 2>/dev/null || echo "")"
    ITER="$(jq -r '.iteration // 0' "$STATE_FILE" 2>/dev/null || echo "0")"
else
    LAST_VERDICT="$(grep -o '"last_verdict": *"[^"]*"' "$STATE_FILE" 2>/dev/null \
        | sed 's/.*"last_verdict": *"\([^"]*\)"/\1/' || echo "")"
    ITER="$(grep -o '"iteration": *[0-9]*' "$STATE_FILE" 2>/dev/null \
        | sed 's/.*"iteration": *\([0-9]*\)/\1/' || echo "0")"
fi
```

### upgrade.sh changes

**Read `.last_verdict` and `.plan` from OLD_STATE:**

Replace:
```bash
if ! command -v python3 &>/dev/null; then
    echo "  WARNING: python3 not found. Cannot auto-migrate ${OLD_STATE}." >&2
    …
else
    LAST_VERDICT="$(python3 -c "…" 2>/dev/null || echo "")"
    PLAN_PATH="$(python3 -c "…" 2>/dev/null || echo "")"
```

With:
```bash
if ! command -v jq &>/dev/null; then
    echo "  WARNING: jq not found. Cannot auto-migrate ${OLD_STATE}." >&2
    echo "  Install jq (e.g. sudo apt-get install jq) and re-run, or migrate manually." >&2
    echo "  See SCHEMAS.md §8 for the required per-plan file format." >&2
else
    LAST_VERDICT="$(jq -r '.last_verdict // .verdict // ""' "$OLD_STATE" 2>/dev/null || echo "")"
    PLAN_PATH="$(jq -r '.plan // ""' "$OLD_STATE" 2>/dev/null || echo "")"
```

**Write migrated JSON (replace `python3 -c "import json…"` write block):**

The migration write produces a JSON object that merges `plan_slug` into the
existing state. Replace with `jq` merge:
```bash
jq --arg slug "$SLUG" '{plan_slug: $slug} + (
    if has("verdict") and (has("last_verdict") | not)
    then del(.verdict) + {last_verdict: .verdict}
    else .
    end
)' "$OLD_STATE" > "$NEW_STATE" 2>/dev/null
```

Exact `jq` expression to use (handles the `verdict` → `last_verdict` rename):
```bash
jq --arg slug "$SLUG" \
   'if has("verdict") and (has("last_verdict") | not)
    then {plan_slug: $slug} + (del(.verdict) + {last_verdict: .verdict})
    else {plan_slug: $slug} + .
    end' "$OLD_STATE" > "$NEW_STATE"
```

**Write deprecation stub (replace second `python3 -c` write block):**

Replace with a `printf` heredoc — no tool dependency:
```bash
printf '{\n  "plan_slug": "_deprecated_migrated",\n  "iteration": 0,\n  "last_model": "upgrade.sh",\n  "last_verdict": "PASSED",\n  "last_review_date": "%s",\n  "_note": "Deprecated by upgrade.sh. Superseded by %s. Safe to delete."\n}\n' \
    "$(date +%Y-%m-%d)" "$NEW_STATE" > "$OLD_STATE"
```

### Known Risks / Failure Modes

| Risk | Mitigation |
|------|-----------|
| `jq` absent in target env | Fallback to grep/sed for reads; printf for writes. Warn clearly. |
| `jq` expression produces `null` literal instead of empty string | Use `// ""` (alternative operator) in all `jq -r` reads |
| JSON key rename (`verdict` → `last_verdict`) in jq pipeline | Covered by the explicit `if has("verdict")` branch; tested in AC |
| `.adversarial/` absent when gate-review.sh runs | Existing early-exit guard unchanged; jq not reached |

## Acceptance Criteria

All criteria verified by running the scripts in a WSL/Linux shell.

### gate-review.sh

```sh
# AC1: With jq present and valid state file — reads verdict correctly
echo '{"plan_slug":"test","last_verdict":"PASSED","iteration":3}' \
    > /tmp/test-state.json
# Manually invoke the parsing block with STATE_FILE=/tmp/test-state.json
# Expected: LAST_VERDICT=PASSED, ITER=3

# AC2: With jq absent (PATH without jq) — grep/sed fallback produces same result
PATH_WITHOUT_JQ="$(echo "$PATH" | tr ':' '\n' | grep -v "$(dirname "$(which jq)")" | tr '\n' ':')"
# Run the block with modified PATH — expected: same PASSED / 3

# AC3: Full script run in a project with a PASSED state file exits 0
bash scripts/gate-review.sh   # exit code 0

# AC4: Full script run with a FAILED state file exits non-zero
# (behavioral parity with pre-change)
```

### upgrade.sh

```sh
# AC5: Migration of a legacy state.json with 'verdict' key produces correct output
echo '{"verdict":"CONDITIONAL","iteration":2,"plan":"docs/specs/foo.md"}' \
    > /tmp/.adversarial-test/state.json
bash scripts/upgrade.sh --target-dir /tmp/test-project
# Expected: /tmp/.adversarial-test/foo.json contains last_verdict=CONDITIONAL
#           and plan_slug=foo

# AC6: Deprecation stub written with correct fields (no python3 invoked)
# Expected: /tmp/.adversarial-test/state.json contains _deprecated_migrated plan_slug
#           and _note field

# AC7: upgrade.sh completes without invoking python3
# Verify: strace or bash -x output contains no 'python3' invocations

# AC8: With jq absent — warning printed, migration skipped gracefully, exit 0
```

### Regression

```sh
# AC9: grep -rn "python3" scripts/gate-review.sh scripts/upgrade.sh → 0 matches
grep -rn "python3" scripts/gate-review.sh scripts/upgrade.sh
# Expected: no output
```

## Files

| File | Action | What changes |
|------|--------|-------------|
| `scripts/gate-review.sh` | Modify | Replace `python3 -c` blocks with `jq -r`; add jq/grep-sed branch |
| `scripts/upgrade.sh` | Modify | Replace four `python3 -c` blocks; update availability check comment; use printf for JSON writes |

## Session Notes

<!-- Agent: append timestamped notes here during implementation. -->

2026-04-22 — EsquissePlan — Created. The JSON-write blocks in upgrade.sh that
produce structured output are replaced with printf (not jq) because jq is a JSON
reader, not a template engine. The read blocks use jq. This is the correct split.
The `uv` constraint is satisfied by eliminating Python entirely; no venv is needed.
