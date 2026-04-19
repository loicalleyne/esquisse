# P1-003 — Adversarial Review Skill: Crush Adaptation

**Phase:** P1  
**Status:** Done  
**Created:** 2026-04-17  
**Requires adversarial review:** minimum 3 rounds before implementation

---

## Goal

Update `skills/adversarial-review/SKILL.md` and add a companion reference
file `skills/adversarial-review/crush-models.md` so that the adversarial
review workflow operates correctly in Crush, using `crush run --model` for
cross-model reviewer dispatch as a direct equivalent of VS Code's
`runSubagent("Adversarial-r{n}", ...)`.

---

## Background

The adversarial review skill's Step 5 dispatches a reviewer using
`runSubagent("Adversarial-r{slot}", prompt)`. After P1-001 tool-name
translation, `runSubagent` becomes `agent` in Crush. However, Crush's `agent`
tool is **permanently read-only** — it can research but cannot write files.
The reviewer must write `.adversarial/{slug}.json` to record its verdict.

Additionally, Crush has no named agent dispatch. The `@Adversarial-r0` name
has no meaning in Crush; there are no `.agent.md` files with per-agent model
fields.

**Root cause:** the VS Code adversarial review workflow relies on two VS Code
capabilities that Crush lacks:
1. Named agent dispatch with model frontmatter (`.agent.md`)
2. Write-capable sub-agents

**Solution:** `crush run --model {provider/model} "{prompt}"` creates a fully
isolated non-interactive Crush session with a specified model. The child
process has full write access and can produce the verdict file. The main
session waits for the bash call to return, then reads the verdict file.

This is a semantically equivalent replacement: different model, isolated
context, produces the same `.adversarial/{slug}.json` artifact.

---

## In Scope

### 1 — New companion file: `skills/adversarial-review/crush-models.md`

Create `skills/adversarial-review/crush-models.md` with:

```markdown
# Adversarial Review: Crush Model Reference

When running in Crush (i.e., `runSubagent` is NOT in your tool list but `agent`
is), use the two-step bash approach below to invoke the reviewer. The rotation
slots map to these models:

| Slot | `iteration % 3` | `--model` flag value |
|------|----------------|---------------------|
| 0 | 0 | `openai/gpt-4.1` |
| 1 | 1 | `anthropic/claude-opus-4-5-20251101` |
| 2 | 2 | `openai/gpt-4o` |

These mirror the VS Code rotation (Adversarial-r0 = GPT-4.1, r1 = Claude
Opus, r2 = GPT-4o). The models are deliberately cross-provider to prevent
self-review bias.

Model strings are pinned (not `latest`). Update this file when a model is
deprecated. Deprecation policy: when a model string returns a 404 or
"model not found" error from the provider, update this file and increment
all affected task doc Session Notes.

## Constructing the bash call — SECURITY INVARIANT

Plan content MUST NEVER appear in the shell command line. It must be
written to a file first, then passed via stdin redirect. This prevents
shell injection from plan content containing `$(...)`, backticks, or
single-quote sequences.

**Step 1** — write plan content using the `write` tool:
```
write(
  path: ".adversarial/tmp/esquisse-review-{slug}-{timestamp}.txt",
  content: "{preamble}\n\n--- PLAN CONTENT ---\n{plan_content}"
)
```

Where `{preamble}` is:
```
You are Adversarial-r{slot}. Apply the 7-attack protocol to the plan below.
Write your report to .adversarial/reports/review-{date}-iter{iteration}-{plan-slug}.md
Write state to .adversarial/{plan-slug}.json with fields:
  plan_slug, iteration (={next_iteration}), last_model, last_verdict, last_review_date.
Schema: SCHEMAS.md §8. Verdict must be PASSED, CONDITIONAL, or FAILED.
```

**Step 2** — invoke the reviewer with stdin redirect (NO plan content in the command):
```bash
crush run --model {model_string} --quiet < ".adversarial/tmp/esquisse-review-{slug}-{timestamp}.txt"
```

**Step 3** — clean up:
```bash
rm -f ".adversarial/tmp/esquisse-review-{slug}-{timestamp}.txt"
```

## Platform detection

- **VS Code Copilot Chat**: `runSubagent` is in your tool list → use Steps 5/6 (named agent dispatch).
- **Crush**: `runSubagent` is NOT in your tool list; `agent` IS → use the two-step bash approach above.
## Environment prerequisite

The specified model providers must be configured in `~/.config/crush/crush.json`
(or the project-local `crush.json`). If the provider is not configured, the
`crush run` call will fail with "no providers configured".
```

### 2 — Update `skills/adversarial-review/SKILL.md`

Add a **"Platform Detection"** section immediately before the existing Step 5.

The new step instructs the agent to choose the dispatch method based on the
runtime platform:

```markdown
### Step 4b: Platform detection — choose dispatch method

Before dispatching the reviewer, determine which platform you are running on:

**VS Code Copilot Chat:**
- `runSubagent` is listed in your tool set.
- Proceed to Step 5 (named agent dispatch).

**Crush:**
- `runSubagent` is NOT listed in your tool set; `agent` IS listed.
  (`agent` in Crush is read-only; it cannot write `.adversarial/` files.)
- Load `skills/adversarial-review/crush-models.md` (or the project-local
  copy under the skills directory).
- Use the two-step bash approach defined in `crush-models.md`:
  1. Write the full review prompt (including plan content) to a temp file
     using the `write` tool.
  2. Call `bash: crush run --model {model} --quiet < {temp_file}`.
  3. Delete the temp file.
- The child process writes `.adversarial/{slug}.json`; read it after `bash`
  returns.
- Skip Step 5; proceed directly to Step 6 (present verdict).

**SECURITY INVARIANT:** Plan content must never appear in the shell command
line. Always use the write-then-stdin-redirect approach. See `crush-models.md`.

**Detection rule:** `runSubagent` in tool list → VS Code. `runSubagent` NOT
in tool list → Crush → use the two-step bash approach above.
```

### 3 — No changes to the existing Steps 1–4 or Steps 6–end

The platform detection is additive. It does not modify the rotation slot
calculation (Step 2), the reference file loading (Step 3), or the verdict
presentation (Step 6). The MCP shortcut (Step 4c) is out of scope for this
task — it is added by P1-004 once the server exists.

### 4 — `crush-models.md` is installed alongside `SKILL.md`

Since `scripts/init.sh` copies entire skill directories (`cp -RL`), the new
`crush-models.md` file is automatically included in both VS Code and Crush
skill installations. No init.sh changes are required for this task.

---

## Out of Scope

- Changing the adversarial review slot rotation logic
- Implementing the `esquisse-mcp` server (that is P1-004)
- Modifying `.agent.md` files for VS Code
- Updating `gate-review.sh`
- Handling the case where `crush` binary is not installed (fail with clear error)
- Installing or configuring `crush.json` provider entries

---

## Files

| Path | Action | What Changes |
|------|--------|-------------|
| `skills/adversarial-review/SKILL.md` | Modify | Add Step 4b (platform detection + two-step bash approach) |
| `skills/adversarial-review/crush-models.md` | Create | New reference file with Crush model rotation table and bash template |

---

## Acceptance Criteria

1. `skills/adversarial-review/SKILL.md` contains a "Platform Detection" section
   that appears before the reviewer dispatch step.
2. `skills/adversarial-review/crush-models.md` exists and contains the 3-slot
   rotation table with `openai/gpt-4.1`, `anthropic/claude-opus-4-5-20251101`,
   and `openai/gpt-4o`.
3. The `crush-models.md` bash template uses heredoc or stdin redirection (not
   direct string interpolation) for the prompt content.
4. After `scripts/init.sh` is run (P1-001 implemented), the Crush skills
   directory contains `adversarial-review/crush-models.md`.
5. The updated `SKILL.md` does not break the VS Code path (Steps 5/6 unchanged).

---

## Session Notes

- The `crush run` child process runs as a separate Crush session. It will be
  recorded in the Crush sessions database. This is acceptable behavior.
- `--quiet` suppresses the spinner but not the final text output. The output
  contains the reviewer's full response, from which the verdict line is extracted.
- The child process writes `.adversarial/` files. The parent session reads them.
  Race conditions are impossible (bash call is synchronous from parent's view).
- Model strings are intentionally pinned (not `latest`). Pinning ensures
  reproducibility across review iterations. Update `crush-models.md` when
  models are deprecated.
