#!/usr/bin/env bash
set -euo pipefail

case "$(uname -s 2>/dev/null)" in
    MINGW*|CYGWIN*|MSYS*) echo "ERROR: Windows not supported. Use WSL." >&2; exit 1 ;;
esac

usage() {
    cat >&2 <<EOF
Usage: $(basename "$0") [OPTIONS]

Wire Esquisse into an existing Crush installation.

Options:
  --crush-config PATH   Override auto-detected crush.json path
  --mcp-binary PATH     Path to esquisse-mcp binary (skips discovery)
  --target-dir DIR      Esquisse repo root (default: parent of this script)
  --dry-run             Print what would change without writing anything
  -h, --help            Show this help message
EOF
}

# Argument parsing
DRY_RUN=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        --crush-config) CRUSH_CONFIG="$2"; shift 2 ;;
        --mcp-binary)   MCP_BINARY="$2";   shift 2 ;;
        --target-dir)   ESQUISSE_DIR="$2"; shift 2 ;;
        --dry-run)      DRY_RUN=true;      shift   ;;
        -h|--help)      usage; exit 0 ;;
        *) echo "Unknown option: $1" >&2; usage; exit 1 ;;
    esac
done

# ESQUISSE_DIR defaults to the repo root (parent of scripts/) if not set by flag
ESQUISSE_DIR="${ESQUISSE_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && cd .. && pwd)}"

# Termux detection (informational)
if [[ -n "${TERMUX_VERSION:-}" ]]; then
    echo "Detected Termux ${TERMUX_VERSION} on Android."
    if ! command -v jq &>/dev/null; then
        echo "  HINT: install jq with: pkg install jq" >&2
        exit 1
    fi
fi

# jq sanity check
if ! command -v jq &>/dev/null; then
    echo "ERROR: jq is required but not found. Install jq and retry." >&2
    [[ -n "${TERMUX_VERSION:-}" ]] && echo "  Hint: pkg install jq" >&2
    cat >&2 <<'EOF'
Manual steps (without jq):
  Open crush.json and add under "mcp":
    "esquisse": {
      "type": "stdio",
      "command": "/path/to/esquisse-mcp",
      "args": ["--project-root", "."]
    }
EOF
    exit 1
fi
if ! jq --null-input '"ok"' &>/dev/null; then
    echo "ERROR: jq is not functional. Install jq and retry." >&2
    [[ -n "${TERMUX_VERSION:-}" ]] && echo "  Hint: pkg install jq" >&2
    exit 1
fi

# crush.json path detection (AFTER arg parsing so --crush-config is preserved)
CRUSH_CONFIG="${CRUSH_CONFIG:-${XDG_CONFIG_HOME:-$HOME/.config}/crush/crush.json}"
CRUSH_CONFIG_DIR="$(dirname "$CRUSH_CONFIG")"
if [[ "$DRY_RUN" == false ]]; then
    mkdir -p "$CRUSH_CONFIG_DIR"
    if [[ ! -f "$CRUSH_CONFIG" ]]; then
        printf '{}' > "$CRUSH_CONFIG"
        echo "  created  $CRUSH_CONFIG  (empty)"
    fi
else
    echo "[dry-run] crush.json path: $CRUSH_CONFIG"
fi

# Register cleanup trap BEFORE temp file could be created
trap 'rm -f "${CRUSH_CONFIG}.tmp"' ERR EXIT

# Binary discovery function
find_mcp_binary() {
    local _dir="$1"
    # 1. PATH
    if command -v esquisse-mcp &>/dev/null; then
        echo "$(command -v esquisse-mcp)"; return 0
    fi
    # 2. GOPATH/bin
    local gobin="${GOPATH:-$HOME/go}/bin/esquisse-mcp"
    if [[ -x "$gobin" ]]; then echo "$gobin"; return 0; fi
    # 3. Local build in repo
    local local_bin="${_dir}/esquisse-mcp/esquisse-mcp"
    if [[ -x "$local_bin" ]]; then echo "$local_bin"; return 0; fi
    return 1
}

# Binary discovery flow
if [[ -n "${MCP_BINARY:-}" ]]; then
    # --mcp-binary flag: verify it is executable
    [[ -x "$MCP_BINARY" ]] || { echo "ERROR: --mcp-binary $MCP_BINARY is not executable" >&2; exit 1; }
else
    MCP_BINARY="$(find_mcp_binary "$ESQUISSE_DIR" || true)"
    if [[ -z "$MCP_BINARY" ]]; then
        if command -v go &>/dev/null; then
            if [[ "$DRY_RUN" == false ]]; then
                echo "Building esquisse-mcp..." >&2
                CGO_ENABLED=0 go build -o "${ESQUISSE_DIR}/esquisse-mcp/esquisse-mcp" \
                    "${ESQUISSE_DIR}/esquisse-mcp" || {
                    echo "ERROR: go build failed." >&2
                    echo "  go install github.com/loicalleyne/esquisse-mcp@latest" >&2
                    exit 1
                }
                MCP_BINARY="${ESQUISSE_DIR}/esquisse-mcp/esquisse-mcp"
            else
                echo "[dry-run] would build esquisse-mcp (Go available)" >&2
                MCP_BINARY="${ESQUISSE_DIR}/esquisse-mcp/esquisse-mcp (would be built)"
            fi
        else
            echo "ERROR: esquisse-mcp not found and Go not available." >&2
            echo "  On Termux:  pkg install golang && cd ${ESQUISSE_DIR}/esquisse-mcp && go build -o esquisse-mcp ." >&2
            echo "  On other:   go install github.com/loicalleyne/esquisse-mcp@latest" >&2
            echo "  Or pass:    --mcp-binary /path/to/esquisse-mcp" >&2
            exit 1
        fi
    fi
fi
# Guard: MCP_BINARY must be non-empty
[[ -z "$MCP_BINARY" ]] && { echo "ERROR: MCP_BINARY is empty after discovery" >&2; exit 1; }
# Make absolute path (skip in dry-run when placeholder value may be set)
if [[ "$DRY_RUN" == false ]]; then
    [[ "$MCP_BINARY" != /* ]] && MCP_BINARY="$(cd "$(dirname "$MCP_BINARY")" && pwd)/$(basename "$MCP_BINARY")"
fi

# jq merge into crush.json
if [[ "$DRY_RUN" == false ]]; then
    jq --arg cmd "$MCP_BINARY" \
       '.mcp.esquisse = {"type":"stdio","command":$cmd,"args":["--project-root","."]}' \
       "$CRUSH_CONFIG" > "${CRUSH_CONFIG}.tmp"
    mv "${CRUSH_CONFIG}.tmp" "$CRUSH_CONFIG"
    echo "  updated  $CRUSH_CONFIG"
else
    echo "[dry-run] would write to: $CRUSH_CONFIG"
    # crush.json may not exist in dry-run mode — fall back to empty object
    _dry_input="$(cat "$CRUSH_CONFIG" 2>/dev/null || echo '{}')"
    jq --arg cmd "$MCP_BINARY" \
       '.mcp.esquisse = {"type":"stdio","command":$cmd,"args":["--project-root","."]}' \
       <<< "$_dry_input"
fi

# Skills copy
# SYNC: same logic in init.sh and upgrade-crush.sh
CRUSH_SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/crush/skills"
if [[ "$DRY_RUN" == false ]]; then
    mkdir -p "$CRUSH_SKILLS_DIR"
    shopt -s nullglob
    for src in "${ESQUISSE_DIR}/skills"/*/; do
        skill_name="$(basename "$src")"
        dst="${CRUSH_SKILLS_DIR}/${skill_name}"
        # Copy to .tmp first; only replace $dst when copy succeeds
        rm -rf "${dst}.tmp"
        cp -RL "$src" "${dst}.tmp" || {
            echo "ERROR: cp failed for $skill_name" >&2
            rm -rf "${dst}.tmp"
            exit 1
        }
        # cp succeeded — now safely replace
        rm -rf "$dst"
        mv "${dst}.tmp" "$dst"
        echo "  updated  ${CRUSH_SKILLS_DIR}/${skill_name}/"
    done
else
    echo "[dry-run] skills that would be copied to ${CRUSH_SKILLS_DIR}:"
    for src in "${ESQUISSE_DIR}/skills"/*/; do
        echo "  $(basename "$src")"
    done
fi

# Post-install output
echo ""
echo "Esquisse Crush integration installed."
echo ""
echo "  crush.json:  $CRUSH_CONFIG"
echo "  MCP binary:  $MCP_BINARY"
# Count installed skills
_skill_count=0
for _d in "${CRUSH_SKILLS_DIR}"/*/; do [[ -d "$_d" ]] && (( _skill_count++ )) || true; done
echo "  Skills:      ${CRUSH_SKILLS_DIR}/ (${_skill_count} dirs)"
echo ""
echo "Next steps:"
echo "  1. If 'copilot/' models are not available (e.g. on Termux), add an"
echo "     ESQUISSE_MODELS env key to the mcp.esquisse entry in crush.json:"
echo "       \"env\": { \"ESQUISSE_MODELS\": \"anthropic/claude-3-5-sonnet,openai/gpt-4o,gemini/gemini-2.0-flash\" }"
echo "  2. Verify Crush sees the MCP server: start Crush and check that esquisse-mcp appears"
echo "  3. In any project, run: bash scripts/gate-review.sh --strict"
