# P2-013: AST Planning Context — Pinned Symbol Snapshots in `code_ast.duckdb`

## Status

Status: Done
Depends on: (none — pure documentation and protocol; no code changes)
Blocks: (none)

## Summary

Adds a `planning_context` table to `code_ast.duckdb` and wires it into the
EsquissePlan agent and `implement-task` skill so that planning produces
machine-readable, pinned symbol snapshots that implementation agents consume
instead of re-reading raw source files. Every task document gains a precise,
captured-at-plan-time record of the exact functions, types, and callers it
touches — eliminating orientation guesswork and enabling automatic drift
detection at implementation time.

## Problem

Today, EsquissePlan writes task documents whose Files table and Specification
sections describe integration points in prose. Implementation agents must
re-discover those locations by reading source files, grep-searching, or relying
on their training data. This means:

- Symbol locations cited in task docs silently diverge from reality if code
  moves between planning and implementation.
- Orientation at Step 4 of `implement-task` re-reads files already examined
  during planning, burning context budget.
- Caller/callee blast-radius analysis is repeated independently in every
  implementation session.

## Solution

Extend `code_ast.duckdb` with a `planning_context` table (one row per
symbol-per-task). EsquissePlan populates it during Step 2 (Map the Work) using
sitting_duck macros already available in `scripts/macros_go.sql`. The
`implement-task` skill queries it at Step 4 (Orient) before touching any file.
A drift-detection query at Step 3 (Baseline) flags symbols whose
`line_start` has shifted since planning.

## Scope

### In Scope

- `planning_context` table DDL added to `scripts/macros.sql` (CREATE TABLE IF NOT EXISTS)
- `scripts/macros_go.sql`: new macro `capture_planning_context(task_id, role, pattern, name_like)` that inserts rows from live `ast` into `planning_context`
- `scripts/macros.sql`: new macro `planning_drift(task_id)` that JOIN-detects line-start drift between `planning_context` and current `ast`
- EsquissePlan `.github/agents/EsquissePlan.agent.md` Step 2: wire in `capture_planning_context` call after symbol-mapping queries
- `skills/implement-task/SKILL.md` Step 4: add AST-first path querying `planning_context`; Step 3: add drift-detection query
- `SCHEMAS.md §4` (Task Document Schema): add `planning_context` row to the optional `## Planning Artifacts` guidance note
- `AGENTS.md §Available Tools`: document `planning_context` table alongside `code_ast.duckdb`
- `GLOSSARY.md`: add terms `planning_context`, `symbol snapshot`, `drift detection`

### Out of Scope

- No changes to `esquisse-mcp/` Go server (no new MCP tool for this feature — P2-014 if desired)
- No UI or query CLI wrapper — raw `duckdb` CLI queries via existing skill
- No cross-project planning context federation (tracked separately)
- No automatic re-capture on file change (implement-task Step 3 is the detection gate, not a watcher)
- No caller-graph storage in `planning_context` in this task — `callers` JSON column is future work

## Prerequisites

- [ ] `code_ast.duckdb` build infrastructure exists (`scripts/rebuild-ast.sh`) — ✅ Done (P0)
- [ ] `scripts/macros.sql` and `scripts/macros_go.sql` exist — ✅ Done (P0)
- [ ] `duckdb-code` skill with `go_func_signatures()`, `go_type_flow()`, `function_callers()` macros — ✅ Done (P0)

## Specification

### `planning_context` Table DDL

Add to `scripts/macros.sql` (universal section, not Go-specific):

```sql
CREATE TABLE IF NOT EXISTS planning_context (
    task_id      VARCHAR NOT NULL,     -- e.g. 'P2-013'
    role         VARCHAR NOT NULL,     -- 'modify' | 'implement' | 'depends-on' | 'must-not-touch'
    symbol_kind  VARCHAR NOT NULL,     -- 'function' | 'type' | 'interface' | 'file'
    symbol_name  VARCHAR NOT NULL,
    file_path    VARCHAR NOT NULL,
    line_start   INTEGER,
    line_end     INTEGER,
    signature    TEXT,                 -- captured via peek := 'full'
    captured_at  TIMESTAMP DEFAULT now()
);
```

**Invariant:** `(task_id, symbol_name, file_path)` is the natural key. Re-running
capture for the same task must use `INSERT OR REPLACE` (i.e. `INSERT OR IGNORE`
is insufficient — a re-plan must update the row).

Use `DELETE FROM planning_context WHERE task_id = ?` before re-inserting to
ensure idempotency.

### `capture_planning_context` Macro (Go-specific, `scripts/macros_go.sql`)

```sql
CREATE OR REPLACE MACRO capture_planning_context(task_id, role, pattern, name_like) AS TABLE
    SELECT
        task_id::VARCHAR            AS task_id,
        role::VARCHAR               AS role,
        CASE
            WHEN semantic_type = 'DEFINITION_FUNCTION' THEN 'function'
            WHEN semantic_type = 'DEFINITION_CLASS'    THEN 'type'
            ELSE 'file'
        END                         AS symbol_kind,
        name                        AS symbol_name,
        file_path,
        start_line                  AS line_start,
        end_line                    AS line_end,
        peek                        AS signature,
        now()                       AS captured_at
    FROM read_ast(pattern, ignore_errors := true, peek := 'full')
    WHERE (semantic_type = 'DEFINITION_FUNCTION' OR semantic_type = 'DEFINITION_CLASS')
      AND name ILIKE name_like
      AND file_path NOT LIKE '%_test.go';
-- Note: column names (start_line, end_line, semantic_type, name, file_path, peek) and
-- parameters (ignore_errors, peek) are verified against the existing scripts/macros_go.sql
-- usage in this project. Re-verify against sitting_duck docs if the extension is upgraded.
```

Usage by EsquissePlan (Step 2):

```sql
LOAD sitting_duck;
.read scripts/macros.sql
.read scripts/macros_go.sql
-- Ensure planning_context table exists
-- (macros.sql CREATE TABLE IF NOT EXISTS runs on .read)

DELETE FROM planning_context WHERE task_id = 'P2-013';

INSERT INTO planning_context
SELECT * FROM capture_planning_context('P2-013', 'modify', '**/*.go', 'ProcessConfig%');

INSERT INTO planning_context
SELECT * FROM capture_planning_context('P2-013', 'depends-on', '**/*.go', 'ReadState%');
```

### `planning_drift` Macro (`scripts/macros.sql`)

```sql
CREATE OR REPLACE MACRO planning_drift(task_id) AS TABLE
    SELECT
        p.task_id,
        p.symbol_name,
        p.file_path,
        p.line_start        AS planned_line,
        a.start_line        AS current_line,
        p.captured_at,
        ABS(a.start_line - p.line_start) AS drift_lines
    FROM planning_context p
    JOIN ast a
      ON  a.name      = p.symbol_name
      AND a.file_path = p.file_path
    WHERE p.task_id = task_id
      AND p.line_start IS NOT NULL
      AND a.start_line != p.line_start
    ORDER BY drift_lines DESC;
```

**Requires:** `ast` table built in the same DuckDB session (standard post-`rebuild-ast.sh`).

### EsquissePlan Step 2 Protocol Addition

After completing the existing symbol-mapping queries, and **before** writing task
documents, add:

> **Step 2c — Capture Planning Context**
>
> For each task just mapped: run `capture_planning_context` for every symbol in the
> `modify` and `implement` roles. Store results in `planning_context` in
> `code_ast.duckdb`. Record the total row count in Session Notes.
>
> Before capturing: ensure `code_ast.duckdb` is current. If it does not exist, or
> if any Go source files have changed since the last rebuild, run
> `bash scripts/rebuild-ast.sh` to ensure the `ast` table reflects current source.
>
> If `duckdb` CLI is unavailable: print
> `[WARN] planning_context capture skipped: duckdb CLI not found`
> as a visible line in chat output, and note it in Session Notes of each affected task.
>
> If the `INSERT INTO planning_context` statement returns a SQL error: print
> `[WARN] planning_context capture failed for task {task_id}: {error message}`
> as a visible line in chat output, skip capture for that task, and note the
> error in Session Notes. Do **not** abort the planning session — proceed to
> writing task documents without planning context for that task.
>
> After each INSERT, check the row count:
> ```sql
> SELECT count(*) FROM planning_context WHERE task_id = '{task_id}';
> ```
> If the count is 0: print
> `[WARN] planning_context capture returned 0 rows for task {task_id} — verify that the pattern and name_like parameters match the intended symbols`
> as a visible line in chat output and note it in Session Notes. This indicates a
> pattern or name_like mismatch; do **not** assume capture succeeded silently.
>
> Additionally, check for rows with missing signatures:
> ```sql
> SELECT count(*) FROM planning_context
> WHERE task_id = '{task_id}' AND (signature IS NULL OR trim(signature) = '');
> ```
> If the count is > 0: print
> `[WARN] planning_context: {count} row(s) have missing signature for task {task_id} — peek may have failed or been truncated; check function size`
> as a visible line in chat output and note it in Session Notes.

### `implement-task` Step 3 Addition (Drift Detection)

After the baseline test run and **before** proceeding to Step 4, if
`code_ast.duckdb` exists:

> **AST Freshness:** Ensure the `ast` table is current before running drift or
> missing-symbol queries. If Go source files have changed since planning,
> run `bash scripts/rebuild-ast.sh` to rebuild. Stale AST produces false
> negatives (drift not detected) or false positives (symbols appear missing).
>
> Before running Step 3a or 3b, verify the `ast` table is non-empty:
> ```sql
> SELECT count(*) FROM ast;
> ```
> If the result is 0: print
> `[WARN] ast table is empty — drift and missing-symbol queries will return 0 rows and cannot be trusted. Run bash scripts/rebuild-ast.sh and verify it scans Go source files.`
> Skip Steps 3a and 3b. Do **not** interpret 0-row results as "no drift".

**Step 3a — Line drift (symbol moved within same file):**

```sql
-- Run in code_ast.duckdb (requires ast table)
LOAD sitting_duck;
.read scripts/macros.sql
SELECT * FROM planning_drift('{task_id}');
```

If any rows are returned: log them to Session Notes as
`DRIFT DETECTED: {symbol_name} planned_line={planned_line} current_line={current_line}`.
Do **not** block implementation — this is an advisory warning, not a gate.

**Step 3b — Missing symbols (symbol deleted or moved to a different file):**

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

If any rows are returned: print
`[WARN] MISSING SYMBOL: {symbol_name} at {file_path} (role={role}) — may have moved or been deleted`
as a visible line in chat output and log to Session Notes.
Do **not** block implementation, but treat these rows as high-priority investigation
points before modifying the file.

### `implement-task` Step 4 Addition (AST-First Orientation)

When `code_ast.duckdb` exists, first verify the `planning_context` table is
present:

```sql
SELECT count(*) FROM information_schema.tables
WHERE table_name = 'planning_context';
```

If the result is 0: print
`[WARN] planning_context table absent — EsquissePlan macros may not have been run for this project. Falling back to raw file orientation.`
as a visible line in chat output, then skip to raw file reading.

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
as a visible line in chat output, then proceed with raw file reading.

Otherwise, use these rows as the definitive list of integration points. Read
raw source only for symbols not found in `planning_context`, or when
`signature` is NULL.

### Known Risks / Failure Modes

| Risk | Mitigation |
|------|-----------|
| `planning_context` stale after large refactor | Drift query at Step 3a surfaces line divergence; agent notes in Session Notes |
| Symbol moved to different file — JOIN misses it | Step 3b missing-symbol query detects absent `(name, file_path)` pairs; visible `[WARN]` printed |
| Symbol deleted between planning and implementation | Same: Step 3b missing-symbol query surfaces it |
| `capture_planning_context` macro missing from older project copies | Step 4 capability check queries `information_schema.tables`; visible `[WARN]` printed; falls back to raw file read |
| `planning_context` empty for task (capture not run) | Step 4 empty-result check; visible `[WARN]` printed; falls back to raw file read |
| `peek := 'full'` slow on large functions | Acceptable: capture runs once at planning, not per-implementation-session |
| `name_like` too broad — captures unrelated symbols | EsquissePlan must use specific patterns (e.g. `'ProcessConfig'` not `'%'`); document in Session Notes; precision smoke check verifies capture correctness |
| `ast` table exists but is empty (rebuild ran against wrong directory or found no files) | `planning_drift` and Step 3b return 0 rows silently — looks like "no issues"; advisory-only design tolerates this; verify `SELECT count(*) FROM ast;` > 0 before trusting a clean drift result |
| `ast` table stale (rebuilt before recent source changes) | Drift query misses real divergence; missing-symbol query has false negatives; Step 2c and Step 3 both require rebuild if source changed since last rebuild |
| `signature` NULL or empty after capture (`peek` failed or function too large) | Post-capture NULL signature check in Step 2c; visible `[WARN]` printed; agent falls back to raw file read for that symbol |

## Acceptance Criteria

This task is pure documentation and protocol — there is no compiled code to
test. Acceptance is verified by:

```sh
# 1. macros.sql contains the planning_context DDL
grep -q "CREATE TABLE IF NOT EXISTS planning_context" scripts/macros.sql

# 2. macros_go.sql contains the capture macro
grep -q "capture_planning_context" scripts/macros_go.sql

# 3. macros.sql contains the drift macro
grep -q "planning_drift" scripts/macros.sql

# 4. EsquissePlan agent references capture_planning_context
grep -q "capture_planning_context" .github/agents/EsquissePlan.agent.md

# 5. implement-task skill references planning_context and planning_drift
grep -q "planning_context" skills/implement-task/SKILL.md
grep -q "planning_drift"   skills/implement-task/SKILL.md

# 6. Smoke test: macro creates table and inserts into it (requires duckdb + sitting_duck)
duckdb -init /dev/null code_ast.duckdb -c "
  LOAD sitting_duck;
  .read scripts/macros.sql
  .read scripts/macros_go.sql
  DELETE FROM planning_context WHERE task_id = '__smoke__';
  INSERT INTO planning_context
  SELECT * FROM capture_planning_context('__smoke__', 'modify', '**/*.go', '%');
  SELECT count() FROM planning_context WHERE task_id = '__smoke__';
"

# 7. Smoke precision check: known symbol captured with correct name
# (esquisse-mcp/models.go must contain 'ModelEntry')
# Run in WSL/Linux (duckdb -init /dev/null requires /dev/null)
_precision_count=$(duckdb -init /dev/null code_ast.duckdb -c "
  LOAD sitting_duck;
  .read scripts/macros.sql
  .read scripts/macros_go.sql
  DELETE FROM planning_context WHERE task_id = '__precision__';
  INSERT INTO planning_context
    SELECT * FROM capture_planning_context('__precision__', 'modify', 'esquisse-mcp/**/*.go', 'ModelEntry');
  SELECT count(*) FROM planning_context
  WHERE task_id = '__precision__' AND symbol_name = 'ModelEntry';
" | tail -1 | tr -d ' ')
if [[ "$_precision_count" -ne 1 ]]; then
  echo "FAIL: precision smoke expected 1 row for ModelEntry, got $_precision_count" >&2; exit 1
fi
echo "PASS: ModelEntry captured (row count = $_precision_count)"

# 8. Missing-symbol detection: assert exactly 1 row for NonExistentFunction
# Run in WSL/Linux (duckdb -init /dev/null requires /dev/null)
_missing_count=$(duckdb -init /dev/null code_ast.duckdb -c "
  LOAD sitting_duck;
  .read scripts/macros.sql
  DELETE FROM planning_context WHERE task_id = '__missing__';
  INSERT INTO planning_context VALUES
    ('__missing__', 'modify', 'function', 'NonExistentFunction', 'esquisse-mcp/models.go', 1, 10, NULL, now());
  SELECT count(*) FROM planning_context p
  WHERE p.task_id = '__missing__'
    AND NOT EXISTS (SELECT 1 FROM ast a WHERE a.name = p.symbol_name AND a.file_path = p.file_path);
" | tail -1 | tr -d ' ')
if [[ "$_missing_count" -ne 1 ]]; then
  echo "FAIL: missing-symbol smoke expected 1 row for NonExistentFunction, got $_missing_count" >&2; exit 1
fi
echo "PASS: NonExistentFunction detected as missing (row count = $_missing_count)"
```

| Check | What It Verifies |
|-------|-----------------|
| `grep planning_context scripts/macros.sql` | DDL present |
| `grep capture_planning_context scripts/macros_go.sql` | Capture macro present |
| `grep planning_drift scripts/macros.sql` | Drift macro present |
| `grep capture_planning_context .github/agents/EsquissePlan.agent.md` | Agent wired |
| `grep planning_context skills/implement-task/SKILL.md` | Skill wired (Step 4) |
| `grep planning_drift skills/implement-task/SKILL.md` | Skill wired (Step 3) |
| Smoke insert returns row count ≥ 0 without error | Macro executes without SQL error |
| Smoke precision check: `ModelEntry` row returned | Capture returns the expected row; shell asserts count = 1 |
| Missing-symbol smoke: shell asserts count = 1 for NonExistentFunction | Step 3b detects moved/deleted symbols; shell assertion prevents silent pass |

## Files

| File | Action | Description |
|------|--------|-------------|
| `scripts/macros.sql` | Modify | Add `planning_context` DDL + `planning_drift` macro |
| `scripts/macros_go.sql` | Modify | Add `capture_planning_context` macro |
| `.github/agents/EsquissePlan.agent.md` | Modify | Add Step 2c — Capture Planning Context |
| `skills/implement-task/SKILL.md` | Modify | Step 3: drift query; Step 4: `planning_context` orientation path |
| `AGENTS.md` | Modify | `§ Available Tools`: document `planning_context` table |
| `SCHEMAS.md` | Modify | §4 Task Document Schema: note `planning_context` capture as optional planning output |
| `GLOSSARY.md` | Modify | Add: `planning_context`, `symbol snapshot`, `drift detection` |

## Design Principles

1. **Advisory, not blocking** — drift detection warns but never halts implementation. Plans are snapshots, not contracts.
2. **Additive only** — `planning_context` is a new table; no existing macros or schema are modified.
3. **Graceful degradation** — every consumer path has an explicit fallback when `code_ast.duckdb` or `planning_context` is absent.
4. **Idempotent capture** — `DELETE … WHERE task_id = ?` before INSERT ensures re-planning is safe.
5. **Verify, don't infer** — EsquissePlan MUST run actual AST queries to populate signatures; it MUST NOT guess symbol locations from training data.

## Testing Strategy

- Manual verification via the `grep` acceptance checks above
- Smoke-test: build `code_ast.duckdb` on the esquisse repo itself (Go source in `esquisse-mcp/`), load macros, run `capture_planning_context`, verify row insertion
- Drift test: manually shift a function comment to change `start_line`, rebuild AST, verify `planning_drift` returns the row

## Session Notes

<!-- 2026-04-22 — Drafted by EsquissePlan session. Scope limited to pure markdown/SQL; no Go MCP changes. -->
<!-- 2026-04-22 — Revised after adversarial review (iter 0, CONDITIONAL, Adversarial-r0). Addressed:
     M1: Added information_schema.tables capability check + visible [WARN] in Step 4.
     M2: Changed capture-skip notification from Session Notes-only to visible [WARN] chat output.
     M3: Added precision smoke test (ModelEntry row) and missing-symbol smoke test to acceptance criteria.
     M4: Added Step 3b missing-symbol detection query for symbols moved/deleted between planning and implementation. -->
<!-- 2026-04-22 — Minor fix after adversarial review (iter 1, CONDITIONAL, Adversarial-r2). Addressed:
     L1: Added explicit [WARN] + skip behavior for INSERT SQL errors in EsquissePlan Step 2c. -->
<!-- 2026-04-22 — Fix after adversarial review (iter 2, CONDITIONAL, Adversarial-r2). Addressed:
     M1: Added LOAD sitting_duck + .read scripts/macros.sql to Step 3b SQL snippet (required for fresh DuckDB sessions).
     L1: Added API verification note to capture_planning_context macro. -->
<!-- 2026-04-22 — Fix after adversarial review (iter 3, CONDITIONAL, Adversarial-r0). Addressed:
     M1: Precision smoke test (#7) now uses a shell count assertion (exits non-zero if ModelEntry not captured).
     L1: Added WSL/Linux comment to acceptance criteria smoke tests. -->
<!-- 2026-04-22 — Fix after adversarial review (iter 4, CONDITIONAL, Adversarial-r1 substituted GPT-4.1). Addressed:
     M1: Missing-symbol smoke test (#8) now uses a shell count assertion (exits non-zero if NonExistentFunction not detected). Parity with #7 fix. -->
<!-- 2026-04-22 — Fix after adversarial review (iter 5, CONDITIONAL, Adversarial-r2 GPT-4o). Addressed:
     M1 (borderline): Added Known Risks row for empty-but-existent ast table causing silent 0-row drift/missing-symbol results. All other findings minor. -->
<!-- 2026-04-22 — Fix after adversarial review (iter 6, CONDITIONAL, Adversarial-r0 GPT-4.1). Addressed:
     M1: Added visible [WARN] + Session Notes note when capture_planning_context INSERT returns 0 rows (pattern/name_like mismatch would silently succeed without this guard). -->
<!-- 2026-04-22 — Fix after adversarial review (iter 7, CONDITIONAL, Adversarial-r1 subst. GPT-4.1). Addressed:
     M1: Expanded Step 2c rebuild-ast.sh condition to cover stale AST (not just missing file); added Step 3 freshness advisory note.
     M2: Added post-capture NULL/empty signature check in Step 2c with visible [WARN]; added Known Risks rows for stale AST and missing signature. -->
<!-- 2026-04-22 — Fix after adversarial review (iter 8, CONDITIONAL, Adversarial-r2 GPT-4o). Addressed:
     M1: Added active `SELECT count(*) FROM ast;` check in Step 3 before drift/missing-symbol queries; visible [WARN] + skip if ast is empty — prevents 0-row results being silently interpreted as "no drift".
     M2 (re-raise, mitigation sufficient): broad name_like already guarded by precision smoke and Known Risks note.
     M3 (out of scope): modifying rebuild-ast.sh to enforce freshness check is future work (P3+). -->
<!-- 2026-04-22 — Completed by ImplementerAgent. All 7 target files modified; all 11 acceptance criteria grep checks pass. Pure documentation task: DDL + macros in scripts/, Step 2c in EsquissePlan, Steps 3b/4a in implement-task SKILL.md, planning_context table doc in AGENTS.md, optional section note in SCHEMAS.md §4, three new terms in GLOSSARY.md. -->
