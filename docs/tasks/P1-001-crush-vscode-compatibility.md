# P1-001 — Esquisse Crush & VSCode Compatibility

**Phase:** P1
**Status:** Done
**Created:** 2026-04-17
**Requires adversarial review:** minimum 3 rounds before implementation

---

## Goal

Update `scripts/init.sh` to install Esquisse skills globally for VS Code Copilot Chat and for Crush, using the same opt-in pattern already used by `init.sh` for Copilot agents. VS Code Copilot Chat skill installation follows the same existing pattern (copy to `~/.copilot/skills/`, WSL-aware path resolution). Crush skill installation copies to `${XDG_CONFIG_HOME:-$HOME/.config}/crush/skills/` (always the Linux/WSL-native path) and applies inline tool-name translation.

---

## Background

`init.sh` already installs Copilot agents globally to `~/.copilot/agents/` as part of the normal init flow, without a separate opt-in flag. This plan aligns Copilot skill installation and Crush skill installation with that same pattern: both happen in the existing "Adversarial review infrastructure" section of `init.sh`, alongside agent installation. No new flag is added.

VS Code Copilot Chat and Crush use different tool primitive names (`runSubagent` vs `agent`, `run_in_terminal` vs `bash`). SKILL.md files are markdown prose instruction documents — not JSON tool-call payloads — so translating tool names by string substitution is sufficient for Crush to recognise and invoke the correct native tools. VS Code skills are installed verbatim (no translation required).

---

## In Scope

### 1 — VS Code Copilot Chat skills (`~/.copilot/skills/`)

- Resolve `COPILOT_SKILLS_DIR` by reusing the WSL detection already done for the Copilot agents block. `_win_home` is only set inside the `if command -v cmd.exe` branch; it must not be referenced outside that branch (violates `set -u`). Set `COPILOT_SKILLS_DIR` with the same guard:
  ```sh
  if command -v cmd.exe &>/dev/null 2>&1; then
      COPILOT_SKILLS_DIR="${_win_home}/.copilot/skills"
  else
      COPILOT_SKILLS_DIR="$HOME/.copilot/skills"
  fi
  ```
  Do not re-invoke `cmd.exe`; `_win_home` is already set from the agents block.
- `mkdir -p "$COPILOT_SKILLS_DIR"`.
- For each skill directory `$src` in `$ESQUISSE_DIR/skills/*/`:
  - `dst="$COPILOT_SKILLS_DIR/$(basename "$src")"`.
  - **Verify source exists** (`[ -d "$src" ]`) before any destructive operation; if absent, print `WARN: source skill not found: $src` and `continue`.
  - If `$dst` exists but is **not a directory**: print `WARN: unexpected file at $dst — skipping` and `continue`.
  - **Pre-flight:** `rm -rf "${dst}.tmp"` (unconditionally removes any stale staging artifact — handles regular files, directories, and symlinks; no separate type check needed).
  - Copy to staging: `cp -RL "$src" "${dst}.tmp"`. If this fails, `rm -rf "${dst}.tmp"` and abort with error.
  - If `$dst` exists as a directory: remove old (`rm -rf "$dst"`); rename staging (`mv "${dst}.tmp" "$dst"`); print `UPDATED: ~/.copilot/skills/$(basename "$src")`.
  - If `$dst` does not exist: rename staging (`mv "${dst}.tmp" "$dst"`); print `CREATED: ~/.copilot/skills/$(basename "$src")`.
  - Use `cp -RL` (dereference symlinks) to produce a clean file tree rather than replicating symlink structure.
- If no skill directories are found in `$ESQUISSE_DIR/skills/`, print an informational message and exit the section.

### 2 — Crush skills (`~/.config/crush/skills/`)

- Crush is a Linux/WSL CLI tool and reads config from its Linux-native path. The Crush skills directory is **always** `${XDG_CONFIG_HOME:-$HOME/.config}/crush/skills` (the Linux/WSL `$HOME` path). The Windows `$USERPROFILE` path is **not** used for Crush. Only VS Code Copilot Chat (a Windows process) requires Windows-side path resolution.
- Set: `CRUSH_SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/crush/skills"`
- `mkdir -p "$CRUSH_SKILLS_DIR"`.
- For each skill directory `$src` in `$ESQUISSE_DIR/skills/*/`:
  - `dst="$CRUSH_SKILLS_DIR/$(basename "$src")"`.
  - **Verify source exists** (`[ -d "$src" ]`) before any destructive operation; if absent, print `WARN: source skill not found: $src` and `continue`.
  - If `$dst` exists but is **not a directory**: print `WARN: unexpected file at $dst — skipping` and `continue`.
  - **Pre-flight:** `rm -rf "${dst}.tmp"` (unconditionally removes any stale staging artifact — handles regular files, directories, and symlinks; no separate type check needed).
  - Copy to staging: `cp -RL "$src" "${dst}.tmp"`. If this fails, `rm -rf "${dst}.tmp"` and abort with error.
  - **Translation Step** (after staging copy, before rename): if `"${dst}.tmp/SKILL.md"` exists, apply `sed -i.bak` with the following substitutions in strictly this longest-match-first order to prevent substring collisions:
    1. `multi_replace_string_in_file` → `multiedit`
    2. `replace_string_in_file` → `edit`
    3. `runSubagent` → `agent`
    4. `run_in_terminal` → `bash`
    5. `create_file` → `write`
    6. `manage_todo_list` → `todos`
    7. `read_file` → `view`
    8. `grep_search` → `grep`
    9. `list_dir` → `ls`
    10. `file_search` → `glob`
  - Remove backup file explicitly: `rm -f "${dst}.tmp/SKILL.md.bak"` (explicit filename, not glob).
  - If `$dst` exists as a directory: remove old (`rm -rf "$dst"`); rename staging (`mv "${dst}.tmp" "$dst"`); print `UPDATED: crush/skills/$(basename "$src")`.
  - If `$dst` does not exist: rename staging (`mv "${dst}.tmp" "$dst"`); print `CREATED: crush/skills/$(basename "$src")`.
- If no skill directories are found in `$ESQUISSE_DIR/skills/`, print an informational message and exit the section.

### 3 — Integration into `init.sh`

- Copilot skill installation follows immediately after the existing Copilot agent installation block.
- Crush skill installation follows Copilot skill installation, as a new clearly labelled section.
- Both sections use `ESQUISSE_DIR` as set by the `ESQUISSE_DIR` assignment in the framework docs section of `init.sh`.

---

## Known Limitations

- `sed` substitutions use unanchored patterns (e.g. `s/read_file/view/g`). A tool name that appears as a prefix of a longer identifier (e.g. `read_file_path`) would be incorrectly translated to `view_path`. Adding word-boundary anchors is not cross-platform (`\b` = GNU, `[[:<:]]` = BSD), so this is accepted as-is; Esquisse SKILL.md files do not currently contain such compound identifiers. This is the same accepted limitation as in P6-007.
- Several VS Code Copilot Chat tool names have no direct Crush equivalent (`semantic_search`, `get_errors`, `view_image`, `vscode_listCodeUsages`, `vscode_askQuestions`). These are left untranslated; Crush will encounter them in skill prose but will not error.
- The `mv` step is atomic only when source and destination are on the same filesystem. Since both `${dst}.tmp` and `$dst` are within the same skills directory, this is guaranteed.

---

## Out of Scope

- `--install-global-skills` flag (no new flag; follows existing `init.sh` agent pattern).
- Windows-native path resolution for Crush (Crush is a Linux/WSL CLI tool).
- Uninstall logic.
- Validating SKILL.md frontmatter.
- macOS `~/Library/Application Support` path for Crush.

---

## Acceptance Criteria

1. Running `init.sh` on any project (first run or re-run) copies skills into `~/.copilot/skills/` and `${XDG_CONFIG_HOME:-$HOME/.config}/crush/skills/` alongside the existing agent installation.
2. Re-running `init.sh` updates both destinations (prints `UPDATED:`); no orphaned files remain.
3. Crush SKILL.md files contain Crush-native tool names; VS Code SKILL.md files are verbatim copies of source.
4. `cp -RL` is used for all copies; no symlinks in destination directories.
5. If staging copy fails during an update (dst previously existed), the old skills directory is preserved intact; staging directory is cleaned up, error printed, script aborts.
6. Crush skills directory always resolves to `${XDG_CONFIG_HOME:-$HOME/.config}/crush/skills`; it never uses `$USERPROFILE`.

---

## Files

| Path | Action | What Changes |
|------|--------|-------------|
| `scripts/init.sh` | Modify | Add Copilot skills and Crush skills installation sections |

---

### Round 1 Adversarial Review (iter0 — r0 / Adversarial-r0)

**Verdict:** FAILED — three critical issues resolved in revision:
1. **C1 (supply-chain/global installation):** Installation now follows existing `init.sh` agent pattern; no new flag. Consistent paradigm — skills installed alongside agents.
2. **C2 (missing `create_file` translation):** `create_file` → `write` added to translation table.
3. **C3 (`sed -i` cross-platform failure):** Mandated `sed -i.bak` everywhere (BSD/macOS safe).

### Round 2 Adversarial Review (iter1 — r1 / Adversarial-r1)

**Verdict:** FAILED — two critical issues and two majors resolved in revision:
1. **C1 (WSL symlink resolution for Windows VS Code):** Switched to copy (`cp -RL`) instead of symlink for both destinations.
2. **C2 (supply chain via symlink):** No symlinks in destination; copies are immutable snapshots.
3. **M1 (inconsistent update lifecycle):** Unconditional overwrite (stage → remove old → rename) for all cases.
4. **M2 (orphaned `.bak` files in subdirs):** Backup file removed by explicit path `rm -f "$dst.tmp/SKILL.md.bak"` (not glob).

### Round 3 Adversarial Review (iter2 — r2 / Adversarial-r2)

**Verdict:** FAILED — three critical issues and two majors resolved in revision:
1. **C1 (`cp -R` preserves symlinks):** Changed to `cp -RL` throughout to dereference symlinks.
2. **C2 (Crush path incorrectly routed through `$USERPROFILE` in WSL):** Crush always uses Linux-native `${XDG_CONFIG_HOME:-$HOME/.config}/crush/skills`; Windows path used only for VS Code Copilot.
3. **C3 (inconsistent paradigm — `--install-global-skills` flag vs always-on agents):** Removed flag; skill installation follows the always-on agent pattern.
4. **M1 (`rm -rf` without rollback):** Introduced atomic staging pattern (`cp -RL` to `$dst.tmp`, `rm -rf $dst`, `mv $dst.tmp $dst`); old skills preserved if staging copy fails.
5. **M2 (tool parameter coupling concern):** Addressed in Background — SKILL.md files are markdown prose, not JSON call specs; name translation is sufficient.

### Round 4 Adversarial Review (iter3 — r0 / Adversarial-r0)

**Verdict:** CONDITIONAL — two majors and three minors resolved in revision:
1. **M1 (stale `${dst}.tmp` corrupts copy):** Added explicit `rm -rf "${dst}.tmp"` pre-flight before every staging copy in both VS Code and Crush sections.
2. **M2 (conflict case — `$dst` is regular file — silently unhandled):** Added explicit non-directory check: print `WARN: unexpected file at $dst — skipping` and `continue`.
3. **L1 (WSL detection reuse ambiguous):** Specified that `_win_home` from agents block is reused; `cmd.exe` is not invoked again.
4. **L2 (zero skills — no message):** Added informational message if no skill directories found.
5. **L3 (AC5 scope too broad):** AC5 narrowed to update path only (old installation preserved if staging fails during an update).

### Round 5 Adversarial Review (iter4 — r1 / Adversarial-r1)

**Verdict:** CONDITIONAL — one major and two minors resolved in revision:
1. **M1 (`_win_home` unbound under `set -u` on non-WSL):** Replaced bare `${_win_home}` reference with a full `if command -v cmd.exe` guard in the Copilot skills block, mirroring the agents block. `_win_home` is only referenced inside the positive branch where it is guaranteed to be set.
2. **L1 (`CRUSH_SKILLS_DIR` never explicitly assigned):** Added explicit assignment line `CRUSH_SKILLS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/crush/skills"` before `mkdir -p`.
3. **L2 (`${dst}.tmp` pre-flight note for implementors):** Clarified that `rm -rf "${dst}.tmp"` handles all types (file, directory, symlink); no additional type check is needed.

### Round 6 Adversarial Review (iter5 — r2 / Adversarial-r2)

**Verdict:** PASSED — two minors applied post-review:
1. **L1 (line number reference drifts):** Removed line-number citation from `ESQUISSE_DIR` reference in §3.
2. **L2 (AC1 "new project" wording):** AC1 updated to "any project (first run or re-run)" to prevent implementors from adding an erroneous first-run guard.
