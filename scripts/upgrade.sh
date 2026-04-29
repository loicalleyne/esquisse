#!/bin/bash
# =============================================================================
# upgrade.sh — Upgrade an existing Esquisse-initialized project to the current
#              Esquisse version.
#
# Safe to run on any project that was bootstrapped with init.sh. Overwrites
# only Esquisse-managed infrastructure files. Never touches:
#   - AGENTS.md, ONBOARDING.md, GLOSSARY.md, docs/planning/NEXT_STEPS.md
#   - docs/tasks/*, docs/planning/*, docs/adr/*
#   - .github/hooks/hooks.json (project may have customised it)
#
# What gets overwritten (Esquisse infrastructure — not user-authored):
#   - docs/artifacts/ directory (created if absent, contents never touched)
#   - scripts/gate-review.sh
#   - scripts/gate-check.sh
#   - scripts/new-task.sh
#   - scripts/rebuild-ast.sh
#   - scripts/macros.sql, scripts/macros_go.sql
#   - scripts/hooks/protect-adversarial.sh  (Crush PreToolUse hook)
#   - ~/.copilot/agents/EsquissePlan.agent.md  (user-level — shared across projects)
#   - ~/.copilot/agents/Adversarial-r0.agent.md  (user-level)
#   - ~/.copilot/agents/Adversarial-r1.agent.md  (user-level)
#   - ~/.copilot/agents/Adversarial-r2.agent.md  (user-level)
#   - ~/.copilot/skills/*/  (all Esquisse skills — user-level global VS Code install)
#   - skills/adversarial-review/  (full overwrite)
#   - skills/write-spec/          (full overwrite)
#   - skills/explore-codebase/    (full overwrite)
#   - skills/implement-task/      (full overwrite)
#
# Migration performed automatically (in addition to state.json above):
#   - .github/agents/EsquissePlan.agent.md removed from target project
#   - .github/agents/Adversarial-r*.agent.md removed from target project
#     (all four agents are now user-level; per-project copies cause duplicates
#      in the VS Code agent selector)
#   - skills/adversarial-review/  (full directory overwrite)
#   - docs/SCHEMAS.md
#
# Migration performed automatically:
#   - .adversarial/state.json (old single-plan schema) → per-plan .json files
#     See SCHEMAS.md §8 for schema details.
#
# IMPORTANT: This script must run under Linux/WSL bash.
#
# Usage:
#   bash /path/to/esquisse/scripts/upgrade.sh [--target-dir DIR]
#
# Defaults: target dir = current working directory.
# =============================================================================

set -euo pipefail

# ── Platform guard ────────────────────────────────────────────────────────────
if [[ "$(uname -s)" == MINGW* || "$(uname -s)" == CYGWIN* || "$(uname -s)" == MSYS* ]]; then
    echo "ERROR: upgrade.sh must run under Linux or WSL." >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ESQUISSE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
TARGET_DIR="$(pwd)"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --target-dir) TARGET_DIR="$2"; shift 2 ;;
        -h|--help)
            echo "Usage: bash upgrade.sh [--target-dir DIR]"
            exit 0 ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

TARGET_DIR="$(cd "$TARGET_DIR" && pwd)"
cd "$TARGET_DIR"

echo "Upgrading Esquisse project at: $TARGET_DIR"
echo "Esquisse source:               $ESQUISSE_DIR"
echo ""

# ── Helper: overwrite a file unconditionally ──────────────────────────────────
overwrite_file() {
    local src="$1"
    local dst="$2"
    local label="$3"
    if [[ -f "$src" ]]; then
        cp "$src" "$dst"
        echo "  updated  $label"
    else
        echo "  missing  $label  (not found in Esquisse dir — skipped)"
    fi
}

# ── 1. Scripts ────────────────────────────────────────────────────────────────
echo "Scripts:"
mkdir -p scripts
mkdir -p scripts/hooks
mkdir -p docs/artifacts
echo "  ensured  docs/artifacts/"
for item in gate-review.sh gate-check.sh new-task.sh rebuild-ast.sh macros.sql macros_go.sql; do
    src="${SCRIPT_DIR}/${item}"
    dst="${TARGET_DIR}/scripts/${item}"
    if [[ -f "$src" ]]; then
        cp "$src" "$dst"
        [[ "$item" == *.sh ]] && chmod +x "$dst"
        echo "  updated  scripts/${item}"
    else
        echo "  missing  scripts/${item}  (not found in Esquisse dir — skipped)"
    fi
done

# Crush PreToolUse hook
src="${SCRIPT_DIR}/hooks/protect-adversarial.sh"
dst="${TARGET_DIR}/scripts/hooks/protect-adversarial.sh"
if [[ -f "$src" ]]; then
    cp "$src" "$dst"
    chmod +x "$dst"
    echo "  updated  scripts/hooks/protect-adversarial.sh"
else
    echo "  missing  scripts/hooks/protect-adversarial.sh  (not found in Esquisse dir — skipped)"
fi

# ── 2. Agent files ────────────────────────────────────────────────────────────
echo ""
echo "Agent files:"

# All four planning agents — user-level (shared across all Esquisse projects).
# Resolve ~/.copilot/agents: WSL uses Windows USERPROFILE, Linux/macOS use HOME.
if command -v cmd.exe &>/dev/null 2>&1; then
    _win_home="$(wslpath "$(cmd.exe /c 'echo %USERPROFILE%' 2>/dev/null | tr -d '\r')")";
    COPILOT_AGENTS_DIR="${_win_home}/.copilot/agents"
else
    COPILOT_AGENTS_DIR="${HOME}/.copilot/agents"
fi
mkdir -p "$COPILOT_AGENTS_DIR"

for agent in EsquissePlan.agent.md Adversarial-r0.agent.md Adversarial-r1.agent.md Adversarial-r2.agent.md; do
    overwrite_file \
        "${ESQUISSE_DIR}/.github/agents/${agent}" \
        "${COPILOT_AGENTS_DIR}/${agent}" \
        "~/.copilot/agents/${agent}  (user-level)"
done

# Remove any per-project agent copies (migration: per-project → user-level).
for agent in EsquissePlan.agent.md Adversarial-r0.agent.md Adversarial-r1.agent.md Adversarial-r2.agent.md; do
    per_project="${TARGET_DIR}/.github/agents/${agent}"
    if [[ -f "$per_project" ]]; then
        rm "$per_project"
        echo "  removed  .github/agents/${agent}  (superseded by user-level copy)"
    fi
done

# ── 3. Adversarial review skill ───────────────────────────────────────────────
echo ""
echo "Adversarial review skill:"
src="${ESQUISSE_DIR}/skills/adversarial-review"
dst="${TARGET_DIR}/skills/adversarial-review"
if [[ -d "$src" ]]; then
    rm -rf "$dst"
    cp -r "$src" "$dst"
    echo "  updated  skills/adversarial-review/"
else
    echo "  missing  skills/adversarial-review/  (not found in Esquisse dir — skipped)"
fi

# ── 3b. Global VS Code skills (~/.copilot/skills/) ────────────────────────────
echo ""
echo "VS Code global skills (~/.copilot/skills/):"
if command -v cmd.exe &>/dev/null 2>&1; then
    _win_home_skills="$(wslpath "$(cmd.exe /c 'echo %USERPROFILE%' 2>/dev/null | tr -d '\r')")"
    COPILOT_SKILLS_DIR="${_win_home_skills}/.copilot/skills"
else
    COPILOT_SKILLS_DIR="${HOME}/.copilot/skills"
fi
mkdir -p "$COPILOT_SKILLS_DIR"
shopt -s nullglob
for src_skill in "${ESQUISSE_DIR}/skills"/*/; do
    skill_name="$(basename "$src_skill")"
    dst_skill="${COPILOT_SKILLS_DIR}/${skill_name}"
    rm -rf "${dst_skill}.tmp"
    cp -RL "$src_skill" "${dst_skill}.tmp" || {
        echo "  ERROR: cp failed for $skill_name" >&2
        rm -rf "${dst_skill}.tmp"
        exit 1
    }
    rm -rf "$dst_skill"
    mv "${dst_skill}.tmp" "$dst_skill"
    echo "  updated  ~/.copilot/skills/${skill_name}/"
done
# Note: shopt -s nullglob remains active for the rest of the script.

# ── 4. Other skills (Batch 1): write-spec, explore-codebase, implement-task ───
echo ""
echo "Workflow skills (Batch 1):"
for skill_dir in write-spec explore-codebase implement-task; do
    src="${ESQUISSE_DIR}/skills/${skill_dir}"
    dst="${TARGET_DIR}/skills/${skill_dir}"
    if [[ -d "$src" ]]; then
        rm -rf "$dst"
        cp -r "$src" "$dst"
        echo "  updated  skills/${skill_dir}/"
    else
        echo "  missing  skills/${skill_dir}/  (not found in Esquisse dir — skipped)"
    fi
done

# ── 5. docs/SCHEMAS.md ────────────────────────────────────────────────────────
echo ""
echo "Framework docs:"
overwrite_file \
    "${ESQUISSE_DIR}/SCHEMAS.md" \
    "${TARGET_DIR}/docs/SCHEMAS.md" \
    "docs/SCHEMAS.md"
# ── 5b. Ensure docs/planning/NEXT_STEPS.md exists (create or migrate root copy) ─────
echo ""
echo "Session state:"
mkdir -p "${TARGET_DIR}/docs/planning"
NEXT_STEPS_CANONICAL="${TARGET_DIR}/docs/planning/NEXT_STEPS.md"
NEXT_STEPS_ROOT="${TARGET_DIR}/NEXT_STEPS.md"
if [[ -f "$NEXT_STEPS_CANONICAL" ]]; then
    echo "  exists   docs/planning/NEXT_STEPS.md  (not modified)"
elif [[ -f "$NEXT_STEPS_ROOT" ]]; then
    mv "$NEXT_STEPS_ROOT" "$NEXT_STEPS_CANONICAL"
    echo "  migrated NEXT_STEPS.md → docs/planning/NEXT_STEPS.md"
else
    printf '# NEXT_STEPS.md\n\n## Active Task\n\nNone.\n\n## Session Notes\n\n<!-- Agent: append a timestamped bullet after each session. -->\n\n## Blocked Items\n\nNone.\n\n## Deferred Work\n\nNone.\n' \
        > "$NEXT_STEPS_CANONICAL"
    echo "  created  docs/planning/NEXT_STEPS.md"
fi
# ── 6. Migrate .adversarial/state.json → per-plan files ──────────────────────
echo ""
echo "Adversarial state migration:"
OLD_STATE=".adversarial/state.json"

if [[ ! -f "$OLD_STATE" ]]; then
    echo "  no state.json found — nothing to migrate."
else
    if ! command -v jq &>/dev/null; then
        echo "  WARNING: jq not found. Cannot auto-migrate ${OLD_STATE}." >&2
        echo "  Install jq (e.g. sudo apt-get install jq) and re-run, or migrate manually." >&2
        echo "  See SCHEMAS.md §8 for the required per-plan file format." >&2
    else
        LAST_VERDICT="$(jq -r '.last_verdict // .verdict // ""' "$OLD_STATE" 2>/dev/null || echo "")"
        PLAN_PATH="$(jq -r '.plan // ""' "$OLD_STATE" 2>/dev/null || echo "")"

        # Derive kebab-case slug from plan path, or fall back to "migrated-state".
        SLUG="$(echo "$PLAN_PATH" | xargs -I{} basename {} .md | tr '[:upper:]' '[:lower:]' | tr ' ' '-')"
        [[ -z "$SLUG" ]] && SLUG="migrated-state"

        NEW_STATE=".adversarial/${SLUG}.json"

        if [[ -f "$NEW_STATE" ]]; then
            echo "  per-plan file already exists: ${NEW_STATE} — skipping migration of ${OLD_STATE}."
            echo "  Rename or delete ${OLD_STATE} manually after verifying."
        else
            # Add plan_slug field to the existing JSON and write new file.
            # Handle legacy 'verdict' → 'last_verdict' rename if needed.
            # Write to a temp file first to avoid creating a 0-byte $NEW_STATE on jq failure,
            # which would permanently block future migration attempts.
            TMP_STATE="${NEW_STATE}.tmp"
            if jq --arg slug "$SLUG" \
                'if has("verdict") and (has("last_verdict") | not)
                 then {plan_slug: $slug} + (del(.verdict) + {last_verdict: .verdict})
                 else {plan_slug: $slug} + .
                 end' "$OLD_STATE" > "$TMP_STATE" 2>/dev/null; then
                mv "$TMP_STATE" "$NEW_STATE"
                echo "  migrated  ${OLD_STATE} → ${NEW_STATE}  (verdict: ${LAST_VERDICT:-unknown})"
                # Mark old file as deprecated stub so gate-review.sh won't double-count it.
                printf '{\n  "plan_slug": "_deprecated_migrated",\n  "iteration": 0,\n  "last_model": "upgrade.sh",\n  "last_verdict": "PASSED",\n  "last_review_date": "%s",\n  "_note": "Deprecated by upgrade.sh. Superseded by %s. Safe to delete."\n}\n' \
                    "$(date +%Y-%m-%d)" "$NEW_STATE" > "$OLD_STATE" \
                    && echo "  stubbed   ${OLD_STATE}  (safe to delete)"
            else
                rm -f "$TMP_STATE"
                echo "  ERROR: migration failed — check ${OLD_STATE} manually against SCHEMAS.md §8." >&2
            fi
        fi
    fi
fi

echo ""
echo "Upgrade complete."
echo ""
echo "Next steps:"
echo "  1. Review any CONDITIONAL state files in .adversarial/ and resolve open issues."
echo "  2. Delete .adversarial/state.json once you have confirmed the gate passes."
echo "  3. Run 'bash scripts/gate-review.sh' to verify the hook passes locally."
