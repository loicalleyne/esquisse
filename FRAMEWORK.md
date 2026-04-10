# Esquisse — AI-Agent-Ready Project Initialization Framework

> *Esquisse* (French): a rough sketch or first draft — the foundational shape before the detail.

This framework defines a project structure, document schema, and working conventions that allow AI coding agents (Claude Sonnet, GPT-4.1, GPT-4o, GPT-5-mini, Copilot, etc.) to successfully implement complete, correct units of work with minimal ambiguity and minimal context waste.

---

## Table of Contents

1. [Philosophy](#1-philosophy)
2. [Document Hierarchy](#2-document-hierarchy)
3. [Directory Layout](#3-directory-layout)
4–9. [Document Schemas → SCHEMAS.md](SCHEMAS.md)
10. [Phase Structure and Gates](#10-phase-structure-and-gates)
11. [Guardrails and Hard Rules](#11-guardrails-and-hard-rules)
12. [Go-Specific Conventions](#12-go-specific-conventions)
13. [llms.txt / llms-full.txt Convention](#13-llmstxt--llms-fulltxt-convention)
14. [Project Initialization Checklist](#14-project-initialization-checklist)
    - [Per-Task Startup Protocol](#per-task-startup-protocol)
    - [Per-Task Completion Protocol](#per-task-completion-protocol)
    - [Artifact Update Trigger Table](#artifact-update-trigger-table)
15. [Adopting in an Existing Project](#15-adopting-in-an-existing-project)
16. [Helper Scripts](#16-helper-scripts)
17. [AST-Aided Codebase Navigation](#17-ast-aided-codebase-navigation)

[Templates and Language Adapters → TEMPLATES.md](TEMPLATES.md)

---

## 1. Philosophy

### Core Insight

AI agents fail not because of capability gaps but because of context gaps. An agent that does not know:

- what invariants must be preserved
- what the acceptance criteria are
- which files to touch and which to leave alone
- what the domain vocabulary means

...will produce code that is locally correct but globally wrong.

### Five Principles

| # | Principle | Implication |
|---|-----------|-------------|
| 1 | **Specificity over cleverness** | Documents must be exact enough for a first-time reader to implement without asking questions. |
| 2 | **Tests define correctness** | Acceptance criteria are test function names and assertions, not prose descriptions. Never change a test to match incorrect output. |
| 3 | **Scope is a contract** | Every task has an explicit out-of-scope list. Agents must not gold-plate. |
| 4 | **Gotchas live in writing** | Hard-won lessons about libraries, language quirks, and framework edge cases belong in `AGENTS.md`, not in someone's head. |
| 5 | **Progressive context disclosure** | A model reading `AGENTS.md` → `ONBOARDING.md` → a task doc should have exactly the context it needs — no more, no less. |
| 6 | **The interface is the prompt** | Function names, parameter names, and type names are the only documentation an agent reliably reads. If the intent is not in the signature, it does not exist for the agent. |
| 7 | **Constraint over power** | A system that makes entire classes of failure impossible is better than a flexible system that relies on the agent to behave correctly. Every degree of freedom left open will be explored. |

---

## 2. Document Hierarchy

```
Read in order for a new task:

1. AGENTS.md          ← Project constitution (architecture, build, conventions, gotchas)
2. ONBOARDING.md      ← Orient: what to read first, load order, mental model
3. GLOSSARY.md        ← Domain vocabulary (canonical names for types, concepts)
4. llms.txt           ← Concise public API index
5. llms-full.txt      ← Full API reference for implementation tasks
6. docs/decisions/    ← WHY decisions were made (ADRs)
7. docs/tasks/        ← WHAT to do next (task plans)
8. skills/            ← HOW to work (reusable workflows)
```

**Rule:** If a model needs to ask a question that one of these files should answer, the file is incomplete. Fix the file, not the prompt.

---

## 3. Directory Layout

```
project-root/
│
├── AGENTS.md                      # Project constitution for AI agents
├── ONBOARDING.md                  # Agent orientation guide (read order, mental model)
├── GLOSSARY.md                    # Canonical domain vocabulary
├── README.md                      # Human-facing project description
├── llms.txt                       # Concise LLM-optimized API index
├── llms-full.txt                  # Full LLM-optimized API reference
├── code_ast.duckdb                # AST cache for the sitting_duck skill (optional, gitignored)
│
├── docs/
│   ├── decisions/                 # Architecture Decision Records (ADRs)
│   │   └── ADR-{nnnn}-{slug}.md
│   ├── tasks/                     # Granular task plans
│   │   └── P{phase}-{nnn}-{slug}.md
│   ├── planning/
│   │   ├── ROADMAP.md             # Phase-level plan with status tracking
│   │   └── NEXT_STEPS.md         # Current session state (updated after each work session)
│   └── specs/                     # Feature design documents (pre-task)
│       └── {date}-{feature}.md
│
├── skills/                        # Reusable agent workflow prompts
│   ├── explore-codebase.md
│   ├── implement-task.md
│   ├── debug-issue.md
│   └── review-changes.md
│
├── {source-dirs}/                 # Language-specific source tree
├── {test-dirs}/                   # Tests
└── {generated-dirs}/              # Generated code — NEVER edit manually
```

**Language-specific source layouts:**

| Language | Typical Layout |
|----------|---------------|
| Go | `cmd/`, `internal/`, `pkg/`, `gen/` |
| Python | `src/{package}/`, `tests/` |
| TypeScript/JS | `src/`, `tests/` or `spec/` |
| Rust | `src/`, `tests/` |

---

## Document Schemas

All document schemas (AGENTS.md, ONBOARDING.md, GLOSSARY.md, Task, ADR, Skill) are defined in **[SCHEMAS.md](SCHEMAS.md)**. Reference that file when creating or auditing any framework document.

---

## 10. Phase Structure and Gates

### Phase Numbering

| Phase | Purpose | Example Contents |
|-------|---------|-----------------|
| P0 | Foundation | Module setup, CI, AGENTS.md, core types, interfaces |
| P1 | Core functionality | Primary happy path, basic tests |
| P2 | Complete feature set | All tools/endpoints/handlers, full test coverage |
| P3 | Production hardening | Error handling, edge cases, benchmarks, profiling |
| P4 | Intelligence / UX | Smart defaults, compound workflows, developer experience |
| P5+ | Ecosystem | Plugin systems, external integrations, documentation site |

### Phase Gate Criteria

A phase is DONE when ALL of the following are true:

```markdown
## P{n} Gate Checklist
- [ ] All P{n} tasks have Status: Done
- [ ] `{full test command}` passes with 0 failures
- [ ] `{race detector command}` passes (if concurrent)
- [ ] `{lint command}` produces 0 errors
- [ ] `go build ./...` (or language equivalent) produces 0 errors
- [ ] Test coverage ≥ {project-defined target, default 80%} for production code
- [ ] AGENTS.md updated with any new gotchas discovered during P{n}
- [ ] llms.txt and llms-full.txt updated to reflect new API surface
- [ ] ROADMAP.md updated: P{n} marked complete, P{n+1} tasks reviewed
- [ ] No TODO/FIXME/ASSUMPTION comments remain unresolved (or triaged to tasks)
```

### ROADMAP.md Structure

```markdown
# {Project} Roadmap

## Current Phase: P{n} — {Name}
**Target:** {brief goal}
**Gate criteria:** [P{n} Gate Checklist](docs/planning/ROADMAP.md#{gate-anchor})

### P{n} Tasks
| Task | Status | Summary |
|------|--------|---------|
| [P{n}-001-slug](docs/tasks/P{n}-001-slug.md) | ✅ Done | ... |
| [P{n}-002-slug](docs/tasks/P{n}-002-slug.md) | 🔄 In Progress | ... |
| [P{n}-003-slug](docs/tasks/P{n}-003-slug.md) | ⬜ Ready | ... |

## Upcoming: P{n+1} — {Name}
...

## Completed
### P{n-1} — {Name} ✅
Completed {date}. Key outcomes:
-
```

---

## 11. Guardrails and Hard Rules

These rules apply to ALL tasks in ALL projects using this framework. They belong in `AGENTS.md` as the first section.

### Universal Hard Rules

```markdown
## THE MOST IMPORTANT RULES

1. **Tests define correctness.** If a test fails, fix the code — never change
   test expectations to match incorrect output.

2. **Scope is a contract.** Do not implement anything in a task's "Out of Scope"
   section. Create a new task for it instead.

3. **Generated files are read-only.** Files under `gen/` or marked "DO NOT EDIT
   MANUALLY" must never be hand-edited. Regenerate them with the documented command.

4. **No silent fallbacks.** If a function cannot complete its contract, return
   an error. Do not return zero-values and continue.

5. **No global state.** All types are constructed explicitly. No `init()` side
   effects that modify package-level variables.

6. **All errors returned.** Never `panic` on expected failure conditions.
   Programmer-contract violations in constructors may panic if documented in godoc.

7. **One responsibility per commit.** Each implementation session should be
   containable in a single coherent commit with a semantic commit message.

8. **Make invalid states unrepresentable.** Use typed enums/constants instead of
   bare strings or booleans for any parameter with a constrained value set.
   An agent given a `status: string` will invent values. An agent given a
   `status: OrderStatus` cannot.

9. **The function name is the spec.** Agent-facing functions must be named for
   their exact, singular effect. Avoid boolean flag parameters that alter
   behaviour — split them into separate functions instead.
   - Wrong: `DeleteFile(path string, force bool)`
   - Right: `DeleteFile(path string)` and `PermanentlyDeleteFile(path string)`

10. **Errors must be actionable.** Every error returned on an agent-facing path
    must encode: (a) what failed, (b) the current state of the resource, and
    (c) the exact next step to recover. Vague errors cause thrashing — the agent
    will retry indefinitely with mutated inputs.

11. **Encode workflows, not just operations.** If achieving a goal requires N
    sequential calls in a fixed order, provide a single compound function that
    executes the workflow. Do not require the agent to orchestrate the sequence —
    it will get the order wrong or skip steps.

12. **Ambiguity resolution: conservative choice + documented assumption.** When
    the task doc is unclear, take the most conservative/reversible interpretation.
    Mark every assumption with an `// ASSUMPTION: {reason}` comment at the
    decision point and record it in Session Notes. Do not guess silently.

13. **Bug fixes require a failing test first.** Before fixing any defect: write
    a test that reproduces the failure. Fix the code only after the test is red.
    Do not close a bug without a regression test that would have caught it.

14. **Never leave the codebase uncompilable.** If a session ends mid-task, every
    unfinished function must compile. Stub incomplete bodies with:
    `panic("TODO P{n}-{nnn}: {slug} — {what is missing}")`
    Record the incomplete state in Session Notes before ending the session.

15. **Breaking changes require their own task.** Do not change a public API
    signature as a side effect of another task. If a task requires a breaking
    change: (a) create a new task with Status: Breaking Change, (b) list all
    call sites that must be updated in the Files table, (c) update all call
    sites in the same commit — never leave call sites in a broken state.

16. **The 3-attempt bail-out rule.** If an agent fails to fix a failing test or
    compilation error after 3 distinct, meaningfully different attempts:
    - Stop modifying code immediately.
    - Revert the specific broken function/block to its last known-good state
      using a stub: `panic("STUCK P{n}-{nnn}: {what was attempted, what failed}")`
    - Record the full failure in `docs/planning/NEXT_STEPS.md` under a new
      `## Blocked` entry, including the 3 approaches tried and their outcomes.
    - Update the task Status to: `Blocked`.
    - Yield to the user. Do not attempt a 4th approach in the same session.

    **Rationale:** An agent retry loop is indistinguishable from forward progress
    to the agent. The 3-attempt limit is the circuit breaker. Three attempts means
    three approaches that are structurally different, not three runs of the same fix.
```

### Anti-Pattern Catalog Template

Add to `AGENTS.md` as patterns are discovered:

```markdown
## Common Mistakes to Avoid

1. **[Category] Mistake description.**
   - Wrong: `code example`
   - Right: `code example`
   - Why: explanation of the invariant being violated.

2. **[Performance] Constructing {expensive type} inside a processing loop.**
   - Wrong: `type := New(...)` inside a `for` loop
   - Right: pre-construct outside the loop, reuse per iteration
   - Why: construction costs ~{N}µs; at {M}/sec throughput this adds {X}% overhead.
```

### Context Window Hygiene

Rules for tasks to stay within model context limits:

| Rule | Rationale |
|------|-----------|
| Task spec fits in ~2000 tokens | Leaves budget for code, tests, and reasoning |
| No task touches more than 10 files | Prevents agent from losing coherence mid-task |
| Acceptance criteria are ≤ 20 test cases | More than this means the task should be split |
| Each task has ≤ 3 upstream dependencies | Deeper dependency chains require too much prior context |

---

## 12. Go-Specific Conventions

When the primary language is Go, add the following to `AGENTS.md`:

### Required Godoc Rules

```
All exported types, functions, methods, and package-level variables MUST have godoc.
Tests must have godoc unless trivially named (TestFoo_HappyPath is sufficient).
Example functions (Example*) should appear for all non-trivial public APIs.
```

### Standard Error Patterns

```go
// Sentinel errors — defined in errors.go at the package root.
var (
    ErrNilInput   = errors.New("{package}: nil input")
    ErrClosed     = errors.New("{package}: writer is closed")
    ErrNotFound   = errors.New("{package}: not found")
)

// Wrap with context:
return fmt.Errorf("{package}.{Function}: %w", ErrNilInput)
```

### Goroutine Safety Contract

Document one of the following for every exported type:

```
// {TypeName} is safe for concurrent use.       (immutable or internally synchronized)
// {TypeName} is NOT safe for concurrent use.   (use Clone() or sync.Pool per goroutine)
// {TypeName} is read-only after construction.  (safe to share after New() returns)
```

### Interface Location Convention

```
- Define interfaces in the CONSUMING package, not the implementing package.
- Keep interfaces small (1-3 methods preferred).
- Always include the interface definition in the task doc for any new interface.
```

### Agent-Facing Interface Design

When a function or type will be called by an AI agent (via a tool, MCP call, or
generated code), apply stricter naming and typing rules than for human-authored calls:

**Naming:**
```go
// Wrong — agent must guess what 'force' does:
func Delete(path string, force bool) error

// Right — single-purpose, self-documenting:
func Delete(path string) error
func DeletePermanently(path string) error  // name communicates irreversibility
```

**Parameters:**
```go
// Wrong — bare string: agent will invent values:
func SetStatus(id int, status string) error

// Right — typed enum: invalid values are compile errors:
type OrderStatus string
const (
    OrderStatusPending   OrderStatus = "pending"
    OrderStatusFulfilled OrderStatus = "fulfilled"
    OrderStatusShipped   OrderStatus = "shipped"
)
func SetStatus(id int, status OrderStatus) error
```

**Godoc for agent-facing functions:**
```go
// ProcessOrder fulfills an order by reserving inventory and marking it ready
// for shipment. The order must be in StatusPending; call FulfillOrder first
// if it is in StatusDraft. Returns ErrInvalidTransition if the order is
// already shipped or cancelled.
func ProcessOrder(ctx context.Context, id OrderID) error
```

The godoc comment is the only text an agent reads about this function.
Treat it as a prompt: state preconditions, the required prior call, and the
exact error to expect on misuse.

**Error types on agent-facing paths:**
```go
// AgentError is returned on all agent-facing paths.
// It provides the information an agent needs to recover without thrashing.
type AgentError struct {
    Code         string   // machine-readable error code
    Message      string   // what failed
    CurrentState string   // state of the resource when the error occurred
    NextSteps    []string // ordered list of exact function calls to recover
}
```

### Required Files per Package

| File | Contents |
|------|----------|
| `doc.go` | Package-level godoc comment + `package {name}` |
| `errors.go` | All sentinel errors for this package |
| `{type}.go` | One primary type per file, named after the type |
| `{type}_test.go` | Tests for that type |
| `example_test.go` | Package-level example functions |

### Package Import Rules

Every multi-package module must declare its import direction in `AGENTS.md`.
The rule: dependencies flow one direction only — never in a cycle.

```
## Package Import Rules  (add to AGENTS.md, adapt per project)

Allowed import direction:
  cmd/        → internal/
  internal/X  → internal/Y  (specify which layers may import which)

Forbidden:
  internal/storage  must NOT import  internal/api
  internal/domain   must NOT import  internal/infra

Verification: `go build ./...` fails on circular imports.
Enforcement: golangci-lint depguard rule (add to .golangci.yml if enforced).
```

When creating a new package, add it to the import direction diagram in `AGENTS.md`
before writing any import statements.

### Module and Dependency Rules

```
- go.mod module path must match the intended import path exactly.
- Do not add dependencies without a task that documents the reason.
- Every new dependency gets an entry in AGENTS.md Key Dependencies table.
- Pin maximum major versions: "Arrow v18 — do not drift without testing all paths."
```

---

## 13. llms.txt / llms-full.txt Convention

These files serve as LLM-optimized API references. They allow a model to understand your public API without reading source files.

### llms.txt (concise index)

```markdown
# {Project} — LLM Quick Index

## Module
{module path}

## Public Types
{TypeName} — {one-line description}

## Key Functions
{FunctionName}({params}) {returns} — {one-line description}

## Key Interfaces
{InterfaceName} — {one-line description}

## CLI / Commands (if applicable)
{command} — {description}

## Important Notes for AI Agents
- {critical invariant}
- {performance constraint}
- {goroutine safety summary}
```

### llms-full.txt (complete reference)

One entry per exported symbol, in the format:

```markdown
### {TypeName}

**Package:** `{import path}`
**File:** `{relative path}`
**Thread safety:** {Safe | Unsafe | Read-only after construction}

{Full godoc comment, verbatim}

**Constructor:**
```go
func New{TypeName}({params}) (*{TypeName}, error)
```

**Methods:**
```go
func (t *{TypeName}) {Method}({params}) {returns}
// {description}
```

**Example:**
```go
{runnable example}
```
```

### Update Rule

`llms.txt` and `llms-full.txt` are updated:
- At the end of every phase (required by phase gate).
- When a public API changes (any task touching exported symbols).
- Never during a task — editing them mid-task wastes context.

---

## 14. Project Initialization Checklist

Run this checklist when bootstrapping a new project with this framework.

### Day 0: Foundation (P0)

```markdown
## P0 Init Checklist

### Repository
- [ ] Create repo with appropriate `.gitignore` (include `code_ast.duckdb` — it is a build artifact)
- [ ] Add `LICENSE`
- [ ] Add `README.md` (minimal: name, one-sentence description, build command)

### Framework Documents
- [ ] `AGENTS.md` — fill Architecture, Build Commands, Key Dependencies, Code Conventions
- [ ] `ONBOARDING.md` — fill Mental Model and Read-Order (stub other sections)
- [ ] `GLOSSARY.md` — add domain terms as they emerge (start with ≥5)
- [ ] `docs/planning/ROADMAP.md` — define P0 through P2 tasks
- [ ] `docs/planning/NEXT_STEPS.md` — current session state

### First Task Document
- [ ] `docs/tasks/P0-001-{slug}.md` — first implementation unit
  - Status: Ready
  - Specification includes exact interface/type signatures
  - Acceptance criteria includes specific test names

### Skills
- [ ] `skills/explore-codebase.md` (copy/adapt from this framework)
- [ ] `skills/implement-task.md` (copy/adapt from this framework)

### Go-Specific
- [ ] `go mod init {module-path}`
- [ ] `cmd/{name}/main.go` — entrypoint stub
- [ ] first package with `doc.go` and `errors.go`
- [ ] `Makefile` with `build`, `test`, `test-race`, `lint`, `generate` targets

### CI
- [ ] `.github/workflows/ci.yml` — build + test on push/PR
- [ ] Lint step (`golangci-lint` for Go)
- [ ] Race detector test step

### After First Test Passes
- [ ] `llms.txt` — stub (expand at P1 gate)
- [ ] `llms-full.txt` — stub (expand at P1 gate)
```

### Per-Task Startup Protocol

When an agent begins a task, it should execute this protocol before writing any code:

```markdown
## Task Startup Protocol

1. Read AGENTS.md → internalize invariants and gotchas.
2. Read GLOSSARY.md → confirm vocabulary for this domain area.
3. Read the task doc (P{n}-{nnn}-{slug}.md) completely.
4. Read the "In Scope" and "Out of Scope" lists carefully.

5. **Establish baseline.** Run the full test suite NOW, before touching any code.
   Record the pass/fail count. If tests are already failing, document this in
   Session Notes before proceeding — do not allow pre-existing failures to be
   attributed to your changes.

6. Locate each file in the "Files" table — read headers/interfaces.
   If a `code_ast.duckdb` cache exists at the project root (see §17), prefer
   AST queries for structural questions ("what implements this interface?",
   "what calls this function?") before reading raw files. This preserves token
   budget for the implementation itself.
7. For Modify actions: read the full function/type being changed.
8. Read the Acceptance Criteria — understand exactly what passing means.

9. **State assumptions explicitly.** If the task doc is ambiguous about any
   implementation detail, write your assumptions down before starting. Each
   assumption will become an `// ASSUMPTION:` comment in the code (Hard Rule 12).

10. **Follow the edit order** to avoid implementing against interfaces that do
    not yet exist:
    1. Define types and interfaces first.
    2. Implement production code.
    3. Write tests.
    4. Update documentation (godoc, llms.txt if API changed).

11. Implement. Run tests after each logical unit.
```

### Per-Task Completion Protocol

When tests pass and the task acceptance criteria are met, execute this before closing the session:

```markdown
## Task Completion Protocol

### 1. Task document
- Set `Status: Done`.
- Append to `## Session Notes`:
  `{YYYY-MM-DD} — {author} — Completed. {one sentence on approach or key decision.}`
- If any "Out of Scope" item was nearly touched, note it with a reference to
  the task that should own it.

### 2. AGENTS.md
- For every workaround, surprising behaviour, or near-mistake encountered:
  add an entry to `## Common Mistakes to Avoid`.
- For every new exported type or package added:
  add a row to `## Key Dependencies` (if external) or update Repository Layout.

### 3. GLOSSARY.md
- For every new domain term introduced in this task (type name, concept, event):
  add a row to GLOSSARY.md.
- **Trigger:** if you named something that isn't already in the glossary, add it.

### 4. ONBOARDING.md
- If a new package or key file was created: add a row to the Key Files Map.
- If the primary data flow changed: update the Data Flow diagram.

### 5. ROADMAP.md
- Mark this task ✅ Done in the phase table.
- If new tasks were identified during implementation:
  create stub task docs (Status: Draft) and add them to ROADMAP.md.

### 6. docs/decisions/ (ADRs)
- If a non-obvious design choice was made (an alternative was considered and
  rejected): create `ADR-{nnnn}-{slug}.md`.
- **Trigger:** if the phrase "we chose X over Y because" appears in your
  reasoning, that reasoning belongs in an ADR.

### 7. docs/planning/NEXT_STEPS.md
- Append a dated entry under `## Session Log`:
  ```
  {YYYY-MM-DD} — {task-id} — {one sentence summary of what was done}
  Files changed: {comma-separated list}
  Up next: {next task id and slug}
  ```

### 8. llms.txt / llms-full.txt
- Update ONLY if a public API was added or changed in this task.
- Do not update mid-task — only after tests pass.

### 9. Phase gate (only when all phase tasks are Done)
- Run the full phase gate checklist from ROADMAP.md.
- Do not mark a phase complete until every gate criterion is ticked.

### 10. PR / Handoff
Only when the task is complete and all documentation is updated:
- Create a pull request using the **PR Schema** (see SCHEMAS.md §7).
- The PR description must be machine-readable enough for the reviewer to test
  the change without reading the source code.
- Do not include implementation detail that is already in the task doc.
  Link to the task doc instead.
```

### Artifact Update Trigger Table

Quick reference: which artifact to update and when.

| Artifact | Update When |
|----------|------------|
| `AGENTS.md` — Common Mistakes | Any workaround, surprising API behaviour, or near-mistake found during a task |
| `AGENTS.md` — Repository Layout | New package or significant directory added |
| `AGENTS.md` — Key Dependencies | New external dependency added |
| `GLOSSARY.md` | New domain term, type name, or concept introduced in any task |
| `ONBOARDING.md` — Key Files Map | New package or key file created |
| `ONBOARDING.md` — Data Flow | Primary data path changes |
| `docs/decisions/ADR-*.md` | Any design choice where an alternative was considered and rejected |
| `docs/planning/ROADMAP.md` | Task status changes; new tasks identified |
| `docs/planning/NEXT_STEPS.md` | End of every work session |
| `llms.txt` / `llms-full.txt` | Public API added or changed; phase gate |
| Task doc `Status` field | Any status transition (Draft→Ready→In Progress→Done) |
| Task doc `Session Notes` | End of every work session on that task |

---

## 15. Adopting in an Existing Project

### The Core Constraint

You cannot retroactively make all documents authoritative at once. Attempting a big-bang adoption produces low-quality documents quickly and creates more confusion than none. The correct approach is **incremental adoption**, task-by-task.

### Phase 1: Describe What Exists (read-only, no code changes)

Complete these in order in a single session. Do not write task docs for already-completed work.

```markdown
## Existing Project Adoption Checklist — Phase 1

- [ ] AGENTS.md — fill Architecture and Repository Layout from what's actually there.
       Add Build/Test commands. Leave Common Mistakes empty for now.
- [ ] GLOSSARY.md — extract every domain term already in the code:
       type names, config keys, event names. Archaeology, not invention.
       Do not invent new names; record the names already used.
- [ ] ONBOARDING.md — fill Mental Model and Key Files Map for the
       5–10 most important files only. Stub everything else.
- [ ] docs/planning/ROADMAP.md — audit current state honestly.
       Assign a phase label ("we are mid-P2"). List known remaining
       work as task stubs with Status: Draft.
- [ ] docs/planning/NEXT_STEPS.md — snapshot of current state.
- [ ] llms.txt — generate from `go doc ./...` (or language equivalent)
       as a first draft. Refine at the next phase gate.
- [ ] DO NOT write task docs for work already done.
- [ ] DO NOT invent an ADR for a decision that was made before the framework.
       Only write ADRs for decisions made going forward.
```

**Time budget:** 2–4 hours. If it takes longer, the codebase is being over-documented. Stub where uncertain; accuracy matters more than completeness at this stage.

### Phase 2: First Task Under Framework

From this point forward, every new feature or non-trivial fix gets a task doc *before* work starts. This is the migration boundary: the first unit of work done in-framework.

```markdown
## Existing Project Adoption Checklist — Phase 2 (ongoing)

- [ ] Create a task doc for the next piece of work.
       Status: Ready. Include exact interface signatures and test names.
- [ ] Follow the Task Startup Protocol and Task Completion Protocol
       exactly as for a greenfield project.
- [ ] After the first completed task: run the full Completion Protocol.
       This is the first real population of Common Mistakes and ADRs.
```

### Phase 3: Backfill Selectively

Over 3–6 months the framework documents become authoritative for the whole codebase, not just new work. This happens organically:

- **AGENTS.md Common Mistakes** grows as agents encounter bugs in existing code.
- **ADRs** are written retroactively when a code review surfaces a non-obvious past decision.
- **GLOSSARY.md** fills in as new terms are found in code that wasn't covered at Phase 1.
- **llms.txt / llms-full.txt** are brought fully up to date at the first phase gate.

### The Key Asymmetry

| | Greenfield | Existing |
|---|---|---|
| Documents | Written before code | Written to describe code that exists |
| GLOSSARY | Invented (design session) | Extracted from the codebase |
| Task docs | Required for all work, from P0 | Required only for new work forward |
| AGENTS.md Common Mistakes | Populated by agents during work | Seed from `// FIXME`, `// HACK` comments and Git log |
| Phase numbering | Starts at P0 | Assign current state honestly (P2, P3) — do not pretend to be at P0 |
| `llms-full.txt` | Built incrementally | Generate from `go doc ./...` as a first draft |
| ADRs | Written when decisions are made | Written retroactively only for decisions with active risk of reversal |

### What NEVER Gets Faked

Even for an existing project, these must be accurate from day one:

- **Build and test commands** in AGENTS.md — must be copy-paste runnable.
- **The Most Important Rule** — must reflect the actual non-negotiable invariant, not an aspirational one.
- **GLOSSARY terms** — must use the names actually in the code, not cleaned-up alternatives.
- **Phase label in ROADMAP.md** — must reflect where the project actually is.

Faking these produces documents that actively mislead agents, which is worse than no documents at all.

---

## 16. Helper Scripts

Three scripts in `scripts/` automate the mechanical friction of following the framework. They have no opinions about your code — they only scaffold and verify documents.

### `scripts/init.sh`

Bootstraps a new project in the current directory. Creates the full directory scaffold and copies starter templates from TEMPLATES.md into place. Run once, at project creation.

```sh
bash scripts/init.sh
```

Creates: `docs/tasks/`, `docs/adr/`, `skills/`, `plan/`, `AGENTS.md`, `ONBOARDING.md`, `GLOSSARY.md`, `docs/planning/ROADMAP.md`, `NEXT_STEPS.md`.

**What it does not do:** fill in project-specific content. Every generated file contains a `TODO:` placeholder for each required section. The agent or developer fills those in.

### `scripts/new-task.sh <phase> <slug>`

Creates the next task document in a phase. Auto-sequences by reading existing files under `docs/tasks/`.

```sh
bash scripts/new-task.sh 1 kafka-ingester
# Creates docs/tasks/P1-003-kafka-ingester.md  (if 002 is the current highest)
```

The created file is a copy of the task document minimal template (from TEMPLATES.md) with `Phase`, `Seq`, and `Slug` pre-filled.

**To fill in the task document,** prompt the agent like this:

> Fill in the task document at `docs/tasks/P1-003-kafka-ingester.md`.
> Goal: [one sentence — what does this task produce].
> In scope: [bullet list].
> Out of scope: [bullet list].
> Acceptance criteria must be written as test function names, not prose.
> Note any known interface constraints in the Interface Sketch section.

Do not ask the agent to "plan the implementation" in the same prompt — that conflates scoping (the task doc's job) with implementation (the next session's job). Do not pre-specify the sequence number; the script sets it.

| Include in the prompt | Why |
|---|---|
| One-sentence goal | Forces a precise `Goal:` line; vague goals produce vague implementations |
| Explicit in/out of scope | Prevents scope creep in `Implementation Notes` |
| "test function names" | Prevents prose acceptance criteria that cannot be verified |
| Known interface constraints | Agent fills `Interface Sketch` correctly instead of inventing signatures |

### `scripts/gate-check.sh [phase]`

Validates the phase gate criteria from §10 before you promote to the next phase. Exits non-zero on any failure — usable in CI or as a pre-commit hook.

```sh
bash scripts/gate-check.sh 1
```

**Checks performed:**
1. `go build ./...` succeeds (no stubs left in a broken state).
2. `go test ./...` passes.
3. No `panic("TODO` strings remain in `*.go` files.
4. `// ASSUMPTION:` and `// TODO` count is printed (informational; not a hard fail unless >0 is passed as `--strict`).
5. `go test -coverprofile` is run; prints coverage percentage. Exits non-zero if coverage is below the threshold (default 80%; override with `COVERAGE_THRESHOLD=70`).
6. All task docs in `docs/tasks/P{phase}-*` have `Status: Completed` — warns on any that do not.

Python and TypeScript projects: set `LANG_ADAPTER=python` or `LANG_ADAPTER=ts` to substitute the appropriate build/test commands.

---

## 17. AST-Aided Codebase Navigation

For large codebases, reading source files to answer structural questions
(*"what implements this interface?"*, *"what calls this function?"*, *"where is this type defined?"*)
wastes tokens on code that is irrelevant to the current task. An AST cache built once
with the [`sitting_duck`](https://sitting-duck.readthedocs.io) DuckDB extension answers
these questions in a single SQL query — preserving context budget for the work itself.

This is the concrete mechanism behind Principle 5: **Progressive context disclosure.**

### When to use AST queries vs. file reads

| Question type | Best tool |
|---|---|
| "Where is `FooService` defined?" | AST query (`DEFINITION_TYPE`) |
| "What calls `writeRecord`?" | AST query (Workflow B — execution flow) |
| "What implements the `Writer` interface?" | AST query (Workflow C — interfaces) |
| "What does `writeRecord` actually do?" | Read the file (with `peek := 'full'` or `read_file`) |
| "Is there a bug in this function?" | Read the file |
| "What is the overall structure of this repo?" | AST query (Workflow A — orientation) |

Rule: **Use AST queries to find the right file, then read only that file.**
Never read a file to answer a question that a single SQL query can resolve.

### Setup — building the cache

The cache file `code_ast.duckdb` lives at the project root (gitignored). Build it once;
rebuild only when source files change.

```bash
# One-time: install the extension
duckdb -init /dev/null :memory: -c "INSTALL sitting_duck FROM community;"

# Build cache (Go project example — adapt glob for other languages)
cd /path/to/project
duckdb -init /dev/null code_ast.duckdb <<'SQL'
LOAD sitting_duck;
CREATE OR REPLACE TABLE ast AS
SELECT * FROM read_ast('**/*.go', ignore_errors := true, peek := 200);
SQL
```

For multi-language projects use a pattern array:
```sql
SELECT * FROM read_ast(['**/*.go', '**/*.py'], ignore_errors := true, peek := 200)
```

Add `code_ast.duckdb` to `.gitignore` — it is a build artifact, not source.

### Querying the cache

All queries load the extension and then query the `ast` table:

```bash
duckdb -init /dev/null code_ast.duckdb -c "
LOAD sitting_duck;
-- Find all function definitions
SELECT name, file_path, start_line
FROM ast
WHERE is_function_definition(semantic_type)
ORDER BY file_path, start_line;
"
```

**Critical rule**: `node_id` is per-file, not globally unique. All JOINs on
`node_id`, `parent_id`, or `descendant_count` **must** also match on `file_path`:

```sql
-- WRONG (joins across files)
JOIN ast child ON child.parent_id = parent.node_id
-- CORRECT
JOIN ast child ON child.parent_id = parent.node_id
                AND child.file_path = parent.file_path
```

### Documenting the cache in AGENTS.md

If a project has a `code_ast.duckdb` cache, add an entry to the
**Available Tools & Services** section of `AGENTS.md`:

```markdown
### AST Cache
| Tool | Purpose | Invocation |
|------|---------|------------|
| `code_ast.duckdb` | Structural codebase queries via sitting_duck | `duckdb -init /dev/null code_ast.duckdb -c "LOAD sitting_duck; <SQL>"` |

Prefer AST queries over file reads for: finding definitions, locating callers,
mapping interface implementations, and orientation queries.
Rebuild cache with: `bash scripts/rebuild-ast.sh` (or see §17 of FRAMEWORK.md).
```

### Reusable Macros

Two script files in `scripts/` provide named, documented macros that an agent
can load once and call by name — replacing the need to write raw SQL for common
codebase questions.

**`scripts/macros.sql`** — universal macros, all 27 sitting_duck languages:

| Macro | Answers |
|---|---|
| `find_definitions(pattern, name_like)` | Where is X defined? |
| `find_calls(pattern, name_like)` | Where is X called? |
| `find_imports(pattern)` | What does this file depend on? |
| `find_in_ast(pattern, kind, name_like)` | Find by semantic category (calls/imports/loops/conditionals/…) |
| `code_structure(pattern)` | Which functions are large or complex? |
| `complexity_hotspots(pattern, n)` | Top-N most complex functions by cyclomatic complexity |
| `function_callers(pattern, func_name)` | Who calls a specific function? |
| `structural_diff(file, from, to)` | What definitions changed between two git revisions? (requires `duck_tails`) |
| `changed_function_summary(from, to, pattern)` | Functions in changed files ranked by complexity (requires `duck_tails`) |

**`scripts/macros_go.sql`** — Go-specific macros (query the cached `ast` table):

| Macro | Answers |
|---|---|
| `go_struct_fields()` | All struct fields with Go types |
| `go_func_signatures()` | All functions/methods with typed params and returns |
| `go_type_flow(type_name)` | Where is type X defined, referenced, and used? |
| `go_external_types()` | External package types and how often they appear |
| `go_interfaces()` | All interface definitions with method count |
| `go_interface_impls(iface_name)` | Candidate implementors of a named interface (heuristic) |

**Loading macros into a session:**

```bash
# Load universal macros into the project cache:
duckdb -init /dev/null code_ast.duckdb <<'SQL'
LOAD sitting_duck;
.read scripts/macros.sql
.read scripts/macros_go.sql   -- (Go projects only)
SELECT * FROM find_definitions('**/*.go', 'Transcoder%');
SQL
```

The macros in `macros.sql` use `read_ast()` directly and do not require a
pre-built cache. The macros in `macros_go.sql` query the `ast` table and
require the cache to be built first.

---

## Templates and Language Adapters

Copy-paste starters for all framework documents are in **[TEMPLATES.md](TEMPLATES.md)**:
- AGENTS.md minimal template
- NEXT_STEPS.md minimal template
- ROADMAP.md minimal template
- Task document minimal template
- Language adapters: Python, TypeScript, Rust, C/C++, React/TypeScript Frontend, Go
- Helper scripts: `scripts/init.sh`, `scripts/new-task.sh`, `scripts/gate-check.sh`
- AST macros: `scripts/macros.sql` (universal), `scripts/macros_go.sql` (Go)


---

*This framework is itself a living document. Update it when new patterns improve agent success rates. Version-control every AGENTS.md change with the commit message `docs(agents): {reason for change}`.*
