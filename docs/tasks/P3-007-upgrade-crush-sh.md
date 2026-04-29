# P3-007 — upgrade-crush.sh

## Status

Ready

## Goal

Create `scripts/upgrade-crush.sh` — an idempotent script that syncs the
Esquisse Crush integration (skills + hooks) from the Esquisse source directory
to the user's global Crush configuration directory, without touching `crush.json`
or repeating the full first-time setup of `install-crush.sh`.

## In Scope

| File | Action | What changes |
|------|--------|-------------|
| `scripts/upgrade-crush.sh` | Create | New script — skills + hooks sync |
| `scripts/install-crush.sh` | Modify | Update SYNC comment (~line 155) to reference `upgrade-crush.sh` |
| `AGENTS.md` | Modify | Add `upgrade-crush.sh` to layout tree |
| `docs/planning/ROADMAP.md` | Modify | Add P3-007 row with `Status: Done` after implementation |

## Out of Scope

- Changes to `install-crush.sh` — install and upgrade are separate concerns
- Changes to `upgrade.sh` (VS Code / project-level upgrade)
- Modifying `crush.json` — that is install's job
- Rebuilding or updating the `esquisse-mcp` binary — tell user to re-run `install-crush.sh` if needed
- Any changes to skill content or hook content

## Specification

### Script: `scripts/upgrade-crush.sh`

**Purpose:** Sync Esquisse Crush integration after skills or hooks change.
Re-running `install-crush.sh` is also correct, but its name implies first-time
setup and it requires `jq` for `crush.json` manipulation. `upgrade-crush.sh`
needs no `jq` and is clearly named for the update use-case.

**Header comment** must document:
- What it overwrites (skills, hooks)
- What it does NOT touch (crush.json, MCP binary)
- When to use vs `install-crush.sh`

**Platform guard:** exit 1 with clear message on MINGW/CYGWIN/MSYS (Windows
native shell). Linux and WSL are supported.

**Shebang and shell options:** the script MUST begin with:
```bash
#!/usr/bin/env bash
set -euo pipefail
```
`set -euo pipefail` ensures errors in individual commands are not silently swallowed,
undefined variables are caught, and pipeline failures propagate.

**Argument parsing:**

| Flag | Description | Default |
|------|-------------|---------|
| `--target-dir DIR` | Esquisse repo root override | parent of `scripts/` |
| `--dry-run` | Print what would change without writing | false |
| `-h`, `--help` | Usage text | — |

No `--crush-config` flag — this script does not touch `crush.json`.

**`ESQUISSE_DIR` detection:** `$(cd "$(dirname "${BASH_SOURCE[0]}")" && cd .. && pwd)`

**`CRUSH_SKILLS_DIR` detection:** `${CRUSH_SKILLS_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/crush/skills}`

This honours the `CRUSH_SKILLS_DIR` env var when Crush is configured to read skills from a
non-default location (Crush checks `os.Getenv("CRUSH_SKILLS_DIR")` and uses it exclusively
when set). An unconditional assignment would write to the wrong path while Crush looks elsewhere.

**`CRUSH_HOOKS_DIR` detection:** `${XDG_CONFIG_HOME:-$HOME/.config}/crush/hooks`

**Sanity check:** verify `${ESQUISSE_DIR}/skills/` directory exists; exit 1 if not
(catches `--target-dir` pointing to wrong path).

**Skills copy logic** (identical two-phase pattern from `install-crush.sh`):

The entire block — including `mkdir`, the loop, and the `rm/cp/mv` operations — must be
wrapped in a `DRY_RUN` gate. Show the complete structure:

```bash
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
```

The `cp -RL` flag dereferences symlinks on copy (same as `install-crush.sh`).
`# SYNC: same logic in install-crush.sh` comment on the block.

**Hooks copy** (gated by `DRY_RUN`):

```bash
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
```

**Dry-run mode:** ALL writes (mkdir, cp, mv, chmod, rm) must be gated by
`if [[ "$DRY_RUN" == false ]]; then`. The dry-run and live paths are separate
branches inside a single `if/else/fi` block — never interleaved. The code
snippets above show this structure explicitly. Both branches must call
`shopt -s nullglob` before any glob expansion to prevent literal `*/` output
when `skills/` is empty.

**Summary output** (only printed when `DRY_RUN == false`; use variables, not
hardcoded paths; N is computed by counting dirs already in `$CRUSH_SKILLS_DIR`):
```bash
if [[ "$DRY_RUN" == false ]]; then
    _skill_count=0
    for _d in "${CRUSH_SKILLS_DIR}"/*/; do [[ -d "$_d" ]] && (( _skill_count++ )) || true; done
    echo ""
    echo "Crush integration updated."
    echo ""
    echo "  Skills:  ${CRUSH_SKILLS_DIR}/  (${_skill_count} dirs)"
    echo "  Hooks:   ${CRUSH_HOOKS_DIR}/protect-adversarial.sh"
    echo ""
    echo "To update the MCP binary or crush.json entry, re-run:"
    echo "  bash scripts/install-crush.sh"
fi
```

Note: `# SYNC: same logic in install-crush.sh` — when updating `install-crush.sh`'s skills
copy block, apply the same change here, and vice versa.

### AGENTS.md change

Add `scripts/upgrade-crush.sh` to the layout tree under `scripts/`, after
`install-crush.sh`. One-line description: `← sync skills/hooks to ~/.config/crush/ (no crush.json changes)`.

### ROADMAP.md change

Add row to the P3 Tasks table:
```
| [P3-007-upgrade-crush-sh](../tasks/P3-007-upgrade-crush-sh.md) | ✅ Done | `scripts/upgrade-crush.sh` — sync Esquisse skills and hooks to global Crush config |
```
(Status changes to Done after implementation.)

## Acceptance Criteria

1. `bash scripts/upgrade-crush.sh --dry-run` prints the list of skills and
   hook that would be copied without writing any files and exits 0.
2. `bash scripts/upgrade-crush.sh` copies every directory under `esquisse/skills/`
   to `~/.config/crush/skills/`. Verified by:
   `ls ~/.config/crush/skills/adversarial-review/SKILL.md` (exits 0).
3. `bash scripts/upgrade-crush.sh` copies `protect-adversarial.sh` to
   `~/.config/crush/hooks/protect-adversarial.sh`. Verified by:
   `diff scripts/hooks/protect-adversarial.sh ~/.config/crush/hooks/protect-adversarial.sh`
   (exits 0).
4. Running the script twice produces identical results (idempotent): the second
   run prints the same "updated" lines and exits 0.
5. `bash scripts/upgrade-crush.sh` on a Windows native shell (MINGW) exits
   non-zero with `ERROR:` on stderr.
6. `bash scripts/upgrade-crush.sh --target-dir /nonexistent` exits non-zero
   with a message referencing the missing skills directory.
7. If `XDG_CONFIG_HOME` is set, skills are copied to `$XDG_CONFIG_HOME/crush/skills/`
   rather than `~/.config/crush/skills/`.
   - Dry-run: `XDG_CONFIG_HOME=/tmp/test-crush bash scripts/upgrade-crush.sh --dry-run`
     prints `/tmp/test-crush/crush/skills` in its output and exits 0 without
     creating `/tmp/test-crush/`.
   - Live-run summary: `XDG_CONFIG_HOME=/tmp/test-crush bash scripts/upgrade-crush.sh`
     prints `/tmp/test-crush/crush/skills` (not `~/.config/crush/skills`) in the
     summary output.
8. `AGENTS.md` contains `upgrade-crush.sh` in the layout tree.
9. `ROADMAP.md` P3 table contains a P3-007 row.
10. If `CRUSH_SKILLS_DIR` is already exported in the caller's environment, the script
    writes skills to that path, not to the XDG default.
    - Dry-run: `CRUSH_SKILLS_DIR=/tmp/test-skills bash scripts/upgrade-crush.sh --dry-run`
      prints `/tmp/test-skills` (not `~/.config/crush/skills`) in its dry-run output,
      exits 0, and does not create `/tmp/test-skills/`.
    - Live-run: `CRUSH_SKILLS_DIR=/tmp/test-skills bash scripts/upgrade-crush.sh`
      writes skills to `/tmp/test-skills/` and the summary output shows that path.

## Session Notes

- **CRUSH_SKILLS_DIR env var (verified from Crush source):** Crush's
  `GlobalSkillsDirs()` checks `os.Getenv("CRUSH_SKILLS_DIR")` and when set,
  returns that path exclusively. The script MUST use
  `${CRUSH_SKILLS_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/crush/skills}` to
  honour a non-default Crush skills location. An unconditional assignment would
  write skills to the wrong directory silently.
- `install-crush.sh` uses jq to merge `mcp.esquisse` into crush.json; this
  script deliberately omits that step to keep it jq-free and focused.
- Two-phase `cp`→`mv` pattern from `install-crush.sh` is the required idiom
  for atomic skill replacement (prevents partial reads during copy). Use
  `# SYNC: same logic in install-crush.sh` comment to keep both in sync.
- Crush reads skills from `~/.config/crush/skills/` (observed from installed
  state: `\\wsl.localhost\Ubuntu-22.04\home\lalleyne\.config\crush\skills\`).
  `$XDG_CONFIG_HOME` override is respected per XDG spec.
- No `jq` dependency — makes the script safe for Termux where jq may not be
  installed.
