---
name: implement-task
description: >
  Central implementation workflow for Esquisse task documents. USE when a user
  wants to execute a specific task from docs/tasks/. Encodes the full
  Per-Task Startup Protocol (11 steps) and Per-Task Completion Protocol (10
  steps) from FRAMEWORK.md §9. Triggers on: "implement task P{n}-{nnn}",
  "start working on {slug}", "execute this task", "let's implement",
  "do the work for task X".
  DO NOT invoke this skill when: the user wants to explore the codebase first
  (use explore-codebase); there is a bug to fix (use debug-issue); the user
  wants to write a spec (use write-spec).
---

# Implement-Task

## Overview

Wraps the complete Esquisse task lifecycle: startup orientation, implementation,
and full completion protocol. Never skips steps. Every session ends with a
handoff prompt.

**Announce at start:** "I'm using the implement-task skill."

## Prerequisites

- [ ] A task document exists at `docs/tasks/P{n}-{nnn}-{slug}.md`.
- [ ] `AGENTS.md` exists layered with invariants and conventions.
- [ ] `GLOSSARY.md` exists (or can be bootstrapped from AGENTS.md vocabulary).

**Tools:** `read_file`, `replace_string_in_file`, `create_file`, `run_in_terminal`, `runSubagent`

## Workflow

### Phase 1: Startup

#### Step 1 — Identify Task

Locate the task document. If the user did not specify a full path, search
`docs/tasks/` for matching slug or task number. Ask if ambiguous.

#### Step 2 — Load Context

Read in parallel:
- `AGENTS.md` — internalize all invariants, common mistakes, conventions.
- `GLOSSARY.md` — confirm vocabulary for this domain area.
- The task doc — read **completely**: Summary, In Scope, Out of Scope,
  Acceptance Criteria, Files table, Specification.

Extract and hold:
- Acceptance criteria (the definition of done)
- Files table (what to create/modify)
- Out of Scope list (guard against scope creep)
- Any `Depends on:` prerequisites

#### Step 2b — Load Planning Artifacts

If the task doc contains a `## Planning Artifacts` section:

For each row in the Planning Artifacts table:
1. Read the linked artifact file from `docs/artifacts/`.
2. Read **only the sections named in "What to read from it"** — not the full file.
   Example: "API Surface, Anti-Patterns" → read those two sections, skip the rest.
3. Hold the loaded facts as working context for Steps 6+.
4. If the artifact's word count is not known: after reading the requested sections,
   estimate whether the content is unusually large (> 600 words). If so, note it in
   Session Notes and read only the subsections most directly relevant to the current task.

If an artifact file is missing (path in table but file does not exist):
- Note in Session Notes: `MISSING ARTIFACT: {path} — proceeding without it.`
- Do NOT attempt to reconstruct the artifact's content from training data.
  Hallucinated API surfaces are the failure mode this mechanism exists to prevent.

If the task doc has no `## Planning Artifacts` section, skip this step entirely.
A missing section means no external research was needed — do not search for artifacts speculatively.

#### Step 3 — Establish Baseline

Run the full test suite **before touching any code**:

```sh
# Detect and run appropriate test command from AGENTS.md build commands
# Go example:
go test -count=1 -timeout 180s ./...
```

Record the pass/fail count. If tests are already failing: document in
Session Notes before writing any code. Do NOT let pre-existing failures
be attributed to changes made in this session.

#### Step 3b — Planning Context Drift Check

If `code_ast.duckdb` exists, check for symbol drift and missing symbols before touching any file:

> **AST Freshness:** Ensure the `ast` table is current before running drift or
> missing-symbol queries. If Go source files have changed since planning,
> run `bash scripts/rebuild-ast.sh` to rebuild. Stale AST produces false
> negatives (drift not detected) or false positives (symbols appear missing).
>
> Before running queries below, verify `ast` is non-empty:
> ```sql
> SELECT count(*) FROM ast;
> ```
> If the result is 0: print
> `[WARN] ast table is empty — drift and missing-symbol queries will return 0 rows and cannot be trusted. Run bash scripts/rebuild-ast.sh and verify it scans Go source files.`
> Skip the drift and missing-symbol queries. Do **not** interpret 0-row results as "no drift".

**Line drift (symbol moved within same file):**
```sql
LOAD sitting_duck;
.read scripts/macros.sql
SELECT * FROM planning_drift('{task_id}');
```
If any rows returned: log `DRIFT DETECTED: {symbol_name} planned_line={planned_line} current_line={current_line}` to Session Notes. Advisory — do not block implementation.

**Missing symbols (symbol deleted or moved to different file):**
```sql
LOAD sitting_duck;
.read scripts/macros.sql
SELECT p.symbol_name, p.file_path, p.role
FROM planning_context p
WHERE p.task_id = '{task_id}'
  AND NOT EXISTS (
      SELECT 1 FROM ast a
      WHERE a.name      = p.symbol_name
        AND a.file_path = p.file_path
  );
```
If any rows returned: print
`[WARN] MISSING SYMBOL: {symbol_name} at {file_path} (role={role}) — may have moved or been deleted`
as a visible line in chat output and log to Session Notes.
Advisory — do not block implementation, but treat as high-priority investigation points.

#### Step 4 — Orient

Orient to the codebase using the sub-steps below before writing any code.

#### Step 4a — Query planning_context First

When `code_ast.duckdb` exists, first verify the `planning_context` table is present:
```sql
SELECT count(*) FROM information_schema.tables
WHERE table_name = 'planning_context';
```
If the result is 0: print
`[WARN] planning_context table absent — EsquissePlan macros may not have been run for this project. Falling back to raw file orientation.`
then skip to Step 4b.

If the table exists, query it:
```sql
LOAD sitting_duck;
SELECT task_id, role, symbol_kind, symbol_name, file_path, line_start, line_end
FROM planning_context
WHERE task_id = '{task_id}'
  AND role IN ('modify', 'implement')
ORDER BY role, file_path, line_start;
```
If the result is empty: print
`[WARN] planning_context has no rows for task {task_id} — capture may not have run. Falling back to raw file orientation.`
then proceed with Step 4b.

Otherwise, use these rows as the definitive list of integration points. Read
raw source only for symbols not found in `planning_context`, or when
`signature` is NULL.

#### Step 4b — Raw File Orientation

For each file in the task doc's Files table:

- **If `code_ast.duckdb` exists:** use the `explore-codebase` skill or direct
  AST queries to locate the relevant types and functions.
- **Otherwise:** read the file's header, interface declarations, and exported
  function signatures only — not the full implementation.

Goal: understand what already exists at the integration points.

#### Step 5 — State Assumptions

If the task doc is ambiguous about any implementation detail, write assumptions
to Session Notes now — before writing any code. Each assumption will become an
`// ASSUMPTION: {reason}` comment at the decision point in code.

---

### Phase 2: Implementation

#### Step 6 — Follow Edit Order (enforce strictly)

1. **Define interfaces and types first.** The code must compile after this step.
2. **Implement production code.** Interfaces first; consumers second.
3. **Write tests.** Red first (stub), then green (implement).
4. **Update documentation** — godoc, `llms.txt` / `llms-full.txt` only if a
   public API was added or changed.

Never implement step 2 before step 1. Never accumulate multiple untested
changes.

#### Step 7 — Run Tests After Each Logical Unit

After every function or type implementation: run the relevant test.
Do not batch multiple implementations and test them all at once.

#### Step 8 — Bail-Out Trigger (Hard Rule 16)

If the same error or test failure persists after **3 meaningfully different**
approaches:

1. **STOP.** Do not attempt a fourth approach.
2. Revert the failing function to a stub:
   `panic("TODO P{n}-{nnn}: {slug} — {what is missing}")`
3. Record in `docs/planning/NEXT_STEPS.md` under `## Blocked`:
   ```
   {YYYY-MM-DD} — P{n}-{nnn} — STUCK on {function/test}
   Tried: {approach 1}, {approach 2}, {approach 3}
   Blocker: {what is unknown}
   ```
4. Set task Status: `Blocked`.
5. Yield to user. Do NOT attempt more code changes.

---

### Phase 3: Completion

Run after all acceptance criteria pass.

Read the completion protocol from
`skills/implement-task/references/completion-protocol.md` and execute every step.

#### Step 9 — Verify Acceptance Criteria

Run the exact test commands from the task doc's Acceptance Criteria section.
**Every criterion must be green.** Do not mark Done until all pass.

#### Step 10 — Update Task Doc

Set `Status: Done`. Append to `## Session Notes`:
```
{YYYY-MM-DD} — Completed. {one sentence on approach or key decision}
```

#### Step 11 — Update AGENTS.md

For every workaround, surprising behaviour, or near-mistake encountered:
add an entry to `## Common Mistakes to Avoid`.

#### Step 12 — Update GLOSSARY.md

For every new domain term introduced (type name, concept, event):
add a row to `GLOSSARY.md`. Trigger: if you named something that isn't
already in the glossary, add it.

#### Step 13 — Update ONBOARDING.md

If a new package or key file was created: add a row to the Key Files Map.
If the primary data flow changed: update the Data Flow diagram.

#### Step 14 — Update ROADMAP.md

Mark this task ✅ Done in the phase table. If new tasks were identified
during implementation: create stub task docs (Status: Draft) and add them.

#### Step 15 — ADR Trigger

If the phrase "we chose X over Y because" appears in reasoning during this session:

- **If `write-adr` skill is available:** invoke it.
- **Otherwise (Batch 1–2 deploy period):** inline the ADR directly using
  SCHEMAS.md §5 as the template at `docs/decisions/ADR-{nnnn}-{slug}.md`.

#### Step 16 — Update NEXT_STEPS.md

Append to `## Session Log`:
```
{YYYY-MM-DD} — {task-id} — {one sentence summary}
Files changed: {comma-separated list}
Up next: {next task id and slug, or "phase gate" if last task}
```

**Never skip this step.**

#### Step 17 — Update llms.txt / llms-full.txt

Only if a public API was added or changed in this task:

- **If `update-llms` skill is available:** invoke it.
- **Otherwise:** run the language-appropriate doc-gen command:
  - Go: `go doc ./...` → format as llms.txt; `go doc -all ./...` → llms-full.txt
  - Python: `pydoc-markdown` or equivalent
  - Other: see AGENTS.md `## Build Commands` for the doc-gen tool

Update both `llms.txt` and `llms-full.txt` together. Never update one without
the other.

#### Step 18 — Handoff (never skip)

Present to the user:

```
Task {task-id} complete. All acceptance criteria pass.
Documentation updated: AGENTS.md, GLOSSARY.md, ONBOARDING.md, ROADMAP.md, NEXT_STEPS.md.

If all P{n} tasks are now Status: Done or Blocked → invoke the `run-phase-gate`
skill for phase {n}.
Otherwise → proceed to the next task in ROADMAP.md.
```

**This step is mandatory.** Phase gates will be silently skipped if this
handoff is omitted.

## Decision Points

- **Task doc not found:** ask the user to confirm the task ID before searching.
- **Baseline tests already failing:** document in Session Notes; do not stop —
  proceed while noting which failures are pre-existing.
- **Out-of-Scope item tempts implementation:** stop; note it as a new stub
  task in ROADMAP.md; do not implement it.
- **AST cache available:** use explore-codebase skill for orientation (Step 4)
  before reading raw files.
- **bail-out triggered (Step 8):** never proceed to a 4th approach; always
  stub + yield.
- **No write-adr skill yet:** inline ADR at Step 15 — do not skip the decision record.
- **No update-llms skill yet:** run doc-gen manually at Step 17 — do not skip.

## Anti-Patterns

- **Don't** skip the baseline run (Step 3) — pre-existing failures must be
  documented before you write a single line.
- **Don't** implement across multiple untested logical units before running
  tests — accumulation makes failures hard to attribute.
- **Don't** mark Status: Done before all acceptance criteria pass.
- **Don't** skip NEXT_STEPS.md (Step 16) — this is the session log; skipping
  it breaks the audit trail.
- **Don't** skip the handoff prompt (Step 18) — phase gates depend on it.
- **Don't** implement Out-of-Scope items, even small ones — create a stub task.
- **Don't** update only `llms.txt` without `llms-full.txt` — they stay in sync.

## Definition of Done

- [ ] All acceptance criteria from the task doc pass.
- [ ] Task Status set to Done.
- [ ] AGENTS.md, GLOSSARY.md, ONBOARDING.md, ROADMAP.md all updated.
- [ ] NEXT_STEPS.md session log entry appended.
- [ ] ADR written if a non-obvious design choice was made (Step 15).
- [ ] llms.txt + llms-full.txt updated if public API changed (Step 17).
- [ ] Handoff prompt presented to user (Step 18).

---

*Last updated: 2026-04-13*
