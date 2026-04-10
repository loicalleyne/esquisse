#!/bin/bash
# =============================================================================
# new-task.sh — Esquisse task document scaffolder.
#
# Creates the next task document for a given phase under docs/tasks/.
# The sequence number is auto-incremented from the highest existing file in
# that phase. The file is populated with the task document minimal template.
#
# Usage:
#   bash scripts/new-task.sh <phase> <slug>
#
# Examples:
#   bash scripts/new-task.sh 1 kafka-ingester
#   bash scripts/new-task.sh 2 parquet-export
#
# Output:
#   docs/tasks/P1-003-kafka-ingester.md
# =============================================================================

set -euo pipefail

readonly SCRIPT_NAME="$(basename "$0")"
readonly TASKS_DIR="docs/tasks"

# ── Usage ─────────────────────────────────────────────────────────────────────
usage() {
    echo "Usage: $SCRIPT_NAME <phase> <slug>"
    echo ""
    echo "  phase   Integer phase number (e.g. 1, 2, 3)"
    echo "  slug    Short kebab-case identifier (e.g. kafka-ingester)"
    echo ""
    echo "Examples:"
    echo "  $SCRIPT_NAME 1 kafka-ingester"
    echo "  $SCRIPT_NAME 2 parquet-export"
}

if [[ $# -ne 2 ]]; then
    usage >&2
    exit 1
fi

PHASE="$1"
SLUG="$2"

# ── Validate inputs ───────────────────────────────────────────────────────────
if ! [[ "$PHASE" =~ ^[0-9]+$ ]]; then
    echo "Error: phase must be a non-negative integer, got: '$PHASE'" >&2
    exit 1
fi

if ! [[ "$SLUG" =~ ^[a-z0-9][a-z0-9-]*$ ]]; then
    echo "Error: slug must be lowercase alphanumeric with hyphens (kebab-case), got: '$SLUG'" >&2
    exit 1
fi

# ── Ensure tasks dir exists ───────────────────────────────────────────────────
if [[ ! -d "$TASKS_DIR" ]]; then
    mkdir -p "$TASKS_DIR"
fi

# ── Auto-sequence ─────────────────────────────────────────────────────────────
# Find highest existing sequence number for this phase, then increment.
LAST_SEQ=0

while IFS= read -r -d '' f; do
    # Extract sequence number from filename e.g. P1-003-*.md → 3
    fname="$(basename "$f")"
    if [[ "$fname" =~ ^P${PHASE}-([0-9]+)- ]]; then
        seq_val="${BASH_REMATCH[1]}"
        # Strip leading zeros for arithmetic.
        seq_int=$((10#$seq_val))
        if [[ $seq_int -gt $LAST_SEQ ]]; then
            LAST_SEQ=$seq_int
        fi
    fi
done < <(find "$TASKS_DIR" -maxdepth 1 -name "P${PHASE}-*.md" -print0 2>/dev/null)

NEXT_SEQ=$(( LAST_SEQ + 1 ))
# Zero-pad to three digits.
SEQ_PAD="$(printf '%03d' "$NEXT_SEQ")"

FILENAME="P${PHASE}-${SEQ_PAD}-${SLUG}.md"
FILEPATH="${TASKS_DIR}/${FILENAME}"

if [[ -f "$FILEPATH" ]]; then
    echo "Error: file already exists: $FILEPATH" >&2
    exit 1
fi

# ── Today's date (ISO 8601) ───────────────────────────────────────────────────
TODAY="$(date +%Y-%m-%d)"

# ── Write task document ───────────────────────────────────────────────────────
cat > "$FILEPATH" <<EOF
# ${FILENAME%.md}

| Field | Value |
|-------|-------|
| Phase | P${PHASE} |
| Seq | ${SEQ_PAD} |
| Slug | ${SLUG} |
| Status | Not Started |
| Created | ${TODAY} |
| Owner | |

## Goal

TODO: One sentence — what does this task produce?

## Background

TODO: Why is this task necessary? What problem does it solve?

## Acceptance Criteria

- [ ] TODO: Write as test function names or runnable assertions.
- [ ] All tests pass: \`go test -v -run TestXxx ./...\`
- [ ] No \`panic("TODO\` stubs remain.
- [ ] Coverage ≥ 80% for changed packages.

## Scope

**In scope:**
- TODO

**Out of scope:**
- TODO

## Interface Sketch

\`\`\`go
// TODO: Stub the types and function signatures that this task must implement.
// Follow the agent-facing interface design rules in AGENTS.md.
\`\`\`

### Interface-as-Prompt Checklist

- [ ] Type names state what they are, not how they are implemented.
- [ ] Function names include the verb (Fetch, Write, Validate — not Get/Do/Handle).
- [ ] Parameters are named after their constraint, not their type.
- [ ] Every error return is actionable (the caller knows what to retry or fix).
- [ ] Invalid states are unrepresentable (use types, not runtime checks, as the first line of defence).

## Implementation Notes

TODO: Library constraints, performance requirements, known gotchas.

## Functions / Methods

| Symbol | Package | Responsibility | Recovery on failure |
|--------|---------|----------------|---------------------|
| TODO() | TODO | | |

## Test Plan

\`\`\`sh
go test -v -count=1 -run TestXxx ./...
\`\`\`

## Session Notes

<!-- Agent: append a timestamped bullet after each work session on this task. -->

## Assumptions

<!-- Record any ASSUMPTION: comments created during implementation here. -->

## Definition of Done

- [ ] Acceptance criteria above all checked.
- [ ] \`AGENTS.md\` Common Mistakes updated if any new gotchas were found.
- [ ] \`GLOSSARY.md\` updated for any new domain terms introduced.
- [ ] \`NEXT_STEPS.md\` updated to reflect next task.
- [ ] \`llms.txt\` / \`llms-full.txt\` updated if public API changed.
- [ ] PR / Handoff document written (see SCHEMAS.md §7).
EOF

echo "Created: $FILEPATH"
