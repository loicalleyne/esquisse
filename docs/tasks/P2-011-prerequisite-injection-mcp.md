# P2-011: Prerequisite Blockquote Injection in write_planning_artifact MCP Tool

## Status
Status: Done
Depends on: P2-010 (schema defines the exact blockquote text this tool must inject)
Blocks: none

## Summary
Extends `write_planning_artifact` to automatically inject the prerequisite blockquote
(defined in P2-010) into each `referenced_by` task file that already exists on disk.
Returns three new fields in the tool response indicating which files were updated,
already up-to-date, or not yet on disk.

## Problem
EsquissePlan must currently inject the blockquote manually after creating an artifact.
This is a multi-step, error-prone process: find the title line, choose the right
`replace_string_in_file` anchor, check idempotency. Automating it in the MCP tool
means the blockquote is injected in a single `write_planning_artifact` call.

## Solution
Add `injectPrerequisite(projectRoot, artifactRelPath, taskPath string)` helper and
an `injectionResult` sentinel type to `artifact.go`. Call the helper for each
`referenced_by` entry after the artifact file is written. Extend `artifactOutput`
with three new `omitempty` slice fields. Existing behavior (artifact file written,
path/word count returned) is unchanged.

## Scope

### In Scope
- `esquisse-mcp/artifact.go`:
  - Add `injectionResult` type with three constants
  - Add `injectPrerequisite(projectRoot, artifactRelPath, taskPath string) (injectionResult, error)`
  - Extend `artifactOutput` with `InjectedInto`, `NotFound`, `AlreadyPresent` fields
  - In the handler, call `injectPrerequisite` for each `referenced_by` entry after
    the artifact is written and `relPath` is known
- `esquisse-mcp/artifact_test.go`:
  - Add 6 new test functions (see Acceptance Criteria)

### Out of Scope
- Any tool for bulk-injecting prerequisites into existing task docs (separate concern)
- Injection into task docs listed in `referenced_by` that do NOT start with `# ` (non-standard
  format; return a Go error from `injectPrerequisite`)
- Changes to SCHEMAS.md, EsquissePlan.agent.md, or FRAMEWORK.md (covered by P2-010)
- Modification of the artifact file content itself

## Prerequisites
- [x] P2-009 is done (`write_planning_artifact` tool exists with its current signature)
- [x] P2-010 is done (exact blockquote text defined in schema)
- [x] Go 1.26.2 toolchain available (`go build ./...` passes in `esquisse-mcp/`)

## Specification

### `injectionResult` type

```go
// injectionResult is returned by injectPrerequisite to indicate the outcome.
type injectionResult int

const (
    injectionResultInjected      injectionResult = iota // blockquote was written
    injectionResultAlreadyPresent                        // blockquote already present — skip
    injectionResultNotFound                              // task file does not exist on disk yet
)
```

### `artifactOutput` extension

Replace the current struct (verified in artifact.go lines ~32–35):

```go
// Current:
type artifactOutput struct {
    Path      string `json:"path"`
    WordCount int    `json:"word_count"`
}
```

With:

```go
// artifactOutput is the structured response for write_planning_artifact.
type artifactOutput struct {
    Path           string   `json:"path"`
    WordCount      int      `json:"word_count"`
    InjectedInto   []string `json:"injected_into,omitempty"`    // task files that received the blockquote
    NotFound       []string `json:"not_found,omitempty"`        // referenced_by paths not yet on disk
    AlreadyPresent []string `json:"already_present,omitempty"` // files already containing the blockquote
}
```

### Multiple-artifact protocol

When a task doc is referenced by **multiple** artifacts, `injectPrerequisite` is called once per artifact (one call per `referenced_by` entry). Each call appends a separate blockquote line. The idempotency check is per-artifact (`marker` contains the specific `artifactRelPath`), so calling with artifact A twice skips the second call but calling with artifact A then artifact B injects two separate lines. The resulting task doc looks like:

```
# P2-010: Some Task

> **Prerequisite:** Read `docs/artifacts/2026-04-21-alpha.md` before writing any code in this phase.
> **Prerequisite:** Read `docs/artifacts/2026-04-21-beta.md` before writing any code in this phase.

## Status
```

Order of blockquotes matches the order of calls (i.e. the order of `referenced_by` entries). No deduplication beyond per-artifact idempotency.

### `injectPrerequisite` function

```go
// injectPrerequisite inserts the prerequisite blockquote immediately after the
// title line of taskPath (resolved relative to projectRoot). It is idempotent:
// calling it twice for the same artifact returns injectionResultAlreadyPresent.
// If taskPath does not exist on disk, returns injectionResultNotFound (not an error).
// Returns a Go error only for I/O failures or a malformed title line (first line
// does not start with "# ").
// This function handles one artifact per call — it is NOT a bulk-injection utility.
// Call it once per artifact for each task file; the handler loop in newArtifactHandler
// is the only intended caller.
func injectPrerequisite(projectRoot, artifactRelPath, taskPath string) (injectionResult, error) {
    absTask := filepath.Join(projectRoot, taskPath)
    data, err := os.ReadFile(absTask)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return injectionResultNotFound, nil
        }
        return 0, err
    }
    content := string(data)

    // Idempotency marker — must match the exact blockquote written below.
    marker := "> **Prerequisite:** Read `" + artifactRelPath + "`"
    if strings.Contains(content, marker) {
        return injectionResultAlreadyPresent, nil
    }

    // Require title line as first line.
    lines := strings.SplitN(content, "\n", 2)
    if !strings.HasPrefix(lines[0], "# ") {
        return 0, fmt.Errorf("injectPrerequisite: %s: first line is not a markdown title", taskPath)
    }

    blockquote := "\n" + marker + " before writing any code in this phase.\n"
    var newContent string
    if len(lines) == 2 {
        newContent = lines[0] + blockquote + lines[1]
    } else {
        newContent = lines[0] + blockquote
    }

    if err := os.WriteFile(absTask, []byte(newContent), 0o644); err != nil {
        return 0, err
    }
    return injectionResultInjected, nil
}
```

**Note:** `errors` package must be added to the import block in `artifact.go`.

### Handler modification

After step 10 (computing `relPath`) and before step 11 (computing `wordCount`),
add step 10b. The handler currently reads (approximately lines 109–136 in artifact.go):

```go
        // 10. Compute relative path.
        relPath, err := filepath.Rel(projectRoot, absPath)
        if err != nil {
            return nil, nil, err
        }
        relPath = filepath.ToSlash(relPath)

        // 11. Compute word count.
        wordCount := len(strings.Fields(content))

        // 12. Return success.
        out := artifactOutput{Path: relPath, WordCount: wordCount}
```

Replace with:

```go
        // 10. Compute relative path.
        relPath, err := filepath.Rel(projectRoot, absPath)
        if err != nil {
            return nil, nil, err
        }
        relPath = filepath.ToSlash(relPath)

        // 10b. Inject prerequisite blockquote into existing referenced task files.
        var injectedInto, notFound, alreadyPresent []string
        for _, entry := range input.ReferencedBy {
            result, injErr := injectPrerequisite(projectRoot, relPath, entry)
            if injErr != nil {
                return nil, nil, injErr
            }
            switch result {
            case injectionResultInjected:
                injectedInto = append(injectedInto, entry)
            case injectionResultAlreadyPresent:
                alreadyPresent = append(alreadyPresent, entry)
            case injectionResultNotFound:
                notFound = append(notFound, entry)
            }
        }

        // 11. Compute word count.
        wordCount := len(strings.Fields(content))

        // 12. Return success.
        out := artifactOutput{
            Path:           relPath,
            WordCount:      wordCount,
            InjectedInto:   injectedInto,
            NotFound:       notFound,
            AlreadyPresent: alreadyPresent,
        }
```

### Required imports

Add `"errors"` to the import block in `artifact.go`. The other required packages
(`os`, `filepath`, `strings`, `fmt`) are already imported.

## Acceptance Criteria

```sh
go test -count=1 -run TestWritePlanningArtifact ./...
go test -count=1 -run TestInjectPrerequisite ./...
go test -count=1 ./...
go build ./...
```

### New test functions (all in `artifact_test.go`, `package main`)

| Test | What It Verifies |
|------|-----------------|
| `TestWritePlanningArtifact_InjectsPrerequisite` | Task file exists → `InjectedInto` populated; blockquote present in task file before `## Status` |
| `TestWritePlanningArtifact_InjectIdempotent` | Second handler call → `AlreadyPresent` populated; task file contains exactly one blockquote |
| `TestWritePlanningArtifact_InjectNotFound` | Task file missing → `NotFound` populated; no error |
| `TestInjectPrerequisite_InsertsAfterTitle` | Blockquote inserted as `\n> **Prerequisite:** Read \`relPath\` before writing any code in this phase.\n` after title line |
| `TestInjectPrerequisite_AlreadyPresent` | Second call returns `injectionResultAlreadyPresent`; file content unchanged |
| `TestInjectPrerequisite_NotFound` | Missing file returns `injectionResultNotFound, nil` |
| `TestWritePlanningArtifact_InjectMultipleArtifacts` | Two artifacts both reference the same task file → file contains two separate blockquote lines in order; both in `InjectedInto` |
| `TestInjectPrerequisite_MalformedTitle` | File whose first line does not start with `# ` returns `(0, non-nil error)`; file content is not modified |

### Exact blockquote text in file (invariant)

For artifact at `relPath = "docs/artifacts/2026-04-21-foo.md"` and a task file whose
first line is `# P2-010: Some Task`, the injected content must match exactly:

```
# P2-010: Some Task
> **Prerequisite:** Read `docs/artifacts/2026-04-21-foo.md` before writing any code in this phase.
...rest of file...
```

(One blank line between title and blockquote, because the existing file has a blank
line after the title that ends up in `lines[1]`. The `blockquote` string starts with
`\n`, so the result is `title\n` + `\n> ...\n` + `\n## Status\n...` = double newline
between title and blockquote, and double newline between blockquote and body.)

### Pre-existing tests

All existing `TestWritePlanningArtifact*` tests must continue to pass without
modification. The new `notFound`/`alreadyPresent`/`injectedInto` fields are
`omitempty` — existing tests that don't create task files in the temp dir will simply
get `NotFound` populated, not an error.

## Files
| File | Action | Description |
|------|--------|-------------|
| `esquisse-mcp/artifact.go` | Modify | Add `injectionResult` type, `injectPrerequisite` func, extend `artifactOutput`, call injection loop in handler, add `"errors"` import |
| `esquisse-mcp/artifact_test.go` | Modify | Add 9 new test functions |

## Planning Artifacts
<!-- No external library research needed — all APIs are stdlib (os, errors, strings) already in use. -->

## Design Principles
1. **Injection is best-effort.** Missing task files are not errors — they are the normal
   case when artifacts are written before task docs. The `not_found` field informs
   EsquissePlan without blocking the write.
2. **Idempotency is a contract.** Calling `write_planning_artifact` a second time with
   the same slug must not produce duplicate blockquotes.
3. **No new dependencies.** `errors.Is(err, os.ErrNotExist)` is stdlib — no new imports
   beyond `"errors"`.

## Testing Strategy
- Unit tests on `injectPrerequisite` directly (it is unexported but in `package main`
  so accessible from `artifact_test.go`): `TestInjectPrerequisite_InsertsAfterTitle`,
  `TestInjectPrerequisite_AlreadyPresent`, `TestInjectPrerequisite_NotFound`,
  `TestInjectPrerequisite_MalformedTitle`.
- Integration tests via the handler: `TestWritePlanningArtifact_InjectsPrerequisite`,
  `TestWritePlanningArtifact_InjectIdempotent`, `TestWritePlanningArtifact_InjectNotFound`,
  `TestWritePlanningArtifact_InjectMultipleArtifacts`.
- `t.TempDir()` for all temporary directories — no manual cleanup.
- Malformed-file test: create a task file whose first line is NOT `# ` (e.g. starts
  with `Status:`); assert `injectPrerequisite` returns a non-nil Go error and the file
  content is unchanged.

## Session Notes
<!-- 2026-04-21 — EsquissePlan — Created during planning session for prerequisite-injection feature. -->
2026-04-21 — ImplementerAgent — Completed. injectionResult type, injectPrerequisite func, artifactOutput extension, handler step 10b, and 9 new tests all implemented. All tests pass.
