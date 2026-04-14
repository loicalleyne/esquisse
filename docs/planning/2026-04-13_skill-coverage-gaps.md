# Plan: Complete Esquisse Conversational Skill Coverage

**Date:** 2026-04-13
**Status:** Ready
**Adversarial review history:** 4 iterations (CONDITIONAL × 3 → PASSED); all prior blocking/major issues resolved.
**Pre-implementation change (iter 3 fix):** SCHEMAS.md §6 frontmatter template updated with `tools_required` validation rules and a `not:` trigger guard field; `updated:` date field added.

---

## Goal

Every stage of the Esquisse workflow must have a conversational skill so an agent
can be told "use the X skill" and know exactly what to do. SCHEMAS.md §6 defines
a standard skills table that five skills must fill. Additional workflow stages have
no skill and no script coverage.

---

## Workflow Map: What Exists vs. What Is Missing

```
PROJECT LIFECYCLE
  New project         → init-project            ✅ skills/init-project/SKILL.md
  Existing project    → adopt-project           ✅ skills/adopt-project/SKILL.md

DESIGN
  Feature spec        → write-spec              ⚠️  Planned, NOT IMPLEMENTED

PLANNING
  Decompose spec      → EsquissePlan (agent)    ✅ .github/agents/EsquissePlan.agent.md
  Single task         → new-task                ✅ skills/new-task/SKILL.md
  Plan review         → adversarial-review      ✅ skills/adversarial-review/SKILL.md

IMPLEMENTATION
  Orient/navigate     → explore-codebase        ❌ MISSING (in SCHEMAS.md §6 standard table)
  Execute a task      → implement-task          ❌ MISSING (in SCHEMAS.md §6 standard table)
  Fix a bug           → debug-issue             ❌ MISSING (in SCHEMAS.md §6 standard table)
  Refactor safely     → refactor-safely         ❌ MISSING (in SCHEMAS.md §6 standard table)

REVIEW
  Review changes/PR   → review-changes          ❌ MISSING (in SCHEMAS.md §6 standard table)

PHASE LIFECYCLE
  Run gate → promote  → run-phase-gate          ❌ MISSING (gate-check.sh exists, no wrapper)

DOCUMENTATION MAINTENANCE
  Record decision     → write-adr               ❌ MISSING (ADR schema in SCHEMAS.md §5)
  Update API ref      → update-llms             ❌ MISSING (required by phase gate + completion protocol)
```

**Total gap: 8 skills missing (5 are listed as mandatory in SCHEMAS.md §6).**

---

## Protocol-Step → Skill Coverage Map

Every step of the FRAMEWORK.md §9 Per-Task Startup and Completion Protocols must be
assigned to a skill. The table below makes that mapping explicit so no step falls
through the gaps. Steps marked **embedded** are sub-steps inside `implement-task`
and do not warrant a standalone skill.

### Per-Task Startup Protocol (§9, 11 steps)

| # | Protocol Step | Owning Skill | Notes |
|---|---|---|---|
| 1 | Read AGENTS.md | `implement-task` Step 2 | embedded |
| 2 | Read GLOSSARY.md | `implement-task` Step 2 | embedded |
| 3 | Read task doc (In Scope, Out of Scope, AC, Files) | `implement-task` Step 2 | embedded |
| 4 | Establish baseline test run | `implement-task` Step 3 | embedded |
| 5 | Locate files (AST or grep) | `implement-task` Step 4 / `explore-codebase` | AST-heavy queries → `explore-codebase` |
| 6 | Read relevant function headers | `implement-task` Step 4 / `explore-codebase` | deeper exploration → `explore-codebase` |
| 7 | Read acceptance criteria carefully | `implement-task` Step 2 | embedded |
| 8 | State all ASSUMPTION comments | `implement-task` Step 5 | embedded |
| 9 | Determine edit order | `implement-task` Phase 2 | embedded |
| 10 | Implement (types → code → tests → docs) | `implement-task` Phase 2 | embedded |
| 11 | Verify acceptance criteria | `implement-task` Step 9 | embedded |

### Per-Task Completion Protocol (§9, 10 steps)

| # | Protocol Step | Owning Skill | Notes |
|---|---|---|---|
| 1 | Set task Status: Done | `implement-task` Step 10 | embedded |
| 2 | Update AGENTS.md Common Mistakes | `implement-task` Step 11 | embedded |
| 3 | Update GLOSSARY.md | `implement-task` Step 12 | embedded |
| 4 | Update ONBOARDING.md | `implement-task` Step 13 | embedded |
| 5 | Update ROADMAP.md (tick task Done) | `implement-task` Step 14 | embedded |
| 6 | Write ADR if decision was made | `write-adr` (invoked by implement-task Step 15) | delegated |
| 7 | Update NEXT_STEPS.md session log | `implement-task` Step 16 | embedded |
| 8 | Update llms.txt / llms-full.txt | `update-llms` (invoked by implement-task Step 17) | delegated if API changed |
| 9 | Run phase gate if phase is complete | `run-phase-gate` | user-initiated; `implement-task` Step 18 prompts the user |
| 10 | Prepare PR / handoff | `review-changes` | user-initiated; `implement-task` Step 18 prompts the user |

**Coverage: all 21 protocol steps are assigned.** Steps 1–8 of Startup and 1–7 of
Completion are embedded in `implement-task`. Steps 9–10 of Completion are
user-initiated but NOT silently skipped — `implement-task` Step 18 presents an
explicit handoff prompt to the user at task close.

---

## Required Skill Structure

Every SKILL.md created from this plan **must** include the following frontmatter
and body sections, as required by SCHEMAS.md §6. The abbreviated specs below
define workflow steps only — implementers must embed those steps inside this
mandatory structure.

### Frontmatter Template

```yaml
---
name: {skill-name}
description: >
  Concise trigger sentence(s) in natural user language. Include at minimum one
  explicit DO NOT USE case to prevent false-positive invocations on similar phrases
  that should route to a different skill.
triggers:
  - phrase: "{primary trigger phrase}"
  - phrase: "{secondary trigger phrase}"
  - not: "{phrase that must NOT trigger this skill — specify the correct skill}"
tools_required:
  - read_file
  - replace_string_in_file   # if skill writes files
  - run_in_terminal          # if skill invokes scripts
  - runSubagent              # if skill delegates to agents
updated: {YYYY-MM-DD}
---
```

### Mandatory Body Sections

Every SKILL.md body must contain these sections in this order:

1. `**Announce at start:** "I'm using the {skill-name} skill."` — first content line
2. `## Prerequisites` — explicit checklist of pre-conditions that must be true
3. `## Steps` — the numbered workflow from the abbreviated spec
4. `## Decision Points` — named branches (e.g., "If AST cache exists: ... else: ...")
5. `## Anti-Patterns` — 3–5 explicit DON'T instructions
6. `## Definition of Done` — the exit condition; must include any handoff prompt

**Trigger invariant:** A skill with no `triggers:` frontmatter will not auto-invoke.
A `description:` paragraph that matches too broadly causes false positives. Each
skill below lists its false-positive guard; include it in the frontmatter `not:` field.

---

## Priority Order

| Priority | Skill | Rationale |
|---|---|---|
| P1 | `implement-task` | The most-used workflow in the framework. SCHEMAS.md standard. Every task session needs this. |
| P1 | `explore-codebase` | Required before implement-task. SCHEMAS.md standard. |
| P1 | `write-spec` | Unblocks EsquissePlan. Without it, PlanD receives vague inputs. Plan already exists. |
| P2 | `debug-issue` | Required by Hard Rule 13 (failing test first). SCHEMAS.md standard. |
| P2 | `run-phase-gate` | No conversational wrapper for phase promotion. Drives project lifecycle. |
| P2 | `review-changes` | Completes the build/review loop. SCHEMAS.md standard. |
| P3 | `write-adr` | Triggered by task completion protocol. Prevents agents from reversing past decisions. |
| P3 | `refactor-safely` | SCHEMAS.md standard. Less frequent than the above. |
| P3 | `update-llms` | Required at phase gate. Completeness item. |

---

## File Plan

### New Files (8 skills)

| File | Trigger phrases |
|------|----------------|
| `skills/write-spec/SKILL.md` | "write a spec for X", "I want to add a feature", "spec out X" |
| `skills/explore-codebase/SKILL.md` | "explore the codebase", "orient me", "where is X" |
| `skills/implement-task/SKILL.md` | "implement task X", "start working on P{n}-{nnn}", "execute this task" |
| `skills/debug-issue/SKILL.md` | "there's a bug", "debug this", "fix failing test", "reproduce this error" |
| `skills/run-phase-gate/SKILL.md` | "run the phase gate", "is phase N done", "promote to next phase" |
| `skills/review-changes/SKILL.md` | "review these changes", "review this PR", "check this diff" |
| `skills/write-adr/SKILL.md` | "record this decision", "write an ADR", "document why we chose X" |
| `skills/refactor-safely/SKILL.md` | "refactor this", "rename this type", "restructure without breaking" |
| `skills/update-llms/SKILL.md` | "update llms.txt", "update the API reference", "sync llms after phase gate" |

### Reference Files (skills with heavy docs)

| File | Parent skill |
|------|-------------|
| `skills/implement-task/references/startup-protocol.md` | implement-task |
| `skills/implement-task/references/completion-protocol.md` | implement-task |
| `skills/debug-issue/references/root-cause-guide.md` | debug-issue |
| `skills/run-phase-gate/references/gate-criteria.md` | run-phase-gate |

### Modified Files

| File | Change |
|------|--------|
| `scripts/init.sh` | Add all new skills to the copy loop |
| `FRAMEWORK.md §11` | Add all new rows to the Conversational Workflows table |
| `SCHEMAS.md §6` | Replace stub skill list with links to actual skill files |

---

## Skill Specifications

---

### 1. `skills/write-spec/SKILL.md` (P1)

**Full spec already written in `docs/planning/2026-04-13_write-spec-skill.md`.**  
Implement directly from that plan. No redesign needed.

**Triggers:** "write a spec for X", "I want to add a feature", "spec out X",
"help me design X", "write a feature specification"

**Not a trigger for this skill:** "implement task X" (→ `implement-task`),
"create a task document" (→ `new-task`), "write an ADR" (→ `write-adr`)

**Sync rule:** The abbreviated step list in this document is a summary only.
The authoritative spec is `docs/planning/2026-04-13_write-spec-skill.md`. If any
discrepancy is found during implementation, the plan doc wins. Do not
reconcile by editing this document — update the plan doc directly.

Key reference files:
- `skills/write-spec/references/spec-template.md`
- `skills/write-spec/references/interview-guide.md`
- `skills/write-spec/references/approach-options-guide.md`

---

### 2. `skills/explore-codebase/SKILL.md` (P1)

**Triggers:** "explore the codebase", "orient me in this project", "where is X",
"how does this work", "map the architecture", "what calls Y"

**Not a trigger for this skill:** "implement task P{n}-{nnn}" (→ `implement-task`),
"there's a bug" (→ `debug-issue`), "write a spec" (→ `write-spec`)

**Purpose:** First-contact codebase orientation. Produces a mental model before
implementation begins so the implement-task skill can read the minimum set of files.

**Steps:**

1. **Validate**: check if `code_ast.duckdb` exists. If yes: AST-first mode.
   If no: file-scan mode.

2. **Level 1 — Landscape** (always): read `AGENTS.md`, `ONBOARDING.md`, `llms.txt`.
   Report: language, architecture pattern, top-level packages, key types.

3. **Level 2 — Structure** (if question is structural):
   - AST mode: query `code_ast.duckdb` using `duckdb-code` skill:
     `find_definitions`, `go_interfaces`, `go_func_signatures`
   - Scan mode: read `llms-full.txt` or scan for exported symbols with grep
   Report: type hierarchy, interface implementations, key function signatures.

4. **Level 3 — Hot path** (if question is about a specific flow):
   - AST mode: Workflow B (execution flow) — trace call graph from entry point
   - Scan mode: read the specific file identified in Level 2
   Report: data flow diagram, key functions in the path, performance constraints.

5. **Answer the user's question** using findings. If further reads are needed,
   do them now — do not ask the user to read files themselves.

**Constraints:**
- Maximum 2 raw file reads before switching to AST queries (if cache exists).
- Do not read generated files (`gen/`, `*.pb.go`).
- Do not read test files for orientation (they describe what's wrong, not what is).

---

### 3. `skills/implement-task/SKILL.md` (P1)

**Triggers:** "implement task P{n}-{nnn}", "start working on {slug}", "execute
this task", "let's implement", "do the work for task X"

**Not a trigger for this skill:** "explore the codebase" (→ `explore-codebase`),
"there's a bug" (→ `debug-issue`), "write a spec for X" (→ `write-spec`)

**Purpose:** The central implementation workflow. Wraps the Task Startup Protocol
and Task Completion Protocol from FRAMEWORK.md §9 as a single structured skill.

**Reference files:**
- `skills/implement-task/references/startup-protocol.md` — verbatim from §9
- `skills/implement-task/references/completion-protocol.md` — verbatim from §9

**Steps:**

**Phase 1: Startup**

1. **Identify task**: locate `docs/tasks/P{n}-{nnn}-{slug}.md`. Ask if not specified.
2. **Load context**: read `AGENTS.md`, `GLOSSARY.md`, the task doc (In Scope,
   Out of Scope, Acceptance Criteria, Files table).
3. **Establish baseline**: run full test suite. Record pass/fail count.
   If tests already fail: document in Session Notes before writing any code.
4. **Orient**: for each file in the Files table:
   - If `code_ast.duckdb` exists: use AST queries to locate types/functions.
   - Otherwise: read file headers/interfaces only.
5. **State assumptions**: write any ambiguities to Session Notes before coding.

**Phase 2: Implementation**

6. **Edit order** (enforce strictly):
   1. Define interfaces and types first.
   2. Implement production code.
   3. Write tests.
   4. Update godoc / llms.txt if public API changed.
7. **Run tests** after each logical unit. Never accumulate multiple untested changes.
8. **Bail-out trigger**: if the same error persists after 3 meaningfully different
   approaches — STOP. Revert to stub, record in NEXT_STEPS.md under `## Blocked`,
   set task Status: Blocked, yield to user. (Hard Rule 16.)

**Phase 3: Completion**

9. **Verify acceptance criteria**: run the exact test commands from the task doc.
   Every criterion must be green.
10. **Update task doc**: Status: Done, Session Notes entry.
11. **Update AGENTS.md**: add any new Common Mistakes entries.
12. **Update GLOSSARY.md**: add any new domain terms.
13. **Update ONBOARDING.md**: if new packages or key files were created.
14. **Update ROADMAP.md**: tick the task ✅ Done, add any new task stubs.
15. **ADR trigger**: if "we chose X over Y because" appeared:
    - If `write-adr` skill is available: invoke it.
    - Otherwise (Batch 1–2 deploy): inline the ADR directly using SCHEMAS.md §5
      schema at `docs/decisions/ADR-{nnnn}-{slug}.md`.
16. **Update NEXT_STEPS.md**: session log entry.
17. **Update llms.txt/llms-full.txt**: only if a public API changed:
    - If `update-llms` skill is available: invoke it.
    - Otherwise (Batch 1–2 deploy): run the canonical doc-gen command for the
      project language (e.g. `go doc ./...` for Go) and update both `llms.txt`
      and `llms-full.txt` together.
18. **Handoff**: present to the user: "Task Done. If all P{n} tasks are now
    Status: Done or Blocked, invoke the `run-phase-gate` skill for phase {n}.
    Otherwise proceed to the next task."

**Constraints:**
- Never mark Status: Done before all acceptance criteria pass.
- Never skip the baseline run (Step 3).
- Never skip NEXT_STEPS.md update (Step 16).
- Never skip the handoff prompt (Step 18).

---

### 4. `skills/debug-issue/SKILL.md` (P2)

**Triggers:** "there's a bug", "debug this", "fix this failing test", "reproduce
this error", "test X is failing", "something is wrong with Y"

**Not a trigger for this skill:** "implement task P{n}-{nnn}" (→ `implement-task`),
"refactor this code" (→ `refactor-safely`), "run the phase gate" (→ `run-phase-gate`)

**Purpose:** Structured debugging following Hard Rule 13: failing test first,
then fix. Never fix without a reproducing test.

**Reference file:** `skills/debug-issue/references/root-cause-guide.md`

**Steps:**

1. **Describe the failure**: ask for the exact error message, test name, or
   observable symptom if not provided.

2. **Reproduce** (before touching any code):
   - Run the failing test / reproduce the error state.
   - Confirm failure is consistent. If flaky: document conditions and proceed.
   - Record baseline: what passes, what fails.

3. **Write regression test** (if no test exists yet — Hard Rule 13):
   - Create the minimal test that reproduces the failure.
   - Confirm the test is red (fails as expected).
   - Do NOT proceed to fix until the test is red.

4. **Trace root cause** using `root-cause-guide.md`:
   - Binary search: narrow to the function/file responsible.
   - Check: invariant violation? Nil pointer? Incorrect algorithm? Race condition?
   - If `code_ast.duckdb` exists: use call graph to trace how inputs reach the
     failing function.

5. **Fix** only after Step 4 identifies root cause:
   - Minimally invasive change. Do not refactor unrelated code.
   - Run the regression test. Must turn green.
   - Run full test suite. No regressions.

6. **3-attempt bail-out** (Hard Rule 16): if after 3 distinct approaches the test
   is still red, stop. Record in NEXT_STEPS.md `## Blocked`. Yield to user.

7. **Completion**:
   - Add to AGENTS.md `## Common Mistakes to Avoid` if the bug reveals a pattern.
   - Append Session Log to NEXT_STEPS.md.
   - If the fix required a non-obvious design decision: invoke `write-adr` skill.

**Constraints:**
- Never fix without a reproducing test (Hard Rule 13). No exceptions.
- Never change test expectations to match incorrect output (Hard Rule 1).
- Minimal change only — no refactoring in the same commit as a bug fix.

---

### 5. `skills/run-phase-gate/SKILL.md` (P2)

**Triggers:** "run the phase gate", "is phase N done", "can I promote to
the next phase", "check phase gate criteria", "phase N complete?"

**Not a trigger for this skill:** "run the tests" (→ CI/test runner directly),
"review these changes" (→ `review-changes`), "there's a bug" (→ `debug-issue`)

**Purpose:** Conversational wrapper for `scripts/gate-check.sh`. Runs all 8 gate
checks, interprets failures, guides the user to fix them, then updates ROADMAP.md
when the gate passes.

**Reference file:** `skills/run-phase-gate/references/gate-criteria.md`
(full description of each check with fixes for common failures)

**Steps:**

1. **Identify phase**: ask for the phase number if not specified. Read
   `docs/planning/ROADMAP.md` to confirm all tasks in that phase exist.

2. **Pre-flight check**: verify all P{n} task docs have Status: Done or Blocked.
   If any are In Progress or Ready: warn the user and ask to confirm before running.

3. **Run gate-check.sh**:
   ```sh
   bash scripts/gate-check.sh {phase}
   ```
   - **Linux / macOS / WSL:** `bash scripts/gate-check.sh {phase}`
   - **Windows (with WSL installed):** `wsl bash scripts/gate-check.sh {phase}`
   - **Windows (no WSL / pure PowerShell environment):** run the 8 checks manually
     per `skills/run-phase-gate/references/gate-criteria.md` and record pass/fail
     for each. The script is a convenience wrapper; the criteria are the source of truth.
   Capture the output.

4. **Interpret failures**: for each failed check, report:
   - What failed (exact check number and description).
   - The most likely cause.
   - The exact command to fix it.
   Prioritise: build failures > lint > test failures > coverage > task doc status.

5. **Iterate**: after the user addresses failures, re-run the gate. Repeat until
   all checks pass.

6. **Adversarial check** (Check 8): confirm `scripts/gate-review.sh` is present
   and executable. If not: offer to copy from Esquisse installation.

7. **On PASS**: update ROADMAP.md:
   - Mark P{n} as ✅ Completed with today's date.
   - Move P{n+1} to `## Current Phase`.
   - Verify P{n+1} tasks exist (create stubs if not).
   - Announce: "Phase P{n} gate passed. Promoted to P{n+1}."

**Constraints:**
- Never mark a phase complete until every gate check passes.
- Do not modify source code to fix gate failures — only guide the user.
- Do not update ROADMAP.md until the full gate run exits 0.

---

### 6. `skills/review-changes/SKILL.md` (P2)

**Triggers:** "review these changes", "review this PR", "check this diff",
"code review", "look at what changed", "is this ready to merge"

**Not a trigger for this skill:** "is phase N done" (→ `run-phase-gate`),
"refactor this" (→ `refactor-safely`), "implement task X" (→ `implement-task`)

**Purpose:** Structured review of a diff or set of changed files. Checks against
task doc acceptance criteria, AGENTS.md invariants, and code quality standards.
Produces either a pass or a list of required changes.

**Steps:**

1. **Load context**: read `AGENTS.md` (invariants), the task doc for this change
   (acceptance criteria, out-of-scope list, Files table).

2. **Get the diff**:
   - If a PR URL is provided: read it.
   - Otherwise: `git diff $(git merge-base HEAD main) HEAD` to capture all commits
     on the branch. Do NOT use `git diff HEAD~1` — it shows only the last commit
     and will produce an incomplete diff for multi-commit PRs.

3. **File-by-file review**:
   For each changed file:
   - Do comments violate AGENTS.md invariants?
   - Are new types/functions missing godoc?
   - Are there bare `string` or `bool` parameters on agent-facing functions?
   - Are new sentinel errors defined in `errors.go`?
   - Are test files present for every new production file?

4. **Acceptance criteria check**: verify each criterion in the task doc is
   provably addressed by the diff. Flag any criterion not covered.

5. **Scope check**: verify no "Out of Scope" items were implemented.

6. **Performance check** (if hot path involved): look for allocations inside loops,
   shared state between goroutines, `proto.Unmarshal` where raw bytes suffice.

7. **Verdict**:
   - **Ready**: all acceptance criteria addressed, no invariant violations, no
     out-of-scope items. Suggest PR description using the PR schema from SCHEMAS.md §7.
   - **Changes Required**: list each issue with: file, line reference, what is wrong,
     what the correct approach is. Group by severity (blocking / advisory).

**Constraints:**
- Never approve a diff that skips acceptance criteria from the task doc.
- Never approve a diff that violates a stated AGENTS.md invariant.
- Do not request cosmetic changes if they are not required by the code conventions.

---

### 7. `skills/write-adr/SKILL.md` (P3)

**Triggers:** "write an ADR", "record this decision", "document why we chose X",
"create an architecture decision record", "we decided to use X instead of Y"

**Not a trigger for this skill:** "update the documentation" (→ manual edit),
"update llms.txt" (→ `update-llms`), "update AGENTS.md" (embedded in `implement-task`)

**Purpose:** Create an Architecture Decision Record (SCHEMAS.md §5) when a
non-obvious design choice is made. Triggered naturally by the implement-task
completion protocol step 6: "if the phrase 'we chose X over Y because' appears."

**Steps:**

1. **Check sequence number**: list `docs/decisions/ADR-*.md`, find the next number.

2. **Interview** (single message):
   ```
   To write this ADR I need:

   1. What was decided? (first sentence is the headline)
   2. What situation or constraint drove this decision?
   3. What alternatives were considered and why were they rejected?
   4. What are the positive consequences?
   5. What are the negative consequences or trade-offs?
   ```

3. **Write ADR** to `docs/decisions/ADR-{nnnn}-{slug}.md` using the schema from
   SCHEMAS.md §5. Set Status: Accepted.

4. **Link from AGENTS.md**: if the decision prevents a common mistake, add a cross-
   reference in `## Common Mistakes to Avoid`:
   `"See ADR-{nnnn} for why this approach was chosen."`

5. **Announce**: "ADR-{nnnn} written. Link it from the task's Session Notes."

**Constraints:**
- Never write ADRs for decisions made before the project adopted Esquisse.
- Do not write an ADR for obvious choices (using stdlib instead of a custom parser).
- One ADR per decision. If multiple decisions were made, write multiple ADRs.

---

### 8. `skills/refactor-safely/SKILL.md` (P3)

**Triggers:** "refactor this", "rename type X to Y", "restructure this package",
"clean up this API", "move this function", "change this interface"

**Not a trigger for this skill:** "there's a bug" (→ `debug-issue`),
"implement a new feature" (→ `implement-task`), "review these changes" (→ `review-changes`)

**Purpose:** Interface-preserving refactor with test-first safety. Ensures
refactoring does not break callers or change observable behaviour.

**Steps:**

1. **Scope the refactor**: confirm with the user what changes and what does NOT
   change. Any caller-visible change becomes a Breaking Change task (Hard Rule 15).

2. **Establish baseline**: run full test suite. 100% of tests must pass before
   any refactoring begins. If tests already fail: stop and fix them first.

3. **Identify all call sites** (AST-first):
   - If `code_ast.duckdb` exists: `find_calls('**/*.go', '{function}')` / `go_type_flow('{type}')`
   - Otherwise: `grep -rn '{identifier}' --include='*.go'`
   List every call site that will need updating.

4. **Plan the rename/move**: list all files that will change. If > 10 files:
   this belongs in a task doc (Hard Rule: no task touches > 10 files). Create a
   task doc instead of proceeding.

5. **Define new interface first**: write the new signatures/names. Add godoc.
   Compile immediately — do not proceed if it does not compile.

6. **Update implementation**: move/rename production code.
   Run `go build ./...` after each file.

7. **Update call sites**: one file at a time. Compile after each.

8. **Run tests**: all tests must pass. No regressions allowed.

9. **No functional changes**: if during refactoring a bug is found, create a
   separate task for the fix. Do not fix and refactor in the same commit.

10. **Update documentation**:
    - Update `llms.txt` / `llms-full.txt` if public API changed.
    - Add ADR if the refactor reflected a non-obvious design decision.

---

### 9. `skills/update-llms/SKILL.md` (P3)

**Triggers:** "update llms.txt", "update the API reference", "sync the LLM index
after phase gate", "regenerate llms-full.txt"

**Not a trigger for this skill:** "write an ADR" (→ `write-adr`),
"update README" (→ manual edit), "explore the codebase" (→ `explore-codebase`)

**Language scope:** Steps 2–3 use language-specific doc-generation commands.
The skill detects the project language from AGENTS.md and adapts accordingly.
Go is the primary supported language; other language adapters are identified
from the project's CI scripts or AGENTS.md build commands.

**Purpose:** Regenerate `llms.txt` and `llms-full.txt` at phase gate or after a
public API changes. These files are the primary API reference for agents — stale
entries cause agents to call non-existent functions.

**Steps:**

1. **Determine scope**: read the last Session Log in NEXT_STEPS.md. Was a public
   API added or changed in recent tasks? If no API changed: skip with a note.

2. **Generate llms.txt** (concise public index):
   - Go: `go doc ./...` output, reformatted as llms.txt schema (module, public types,
     key functions, key interfaces, CLI commands, notes for agents).
   - Python / TS / Rust: equivalent doc generation command per language adapter.

3. **Generate llms-full.txt** (complete reference):
   - Go: `go doc -all ./...` — full godoc per exported symbol with constructor,
     methods, goroutine safety. Add the "Important Notes for AI Agents" section
     from AGENTS.md at the top.
   - Python: `pydoc-markdown` or equivalent; include class/function signatures
     and docstrings.
   - Other languages: read AGENTS.md `## Build Commands` to identify the
     canonical doc-generation tool for this project and use it.

4. **Diff against previous version**: summarise what was added, what changed,
   what was removed. Present to user before writing.

5. **Write files**: update both `llms.txt` and `llms-full.txt`.

6. **Append to NEXT_STEPS.md**: note `llms.txt updated for P{n} gate`.

**Constraints:**
- Do not update mid-task — only after tests pass and the task is Done.
- Both files must be updated together — never one without the other.
- Do not maintain by hand. Always regenerate from source.

---

## SCHEMA.md §6 Standard Skills Table Update

The current §6 table lists these five skills with stub descriptions. After all
P1/P2 skills are created, update the table to link to actual files:

```markdown
| Skill File | Purpose |
|-----------|---------|
| `skills/explore-codebase/SKILL.md` | First contact orientation — landscape → structure → hot path |
| `skills/implement-task/SKILL.md` | Startup protocol → implement → completion protocol |
| `skills/debug-issue/SKILL.md` | Reproduce → regression test → trace → fix → verify |
| `skills/review-changes/SKILL.md` | Diff review: acceptance criteria → invariants → scope → verdict |
| `skills/refactor-safely/SKILL.md` | Interface-preserving refactor: baseline → rename → call-sites → tests |
```

---

## FRAMEWORK.md §11 Table Additions

Add rows (in workflow order):

```markdown
| `skills/write-spec/SKILL.md` | "write a spec for X", "I want to add a feature", "help me design X" | Constraint audit → approach options → deep interview → writes approved spec doc |
| `skills/explore-codebase/SKILL.md` | "explore the codebase", "orient me", "where is X", "how does Y work" | 3-level orientation: landscape → structure → hot path. AST-first when cache exists. |
| `skills/implement-task/SKILL.md` | "implement task X", "start task P{n}-{nnn}", "execute this task" | Startup protocol → implement (types → code → tests) → full completion protocol |
| `skills/debug-issue/SKILL.md` | "there's a bug", "debug this", "fix failing test" | Reproduce → regression test (failing) → trace → fix → verify |
| `skills/run-phase-gate/SKILL.md` | "run the phase gate", "is phase N done", "promote to next phase" | Runs gate-check.sh, interprets failures, guides fixes, promotes phase in ROADMAP.md |
| `skills/review-changes/SKILL.md` | "review these changes", "review this PR", "is this ready to merge" | Acceptance criteria → invariants → scope → verdict with required changes |
| `skills/write-adr/SKILL.md` | "write an ADR", "record this decision", "document why we chose X" | Interviews user, writes ADR-{nnnn} doc, cross-references from AGENTS.md |
| `skills/refactor-safely/SKILL.md` | "refactor this", "rename X to Y", "restructure this package" | Baseline → identify call sites → define new interface → update → test |
| `skills/update-llms/SKILL.md` | "update llms.txt", "update the API reference", "sync LLM index" | Regenerates llms.txt + llms-full.txt from source, diffs against previous |
```

---

## `scripts/init.sh` Change

Extend the skills copy loop to include all new skills:

```bash
# Before:
for skill_dir in init-project new-task adopt-project adversarial-review write-spec; do

# After:
for skill_dir in init-project new-task adopt-project adversarial-review write-spec \
                 explore-codebase implement-task debug-issue run-phase-gate \
                 review-changes write-adr refactor-safely update-llms; do
```

---

## Implementation Order

### Batch 1 — P1 (implement first; every task session needs these)

1. `skills/write-spec/SKILL.md` + `references/` — full spec in plan doc
2. `skills/explore-codebase/SKILL.md` — orientation before implement-task
3. `skills/implement-task/SKILL.md` + `references/startup-protocol.md` + `references/completion-protocol.md`

### Batch 2 — P2 (drives quality and phase lifecycle)

4. `skills/debug-issue/SKILL.md` + `references/root-cause-guide.md`
5. `skills/run-phase-gate/SKILL.md` + `references/gate-criteria.md`
6. `skills/review-changes/SKILL.md`

### Batch 3 — P3 (maintenance and completeness)

7. `skills/write-adr/SKILL.md`
8. `skills/refactor-safely/SKILL.md`
9. `skills/update-llms/SKILL.md`

After each batch: update `scripts/init.sh` and `FRAMEWORK.md §11`.
As part of Batch 1: add stub entries to `SCHEMAS.md §6` for all 9 new skills with
status "Planned". This prevents agents from seeing an inconsistent standard table
during the Batch 1–2 period when new skills exist but aren't listed.
After Batch 2: replace the stub entries in `SCHEMAS.md §6` with live file links.

---

## Skill Trigger Test Protocol

The acceptance criterion "all skills trigger correctly" requires a repeatable manual
test procedure. Run this checklist after each Batch is installed:

1. Open VS Code Copilot Chat in a project that has the skills symlinked.
2. For each skill in the batch, paste the **first** listed trigger phrase verbatim
   into chat. The skill must auto-invoke (visible in the Copilot Skills sidebar or
   the skill's opening step appears in the response).
3. For `implement-task`: also confirm that pasting a phrase from a *different* skill
   does NOT trigger `implement-task` (false-positive check).
4. Record each result in `tests/triggers/{skill-name}.md` with: phrase used,
   date, VS Code Copilot version, pass/fail.
5. A skill is considered passing when all its listed trigger phrases pass in the
   same VS Code session.

**False-negative procedure** (trigger does not invoke the skill):
1. Confirm the skill is installed: `ls ~/.copilot/skills/{skill-name}/SKILL.md`
2. Confirm the frontmatter `description:` paragraph contains the trigger phrase text verbatim.
3. Reload VS Code (Developer: Reload Window) to force skill discovery.
4. Try a secondary trigger phrase from the skill's list.
5. If still failing after steps 1–4: file an issue in `tests/triggers/{skill-name}.md` under `## Failures` with exact phrase, VS Code Copilot version, and steps tried. Do NOT promote the phase gate until this is resolved.

**Gate integration:** Step 5 of `run-phase-gate` skill must include a prompt:
"Have all new skills been trigger-tested per the Skill Trigger Test Protocol?"
Proceed only on affirmative.

---

## Skill Lifecycle Policy

Skills are living documents. As FRAMEWORK.md evolves, skills may require updates
or, rarely, retirement.

**How to update a skill:**
1. Identify which FRAMEWORK.md or SCHEMAS.md section changed.
2. Edit the affected SKILL.md to reflect the change.
3. Update the trigger test checklist in `tests/triggers/{skill-name}.md`.
4. Bump the `updated:` field in the skill's YAML frontmatter.

**How to deprecate a skill:**
1. Add `deprecated: true` to the skill's YAML frontmatter and a `replaced-by:` field.
2. Remove the skill from the `scripts/init.sh` copy loop so new projects don't
   receive it.
3. Do NOT delete the skill directory — existing projects may still reference it.
4. Write an ADR documenting the retirement decision.

**Review cadence:** At every major phase gate (P2→P3, P4→P5), open each SKILL.md
and confirm its step list still matches the current FRAMEWORK.md protocols. Tag
anything stale in NEXT_STEPS.md.

---

## Skill Automation Boundary

Skills are **conversational guides** — they instruct an agent what steps to take.
They do not replace CI.

**Skills MAY:**
- Invoke shell scripts via `run_in_terminal` (e.g. `wsl bash scripts/gate-check.sh 2`)
- Read files and report findings
- Invoke sub-agents via `runSubagent`
- Write files (SKILL.md files, task docs, ADRs, NEXT_STEPS.md)

**Skills MUST NOT:**
- Compile code autonomously and commit without user review
- Automatically push to remote branches
- Modify `.adversarial/state.json` directly (only adversarial reviewer agents do this)
- Delete files without explicit user confirmation

This boundary is language-agnostic: skills detect language from AGENTS.md and
adjust commands accordingly, but a single SKILL.md covers all supported languages.

---

## Acceptance Criteria

- [ ] All 9 skills trigger correctly on their listed phrases (validated via Skill Trigger Test Protocol above).
- [ ] SCHEMAS.md §6 standard skills table lists actual file links (not stubs).
- [ ] FRAMEWORK.md §11 table has a row for every skill.
- [ ] `scripts/init.sh` copies all 9 skill directories to new projects.
- [ ] `implement-task` explicitly enforces the 3-attempt bail-out rule (Hard Rule 16).
- [ ] `debug-issue` explicitly enforces failing-test-first (Hard Rule 13).
- [ ] `run-phase-gate` updates ROADMAP.md on pass.
- [ ] `review-changes` checks against task doc acceptance criteria, not just code style.
- [ ] `tests/triggers/` directory exists with one checklist file per skill.
- [ ] Protocol coverage map in this document accounts for all 21 startup+completion steps.
- [ ] Every skill's `tools_required` list uses canonical tool names and follows the validation rules in SCHEMAS.md §6.
- [ ] Every skill frontmatter includes a `not:` trigger guard and an `updated:` date field per the updated SCHEMAS.md §6 template.

---

## Out of Scope

- Autonomous compilation and commit without user review.
- Language-specific skill variants (one skill per workflow; language is detected within the skill).
- Skills for VS Code extension development or MCP server configuration.
- A skill for managing git history (out of Esquisse scope).
- Automated CI replacement (skills guide the agent; CI enforces correctness independently).
