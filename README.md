# Esquisse

> *esquisse* (French) — a rough sketch; the foundational shape before the detail.

A framework for initializing AI-agent-ready projects. Gives agents the context they need to implement complete, correct units of work the first time — without reading the entire codebase, asking clarifying questions, or silently violating invariants.

---

## The Problem

AI agents fail not because of capability gaps but because of **context gaps**.

An agent that doesn't know what invariants must be preserved, what the acceptance criteria actually are, which files to touch and which to leave alone, or what the domain vocabulary means will produce code that is _locally correct but globally wrong_.

Esquisse fixes this with a document structure and working protocol that encodes exactly the context an agent needs — and no more.

---

## What's in the Box

| File | Purpose |
|------|---------|
| [`FRAMEWORK.md`](FRAMEWORK.md) | The operational guide — philosophy, protocols, phase gates, guardrails, Go conventions |
| [`SCHEMAS.md`](SCHEMAS.md) | Document schemas — what to write in each artifact type |
| [`TEMPLATES.md`](TEMPLATES.md) | Copy-paste starters — templates + language adapters for Go, Python, TypeScript, Rust, C/C++, React |
| [`scripts/init.sh`](scripts/init.sh) | Bootstraps a new project: creates directories and starter documents |
| [`scripts/new-task.sh`](scripts/new-task.sh) | Scaffolds the next task document in a phase (auto-sequences) |
| [`scripts/gate-check.sh`](scripts/gate-check.sh) | Validates phase gate criteria: build, lint, tests, coverage, stub count, task status |
| [`scripts/macros.sql`](scripts/macros.sql) | Universal DuckDB/sitting_duck AST macros for codebase navigation |
| [`scripts/macros_go.sql`](scripts/macros_go.sql) | Go-specific AST macros: struct fields, signatures, type flow, interfaces |

---

## Quick Start

```bash
# Bootstrap a new project from your project root
bash /path/to/esquisse/scripts/init.sh --project-name myproject

# Scaffold the first task document
bash scripts/new-task.sh 0 foundation

# Fill it in (prompt your agent):
# "Fill in docs/tasks/P0-001-foundation.md.
#  Goal: [one sentence]. In scope: [...]. Out of scope: [...].
#  Acceptance criteria as test function names."

# Validate the phase gate before promoting
bash scripts/gate-check.sh 0
```

---

## Document Hierarchy

```
AGENTS.md          ← Project constitution: invariants, commands, gotchas
ONBOARDING.md      ← Mental model, read order, data flow
GLOSSARY.md        ← Canonical domain vocabulary (names must match the code)
docs/
  tasks/           ← P{phase}-{seq}-{slug}.md — one unit of work each
  adr/             ← Architecture Decision Records
  planning/
    ROADMAP.md     ← Phase status and gate checklists
    NEXT_STEPS.md  ← Session state (updated after every work session)
skills/            ← Reusable agent workflow prompts
llms.txt           ← Concise public API index
llms-full.txt      ← Full API reference for implementation tasks
code_ast.duckdb    ← AST cache (gitignored; built with sitting_duck)
```

Reading order for an agent starting a task: `AGENTS.md` → `ONBOARDING.md` → task doc.

---

## Core Principles

1. **Specificity over cleverness** — documents exact enough for a first-time reader to implement without asking questions.
2. **Tests define correctness** — acceptance criteria are test function names, not prose. Never change a test to match incorrect output.
3. **Scope is a contract** — every task has an explicit out-of-scope list. No gold-plating.
4. **Gotchas live in writing** — hard-won lessons belong in `AGENTS.md`, not in someone's head.
5. **Progressive context disclosure** — `AGENTS.md` → `ONBOARDING.md` → task doc gives the agent exactly the context it needs. No more, no less.
6. **Interface is the prompt** — function names, parameter types, and error messages are instructions to the agent. Design them accordingly.
7. **Constraint over power** — a typed `Status` enum the agent cannot misuse beats a `string` the agent will invent values for.

---

## Key Guardrails (selected)

- **Tests define correctness.** Fix the code, never the test expectations.
- **No `panic("TODO` in production code.** Stubs compile-block the gate.
- **No silent fallbacks.** If a function cannot complete its contract, return an error.
- **Breaking changes require their own task.** Never change a public API as a side effect.
- **3-attempt bail-out.** After 3 failed attempts at the same problem: stop, revert to a stub, record it in `NEXT_STEPS.md` as Blocked, yield to the human.
- **Bug fixes require a failing test first.** Write the red test, then fix the code.

Full list: [`FRAMEWORK.md` §11](FRAMEWORK.md#11-guardrails-and-hard-rules) — 16 rules.

---

## Scripting the Gate

`gate-check.sh` runs 7 sequential checks. For Go projects (default):

1. `go build ./...`
2. `golangci-lint run ./...` (auto-discovers `.golangci.yml`; falls back to `go vet`)
3. `go test -count=1 ./...`
4. No `panic("TODO` stubs
5. `// ASSUMPTION:` and `// TODO` annotation count (warning; `--strict` for hard fail)
6. Coverage ≥ threshold (default 80%; `COVERAGE_THRESHOLD=70` to override)
7. All `P{phase}-*.md` task docs have `Status: Completed`

```bash
bash scripts/gate-check.sh 1          # phase 1 gate
bash scripts/gate-check.sh 1 --strict # zero-tolerance for annotations
COVERAGE_THRESHOLD=70 bash scripts/gate-check.sh 2
```

Python and TypeScript projects: `LANG_ADAPTER=python` or `LANG_ADAPTER=ts`.

---

## AST-Aided Navigation

For large codebases, answering *"what implements this interface?"* by reading files wastes tokens. Build the cache once and use SQL instead:

```bash
# Build cache (Go)
duckdb -init /dev/null code_ast.duckdb <<'SQL'
LOAD sitting_duck;
CREATE OR REPLACE TABLE ast AS
SELECT * FROM read_ast('**/*.go', ignore_errors := true, peek := 200);
SQL

# Load macros and query
duckdb -init /dev/null code_ast.duckdb <<'SQL'
LOAD sitting_duck;
.read scripts/macros.sql
.read scripts/macros_go.sql
SELECT * FROM complexity_hotspots('**/*.go', 10);
SELECT * FROM go_struct_fields() WHERE struct_name = 'MyType';
SQL
```

Universal macros: `find_definitions`, `find_calls`, `code_structure`, `complexity_hotspots`, `function_callers`, `structural_diff`.

Go macros: `go_struct_fields`, `go_func_signatures`, `go_type_flow`, `go_interfaces`, `go_interface_impls`.

Details: [`FRAMEWORK.md` §17](FRAMEWORK.md#17-ast-aided-codebase-navigation).

---

## Applying to an Existing Project

Three phases, each taking a day or less:

1. **Document what already exists** — `AGENTS.md` with build commands, key deps, top gotchas. `GLOSSARY.md` with terms extracted from the code.
2. **Draw the migration boundary** — one task doc for the next piece of real work. Follow the protocol exactly from that point forward.
3. **Backfill selectively** — `AGENTS.md` grows as agents encounter bugs; ADRs written for past decisions only when they carry active reversal risk.

Golden rule: **never fake it**. Build commands must be copy-paste runnable. Phase labels must reflect where the project actually is.

Details: [`FRAMEWORK.md` §15](FRAMEWORK.md#15-adopting-in-an-existing-project).

---

## Language Adapters

Starter `AGENTS.md` conventions, build/test commands, and required file lists for:
Go · Python · TypeScript · Rust · C/C++ · React/TypeScript

[`TEMPLATES.md` → Language Adapters](TEMPLATES.md#language-adapters)

---

## Influenced By

- *["APIs Are Environments Now"](https://medium.com/@tfmv/apis-are-environments-now-b60c94bd4804)* (Thomas McGeehan) — agent-facing interface design principles (Hard Rules 8–11, Go conventions §12).
