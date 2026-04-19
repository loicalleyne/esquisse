# Spec: Esquisse × Crush + VS Code Full Compatibility

**Date:** 2026-04-17  
**Status:** Planning  
**Phase:** P1

---

## Problem Statement

Esquisse is currently VS Code Copilot Chat-native. Its skills reference
`runSubagent`, `run_in_terminal`, `read_file`, and other VS Code-specific
primitives. Crush (charmbracelet/crush) is a fully featured terminal-based AI
coding assistant with its own primitive set (`agent`, `bash`, `view`, `glob`,
etc.), skill loading mechanism, and `crush run --model` non-interactive CLI.

Full Esquisse × Crush compatibility requires solving four independent problems:

| Problem | Status |
|---------|--------|
| Install skills to both `~/.copilot/skills/` and `~/.config/crush/skills/` with translation | P1-001 CONDITIONAL — ready for impl |
| Provide a per-project Crush bootstrap context file (equivalent of `bof-bootstrap.instructions.md`) | **NEW — P1-002** |
| Make adversarial review work in Crush (multi-model rotation via `crush run --model`) | **NEW — P1-003** |
| MCP server for adversarial review dispatch (platform-agnostic, callable from Crush or VS Code) | **NEW — P1-004** |

---

## Architecture Overview

### P1-001 (existing, CONDITIONAL): init.sh skill installation

`scripts/init.sh` is updated to:
1. Copy skills verbatim to `~/.copilot/skills/` (VS Code).
2. Copy skills with tool-name translation to `~/.config/crush/skills/` (Crush).

Translation table (longest-match-first to prevent substring collisions):
`multi_replace_string_in_file` → `multiedit`, `replace_string_in_file` → `edit`,
`runSubagent` → `agent`, `run_in_terminal` → `bash`, `create_file` → `write`,
`manage_todo_list` → `todos`, `read_file` → `view`, `grep_search` → `grep`,
`list_dir` → `ls`, `file_search` → `glob`.

### P1-002 (new): CRUSH.md bootstrap context

In VS Code, `bof-bootstrap.instructions.md` is injected as always-on context
for every session. Crush has no `.instructions.md` mechanism. The equivalent
in Crush is a project `CRUSH.md` context file — loaded at startup as part of
the system prompt.

`scripts/init.sh` generates `CRUSH.md` in the project root (alongside
`AGENTS.md`). It contains:
- Instruction to read `AGENTS.md` first
- The Crush tool name mapping table (for the agent to know what tools exist)
- A standing gate-review instruction: run `bash scripts/gate-review.sh --strict`
  before declaring any planning session complete
- A note that `~/.config/crush/skills/` contains Esquisse skills installed by
  `scripts/init.sh`

### P1-003 (new): Adversarial review skill — Crush adaptation

The current `adversarial-review` SKILL.md dispatches `@Adversarial-r{slot}` via
`runSubagent`. After P1-001 translation, this becomes `agent`, but Crush's
`agent` tool is permanently read-only (research only). It cannot write the
`.adversarial/{slug}.json` verdict file.

The root gap: **Crush has no named agent dispatch and no per-agent model
selection.**

**Resolution**: `crush run --model {provider/model}` runs a completely isolated
non-interactive Crush session with a specified model. This child process has full
write access and can produce the verdict file. It is functionally equivalent to
`runSubagent` — it creates an isolated LLM context with a different model.

The updated `adversarial-review/SKILL.md` adds a **Platform section** instructing
the agent: if dispatching `@Adversarial-r{slot}` is not available (i.e., running
in Crush where `agent` is read-only), use `bash: crush run --model {provider/model}
"..."` to invoke the reviewer instead.

A companion `adversarial-review/crush-models.md` reference file defines the
exact provider/model strings for each rotation slot:

| Slot | Provider | Model string |
|------|----------|-------------|
| 0 | OpenAI | `openai/gpt-4.1` |
| 1 | Anthropic | `anthropic/claude-opus-4-5-20251101` |
| 2 | OpenAI | `openai/gpt-4o` |

These match the same cross-provider rotation strategy as VS Code
(Adversarial-r0/r1/r2).

### P1-004 (new): esquisse-mcp — adversarial review MCP server

An MCP stdio server written in Go (in the `esquisse-mcp/` directory) that
exposes two tools:

**`adversarial_review`**
- Input: `plan_slug` (string), `plan_content` (string)
- Reads `.adversarial/{plan_slug}.json` to get `iteration`; computes `slot = iteration % 3`
- Picks the model for the slot from a config table or env vars
- Shells out to `crush run --model {provider/model} "{adversarial review prompt with plan_content}"`
- Captures stdout; extracts verdict
- Writes `.adversarial/{plan_slug}.json` (increments iteration, stores verdict)
- Returns the full review report text

**`gate_review`**
- Input: `strict` (bool, default false)
- Scans `.adversarial/*.json` for FAILED or missing verdicts
- Returns structured result: `{blocked: bool, reason: string, blocking_plans: []string}`

**Why an MCP server?**

The MCP approach is platform-agnostic: any MCP-capable client (Crush, VS Code
with MCP configured, CI scripts) can invoke adversarial review with a single
tool call. The server handles model selection, state file management, and
`crush run` invocation. The skill's Step 5 becomes identical on all platforms:
call the `adversarial_review` MCP tool.

**Crush configuration** (`crush.json` fragment):

```json
{
  "mcp": {
    "esquisse": {
      "type": "stdio",
      "command": "esquisse-mcp",
      "args": ["--project-root", "."],
      "env": {
        "ESQUISSE_MODEL_SLOT0": "openai/gpt-4.1",
        "ESQUISSE_MODEL_SLOT1": "anthropic/claude-opus-4-5-20251101",
        "ESQUISSE_MODEL_SLOT2": "openai/gpt-4o"
      }
    }
  }
}
```

---

## Task Dependency Graph

```
P1-001  ←── P1-002 (init.sh already exists; CRUSH.md extends it)
P1-003  ←── (standalone: skill update only)
P1-004  ←── P1-003 (MCP server replaces bash-dispatch in the skill's Step 5)
```

P1-003 is a usable interim solution without P1-004. P1-004 makes the workflow
platform-transparent.

---

## Translation Table Maintenance

The tool-name translation table is a point-in-time snapshot. When any Esquisse
SKILL.md gains a new VS Code tool reference, the implementer must also update:
1. The `sed` block in `scripts/init.sh` (P1-001)
2. The Tool Name Reference table in the `CRUSH.md` template (P1-002)

This is documented in `AGENTS.md` Common Mistakes. There is no automated
staleness detection.

---

## Out of Scope

- Windows-native Crush path resolution (Crush is WSL/Linux only)
- Implementing Crush Stop hooks (Crush has no hook mechanism)
- Porting EsquissePlan agent mode to Crush (planning agent is VS Code-only)
- Per-agent model selection inside Crush (not supported; workaround via crush run)
- Uninstall logic for Crush skills (deferred)
- macOS `~/Library/Application Support` path for Crush skills

---

## Security Considerations

- `esquisse-mcp` receives `plan_content` from the agent. This content is passed
  as a shell argument to `crush run`. Shell injection must be prevented:
  pass the plan content via stdin or a temp file, not as a shell argument.
- `adversarial_review` only writes to `.adversarial/`; it must not write outside
  the project root.
- The MCP server must not read or expose secrets from the environment beyond the
  `ESQUISSE_MODEL_SLOT*` vars it explicitly consumes.
