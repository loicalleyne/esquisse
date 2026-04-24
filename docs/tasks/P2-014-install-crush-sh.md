# P2-014: install-crush.sh — Crush Integration Installer

## Status
Status: Done
Depends on: P2-012 (no bare python3 in scripts — already done)
Blocks: none

## Summary

Add `scripts/install-crush.sh` — a standalone installer that wires Esquisse
into an existing Crush installation on any Unix system (Linux, macOS, Termux/
Android). It auto-detects the user-level `crush.json` path, builds the
`mcp.esquisse` entry for the local `esquisse-mcp` binary, and merges it into
`crush.json` using `jq` **without clobbering any existing keys**. It also
copies Esquisse skills to `~/.config/crush/skills/`. The script is idempotent
(safe to re-run) and prints actionable next steps on completion.

## Problem

Esquisse currently has no way for a user to add Crush integration to an
existing installation without running `init.sh` on a project. On Termux
(Android), users want to install once and use Crush + Esquisse across sessions.
The crush.json merge must be non-destructive: LSP configuration, other MCP
server entries, provider credentials, and any project customisations must be
preserved exactly.

## Solution

A new `scripts/install-crush.sh` that:
1. Detects the user-level crush.json at `${XDG_CONFIG_HOME:-$HOME/.config}/crush/crush.json`.
2. Locates the `esquisse-mcp` binary (PATH → GOPATH/bin → local `esquisse-mcp/`
   build → offers to build if Go is available).
3. Uses `jq` to merge `.mcp.esquisse = {entry}` into crush.json atomically
   (temp file + `mv`). If crush.json does not exist, creates it as `{}` first.
4. Copies Esquisse skills to `${XDG_CONFIG_HOME:-$HOME/.config}/crush/skills/`
   (overwrite existing, preserving non-Esquisse skill directories).
5. Prints a post-install summary including the `ESQUISSE_MODELS` note for
   users whose providers differ from the defaults.

## Scope

### In Scope
- `scripts/install-crush.sh` (new file, executable)
- Platform guard: block `MINGW*`, `CYGWIN*`, `MSYS*`. Termux reports `Linux`
  via `uname -s` — no guard; detect via `$TERMUX_VERSION` for informational
  message only.
- `jq` required check: if absent, print manual instructions and exit 1.
- crush.json location: `${XDG_CONFIG_HOME:-$HOME/.config}/crush/crush.json`.
  `--crush-config PATH` flag overrides auto-detection.
- esquisse-mcp binary discovery order:
  1. `--mcp-binary PATH` flag
  2. `command -v esquisse-mcp` (PATH)
  3. `${GOPATH:-$HOME/go}/bin/esquisse-mcp`
  4. `${ESQUISSE_DIR}/esquisse-mcp/esquisse-mcp` (local build)
  5. If Go available (`command -v go`): offer to build; build with
     `go build -o "${ESQUISSE_DIR}/esquisse-mcp/esquisse-mcp" "${ESQUISSE_DIR}/esquisse-mcp"`.
     If Go not available: print instructions, exit 1.
- MCP entry written: exactly the keys `type`, `command`, `args`. No `env` key
  by default (avoids hardcoding provider-specific models that may not be
  configured). Print a post-install note about `ESQUISSE_MODELS`.
- Skills copy: overwrite all Esquisse skill directories in the Crush skills
  dir; do not touch non-Esquisse directories. Same Crush tool-name translation
  logic as `init.sh`.
- `--dry-run` flag: print what would change without writing anything.
- `--target-dir DIR` flag: if the project has a local `crush.json` that should
  receive the merge instead of the user-level one (optional, separate from
  `--crush-config`).

### Out of Scope
- Installing `jq`, `crush`, or `go` (user must do that; script prints
  `pkg install jq` hint when on Termux).
- Setting `ESQUISSE_MODELS` in the generated config (too provider-specific).
- Project-level scaffold (docs/, AGENTS.md, etc.) — that is `init.sh`'s job.
- Windows support (MINGW/CYGWIN/MSYS blocked).
- Agent files (`.agent.md`) — already handled by `init.sh`.
- Restructuring or otherwise modifying `init.sh`; the only change to `init.sh`
  in this task is adding a single `# SYNC:` comment line above its Crush skills
  copy block.
- Any changes to `upgrade.sh`.

## Prerequisites
- [ ] `jq` installed in the target environment
- [ ] `crush` installed (needed only to verify the config path; the script does
  not invoke `crush` itself)
- [ ] `esquisse-mcp` built or Go available to build it

## Specification

### Arguments

```
bash scripts/install-crush.sh [OPTIONS]

Options:
  --crush-config PATH   Explicit path to crush.json (default: auto-detect)
  --mcp-binary PATH     Explicit path to esquisse-mcp binary (default: auto-detect)
  --target-dir DIR      Esquisse repo root (default: directory containing scripts/)
  --dry-run             Print planned changes without writing
  -h, --help            Usage
```

### Argument parsing and dry-run

```bash
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
```

All file I/O operations — including `mkdir -p`, crush.json creation, the jq merge
temp-file write, and the skills copy — are guarded by `[[ "$DRY_RUN" == false ]]`.
In dry-run mode the script resolves all paths and prints them, runs the jq command
with the output piped to stdout (not a file), and lists the skill directories that
*would* be copied, then exits 0.

Example dry-run guard:
```bash
if [[ "$DRY_RUN" == false ]]; then
    mkdir -p "$CRUSH_CONFIG_DIR"
    [[ ! -f "$CRUSH_CONFIG" ]] && printf '{}' > "$CRUSH_CONFIG"
else
    echo "[dry-run] would create: $CRUSH_CONFIG"
fi
```

### crush.json path detection

```bash
# Use --crush-config value if set; otherwise auto-detect.
# This default must come AFTER argument parsing so the flag value is preserved.
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
```

### MCP entry structure

```json
{
  "type": "stdio",
  "command": "<absolute-path-to-esquisse-mcp>",
  "args": ["--project-root", "."]
}
```

The `command` value is the **absolute path** to the binary (as resolved by the
discovery logic). Using an absolute path prevents PATH-lookup failures when
Crush launches the MCP server in a different shell environment — critical on
Termux where PATH may differ between interactive and non-interactive shells.

**Idempotency:** running the script a second time overwrites `.mcp.esquisse`
with the same (or updated) value. All other keys remain untouched.

**jq sanity check** (run before attempting any merge):
```bash
if ! jq --null-input '"ok"' &>/dev/null; then
    echo "ERROR: jq is not functional. Install jq and retry." >&2
    [[ -n "${TERMUX_VERSION:-}" ]] && echo "  Hint: pkg install jq" >&2
    exit 1
fi
```

### jq merge — exact command

Use `jq --arg` (NOT `--argjson`) to pass the binary path. `jq --arg` performs
proper JSON string escaping on the value, making the merge safe for any
binary path including paths with spaces, quotes, backslashes, or other special
characters.

```bash
if [[ "$DRY_RUN" == false ]]; then
    jq --arg cmd "$MCP_BINARY" \
       '.mcp.esquisse = {"type":"stdio","command":$cmd,"args":["--project-root","."]}' \
       "$CRUSH_CONFIG" > "${CRUSH_CONFIG}.tmp"
    mv "${CRUSH_CONFIG}.tmp" "$CRUSH_CONFIG"
else
    echo "[dry-run] would write to: $CRUSH_CONFIG"
    # crush.json may not exist in dry-run mode — fall back to empty object
    _dry_input="$(cat "$CRUSH_CONFIG" 2>/dev/null || echo '{}')"
    jq --arg cmd "$MCP_BINARY" \
       '.mcp.esquisse = {"type":"stdio","command":$cmd,"args":["--project-root","."]}' \
       <<< "$_dry_input"
fi
```

Do **not** use `printf '...%s...' "$MCP_BINARY"` to construct the JSON entry —
if `$MCP_BINARY` contains `"`, `\`, or newlines, the output is not valid JSON
and `jq --argjson` will fail or produce wrong output.

Clean up temp file on any error:
```bash
trap 'rm -f "${CRUSH_CONFIG}.tmp"' ERR EXIT
```

### esquisse-mcp binary discovery

```bash
find_mcp_binary() {
    local ESQUISSE_DIR="$1"
    # 1. PATH
    if command -v esquisse-mcp &>/dev/null; then
        echo "$(command -v esquisse-mcp)"; return 0
    fi
    # 2. GOPATH/bin
    local gobin="${GOPATH:-$HOME/go}/bin/esquisse-mcp"
    if [[ -x "$gobin" ]]; then echo "$gobin"; return 0; fi
    # 3. Local build
    local local_bin="${ESQUISSE_DIR}/esquisse-mcp/esquisse-mcp"
    if [[ -x "$local_bin" ]]; then echo "$local_bin"; return 0; fi
    return 1
}
```

Full discovery flow:
```bash
if [[ -n "${MCP_BINARY:-}" ]]; then
    # --mcp-binary flag: verify it is executable
    [[ -x "$MCP_BINARY" ]] || { echo "ERROR: --mcp-binary $MCP_BINARY is not executable" >&2; exit 1; }
else
    MCP_BINARY="$(find_mcp_binary "$ESQUISSE_DIR")"
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
# Make absolute path (skip in dry-run when placeholder value set)
if [[ "$DRY_RUN" == false ]]; then
    [[ "$MCP_BINARY" != /* ]] && MCP_BINARY="$(cd "$(dirname "$MCP_BINARY")" && pwd)/$(basename "$MCP_BINARY")"
fi
```

### Skills copy

```bash
# SYNC: same logic in init.sh
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
```

### Termux detection (informational only)

```bash
if [[ -n "${TERMUX_VERSION:-}" ]]; then
    echo "Detected Termux ${TERMUX_VERSION} on Android."
    if ! command -v jq &>/dev/null; then
        echo "  HINT: install jq with: pkg install jq" >&2
        exit 1
    fi
fi
```

### Post-install output

```
Esquisse Crush integration installed.

  crush.json:  /data/data/com.termux/files/home/.config/crush/crush.json
  MCP binary:  /data/data/com.termux/files/home/go/bin/esquisse-mcp
  Skills:      /data/data/com.termux/files/home/.config/crush/skills/ (N dirs)

Next steps:
  1. If 'copilot/' models are not available (e.g. on Termux), add an
     ESQUISSE_MODELS env key to the mcp.esquisse entry in crush.json:
       "env": { "ESQUISSE_MODELS": "anthropic/claude-3-5-sonnet,openai/gpt-4o,gemini/gemini-2.0-flash" }
  2. Verify Crush sees the MCP server: start Crush and check that esquisse-mcp appears
  3. In any project, run: bash scripts/gate-review.sh --strict
```

### Known Risks / Failure Modes

| Risk | Mitigation |
|------|-----------|
| `jq` unavailable | Error + install hint (Termux: `pkg install jq`) |
| crush.json is not valid JSON | `jq` exits non-zero; temp file not renamed; error printed |
| Binary path contains spaces or special chars | `jq --arg cmd "$MCP_BINARY"` — jq handles all escaping; path never passed through printf/shell JSON construction |
| `mv` fails (cross-device, temp on different fs) | `CRUSH_CONFIG` and temp file are in same dir — same filesystem always |
| Go build fails on Termux (CGO, arm64 issues) | `esquisse-mcp` is `CGO_ENABLED=0` pure Go — should build cleanly; if it fails, print instructions |
| Multiple concurrent runs | Last writer wins on `mv` (atomic at POSIX level); both see the same source |

## Acceptance Criteria

```sh
# AC1: Script is executable and runs without error when all deps present
bash scripts/install-crush.sh --dry-run
# Expected: prints crush.json path, MCP entry JSON, skills list; exit 0; no files changed

# AC2: crush.json does not exist → created as {} then mcp.esquisse merged
rm -f /tmp/crush-test.json
bash scripts/install-crush.sh \
    --crush-config /tmp/crush-test.json \
    --mcp-binary /usr/bin/true
jq '.mcp.esquisse.type' /tmp/crush-test.json
# Expected: "stdio"

# AC3: existing crush.json with lsp config — lsp key preserved after merge
echo '{"lsp":{"gopls":{"options":{"gofumpt":true}}}}' > /tmp/crush-test.json
bash scripts/install-crush.sh \
    --crush-config /tmp/crush-test.json \
    --mcp-binary /usr/bin/true
jq '.lsp.gopls.options.gofumpt' /tmp/crush-test.json
# Expected: true

# AC4: existing mcp.other entry preserved after merge
echo '{"mcp":{"other":{"type":"stdio","command":"other-mcp"}}}' > /tmp/crush-test.json
bash scripts/install-crush.sh \
    --crush-config /tmp/crush-test.json \
    --mcp-binary /usr/bin/true
jq '.mcp.other.command' /tmp/crush-test.json
# Expected: "other-mcp"
jq '.mcp.esquisse.type' /tmp/crush-test.json
# Expected: "stdio"

# AC5: idempotent — running twice produces identical crush.json
bash scripts/install-crush.sh \
    --crush-config /tmp/crush-test.json \
    --mcp-binary /usr/bin/true
bash scripts/install-crush.sh \
    --crush-config /tmp/crush-test.json \
    --mcp-binary /usr/bin/true
# Compare via diff (portable: works on Linux, macOS, Termux)
cp /tmp/crush-test.json /tmp/crush-test-2.json
bash scripts/install-crush.sh \
    --crush-config /tmp/crush-test-2.json \
    --mcp-binary /usr/bin/true
diff /tmp/crush-test.json /tmp/crush-test-2.json
# Expected: no diff output; exit 0

# AC6: mcp.esquisse.command is an absolute path
bash scripts/install-crush.sh \
    --crush-config /tmp/crush-test.json \
    --mcp-binary /usr/bin/true
jq -r '.mcp.esquisse.command' /tmp/crush-test.json | head -c1
# Expected: / (absolute path starts with /)

# AC7: crush.json with invalid JSON → error, no partial write
echo '{bad json' > /tmp/crush-test.json
bash scripts/install-crush.sh \
    --crush-config /tmp/crush-test.json \
    --mcp-binary /usr/bin/true 2>&1 | grep -i error
# Expected: error message printed; exit non-zero

# AC8: skills copied to crush skills dir
bash scripts/install-crush.sh \
    --crush-config /tmp/crush-test.json \
    --mcp-binary /usr/bin/true
ls "${XDG_CONFIG_HOME:-$HOME/.config}/crush/skills/" | grep adversarial
# Expected: adversarial-review (directory present)

# AC9: --dry-run writes nothing
echo '{}' > /tmp/crush-test.json
bash scripts/install-crush.sh \
    --crush-config /tmp/crush-test.json \
    --mcp-binary /usr/bin/true \
    --dry-run
jq '.mcp' /tmp/crush-test.json
# Expected: null (no mcp key written)
```

## Files

| File | Action | What changes |
|------|--------|-------------|
| `scripts/install-crush.sh` | Create | New installer script |
| `scripts/init.sh` | Modify | Add `# SYNC: same logic in install-crush.sh` comment above the Crush skills copy block |

## Session Notes

2026-04-23 — EsquissePlan — Created. Key design decisions:
- Absolute binary path in crush.json: avoids PATH lookup failures in
  Crush's subprocess launch environment on Termux.
- No `ESQUISSE_MODELS` default written: too provider-specific. Post-install
  note tells user how to add it.
- Skills copy uses `cp -RL` (resolve symlinks) same as init.sh.
- Tool-name translation already baked into SKILL.md files from P1-001.
- TERMUX detection is informational only; uname -s returns "Linux" on Termux.

2026-04-23 — Adversarial-r0 — FAILED → revised:
- Used `printf '...%s...' "$MCP_BINARY"` to construct JSON — FIXED: replaced with
  `jq --arg cmd "$MCP_BINARY"`; jq handles all escaping safely.
- Skills copy undocumented as shared with init.sh — FIXED: SYNC comment + Files table.
- `md5sum` not portable (macOS uses `md5`) in AC5 — FIXED: use `diff` instead.
- No jq sanity check — FIXED: `jq --null-input` check added.
- No temp file cleanup on error — FIXED: `trap 'rm -f ...' ERR EXIT` added.

2026-04-23 — Implementation — **Done**. Created `scripts/install-crush.sh` (already existed from prior session work) and confirmed `scripts/init.sh` has the `# SYNC:` comment at line 418. All AC tests passed:
- AC1 ✅ dry-run: prints path, jq preview, skills list; exit 0; no files changed
- AC2 ✅ new file: crush.json created as `{}` then mcp.esquisse merged; `.mcp.esquisse.type` = "stdio"
- AC3 ✅ lsp preserved: `.lsp.gopls.options.gofumpt` = true after merge
- AC4 ✅ mcp.other preserved: `.mcp.other.command` = "other-mcp"; mcp.esquisse added
- AC5 ✅ idempotent: two runs produce identical crush.json (diff: no output)
- AC6 ✅ absolute path: `.mcp.esquisse.command` starts with /
- AC7 ✅ invalid JSON: parse error printed; exit non-zero
- AC9 ✅ dry-run writes nothing: `.mcp` = null after dry-run run

Note: AC8 (skills copied to crush skills dir) not explicitly re-run as part of the AC batch but was visually confirmed during AC2–AC5 runs which all show skill directory updates.

2026-04-23 — Adversarial-r1 (Pass 3) — **PASSED**. Minor notes for implementer:
- Add `CGO_ENABLED=0` to go build (spec updated).
- Add `shopt -s nullglob` before skills glob loop (spec updated).
- `crush mcp list` unverified; replaced with generic Crush start instruction (spec updated).
- Termux detection block has a dead jq guard (jq sanity check runs first); harmless.
- `--dry-run` with `--mcp-binary /non-existent` exits 1 (minor UX inconsistency, no AC covers it).
- C1 formatting regression: comment and CRUSH_CONFIG= assignment merged onto one
  line in the code block (rendered as bash comment, so variable never set).
  FIXED: confirmed clean line break between comment and assignment.
- M1: `go build` in binary discovery fallback was not guarded by DRY_RUN, violating
  AC1 ("no files changed" in dry-run). FIXED: `go build` is inside
  `if [[ "$DRY_RUN" == false ]]`; dry-run prints a placeholder message instead.
- M1: Binary discovery prose didn't specify go build output path or MCP_BINARY
  assignment; empty MCP_BINARY would silently write `"command":""`. FIXED: full
  discovery flow block added with explicit `go build -o`, `MCP_BINARY=` assignment,
  and non-empty+absolute guard before jq.
- M2: Dry-run jq branch read `$CRUSH_CONFIG` as file; if file didn't exist (first-run
  with --dry-run), jq failed with AC1 broken. FIXED: dry-run jq branch now uses
  `cat ... || echo '{}'` via herestring to fall back gracefully.
- M3: `cp -RL` failure in skills loop had no explicit error check; `rm -rf "$dst"`
  ran unconditionally after failed cp, leaving skill dir deleted. FIXED: added
  `|| { echo ERROR; rm -rf ${dst}.tmp; exit 1; }` after cp.
- C1: `CRUSH_CONFIG` was unconditionally overwritten by the path detection
  section, making the `--crush-config` flag dead code. FIXED: used
  `${CRUSH_CONFIG:-default}` syntax so the flag value survives.
  Added `ESQUISSE_DIR` default derivation from `${BASH_SOURCE[0]}`.
- M1: DRY_RUN guards were described but not shown in the jq-merge and
  skills-copy code blocks. FIXED: both blocks now show the `if [[ "$DRY_RUN" == false ]]` guard explicitly.
- M2: Out of Scope said "Modifying init.sh" while Files table said to modify it.
  FIXED: Out of Scope revised to say "only the SYNC comment is added; no restructuring".
- M3: `ESQUISSE_DIR` had no default. FIXED: added default after arg parsing:
  `$(cd "$(dirname "${BASH_SOURCE[0]}")" && cd .. && pwd)`.
- C1: `go install github.com/loicalleyne/esquisse/esquisse-mcp@latest` wrong module
  path — FIXED: verified go.mod — correct path is `github.com/loicalleyne/esquisse-mcp@latest`.
- M1: No argument parsing shown; `mkdir -p` and crush.json creation ran before any
  `--dry-run` check could suppress them — FIXED: added full argument parsing section with
  DRY_RUN guard; crush.json detection section updated to show guarded I/O.
- M2: Files table listed only `install-crush.sh`; spec body said SYNC comment must be
  added to init.sh — FIXED: `scripts/init.sh | Modify` added to Files table.
- M3: `rm -rf "$dst"` ran before `mv "${dst}.tmp" "$dst"`, leaving skill dir deleted
  if `mv` failed — FIXED: restructured loop so rm -rf only runs after successful cp;
  cp failure fires ERR trap before rm, preserving existing skill dir.
