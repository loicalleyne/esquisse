# P3-009 — reviewer-fix-type-table

> **Depends on**: P3-008 must be complete before this task (both modify `adversarial.go`; P3-009 edits a different section of `reviewPromptTemplate` than P3-008).

## Status: Done

## Goal
All three adversarial reviewer agents emit a structured Fix Type table in Required Changes instead of a prose bullet list, reference the new nested report path, have their tool lists trimmed to the minimum needed for document review, and the `reviewPromptTemplate` in `esquisse-mcp` is updated to match.

## In Scope
- `.github/agents/Adversarial-r0.agent.md`: Fix Type table in Output Format, new report path, tool list trim
- `.github/agents/Adversarial-r1.agent.md`: same
- `.github/agents/Adversarial-r2.agent.md`: same
- `esquisse-mcp/adversarial.go`: add Fix Type table instruction to `reviewPromptTemplate` Required Output section
- `GLOSSARY.md`: add Fix Type enum definition

## Out of Scope
- EsquissePlan Step 5 response protocol (P3-010)
- `adversarial-review` SKILL.md (P3-010)
- SCHEMAS.md planner-fixes artifact definition (P3-010)
- Reviewer agent 7-attack text — keep exactly as-is
- Core Directive structure — keep exactly as-is except the single "Do not soften them." → "Report issues at full severity." replacement specified in §3

## Files

| Path | Action | What changes |
|---|---|---|
| `.github/agents/Adversarial-r0.agent.md` | Modify | Tool list; Output Format report path; Required Changes format |
| `.github/agents/Adversarial-r1.agent.md` | Modify | Same three sections |
| `.github/agents/Adversarial-r2.agent.md` | Modify | Same three sections |
| `esquisse-mcp/adversarial.go` | Modify | `reviewPromptTemplate` Required Output section: add Fix Type table instruction |
| `GLOSSARY.md` | Modify | Add `Fix Type` entry |

## Specification

### 1. Tool list — all three reviewer agents

Replace the current `tools:` list in each reviewer agent frontmatter with:
```yaml
tools:
  [read/readFile, search/fileSearch, search/textSearch, search/listDirectory, edit/createFile, edit/editFiles]
```

Removed tools (not needed for reviewing plan documents): `vscode/memory`, `read/getNotebookSummary`, `read/problems`, `read/viewImage`, `read/readNotebookCellOutput`, `read/terminalSelection`, `read/terminalLastCommand`, `search/changes`, `search/codebase`, `search/usages`.

`edit/editFiles` is retained because reviewers overwrite the existing state JSON file (`create_file` cannot overwrite existing files). `edit/createFile` is retained for writing new report files.

### 2. Output Format — report path in all three reviewer agents

Locate the `### Output Format` section in each agent. Change the report path instruction from:
```
`.adversarial/reports/review-{YYYY-MM-DD}-iter{N}-r{round}-{plan-slug}.md`
```
to:
```
`.adversarial/{slug}/iter{NN}-{YYYY-MM-DD}-{HHmm}-review.md`
```

Also update the `report_path` value in the example JSON block:
```json
"report_path": ".adversarial/reports/review-{filename}.md"
```
becomes:
```json
"report_path": ".adversarial/{slug}/iter{NN}-{YYYY-MM-DD}-{HHmm}-review.md"
```

**Verify exact current text in each agent file before editing** — path format varies per agent. Read the file verbatim first.

### 3. Required Changes format — all three reviewer agents

The three reviewer agent files do **not** currently have a `## Required Changes` section. Add the following new section to each agent, placed immediately before `## Adversarial Constraints`:

```markdown
## Required Changes

| Priority | Attack | Issue | Fix Type | Concrete Action |
|---|---|---|---|---|
| BLOCKING | A{N} | {one-line description} | `{Fix Type}` | {specific runnable action} |
| ADVISORY | A{N} | {one-line description} | `{Fix Type}` | {specific runnable action} |
```

Valid `Fix Type` values — use exactly one of these per row:

| Fix Type | When to use | Required Concrete Action |
|---|---|---|
| `PLANNING_ARTIFACT` | External API signature unverified | Run `go doc {pkg} {symbol}`; create `docs/artifacts/YYYY-MM-DD-{slug}.md` with API Surface table |
| `DEPENDENCY` | Package absent from `go.mod` / `package.json` | Run `go get {pkg}@{version}`; record exact version in task Specification |
| `TEST_NAME` | Test missing or named vaguely | Provide exact Go test function name to add to Acceptance Criteria |
| `SPEC_EDIT` | Specification text is wrong or ambiguous | Name the exact section to rewrite; provide sourced replacement |
| `TASK_SPLIT` | Task exceeds single-session scope | Name the boundary at which to split |
| `SCOPE_REMOVE` | Out-of-scope work present | Name the exact section or function to delete from the task |

Remove the "Do not soften them." sentence from the Core Directive. Replace with:
```
Report issues at full severity.
```

### 4. MCP preamble — Required Output section in adversarial.go

The reviewer model receives a preamble built by an inline `fmt.Sprintf` in `newAdversarialHandler` (no named const). After the existing "Write a report to: ..." paragraph in that preamble, add:

```
Format your Required Changes section as a markdown table:

| Priority | Attack | Issue | Fix Type | Concrete Action |
|---|---|---|---|---|

Priority: BLOCKING or ADVISORY
Fix Type (use exactly one): PLANNING_ARTIFACT | DEPENDENCY | TEST_NAME | SPEC_EDIT | TASK_SPLIT | SCOPE_REMOVE
Concrete Action: a specific runnable command or edit, not a description of intent.
```

### 5. GLOSSARY.md — Fix Type entry

Add to the glossary (alphabetical position between "F" entries or at the end of the "F" section, or as a new entry if no "F" section exists):

```markdown
**Fix Type**: The required remediation category for a reviewer-flagged issue. One of:
`PLANNING_ARTIFACT`, `DEPENDENCY`, `TEST_NAME`, `SPEC_EDIT`, `TASK_SPLIT`, `SCOPE_REMOVE`.
Used in the Required Changes table produced by adversarial reviewers and consumed by
EsquissePlan's Step 5 Fix Type response protocol.
```

## Acceptance Criteria

- `grep -c "Fix Type" .github/agents/Adversarial-r0.agent.md` returns ≥ 2
- `grep -c "Fix Type" .github/agents/Adversarial-r1.agent.md` returns ≥ 2
- `grep -c "Fix Type" .github/agents/Adversarial-r2.agent.md` returns ≥ 2
- `grep "PLANNING_ARTIFACT" esquisse-mcp/adversarial.go` returns 1 match (in inline preamble)
- `grep "read/readFile" .github/agents/Adversarial-r0.agent.md` returns 1 match (tool list)
- `grep "edit/editFiles" .github/agents/Adversarial-r0.agent.md` returns 1 match (tool list retained)
- `grep "vscode/memory" .github/agents/Adversarial-r0.agent.md` returns 0 matches (removed)
- `grep "Fix Type" GLOSSARY.md` returns ≥ 1 match
- `go test -count=1 ./esquisse-mcp/...` passes (no logic changes, only string constant)

## Session Notes

- Each reviewer agent is ~600 tokens. The tool list trim removes ~12 tool entries per agent.
- The 7-attack text must not change — only Output Format and Required Changes sections.
- "Do not soften them." → "Report issues at full severity." is the only Core Directive change.
- Reviewer agent files live in `esquisse/.github/agents/` (source). Deployment to `~/.copilot/agents/` is via `bash scripts/upgrade.sh --target-dir {project}` (upgrade.sh §2 handles this automatically).
- [WARN] planning_context capture skipped: no `code_ast.duckdb` at esquisse root.

### Session Notes — 2026-04-30

- **Status**: Completed
- **What was implemented**: All five files updated per spec. Tools list trimmed to `[read/readFile, search/fileSearch, search/textSearch, search/listDirectory, edit/createFile, edit/editFiles]` in all three reviewer agents. Report path updated from `.adversarial/reports/review-...` to `.adversarial/{slug}/iter{NN}-...` in Step 3 of each agent. `## Required Changes` section with Fix Type table added before `## Adversarial Constraints` in all three agents. "Do not soften verdicts. FAILED means FAILED." replaced with "Report issues at full severity." Fix Type table instruction added to `adversarial.go` inline preamble. `Fix Type` entry added to `GLOSSARY.md`.
- **Deviations from plan**: §2 mentions a `report_path` JSON block to update — no such block exists in current agent files (P3-008 In Scope explicitly excluded agent files). Change applied only to Step 3 path text. Spec quotes "Do not soften them." but actual text was "Do not soften verdicts. FAILED means FAILED." — replaced as intended.
- **gofmt**: Applied `gofumpt -w adversarial.go` after insertion to normalize indentation.
- **All acceptance criteria**: PASS. Spec review: COMPLIANT. Quality review: APPROVED.
