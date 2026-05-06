# P3-008 — adversarial-report-nesting

## Status: Done

## Goal
Replace the flat `.adversarial/reports/` directory with per-slug nested directories `.adversarial/{plan-slug}/`, and change the filename format so iteration number sorts lexicographically in chronological order.

## In Scope
- `esquisse-mcp/adversarial.go`: `reportsDir` path in `newAdversarialHandler`; filename format in `writeReportFile`; anti-destruction path string in the preamble `fmt.Sprintf`
- `esquisse-mcp/adversarial_test.go` (new file): unit tests for `writeReportFile` path format and directory creation
- No changes to `.github/agents/` markdown files (covered by P3-009)

## Out of Scope
- Fix Type table format in reviewer agents or preamble (P3-009)
- EsquissePlan Step 5 protocol changes (P3-010)
- SCHEMAS.md updates (P3-010)
- Any changes to state.go, tools.go, gate.go, artifact.go
- Moving reports to `docs/` — reports stay under `.adversarial/` (gitignored by root `.gitignore`)

## Files

| Path | Action | What changes |
|---|---|---|
| `esquisse-mcp/adversarial.go` | Modify | `reportsDir` path; filename format in `writeReportFile`; anti-destruction path string in preamble |
| `esquisse-mcp/adversarial_test.go` | Create | Tests for `writeReportFile`: path format, dir creation, zero-padding, non-writable dir |
| `SCHEMAS.md` | Modify | §9 title, file naming convention, path examples |

## Specification

### 1. `newAdversarialHandler` — reports directory

Actual current code (inside `newAdversarialHandler`, before the round loop):
```go
reportsDir := filepath.Join(effectiveRoot, ".adversarial", "reports")
if err := os.MkdirAll(reportsDir, 0o700); err != nil {
    return mcpErr("failed to create reports directory %q: %v", reportsDir, err)
}
```

Change to:
```go
// PlanSlug validated via ReadState (calls validateSlug) earlier in handler — safe as dir component.
reportsDir := filepath.Join(effectiveRoot, ".adversarial", input.PlanSlug)
if err := os.MkdirAll(reportsDir, 0o700); err != nil {
    return mcpErr("failed to create reports directory %q: %v", reportsDir, err)
}
```

Design rationale: reports stay under `.adversarial/` (gitignored via root `.gitignore` entry `.adversarial/`) for consistency with state files. Moving to `docs/adversarial/` would make reports git-tracked, breaking the existing gitignore contract.

Security: `input.PlanSlug` is validated by `ReadState(effectiveRoot, input.PlanSlug)` which calls `validateSlug` — already present earlier in the handler. The inline comment is required to document this invariant.

### 2. `writeReportFile` — filename format

Actual current signature (verify in adversarial.go before editing):
```go
func writeReportFile(reportsDir, date, planSlug, usedModel string, iteration, roundNum, rounds int, body string) error
```

Actual current filename format:
```go
fname := fmt.Sprintf("review-%s-%s-iter%d-r%d.md",
    date, planSlug, iteration, roundNum)
```

Change to (reuse the `now` variable already computed for the header):
```go
now := time.Now().UTC()
fname := fmt.Sprintf("iter%02d-%s-%s-review.md",
    iteration,
    date,
    now.Format("1504"),
)
```

Slug removed from filename — it is already the parent directory name. Zero-padded `iter%02d` ensures lexicographic sort = chronological sort up to iter 99.

### 3. Preamble anti-destruction path string

The preamble `fmt.Sprintf` in `newAdversarialHandler` contains:
```
NEVER delete, overwrite, or move any existing report file under .adversarial/reports/.
```

Change to:
```
NEVER delete, overwrite, or move any existing report file under .adversarial/.
```

(Remove the `/reports` suffix — reports now live directly under `.adversarial/{slug}/`.)

### 4. New test file: `adversarial_test.go`

Package: `package main`

All test calls use the actual 8-param signature:
`writeReportFile(reportsDir, date, planSlug, usedModel string, iteration, roundNum, rounds int, body string) error`

**`TestWriteReportFile_PathFormat`**
- Create a temp dir; call `writeReportFile(tmpDir, "2026-04-30", "my-slug", "gpt-4.1", 3, 1, 1, "body")`.
- Read the directory; assert exactly one file exists.
- Assert the filename matches regexp `^iter03-2026-04-30-\d{4}-review\.md$`.

**`TestWriteReportFile_CreatesDir`**
- Call with a `reportsDir` that does not yet exist (subdir of `t.TempDir()`).
- Assert no error and directory exists after call.

**`TestWriteReportFile_ContentHeader`**
- Call and read the file; assert content contains `# Adversarial Review Report: my-slug`.

**`TestWriteReportFile_ZeroPadsIteration`**
- Call with `iteration = 1`; read dir; assert filename starts with `iter01-`.
- Call with `iteration = 10`; read a fresh temp dir; assert filename starts with `iter10-`.

**`TestWriteReportFile_NonWritableDir`** (Linux/macOS only — `t.Skip` on Windows)
- Create a temp dir; `chmod 0o444` on it.
- Call `writeReportFile(nonWritableDir, "2026-04-30", "slug", "model", 1, 1, 1, "body")`.
- Assert returned error is non-nil.
- `defer os.Chmod(nonWritableDir, 0o755)` before test cleanup.

### 5. SCHEMAS.md §9 — Adversarial Review Report Schema

Current §9 header and naming convention:
```
## 9. Adversarial Review Report Schema (`.adversarial/reports/review-{date}-iter{N}-r{round}-{plan-slug}.md`)
```

Change to:
```
## 9. Adversarial Review Report Schema (`.adversarial/{plan-slug}/iter{NN}-{YYYY-MM-DD}-{HHmm}-review.md`)
```

Current file naming convention block (verbatim from SCHEMAS.md lines 630–631):
```
.adversarial/reports/review-{YYYY-MM-DD}-iter{N}-r{round}-{plan-slug}.md
```

Change to:
```
.adversarial/{plan-slug}/iter{NN}-{YYYY-MM-DD}-{HHmm}-review.md
```

Update the segment table: remove `r{round}` row; add `{HHmm}` row (UTC time of review run). Update `{plan-slug}` row note to read "slug subdirectory — pre-validated by `validateSlug`".

Update the two example rows:
| State file | Report file (round 1 of 1) |
|---|---|
| `.adversarial/roadmap.json` | `.adversarial/roadmap/iter00-2026-04-21-1023-review.md` |
| `.adversarial/P8-002-pipeline.json` | `.adversarial/P8-002-pipeline/iter02-2026-04-21-1455-review.md` |

No other changes to §9 body (immutability rule and required content sections remain).

## Acceptance Criteria

- `go test -count=1 -run TestWriteReportFile ./esquisse-mcp/...` — all 5 tests pass
- `go test -count=1 ./esquisse-mcp/...` — full suite passes (no regressions)
- `grep 'filepath.Join(effectiveRoot, ".adversarial", input.PlanSlug)' esquisse-mcp/adversarial.go` — returns 1 match
- `grep 'iter%02d' esquisse-mcp/adversarial.go` — returns 1 match
- `grep -E 'NEVER delete.*\.adversarial/' esquisse-mcp/adversarial.go` — returns 1 match with no `/reports` suffix
- `grep 'iter{NN}' SCHEMAS.md` — returns ≥ 1 match (new naming convention in §9)

## Session Notes

- **2026-04-30**: Implemented by GitHub Copilot (Claude Sonnet 4.6).
  - `adversarial.go`: `reportsDir` changed from `.adversarial/reports` to `.adversarial/{input.PlanSlug}` with required security comment; preamble anti-destruction path updated (removed `/reports` suffix); `writeReportFile` filename changed to `iter%02d-%s-%s-review.md`; `now` moved before `fname` as specified; added `os.MkdirAll` inside `writeReportFile` (required by `TestWriteReportFile_CreatesDir` — function must be self-contained per test spec).
  - `adversarial_test.go`: created with all 5 tests; all pass.
  - `SCHEMAS.md` §9: header, naming convention block, segment table (removed `r{round}`, added `{HHmm}`, updated `{plan-slug}` note), and example rows updated.
  - All 6 acceptance criteria grep checks pass; full `./...` test suite passes with no regressions.
  - Deviation: `writeReportFile` now calls `os.MkdirAll` internally. The handler-level `MkdirAll` is kept as defense-in-depth — idempotent, no harm.
  - **CodeQualityReview found**: `validateSlug` did not reject `".."` (passes `filepath.Clean` + no-slash checks). Fixed in `state.go`: added `strings.Contains(planSlug, "..")` guard. `os.ReadDir` error discards fixed in two tests. Redundancy comment added to handler `MkdirAll`. Re-review: ✅ APPROVED.
- Original planning notes:
  - No existing `adversarial_test.go` in `esquisse-mcp/` — created from scratch.
  - `validateSlug` is defined in `state.go`; called by `ReadState` earlier in the handler. Inline comment documents this.
  - **Design decision — gitignore**: Reports stay under `.adversarial/{slug}/`. Moving to `docs/adversarial/` rejected — would break existing gitignore contract.
  - **Accepted risk (symlink/race)**: `os.MkdirAll` + `os.WriteFile` not atomic. Accepted for local dev tool; `validateSlug` blocks traversal.
