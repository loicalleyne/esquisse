# Per-Task Startup Protocol

Source: FRAMEWORK.md §9

When beginning any task, execute these steps in order before writing any code.

---

## 1. Read AGENTS.md

Internalize:
- All invariants (must not be violated)
- Common mistakes to avoid
- Code conventions
- Repository layout

## 2. Read GLOSSARY.md

Confirm the vocabulary for this domain area.
If a term appears in the task doc and is not in GLOSSARY.md, note it as a
potential addition for the completion protocol.

## 3. Read the Task Document

Read **completely**:
- Summary
- In Scope and Out of Scope
- Prerequisites
- Specification (interfaces, types, algorithm, config)
- Acceptance Criteria
- Files table

## 4. Establish Baseline

Run the full test suite **before touching any code**.

```sh
# Use the test command from AGENTS.md Build Commands.
# Go example:
go test -count=1 -timeout 180s ./...
```

Record the exact pass/fail count and any currently failing tests.
If tests are already failing: document in Session Notes as "PRE-EXISTING FAILURES"
before proceeding. Do not let pre-existing failures be attributed to your changes.

## 5. Locate Files

For each file in the task doc's Files table:

- **If `code_ast.duckdb` exists:** use AST queries or the `explore-codebase`
  skill for structural questions before reading raw files.
- **Otherwise:** read file header and exported interface/type declarations only.

Goal: understand the integration points, not the full implementation.

## 6. Read Relevant Functions

For each function/type marked `Modify` in the Files table:
read the full function or type being changed. Not just the signature.

## 7. Read Acceptance Criteria Carefully

Understand exactly what "passing" means before writing a line.
If any criterion is ambiguous: note it as an assumption (Step 9 below).

## 8. State Assumptions

Write every ambiguity down before starting.

Format in Session Notes:
```
ASSUMPTION: {what was assumed} because {why the task doc was ambiguous}
```

Each assumption becomes an `// ASSUMPTION: {reason}` comment in the code at
the exact decision point.

## 9. Determine Edit Order

Plan the sequence:
1. Which interfaces/types must exist before production code can be written?
2. Which production functions must exist before tests can be written?
3. Which documentation updates wait until a public API is finalized?

## 10. Implement

Follow the edit order strictly:
1. Define types and interfaces (compile after each)
2. Implement production code
3. Write tests (red first if TDD, then make them green)
4. Update documentation

Run tests after each logical unit.

## 11. Verify

Run acceptance criteria tests. Every criterion must be green before calling
the task complete.
