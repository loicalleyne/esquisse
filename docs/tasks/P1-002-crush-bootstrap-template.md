# P1-002 — CRUSH.md Bootstrap Context Template

**Phase:** P1  
**Status:** Done  
**Created:** 2026-04-17  
**Requires adversarial review:** minimum 3 rounds before implementation

---

## Goal

`scripts/init.sh` generates a `CRUSH.md` context file in the project root
(alongside `AGENTS.md`). This file is loaded by Crush at every session start
as part of the system prompt, serving the same purpose as
`bof-bootstrap.instructions.md` does for VS Code Copilot Chat: ensuring the
agent reads project constitution first, knows available tools, and respects
the gate-review workflow.

---

## Background

VS Code Copilot Chat has an `.instructions.md` injection mechanism
(`bof-bootstrap.instructions.md`) that auto-loads always-on context for every
session. Crush has no equivalent automatic injection mechanism — it uses
context files (`CRUSH.md`, `AGENTS.md`, etc.) that are loaded into the system
prompt at startup. These context files serve the same role but are
per-project, not global.

`scripts/init.sh` already generates `AGENTS.md`, `ONBOARDING.md`, `GLOSSARY.md`,
and other starter documents. Adding `CRUSH.md` generation follows the same
established pattern.

---

## In Scope

### 1 — New `create_file` call in `scripts/init.sh`

After the existing `create_file "AGENTS.md" ...` block, add:

```bash
create_file "CRUSH.md" "# Crush Session Context — ${PROJECT_NAME}

## Priority Order

1. **\`AGENTS.md\`** (highest — project law)
2. **Skills in \`~/.config/crush/skills/\`** (workflow guides)
3. **Default Crush behavior** (lowest)

## Project Context

Before any response, read \`AGENTS.md\`. If \`ONBOARDING.md\` exists, read it.
If \`GLOSSARY.md\` exists, use its vocabulary.

## Tool Name Reference

| What to do | Crush tool |
|---|---|
| Run a shell command | \`bash\` |
| Dispatch a sub-agent (read-only) | \`agent\` |
| Read a file | \`view\` |
| Edit an existing file | \`edit\` / \`multiedit\` |
| Create a new file | \`write\` |
| Find files by name/pattern | \`glob\` |
| Search file contents | \`grep\` |
| List directory | \`ls\` |
| Track tasks | \`todos\` |

## Multi-model Adversarial Review

Crush cannot dispatch named agents with different models. Use the
\`esquisse\` MCP server (\`adversarial_review\` tool) when available.
As a fallback, read \`skills/adversarial-review/crush-models.md\` for the
exact secure bash syntax (write to file, then stdin redirect) to run the
reviewer in a child process:

| Slot | Model string |
|------|-------------|
| 0 | \`openai/gpt-4.1\` |
| 1 | \`anthropic/claude-opus-4-5-20251101\` |
| 2 | \`openai/gpt-4o\` |

## Gate Review (Planning Gate)

Before declaring any planning session complete, run:
\`\`\`bash
bash scripts/gate-review.sh --strict
\`\`\`

This checks \`.adversarial/*.json\` for PASSED/CONDITIONAL verdicts.
A FAILED or missing verdict blocks the session.

## AST Navigation

If \`code_ast.duckdb\` exists at the project root, prefer structural
queries over reading every source file. Rebuild with:
\`\`\`bash
bash scripts/rebuild-ast.sh
\`\`\`
"
```

### 2 — Idempotency

`create_file` in `init.sh` is already idempotent (skips if file exists). No
additional handling needed.

### 3 — Re-run (update) behavior

`init.sh` uses `create_file` which skips existing files. `CRUSH.md` is a
living document (users may edit it). It must not be overwritten on re-run.
This is the same policy as `AGENTS.md` — generate once, maintain manually.

---

## Out of Scope

- Global `~/.config/crush/CRUSH.md` (per-project only; global bootstrap not
  currently supported by Crush config)
- `upgrade.sh` updating existing CRUSH.md (upgrade script touches docs, not
  context files)
- Windows path variants (CRUSH.md is consumed by Crush, a Linux/WSL tool)
- Validation of CRUSH.md contents

---

## Files

| Path | Action | What Changes |
|------|--------|-------------|
| `scripts/init.sh` | Modify | Add `create_file "CRUSH.md" ...` block after AGENTS.md block |

---

## Acceptance Criteria

1. Running `init.sh` on a new project creates `CRUSH.md` in the project root.
2. `CRUSH.md` contains all five sections: Priority Order, Project Context,
   Tool Name Reference, Multi-model Adversarial Review, Gate Review.
3. Re-running `init.sh` on a project that already has `CRUSH.md` prints
   `exists   CRUSH.md  (skipped)` and does not overwrite the file.
4. The generated `CRUSH.md` contains no tab characters; only spaces and
   newlines (bash heredoc must use spaces, not tabs, for indentation).
5. `CRUSH.md` does not appear in `AGENTS.md` as a file that is auto-committed;
   it is a living project file, not a generated artifact.

---

## Session Notes

- CRUSH.md is not a replacement for AGENTS.md — it is a Crush-specific
  orientation that redirects to AGENTS.md.
- The tool name table duplicates the translation table in P1-001; this
  duplication is intentional (the agent in Crush needs both the installed
  skill content and the CRUSH.md to understand the tool mapping).
- The multi-model adversarial review table values must match the model strings
  in P1-003's `crush-models.md`. If model strings change, both files must be
  updated together.
