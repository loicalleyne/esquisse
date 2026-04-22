# P2-009: esquisse-mcp write_planning_artifact Tool

## Status
Status: Done
Depends on: P2-007 (schema defines the artifact format this tool must produce)
Blocks: none

## Summary
Adds a `write_planning_artifact` tool to `esquisse-mcp` that accepts structured
research input from EsquissePlan and writes a correctly formatted Planning Artifact
file to `docs/artifacts/{YYYY-MM-DD}-{slug}.md`. Ensures artifact files are
schema-compliant (correct section headers, immutability metadata) without requiring
EsquissePlan to format markdown manually, and returns the written file path and an
estimated token count so EsquissePlan can check the artifact stays within size guidance.

## Problem
EsquissePlan can write artifact files via `create_file`, but there is no structural
enforcement that the output matches SCHEMAS.md §10. A free-form write can omit required
sections, mis-label headers, or exceed the per-section token guidance — all of which
degrade the implementor agent's ability to use the artifact reliably. A dedicated MCP
tool encapsulates formatting and provides a machine-readable path for downstream
referencing.

## Solution
New file `esquisse-mcp/artifact.go` implementing the `write_planning_artifact` handler.
Register it in `tools.go`. The handler receives structured fields, assembles the §10
schema, writes the file, and returns the path and word count. Path traversal is prevented
the same way `state.go` prevents it: `validateSlug` applied to the artifact slug before
constructing the path. The `docs/artifacts/` directory is created if absent.

## Scope

### In Scope
- `esquisse-mcp/artifact.go`: new file — `artifactInput` struct, `newArtifactHandler`
  func, slug validation (reuse `validateSlug` from state.go), directory creation,
  template assembly, file write, response with path + word count
- `esquisse-mcp/tools.go`: register `write_planning_artifact` tool with description
- `esquisse-mcp/artifact_test.go`: new file — unit tests for happy path, slug
  validation rejection, missing-section omission (Minimal Examples omitted when blank)

### Out of Scope
- Any tool for reading or listing artifacts (agents use `read_file` directly)
- Any tool for updating the `Referenced by:` field after-the-fact (EsquissePlan does
  this in the initial write by passing the list at creation time)
- Changes to SCHEMAS.md, FRAMEWORK.md, or any skill files (P2-007, P2-008)
- Changes to gate-check.sh, gate-review.sh, or adversarial review infrastructure

## Prerequisites
- [ ] P2-007 is done (SCHEMAS.md §10 defines the canonical artifact sections)
- [ ] Go 1.26.2 toolchain available (`go build ./...` passes in esquisse-mcp/)

## Specification

### artifactInput struct

```go
// artifactInput is the input schema for the write_planning_artifact tool.
// All string fields accept multi-line content.
// MinimalExamples may be empty — the section is omitted from the output when blank.
type artifactInput struct {
    // Title is the human-readable artifact title, e.g. "franz-go Consumer API".
    Title string `json:"title"`
    // Slug is the kebab-case identifier used in the filename,
    // e.g. "franz-go-consumer-api". Must not contain path separators.
    Slug string `json:"slug"`
    // Source describes the origin of the research:
    // a URL, module path, or "AST analysis of {package}".
    Source string `json:"source"`
    // ReferencedBy is a slice of workspace-relative paths to task documents
    // that reference this artifact, e.g.:
    //   ["docs/tasks/P2-007-planning-artifact-schema.md",
    //    "docs/tasks/P2-008-planning-artifact-consumption.md"]
    // Each path is rendered as a relative markdown link from docs/artifacts/
    // to the task file. The display text is derived from the filename stem,
    // e.g. "docs/tasks/P2-007-planning-artifact-schema.md" → "[P2-007-planning-artifact-schema](../tasks/P2-007-planning-artifact-schema.md)".
    // Validated: each entry must not contain '..' components after filepath.Clean.
    ReferencedBy []string `json:"referenced_by"`
    // Summary is 2-3 sentences on what this artifact covers and why it matters.
    Summary string `json:"summary"`
    // APISurface is the condensed API facts section content (tables/lists).
    APISurface string `json:"api_surface"`
    // Constraints is the MUST/MUST NOT rules section content.
    Constraints string `json:"constraints"`
    // AntiPatterns is the Wrong/Right/Why patterns section content.
    AntiPatterns string `json:"anti_patterns"`
    // MinimalExamples is the copy-paste code snippets section content.
    // If empty, the section is omitted from the output file.
    MinimalExamples string `json:"minimal_examples,omitempty"`
}
```

### artifactOutput struct

```go
// artifactOutput is the structured response for write_planning_artifact.
type artifactOutput struct {
    // Path is the relative path written, e.g. "docs/artifacts/2026-04-21-franz-go.md".
    Path string `json:"path"`
    // WordCount is a rough word count of the written file (whitespace-split tokens).
    // Guidance: keep below 600 words (~800 tokens) per artifact.
    WordCount int `json:"word_count"`
}
```

### newArtifactHandler func signature

```go
func newArtifactHandler(projectRoot string) func(context.Context, *mcp.CallToolRequest, artifactInput) (*mcp.CallToolResult, any, error)
```

### Preconditions
- `projectRoot` must be an absolute path (validated by caller in `main.go`, same as other handlers).
- `input.Slug` must pass `validateSlug` — returns tool error if invalid.
- `input.Title`, `input.Source`, `input.Summary`, `input.APISurface`, `input.Constraints`, `input.AntiPatterns` must all be non-empty after `strings.TrimSpace` — returns tool error naming the missing field, e.g. `"required field 'summary' is empty"`.
- `input.MinimalExamples` is optional — section omitted when `strings.TrimSpace(input.MinimalExamples) == ""`. Whitespace-only content is treated as empty.
- `input.ReferencedBy` entries are validated fail-fast: iterate entries in order; for the first entry where `strings.Contains(filepath.Clean(entry), "..")` is true, return a tool error immediately identifying the offending path (e.g. `"invalid referenced_by path: ../../../etc/passwd"`). Do not continue to subsequent entries once an invalid one is found. After `filepath.Clean(entry)`, the cleaned path must not equal `".."`, must not start with `"../"` or `"..\\"`, and must not contain the string `".."` as a path component. In code: `strings.Contains(filepath.Clean(entry), "..")` is the guard; if true, return a tool error. An empty slice is valid (artifact not yet linked to any task).

### Postconditions
- File written to `{projectRoot}/docs/artifacts/{today}-{slug}.md` where `{today}` is `time.Now().UTC().Format("2006-01-02")`.
- `docs/artifacts/` directory created if absent via `os.MkdirAll(dir, 0o755)`. If `os.MkdirAll` returns an error, return it as a Go error (not a tool error) — this is an unrecoverable I/O failure.
- File overwrites an existing file at the same path (idempotent re-runs on same date).
- Returns `artifactOutput` where:
  - `Path` is computed as `filepath.Rel(projectRoot, absPath)` — always a forward-slash path relative to the project root, e.g. `docs/artifacts/2026-04-21-franz-go.md`.
  - `WordCount` is `len(strings.Fields(fileContent))` — whitespace-split word count of the full written file content.

### Template assembly
The handler assembles the file using a Go `text/template` or direct string formatting.

**`Referenced by:` link construction:** for each entry in `ReferencedBy`:
- `stem` = `strings.TrimSuffix(filepath.Base(entry), ".md")`, e.g. `"P2-007-planning-artifact-schema"`
- `basename` = `filepath.Base(entry)`, e.g. `"P2-007-planning-artifact-schema.md"`
- rendered link = `"[" + stem + "](../tasks/" + basename + ")"`
- all links joined by `", "`
- example output: `[P2-007-planning-artifact-schema](../tasks/P2-007-planning-artifact-schema.md), [P2-008-planning-artifact-consumption](../tasks/P2-008-planning-artifact-consumption.md)`

The exact template content (no editorial comments — every line below appears verbatim in the written file):
```
# Artifact: {Title}

**Primary Source:** {Source}
**Date:** {today}
**Produced by:** EsquissePlan
**Referenced by:** {rendered links joined by ", "}

---

## Summary
{Summary}

## API Surface / Key Facts
{APISurface}

## Constraints
{Constraints}

## Anti-Patterns
{AntiPatterns}

[## Minimal Examples  ← omit entire section when MinimalExamples is empty]
{MinimalExamples}
```

Note: `APISurface` is a freeform string. The tool does not validate that it contains
a three-column table — schema compliance (Symbol / Exact Signature / Source columns)
is the responsibility of EsquissePlan when calling the tool. The tool's job is
formatting and file I/O only.

### Security
- `input.Slug` is validated with `validateSlug` (same function as `state.go`) before being used in `filepath.Join`. This prevents `../../etc/evil` path traversal.
- `projectRoot` is trusted (passed from `main.go` which validates it at startup).
- File content is agent-controlled text written to `docs/artifacts/` only. No shell execution.

### Tool registration in tools.go
```go
mcp.AddTool(server, &mcp.Tool{
    Name: "write_planning_artifact",
    Description: "Write a Planning Artifact file to docs/artifacts/{date}-{slug}.md. " +
        "Accepts structured research content (API surface, constraints, anti-patterns, " +
        "optional minimal examples) and produces a schema-compliant artifact per SCHEMAS.md §10. " +
        "Returns the written file path (relative to project root) and word count. " +
        "Use during planning, after identifying external libraries needed by ≥ 2 tasks " +
        "or whose API surface exceeds ~400 tokens inline. " +
        "Do NOT use for internal project code — AGENTS.md and llms.txt cover those surfaces.",
}, newArtifactHandler(projectRoot))
```

### Known Risks / Failure Modes
| Risk | Mitigation |
|---|---|
| Slug with `/` or `..` escapes docs/artifacts/ | `validateSlug` rejects any slug with path separators |
| Large artifact bloats context at implementation time | WordCount in response; SCHEMAS.md §10 guidance says keep below 600 words |
| File written on wrong date (time zone) | Always use `time.Now().UTC()` — explicit UTC prevents cross-midnight confusion |
| MinimalExamples contains a section header that disrupts parsing | Section is fenced in the template; no parsing of content occurs — written verbatim |

## Acceptance Criteria

```sh
# From the esquisse-mcp/ directory:
go test -count=1 -run TestWritePlanningArtifact ./...
go test -count=1 -run TestWritePlanningArtifact_SlugValidation ./...
go test -count=1 -run TestWritePlanningArtifact_MinimalExamplesOmitted ./...
go test -count=1 -run TestWritePlanningArtifact_MinimalExamplesWhitespaceOmitted ./...
go test -count=1 -run TestWritePlanningArtifact_MissingRequiredField ./...
go test -count=1 -run TestWritePlanningArtifact_ReferencedByLinks ./...
go test -count=1 -run TestWritePlanningArtifact_ReferencedByTraversal ./...
go test -count=1 -run TestWritePlanningArtifact_ReferencedByEmpty ./...
go build ./...
```

| Test | What It Verifies |
|---|---|
| `TestWritePlanningArtifact` | Happy path: all fields provided, file written at `filepath.Rel(projectRoot, absPath)`, word count == `len(strings.Fields(content))` > 0, `Referenced by:` renders as `[P2-007-planning-artifact-schema](../tasks/P2-007-planning-artifact-schema.md)` |
| `TestWritePlanningArtifact_SlugValidation` | Slug `"../evil"` returns tool error, no file written |
| `TestWritePlanningArtifact_MinimalExamplesOmitted` | Empty `MinimalExamples` → output file has no `## Minimal Examples` section |
| `TestWritePlanningArtifact_MinimalExamplesWhitespaceOmitted` | Whitespace-only `MinimalExamples` (e.g. `"   \n  "`) treated as empty → section omitted |
| `TestWritePlanningArtifact_MissingRequiredField` | Empty `Summary` returns tool error naming the field `"summary"` |
| `TestWritePlanningArtifact_ReferencedByLinks` | `ReferencedBy: ["docs/tasks/P2-007-foo.md"]` → output contains `[P2-007-foo](../tasks/P2-007-foo.md)` |
| `TestWritePlanningArtifact_ReferencedByTraversal` | `ReferencedBy: ["../../../etc/passwd"]` returns tool error, no file written |
| `TestWritePlanningArtifact_ReferencedByEmpty` | `ReferencedBy: []` → `**Referenced by:**` line is present but empty (not omitted) |

All pre-existing `esquisse-mcp` tests must continue to pass: `go test -count=1 ./...`

## Files
| File | Action | Description |
|------|--------|-------------|
| `esquisse-mcp/artifact.go` | Create | `artifactInput`, `artifactOutput`, `newArtifactHandler` |
| `esquisse-mcp/artifact_test.go` | Create | 8 test functions listed in Acceptance Criteria |
| `esquisse-mcp/tools.go` | Modify | Register `write_planning_artifact` via `mcp.AddTool` |

## Design Principles
1. **Reuse `validateSlug`.** The same path-traversal defence used by state files applies to artifact slugs. Do not write a second validation function.
2. **Omit-when-empty for MinimalExamples.** Absence of the section is meaningful — it signals no code examples were needed. Do not emit a blank section.
3. **UTC date, always.** The artifact date is a freshness signal. Always use UTC to prevent agents running in different time zones from producing different filenames for the same planning session.
4. **Tool error vs. Go error.** Input validation failures (bad slug, missing field) return a structured tool error that the MCP caller can act on. Internal I/O failures (can't write file) return a Go error.

## Testing Strategy
- Unit tests use `t.TempDir()` as `projectRoot` — no real filesystem dependencies.
- Test for path traversal by passing `../evil` as slug and asserting the function returns an error and does not write any file.
- Test MinimalExamples omission by checking the written file content does not contain `## Minimal Examples`.
- Test MissingRequiredField by passing empty `Summary` and asserting error message names `"summary"`.

## Session Notes
<!-- Append-only. Never overwrite. -->
<!-- 2026-04-21 — EsquissePlan — Created. Depends on P2-007 for schema. Reuses validateSlug from state.go. -->
<!-- 2026-04-21 — EsquissePlan — Revised after Adversarial-r0 CONDITIONAL (iter 0). Added strings.TrimSpace normalization for all required fields and MinimalExamples (M3, M7). Added whitespace-only MinimalExamples test case. -->
<!-- 2026-04-21 — EsquissePlan — Adversarial-r2 CONDITIONAL (iter 2). Fixed: (1) ReferencedBy validation now explicitly fail-fast — first invalid entry returns tool error naming offending path, no further entries processed. (2) os.MkdirAll failure now explicitly returns Go error with 0o755 permission stated. -->
<!-- 2026-04-21 — ImplementerAgent — Completed. artifact.go and artifact_test.go created; tool registered in tools.go; all 8 tests pass. -->
