#!/usr/bin/env bash
set -euo pipefail

# upgrade-crush.sh — Sync Esquisse Crush integration (skills + hooks) to the
# user's global Crush configuration directory.
#
# What this script OVERWRITES:
#   - All skill directories under ${CRUSH_SKILLS_DIR}/  (default: ~/.config/crush/skills/)
#   - ${CRUSH_HOOKS_DIR}/protect-adversarial.sh
#
# What this script does NOT touch:
#   - crush.json (MCP server registration) — use install-crush.sh for that
#   - The esquisse-mcp binary — use install-crush.sh to rebuild/reinstall it
#
# When to use this vs install-crush.sh:
#   - After pulling new skill/hook content from the Esquisse repo, run THIS script
#     to push the updated files to your global Crush config directory.
#   - Use install-crush.sh for first-time setup or when you need to update
#     crush.json or rebuild the MCP binary.

# ---------------------------------------------------------------------------
# Platform guard — Windows native shells are not supported
# ---------------------------------------------------------------------------
case "$(uname -s 2>/dev/null)" in
    MINGW*|CYGWIN*|MSYS*) echo "ERROR: Windows not supported. Use WSL." >&2; exit 1 ;;
esac

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ESQUISSE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DRY_RUN=false

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Sync Esquisse skills and hooks to the global Crush configuration directory.
Does NOT modify crush.json or the esquisse-mcp binary.

Options:
  --target-dir DIR   Override the Esquisse repo root (default: parent of scripts/)
  --dry-run          Print what would change without writing any files
  -h, --help         Show this help text and exit
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --target-dir)
            [[ -z "${2:-}" ]] && { echo "ERROR: --target-dir requires an argument" >&2; exit 1; }
            ESQUISSE_DIR="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "ERROR: unknown option: $1" >&2
            usage >&2
            exit 1
            ;;
    esac
done

# ---------------------------------------------------------------------------
# Resolve target directories
# ---------------------------------------------------------------------------
# IMPORTANT: honor env vars if already set in the caller's environment.
CRUSH_CONFIG_DIR="${CRUSH_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/crush}"
CRUSH_SKILLS_DIR="${CRUSH_SKILLS_DIR:-${CRUSH_CONFIG_DIR}/skills}"
CRUSH_HOOKS_DIR="${CRUSH_HOOKS_DIR:-${CRUSH_CONFIG_DIR}/hooks}"

# ---------------------------------------------------------------------------
# Sanity checks
# ---------------------------------------------------------------------------
if [[ ! -d "${ESQUISSE_DIR}/skills" ]]; then
    echo "ERROR: skills directory not found: ${ESQUISSE_DIR}/skills" >&2
    echo "  Is --target-dir pointing to the Esquisse repo root?" >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# Skills copy
# SYNC: same logic in init.sh and install-crush.sh
# ---------------------------------------------------------------------------
if [[ "$DRY_RUN" == false ]]; then
    mkdir -p "$CRUSH_SKILLS_DIR" || { echo "ERROR: cannot create $CRUSH_SKILLS_DIR" >&2; exit 1; }
    shopt -s nullglob
    for src in "${ESQUISSE_DIR}/skills"/*/; do
        skill_name="$(basename "$src")"
        dst="${CRUSH_SKILLS_DIR}/${skill_name}"
        rm -rf "${dst}.tmp"
        cp -RL "$src" "${dst}.tmp" || { echo "ERROR: cp failed for $skill_name" >&2; rm -rf "${dst}.tmp"; exit 1; }
        rm -rf "$dst"
        mv "${dst}.tmp" "$dst" || { echo "ERROR: mv failed for $skill_name" >&2; exit 1; }
        echo "  updated  ${CRUSH_SKILLS_DIR}/${skill_name}/"
    done
else
    echo "[dry-run] skills that would be copied to ${CRUSH_SKILLS_DIR}:"
    shopt -s nullglob
    for src in "${ESQUISSE_DIR}/skills"/*/; do
        echo "  $(basename "$src")"
    done
fi
# Note: shopt -s nullglob is now active for the remainder of this script.

# ---------------------------------------------------------------------------
# Hooks copy
# ---------------------------------------------------------------------------
if [[ "$DRY_RUN" == false ]]; then
    mkdir -p "$CRUSH_HOOKS_DIR" || { echo "ERROR: cannot create $CRUSH_HOOKS_DIR" >&2; exit 1; }
    src="${ESQUISSE_DIR}/scripts/hooks/protect-adversarial.sh"
    dst="${CRUSH_HOOKS_DIR}/protect-adversarial.sh"
    if [[ -f "$src" ]]; then
        cp "$src" "$dst" || { echo "ERROR: cp failed for protect-adversarial.sh" >&2; exit 1; }
        chmod +x "$dst" || { echo "ERROR: chmod failed for protect-adversarial.sh" >&2; exit 1; }
        echo "  updated  ${CRUSH_HOOKS_DIR}/protect-adversarial.sh"
    else
        echo "  WARNING: $src not found — hook not updated" >&2
    fi
else
    echo "[dry-run] hook that would be copied: ${CRUSH_HOOKS_DIR}/protect-adversarial.sh"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
if [[ "$DRY_RUN" == false ]]; then
    _skills=( "${CRUSH_SKILLS_DIR}"/*/ )
    _skill_count="${#_skills[@]}"
    echo ""
    echo "Crush integration updated."
    echo ""
    echo "  Skills:  ${CRUSH_SKILLS_DIR}/  (${_skill_count} dirs)"
    echo "  Hooks:   ${CRUSH_HOOKS_DIR}/protect-adversarial.sh"
    echo ""
    echo "To update the MCP binary or crush.json entry, re-run:"
    echo "  bash scripts/install-crush.sh"
fi
