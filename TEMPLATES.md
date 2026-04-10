# Esquisse — Templates and Language Adapters

Copy-paste starters for bootstrapping a new project with the Esquisse framework.
Part of the [Esquisse framework](FRAMEWORK.md). Document schemas are in [SCHEMAS.md](SCHEMAS.md).

---

## AGENTS.md Minimal Template

````markdown
# AGENTS.md — {Project Name} contributor guide for AI agents

## THE MOST IMPORTANT RULE
Tests define correctness. Fix code to match tests, never the reverse.

## Project Overview
**Module:** `{module-path}`
{One paragraph description.}

## Repository Layout
```
{project}/
├── cmd/{name}/    CLI entry point
├── internal/      Private packages
├── pkg/           Public packages (if any)
├── gen/           Generated code — DO NOT EDIT MANUALLY
├── testdata/      Test fixtures
└── docs/          Plans, decisions, tasks
```

## Build Commands
| Command | Description |
|---------|-------------|
| `go build ./...` | Compile all packages |
| `make test` | Run all tests |
| `make test-race` | Run with race detector |
| `make lint` | Run golangci-lint |
| `make generate` | Regenerate generated code |

## Test Commands
```sh
make test
make test-race
go test -v -run TestFoo ./internal/...
```

## Key Dependencies
| Dependency | Role | Notes |
|-----------|------|-------|
| | | |

## Code Conventions
- All errors returned, never panicked (constructor panics must be documented).
- No global state. All types constructed explicitly.
- Exported symbols require godoc.
- Generated files under `gen/` are read-only.

## Common Mistakes to Avoid
<!-- Add entries here as they are discovered during work sessions. -->

## Invariants
<!-- List things that must always be true about this codebase. -->

## References
- [`llms.txt`](llms.txt)
- [`llms-full.txt`](llms-full.txt)
- [`SCHEMAS.md`](docs/SCHEMAS.md) (if copied into project)
````

---

## docs/planning/NEXT_STEPS.md Minimal Template

````markdown
# {Project} — Next Steps

> Last updated: {YYYY-MM-DD}
> Current phase: P{n} — {Name}
> Active task: [P{n}-{nnn}-{slug}](docs/tasks/P{n}-{nnn}-{slug}.md)

## What Exists
<!-- Annotated inventory of completed work. ✅ Done / ⚠️ Partial / ❌ Missing. -->

## Immediate Next
<!-- Ordered list of tasks ready to start. First item is the active task. -->

## Blocked / Waiting
<!-- Tasks that cannot start yet and the blocking reason. -->

## Session Log
<!-- Append-only. Most recent entry first. -->
<!-- {YYYY-MM-DD} — {task-id} — {one sentence summary} -->
<!-- Files changed: {list} -->
<!-- Up next: {task-id and slug} -->
````

---

## docs/planning/ROADMAP.md Minimal Template

````markdown
# {Project} Roadmap

## Current Phase: P{n} — {Name}
**Target:** {brief goal}

### P{n} Gate Checklist
- [ ] All P{n} tasks have Status: Done
- [ ] `{test command}` passes with 0 failures
- [ ] `{race detector command}` passes (if concurrent)
- [ ] `{lint command}` produces 0 errors
- [ ] AGENTS.md updated with gotchas discovered during P{n}
- [ ] llms.txt and llms-full.txt updated to reflect new API surface
- [ ] No TODO/FIXME comments added (or triaged to new tasks)

### P{n} Tasks
| Task | Status | Summary |
|------|--------|---------|
| [P{n}-001-slug](docs/tasks/P{n}-001-slug.md) | ⬜ Ready | ... |

## Upcoming: P{n+1} — {Name}
...

## Completed
<!-- ### P{n-1} — {Name} ✅ -->
<!-- Completed {date}. Key outcomes: -->
````

---

## docs/tasks/P0-001-{slug}.md Minimal Template

````markdown
# P0-001: {Title}

## Status
Status: Ready
Depends on: None
Blocks: P0-002

## Summary
{One paragraph.}

## Problem
{What is missing.}

## Solution
{High-level approach.}

## Scope
### In Scope
-

### Out of Scope
-

## Prerequisites
- [ ] Repository initialized

## Specification

### Types
```go
// Paste exact signatures here.
```

## Acceptance Criteria
```sh
go test -v -run TestFoo ./...
```

| Test | What It Verifies |
|------|-----------------|
| `TestFoo_HappyPath` | ... |

## Files
| File | Action | Description |
|------|--------|-------------|
| `internal/foo/foo.go` | Create | Core type |
| `internal/foo/foo_test.go` | Create | Unit tests |

## Design Principles
1.

## Session Notes
````

---

### Go

```
Build:       go build ./...
Test:        go test -count=1 ./...
Test (race): go test -count=1 -race -timeout 300s -run '^Test' ./...
Lint:        golangci-lint run ./...     (picks up .golangci.yml automatically)
Generate:    buf generate                (if using protobuf)
Audit:       go vet ./...
```

**Conventions:**
- All errors returned, never panicked, except programmer-contract violations in
  constructors (document in godoc).
- No global state. No `init()` side effects that modify package-level variables.
- Godoc required on all exported types, functions, and methods.
- `zero unnecessary allocations on the hot path` — benchmark with `-benchmem`
  before and after changes to performance-sensitive code.
- Sentinel errors defined in `errors.go` at each package root; wrap with context:
  `fmt.Errorf("pkg.FuncName: %w", ErrSentinel)`.
- Interfaces defined in the consuming package, not the implementing package.
  Keep interfaces ≤3 methods.

**Required files per package:**

| File | Contents |
|------|----------|
| `doc.go` | Package godoc |
| `errors.go` | Sentinel errors |
| `{type}.go` | One primary type per file |
| `example_test.go` | Example functions for pkg.go.dev |

**Agent-facing interface design (add verbatim where relevant):**

```markdown
## Code Conventions — Go Interface Rules

1. **Never boolean flag parameters.** Split into separate functions.
   Wrong: `Delete(path string, force bool)`
   Right: `Delete(path string)` and `DeletePermanently(path string)`

2. **Use typed enums, not bare strings.** Any parameter with a constrained value
   set must be a typed constant. Wrong: `status: string`. Right: `status: Status`.

3. **Every error is actionable.** It encodes (a) what failed, (b) current state,
   (c) the exact next step to recover. Vague errors cause thrashing.

4. **Compound workflows.** If N sequential calls are always required in a fixed
   order, provide a single function that executes the workflow.
```

**Go AST quirks for sitting_duck queries (add to AGENTS.md if using code_ast.duckdb):**

```markdown
## Go AST Notes (sitting_duck)

- **Named types**: `type_spec` node, not `is_class_definition()`. The struct body
  is on the `struct_type` child; the name is on the `type_spec` parent.
- **Methods vs functions**: `method_declaration` has a receiver; `function_declaration`
  does not. Both match `is_function_definition()`.
- **Receiver**: `parameters[1].name` gives the receiver (e.g. `q *DB`).
  Use `list_filter(parameters, lambda x: x.type != 'receiver')` for non-receiver params.
- **Exported**: `SUBSTRING(name, 1, 1) = UPPER(SUBSTRING(name, 1, 1))`.
- **Test exclusions**: `name NOT LIKE 'Test%' AND name NOT LIKE 'Benchmark%' AND name NOT LIKE 'Example%'`
- **Generated file exclusions**: `file_path NOT LIKE '%.pb.go' AND file_path NOT LIKE '%_gen.go'`
- **Critical JOIN rule**: `node_id` is per-file. All JOINs on `node_id` MUST also
  match on `file_path` or results will be silently wrong.

Load Go-specific macros: `.read scripts/macros_go.sql` (see FRAMEWORK.md §17).
```

### Python

```
Build:       uv sync && uv build
Test:        uv run pytest
Lint:        uv run ruff check . && uv run mypy .
Format:      uv run ruff format .
Generate:    (protobuf, if applicable) buf generate
```

**Conventions:**
- Type hints required on all public functions and methods.
- `__all__` defined in every `__init__.py`.
- Google-style docstrings on all public symbols.
- Custom exception hierarchy in `exceptions.py` per package.
- Generated files: `src/{package}/py.typed`, protobuf stubs.

**Required files per package:**

| File | Contents |
|------|----------|
| `__init__.py` | `__all__`, re-exports |
| `exceptions.py` | Package exception hierarchy |
| `py.typed` | PEP 561 marker |

---

### TypeScript / Node

```
Build:       pnpm build
Test:        pnpm test          (vitest or jest)
Lint:        pnpm lint          (eslint)
Format:      pnpm format        (prettier)
Type check:  pnpm typecheck     (tsc --noEmit)
```

**Conventions:**
- `"strict": true` in tsconfig. No `any` without a suppression comment justifying it.
- `interface` over `type` for object shapes; `type` for unions and computed types.
- Generated files: `.d.ts` files, GraphQL types — never edited manually.
- Error pattern: extend `Error`, always set `name` to the class name.

---

### Rust

```
Build:       cargo build
Test:        cargo test
Lint:        cargo clippy -- -D warnings
Format:      cargo fmt
Generate:    (protobuf) cargo build   (prost via build.rs)
```

**Conventions:**
- `#[must_use]` on all types and functions whose return value should not be ignored.
- `thiserror` for library error types; `anyhow` for application error types.
- Generated files: `build.rs` outputs, `prost`/`tonic` protobuf types.
- Document every `unsafe` block with a `// SAFETY:` comment.

---

### C / C++

```
Build:       cmake --build build
Test:        cmake --build build && ctest --test-dir build
Lint:        clang-tidy
Format:      clang-format -i
```

**Conventions:**
- No raw owning pointers; use `std::unique_ptr` / `std::shared_ptr`.
- No `using namespace std` in headers.
- `_test.cpp` suffix for test files; Google Test framework.
- Generated files: protobuf `.pb.h`/`.pb.cc` — never edited manually.

---

### React / TypeScript Frontend

```
Build:       pnpm build
Test:        pnpm test        (vitest or jest + @testing-library)
Lint:        pnpm lint        (eslint)
Format:      pnpm format      (prettier)
Type check:  pnpm typecheck   (tsc --noEmit)
```

**Conventions:**

- Strict TypeScript (`"strict": true`). No `any` without a suppression comment
  explaining why.
- `interface` over `type` for component props and object shapes.
- Generated files: GraphQL types, icon sprites, route types — never edited manually.

**Agent-specific rules (add verbatim to project's AGENTS.md):**

```markdown
## Code Conventions (Frontend)

1. **Never invent CSS class names.** Use only classes defined in the existing
   Tailwind config (`tailwind.config.*`). If a required style is missing,
   note it in Session Notes as a new task — do not add a one-off class.

2. **Never hardcode UI copy.** All user-visible strings must go through the
   i18n/localization system. If the project uses react-i18next, every string
   is `t('key')`. If no i18n system exists, file a task before adding copy.

3. **Tests use roles and data-testid, never DOM structure.**
   - Wrong: `container.querySelector('div > ul > li:first-child')`
   - Right: `screen.getByRole('listitem', { name: /first item/i })`
     or `screen.getByTestId('item-0')`
   - Why: DOM structure changes break tests silently; semantic queries are robust.

4. **data-testid is set by the implementing developer, not the test.**
   Add `data-testid` attributes to the component, not as a test-time trick.

5. **No direct DOM manipulation.** Use React state and refs. `document.querySelector`
   inside component code is forbidden.

6. **Component files are co-located with their test and their style.**
   ```
   src/components/Button/
     Button.tsx
     Button.test.tsx
     Button.module.css   (or Button.stories.tsx for Storybook)
   ```

7. **One component per file.** No barrel-file re-exports of implementation details.
   `index.ts` re-exports the public API only.

8. **State mutation is always through the setter, never direct.**
   - Wrong: `list.push(item); setList(list)`
   - Right: `setList(prev => [...prev, item])`
```

**Required files per component:**

| File | Contents |
|------|----------|
| `{Name}.tsx` | Component implementation |
| `{Name}.test.tsx` | Unit tests (Testing Library) |
| `{Name}.stories.tsx` | Storybook stories (if Storybook is used) |
| `index.ts` | Public re-export only |
