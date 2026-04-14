---
name: new-task
description: >
  Use this skill to define a new task for an Esquisse-managed project.
  Triggers on: "create a new task", "add a task", "define task", "new task for phase N",
  "I need to implement {feature}", "what's next?".
  DO NOT USE when the user wants to initialize a brand-new project (use the init-project skill).
  DO NOT USE when the user wants to implement code immediately without a task document first.
---

# Skill: Task Definition

## Purpose

Interview the user to define a task, run `new-task.sh` to create the document, and fill
in all sections — producing a complete task spec ready for an implementation session.

## Prerequisites & Environment

- **Required MCP Servers:** None.
- **Required CLI:** `bash`; `scripts/new-task.sh` must exist in the project root.
- **Required files:** `AGENTS.md` (project must be initialized), `docs/planning/ROADMAP.md`.

## Execution Steps

### Step 1: Validate Context

Verify required files before making any changes:

- Check `AGENTS.md` exists. If not: stop and tell the user to initialize the project with
  the `init-project` skill first.
- Check `scripts/new-task.sh` exists. If not: offer to copy it from the Esquisse
  installation at `/opt/esquisse/scripts/new-task.sh`.
- Check `docs/tasks/` exists. If not: create it with `mkdir -p docs/tasks`.

Then read silently:

1. `docs/planning/ROADMAP.md` — find `## Current Phase: P{n}` for the default phase number.
2. `docs/tasks/` directory listing — existing tasks to avoid duplicate slugs.
3. `NEXT_STEPS.md` `## Immediate Next` — any already-queued tasks.

### Step 2: Interview

Report what you found in one line, then ask:

```
Current phase: P{n} — {Name}. Existing tasks: [list of slugs or "none"].

To create a new task I need:

1. **Phase** (default: {n} — press Enter or give a different number)
2. **Goal** — one sentence: what does this task produce when done?
3. **In scope** — what must be built (bullet list, or describe freely)
4. **Out of scope** — what this task must NOT do (at least one item)
5. **Interface sketch** — the key types/functions this task must produce.
   Can be rough. If you don't know yet, write "TBD" and I'll add a placeholder.
6. **Known constraints** — library gotchas, performance requirements, dependencies on other tasks (or "none")

Provide as a numbered list.
```

Wait for the user's response.

---

### Step 3: Derive the Slug

From the **Goal** sentence, derive a kebab-case slug:

- Extract the core noun phrase (drop articles, verbs like "implement/create/add").
- Max 4 words, lower-case, joined with hyphens.
- Examples:
  - "Implement the Kafka ingester" → `kafka-ingester`
  - "Add DuckDB file rotation with ready-event emission" → `duckdb-file-rotation`
  - "Write gRPC service for file-ready notifications" → `grpc-file-ready`

Show the user the proposed slug before running the script: `Proposed slug: {slug}. OK?`  
If they suggest a different slug, use theirs.

---

### Step 4: Run new-task.sh

```sh
bash scripts/new-task.sh {phase} {slug}
```

Capture the output line `Created: docs/tasks/P{n}-{nnn}-{slug}.md` to get the exact file path.

If the script fails:
- Check that `docs/tasks/` exists (`mkdir -p docs/tasks` if not).
- Check that `scripts/new-task.sh` exists — if not, the project was not initialized with Esquisse; offer to copy the script from the esquisse installation.

---

### Step 5: Fill in the Task Document

Open the created file. Fill in every `TODO:` section using the user's answers from Phase 1.

### Goal
Replace `TODO: One sentence — what does this task produce?` with the user's goal sentence verbatim.

### Background
Write 2-3 sentences explaining:
- Why this task is needed (what problem it solves or what dependency it unblocks).
- Where it fits in the phase — is it a prerequisite for other tasks?
- Reference the task it unblocks by slug if known.

### Acceptance Criteria
Convert the user's in-scope items to concrete, verifiable statements. **Write each criterion as a test function name**, not prose:

| User says | Criterion |
|-----------|-----------|
| "ingester reads kafka messages" | `[ ] TestIngester_ReadsMessages` |
| "file rotates after N records" | `[ ] TestWriter_RotatesFile_AfterNRecords` |
| "gRPC returns queue depth" | `[ ] TestRPCServer_GetQueueStatus_ReturnsDepth` |

Always include:
```
- [ ] All tests pass: `{test command from AGENTS.md}`
- [ ] No `panic("TODO` stubs remain.
- [ ] Coverage ≥ 80% for changed packages.
```

### Scope

**In scope:** The user's in-scope list, one bullet per item. Be specific:
- add the package path where the work happens.
- add "tests for each new function".

**Out of scope:** The user's out-of-scope list. If the user gave none, ask or infer from the goal:
- Common out-of-scope for implementation tasks: "No CLI changes", "No config changes this task", "No public API changes this task".
- Add at minimum: "No changes to packages not listed in Files table."

### Interface Sketch

If the user provided one, paste it verbatim in the code block. 
If they said "TBD", write:

```go
// TODO: Define types and function signatures when the design is clear.
// Follow agent-facing interface design rules in AGENTS.md.
```

If the user described the interfaces in prose instead of code, translate to stub Go signatures (or the appropriate language).

**Interface-as-Prompt Checklist** — check each box to the extent you can verify from the user's description:
- [ ] Type names state what they are, not how they are implemented.
- [ ] Function names include the verb (Fetch, Write, Validate — not Get/Do/Handle).
- [ ] Parameters are named after their constraint, not their type.
- [ ] Every error return is actionable.
- [ ] Invalid states are unrepresentable (use types, not runtime checks).

### Implementation Notes

From the user's **Known constraints** answer. Add:
- Direct quotes of any performance requirements.
- Library constraints (e.g. "must use bufarrowlib.AppendRaw, not proto.Unmarshal").
- Any upstream task this task depends on (with task ID).
- Any link from AGENTS.md Common Mistakes that is relevant.

If none were given, write: `See AGENTS.md Common Mistakes. No additional constraints identified.`

### Functions / Methods Table

Produce one row per function/method named in the Interface Sketch. Use this format:

```
| NewIngester() | internal/ingest | Construct ingester from config | Return ErrInvalidConfig |
```

If the interface is TBD, leave the placeholder row.

### Test Plan

Write the exact `go test` (or equivalent) command to run tests for this task:

```sh
go test -v -count=1 -run Test{Slug_PascalCase} ./{package}/...
```

---

### Step 6: Update Supporting Documents

After filling in the task doc:

1. **ROADMAP.md** — add a row to the `P{n} Tasks` table:
   ```
   | [P{n}-{nnn}-{slug}](docs/tasks/P{n}-{nnn}-{slug}.md) | Ready | {one-line summary} |
   ```

2. **NEXT_STEPS.md** — prepend to `## Immediate Next`:
   ```
   - [ ] [P{n}-{nnn}: {Title}](docs/tasks/P{n}-{nnn}-{slug}.md) — {goal sentence}
   ```

---

### Step 7: Confirm

Report:

```
Task created: docs/tasks/P{n}-{nnn}-{slug}.md

  Goal         [done]
  Acceptance   [done] ({N} criteria as test names)
  Scope        [done] (in: {N} items, out: {N} items)
  Interface    [done / TBD]
  ROADMAP.md   [updated]
  NEXT_STEPS   [updated]

Start an implementation session with:
  "Implement task P{n}-{nnn}-{slug}"
  (The agent will read AGENTS.md then the task doc before writing any code.)
```

---

## Constraints & Security

- DO NOT overwrite a file section that has already been filled in (non-TODO content).
  Only replace text that is verbatim a `TODO:` placeholder.
- DO NOT run `new-task.sh` without first showing the user the proposed slug and
  receiving confirmation.
- DO NOT invent build/test commands. Read them from `AGENTS.md` or `TEMPLATES.md`.
- ALWAYS verify `scripts/new-task.sh` exists before executing (Step 1).
- NEVER add scope items, acceptance criteria, or interface signatures the user did not
  provide. Mark unprovided sections as `TODO:` and note them in Session Notes.
- SECURITY: Do not execute any destructive shell commands. `new-task.sh` only creates
  new files and never modifies existing ones.

## Edge Cases

| Situation | Action |
|-----------|--------|
| User says "what's next?" | Read ROADMAP.md, list queued tasks. If none, offer to create one. |
| User provides goal in vague terms ("improve performance") | Ask: "Complete this sentence: 'This task is done when _____ can be demonstrated by a passing test named _____'." |
| User skips out-of-scope | Add: "No changes outside the listed files." That is always true even if they don't say it. |
| Acceptance criteria are prose, not test names | Convert each one to `TestXxx_YyyCondition` before writing the doc. Brief inline explanation of why. |
| Task spans too many files (>10) | Warn the user: "This task touches {N} files which exceeds the recommended maximum of 10 (§6 Context Window Hygiene). Consider splitting into two tasks. Proceed anyway?" |
| Phase number not found in ROADMAP.md | Create for the specified phase anyway; note the discrepancy in Session Notes. |
