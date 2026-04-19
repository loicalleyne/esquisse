---
name: explore-codebase
description: >
  First-contact codebase orientation. USE when a user wants to understand a
  codebase, find where something lives, map architecture, or answer structural
  questions before starting implementation. Triggers on: "explore the codebase",
  "orient me", "where is X", "how does Y work", "map the architecture", "what
  calls Y", "help me understand this project".
  DO NOT invoke this skill when: the user wants to implement a specific task
  (use implement-task); the user has a bug to fix (use debug-issue);
  the user wants to write a feature spec (use write-spec).
---

# Explore-Codebase

## Overview

Three-level progressive orientation: Landscape → Structure → Hot Path.
Stops as soon as the user's question is answered. Never reads more than
necessary. Prefers AST queries over raw file reads when a cache exists.

**Announce at start:** "I'm using the explore-codebase skill."

## Prerequisites

- [ ] The user has a specific question or orientation goal. Ask if unclear.

**Tools:** `read_file`, `file_search`, `grep_search`, `run_in_terminal`

## Workflow

### Step 1: Validate Context

Run the following checks in order:

1. **Check `duckdb` CLI availability:** run `command -v duckdb` in terminal.
   - **Not found:** File-scan mode for this session. Announce: "duckdb not available — using file-scan mode."
   - **Found:** continue to step 2.

2. **Check `code_ast.duckdb` at project root:**
   - **Exists:** AST-first mode. Announce: "AST cache found — using AST-first mode."
   - **Missing:** Ask the user: "No AST cache found. Run `bash scripts/rebuild-ast.sh` to build it now? (yes / no / skip)"
     - **Yes:** run `bash scripts/rebuild-ast.sh` in terminal. On success, switch to AST-first mode and announce: "AST cache built — using AST-first mode."
     - **No / skip / script missing:** File-scan mode. Announce: "No AST cache — using file-scan mode."

### Level 1 — Landscape (always run)

Read these files in parallel. Stop reading a file once the relevant section is found.

1. `AGENTS.md` — extract: language/runtime, architecture pattern, key types,
   top-level directory layout, invariants.
2. `ONBOARDING.md` (if it exists) — extract: mental model, key files map,
   data flow, read-order for new tasks.
3. `llms.txt` (if it exists) — extract: public API surface, key types,
   key functions.

**Report from Level 1:**
- Language and runtime
- Architecture pattern (e.g. "pipeline: Kafka → transcoder → DuckDB")
- Top-level packages/directories and their roles
- Key types (from llms.txt or ONBOARDING.md Key Files Map)

If the user's question is answered at Level 1, stop here.

### Level 2 — Structure (if question is structural)

For questions like: "What implements this interface?",
"What types exist?", "Where is X defined?"

**AST mode:**
```sql
-- In DuckDB with sitting_duck loaded:
SELECT name, kind, file_path, line FROM ast
WHERE name ILIKE '%{symbol}%'
  AND kind IN ('function', 'type', 'interface', 'class', 'method')
ORDER BY file_path, line
LIMIT 30;
```
Use the `duckdb-code` skill for complex structural queries (call graphs,
interface implementations, type flows).

**File-scan mode:**
- Read `llms-full.txt` if it exists — scan for the symbol.
- Otherwise: `grep_search` for the symbol with `isRegexp: false`.
- Read the identified file's header/interface section only.

**Report from Level 2:**
- Type hierarchy or interface implementations found
- Key function signatures
- File locations for the most relevant symbols

If the user's question is answered at Level 2, stop here.
Maximum 2 raw file reads before switching to AST queries (if cache exists).

### Level 3 — Hot Path (if question is about a specific flow)

For questions like: "How does X get from A to B?",
"What happens when the user calls Y?", "Trace this execution."

**AST mode:**
Use the `duckdb-code` skill Workflow B (execution flow):
- Trace call graph from entry point.
- Map data flow through key functions.

**File-scan mode:**
- Read the specific file identified in Level 2.
- Follow function calls one level deep.

**Report from Level 3:**
- Data flow diagram (ASCII or bullet list)
- Key functions in the path with their file locations
- Performance constraints or concurrency notes from AGENTS.md

### Step 2: Answer

Directly answer the user's original question using findings from the level(s)
reached. If a further specific read is needed to answer precisely, do it now.
Do not ask the user to read files themselves.

## Decision Points

- **Question answered at Level 1:** stop. Do not proceed to Level 2.
- **"What implements X" or "What calls Y":** go directly to Level 2/AST. Skip Level 1 report.
- **AST cache exists but query returns no results:** check spelling, then fall back to File-scan Level 2.
- **No ONBOARDING.md:** infer mental model from AGENTS.md + directory listing.
- **No llms.txt:** infer API surface from exported symbols via grep or AST.

## Anti-Patterns

- **Don't** read more than 2 raw files before using AST queries (if cache exists) —
  AST queries are 3–5× faster and preserve token budget.
- **Don't** read generated files (`gen/`, `*.pb.go`, `*_gen.go`) for orientation —
  they describe generated code, not design intent.
- **Don't** read test files for orientation — they describe what's wrong, not what is.
- **Don't** report file paths without context — always context each path with its role.
- **Don't** ask the user to go read a file — answer the question directly.

## Definition of Done

- [ ] User's original question is answered.
- [ ] Key findings summarised: language, architecture pattern, relevant types/files.
- [ ] If a hot-path trace was requested: data flow described with entry → transform → exit.

---

*Last updated: 2026-04-13*
