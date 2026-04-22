# External Session Prompt: Update `bof` to Use AST Planning Context

**Run this prompt in a fresh session inside the `bof` project root.**
**Prerequisite:** Esquisse P2-013 (`ast-planning-context`) must be merged and
`scripts/macros.sql` + `scripts/macros_go.sql` in `bof`'s copy must be
up to date (run `scripts/upgrade.sh` if bof uses Esquisse-managed macros).

---

## Context

The Esquisse framework has added a `planning_context` table to `code_ast.duckdb`
(task P2-013). It stores pinned symbol snapshots produced by EsquissePlan during
planning: exact `file_path`, `line_start`, `line_end`, and full function/type
`signature` for every symbol a task is expected to touch. Two companion macros
were added to `scripts/macros.sql` and `scripts/macros_go.sql`:

- `capture_planning_context(task_id, role, pattern, name_like)` — inserts rows
  from the live `ast` table into `planning_context` for a given task
- `planning_drift(task_id)` — returns symbols whose `line_start` has shifted
  since the snapshot was captured

The `duckdb-code` global skill has been updated to document this table and its
usage. The bof skills listed below need equivalent updates so that bof planning
and implementation sessions take advantage of this feature.

---

## Your Task

Update the following three `bof` skill files. For each file, the change is
**additive only** — do not remove or restructure existing content.

---

### 1. `skills/writing-plans/SKILL.md`

**Location:** The `## File Structure Mapping` section, under the
`If code_ast.duckdb exists (do this first):` block.

**Add after the existing 4 bullet-point questions**, as a new subsection:

```markdown
#### Step 2c — Capture Planning Context (Esquisse P2-013+)

After the symbol-mapping queries, if `scripts/macros_go.sql` contains
`capture_planning_context` (Esquisse P2-013+), capture a snapshot for each task:

```sql
LOAD sitting_duck;
.read scripts/macros.sql
.read scripts/macros_go.sql

-- Re-plan is safe: DELETE clears the old snapshot first
DELETE FROM planning_context WHERE task_id = '{task_id}';

INSERT INTO planning_context
SELECT * FROM capture_planning_context('{task_id}', 'modify',   '**/*.go', '{symbol_pattern}');

INSERT INTO planning_context
SELECT * FROM capture_planning_context('{task_id}', 'depends-on', '**/*.go', '{dep_pattern}');
```

Record the row count in Session Notes. If the macro is absent (older project
copy), skip this step and note `planning_context capture unavailable` in Session Notes.
```

---

### 2. `skills/implement-task/SKILL.md`

**Two changes:**

#### Change A — Step 3 (Baseline), add drift detection

After the baseline test run, add:

```markdown
**If `code_ast.duckdb` exists and `planning_context` is populated for this task:**

```sql
LOAD sitting_duck;
.read scripts/macros.sql
SELECT * FROM planning_drift('{task_id}');
```

If any rows are returned, log them to Session Notes:
`DRIFT DETECTED: {symbol_name} planned_line={planned_line} current_line={current_line}`
This is advisory — do not block implementation. It signals that the task doc's
line references are stale and the agent should re-verify locations before editing.
```

#### Change B — Step 4 (Orient), add planning_context fast path

Prepend to the existing `If code_ast.duckdb exists:` branch:

```markdown
**If `planning_context` table exists for this task (Esquisse P2-013+):**

```sql
LOAD sitting_duck;
SELECT role, symbol_kind, symbol_name, file_path, line_start, line_end
FROM planning_context
WHERE task_id = '{task_id}'
  AND role IN ('modify', 'implement')
ORDER BY role, file_path, line_start;
```

Use these rows as the definitive integration-point list. Read raw source only
for symbols absent from `planning_context` or when `signature` is NULL.
Verify with `SELECT count() FROM planning_context WHERE task_id = '{task_id}'`
first — if 0 rows, fall through to the standard AST orientation queries below.
```

---

### 3. `skills/explore-codebase/SKILL.md`

**Location:** Level 2 (Structure) — the `AST-first mode` branch.

**Add a note** after the existing orientation queries, before the Level 3 section:

```markdown
**Planning context check (Esquisse P2-013+):**
If the user's question is about a specific task (e.g. "what does P2-013 touch?"),
check `planning_context` first — it may already contain the answer without
re-querying the full `ast`:

```sql
LOAD sitting_duck;
SELECT role, symbol_kind, symbol_name, file_path, line_start
FROM planning_context
WHERE task_id = '{task_id}'
ORDER BY role, file_path;
```

If `planning_context` has no rows for the task, proceed with standard orientation
queries (the task predates P2-013 or was planned without `duckdb` available).
```

---

## Verification

After making all three edits, run:

```sh
grep -q "capture_planning_context" skills/writing-plans/SKILL.md && echo "✅ writing-plans"  || echo "❌ writing-plans"
grep -q "planning_context"         skills/implement-task/SKILL.md && echo "✅ implement-task" || echo "❌ implement-task"
grep -q "planning_drift"           skills/implement-task/SKILL.md && echo "✅ drift"          || echo "❌ drift"
grep -q "planning_context"         skills/explore-codebase/SKILL.md && echo "✅ explore"      || echo "❌ explore"
```

All four lines should print ✅.

---

## Guardrails

- **Do not change** the existing `code_ast.duckdb` cache-build or macro-load
  patterns — this is an additive layer on top of them.
- **Do not add** `planning_context` capture to `explore-codebase` — that skill
  is read-only. Capture belongs only in `writing-plans`.
- **Do not remove** the `grep_search` fallback paths — `planning_context` is
  only available when P2-013 macros are present.
- If `bof`'s `scripts/macros.sql` is managed independently of Esquisse (not a
  symlink), also add the `planning_context` DDL and `planning_drift` macro from
  Esquisse `scripts/macros.sql`, and the `capture_planning_context` macro from
  Esquisse `scripts/macros_go.sql`.
- After edits, update `NEXT_STEPS.md` in bof with a session log entry.
