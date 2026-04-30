# P3-010 — planner-hardening

> **Depends on**: P3-009 (Fix Type enum must be defined in GLOSSARY.md before EsquissePlan references it).

## Status
Done

## Goal
EsquissePlan applies structural Fix Type actions after CONDITIONAL/FAILED verdicts rather than prose patches; Step 2c is compressed to remove redundant SQL examples; guardrail negations are rewritten as affirmative imperatives; a Critical Rules box appears at the top of the file for U-shaped recall mitigation; the adversarial-review skill gains a condensed Fix Type response protocol; SCHEMAS.md gains §11 for the planner-fixes artifact; AGENTS.md Common Mistakes gains the negation anti-pattern.

## In Scope
- `.github/agents/EsquissePlan.agent.md`: critical rules box, Step 2c compression, Step 5 replacement, guardrail negation rewrites
- `skills/adversarial-review/SKILL.md`: Step 5 condensed Fix Type protocol
- `SCHEMAS.md`: new §11 — Planner Fixes Artifact
- `AGENTS.md`: new Common Mistakes entry — negation anti-pattern

## Out of Scope
- Reviewer agent files (P3-009)
- esquisse-mcp Go code (P3-008, P3-009)
- install/deploy scripts (upgrade.sh already handles deployment)
- `implement-task` SKILL.md, `write-spec` SKILL.md, `explore-codebase` SKILL.md (no changes needed)
- EsquissePlan tool list (already updated by the user prior to this task)

## Files

| Path | Action | What changes |
|---|---|---|
| `.github/agents/EsquissePlan.agent.md` | Modify | Add critical rules box; compress Step 2c; replace Step 5; rewrite 4 guardrail lines |
| `skills/adversarial-review/SKILL.md` | Modify | Replace Step 5 prose with condensed Fix Type protocol |
| `SCHEMAS.md` | Modify | Append §11 — Planner Fixes Artifact |
| `AGENTS.md` | Modify | Append negation anti-pattern to Common Mistakes |

## Specification

### 1. EsquissePlan.agent.md — Critical Rules box

Insert immediately after the closing `---` of the YAML frontmatter, before the first prose line:

```markdown
> **Critical Rules (always active)**
> 1. Write documents only — ImplementerAgent writes all code.
> 2. Every API fact in a Specification requires a sourced retrieval performed in this session.
> 3. Hand off to ImplementerAgent only when verdict is PASSED, or CONDITIONAL with all BLOCKING fixes resolved.
> 4. Write to `docs/` only — all other paths belong to ImplementerAgent.
```

### 2. EsquissePlan.agent.md — Step 2c compression

Replace the entire `### Step 2c — Capture Planning Context` section (from the `### Step 2c` heading through the closing SQL code block) with:

```markdown
### Step 2c — Capture Planning Context

Capture `modify`/`implement` symbols into `planning_context` before writing task docs.

**Preconditions (check in order):**
- `code_ast.duckdb` missing or any `.go` files changed since last build → `bash scripts/rebuild-ast.sh`
- `duckdb` CLI unavailable → emit `[WARN] planning_context capture skipped` in Session Notes; skip this step

**Per task** (full macro syntax: `scripts/macros_go.sql`):
1. `DELETE FROM planning_context WHERE task_id = '{task_id}';`
2. `INSERT INTO planning_context SELECT * FROM capture_planning_context('{task_id}', 'modify', '**/*.go', '{Symbol}%');`
3. `SELECT count(*) FROM planning_context WHERE task_id = '{task_id}';`
   - 0 rows → emit `[WARN] 0 rows captured for {task_id}` in Session Notes
   - Any row with null signature → emit `[WARN] missing signature for {task_id}` in Session Notes
   - SQL error → emit `[WARN] capture failed for {task_id}: {error}` in Session Notes; proceed without capture
```

### 3. EsquissePlan.agent.md — Step 5 replacement

Replace the entire `### Step 5: Respond to verdict` section (from the `### Step 5` heading through the closing `> RULE:` blockquote) with:

```markdown
### Step 5: Respond to verdict

**PASSED**: Inform the user. Hand off to ImplementerAgent.

**CONDITIONAL or FAILED**:

1. **Read all past reports for this slug.**
   `list_dir(".adversarial/{slug}/")` → `read_file` each `*-review.md`.
   Build an issue table with columns: Resolved / Recurring / New.

2. **Execute Fix Type actions** — no prose patches:
   - `PLANNING_ARTIFACT` → run `go doc {pkg} {symbol}` or fetch docs; create
     `docs/artifacts/{YYYY-MM-DD}-{slug}.md` with three-column API Surface table
   - `DEPENDENCY` → run `go get {pkg}@{version}` in terminal; record exact
     version in the task Specification
   - `TEST_NAME` → add the exact named function to Acceptance Criteria
   - `SPEC_EDIT` → rewrite the named Specification section using sourced facts only
   - `TASK_SPLIT` → create a new task doc at the named boundary
   - `SCOPE_REMOVE` → delete the named section from the task

3. **Self-verify each BLOCKING fix.** Re-read the revised task section and confirm
   the reviewer's original objection is structurally satisfied — not just mentioned.

4. **Write planner-fixes artifact** to
   `docs/adversarial/{slug}/iter{NN}-{YYYY-MM-DD}-{HHmm}-planner-fixes.md`
   (schema: SCHEMAS.md §11). This replaces reporting to the user.

5. Return to Step 4. Next reviewer slot = current `.adversarial/{slug}.json` iteration % 3.
```

### 4. EsquissePlan.agent.md — Guardrail negation rewrites

In the `## Guardrails` section, make these four targeted replacements:

| Find (exact text) | Replace with |
|---|---|
| `- Never write code. EsquissePlan writes documents; implementation agents write code.` | `Write documents only. ImplementerAgent writes all code.` |
| `- Never mark a task Status: Completed. Only implementation agents do this.` | `Set task Status to \`Ready\` or \`In Progress\` only. ImplementerAgent sets \`Completed\`.` |
| `- Never modify files outside \`docs/\` during planning.` | `Write to \`docs/\` only. All other paths belong to ImplementerAgent.` |
| `- **Never put ungrounded API facts in a Specification.** If a task Specification references an external library API, every signature, field name, and behavioural rule must come from an actual retrieval (running \`go doc\`, reading source, fetching documentation) performed during this planning session — not from training data. An ungrounded API claim in a task doc is a hallucination waiting to happen at implementation time. If retrieval is impractical, omit the fact and note the gap in Session Notes.` | `**Every API fact in a Specification requires a sourced retrieval performed in this session.** Run \`go doc\`, read source, or fetch documentation — never rely on training data. If retrieval is impractical, omit the fact and record the gap in Session Notes.` |

### 5. adversarial-review SKILL.md — Step 5 replacement

Replace the current `## Step 5 — React to verdict` section (from the `## Step 5` heading through the final `> Do not hand off...` blockquote) with:

```markdown
## Step 5 — React to verdict

**PASSED**: Signal that implementation may begin.

**CONDITIONAL or FAILED**:

1. Read all past reports: `list_dir(".adversarial/{slug}/")` → `read_file` each `*-review.md`.
   Build: Resolved / Recurring / New issue table.
2. Execute Fix Type actions (no prose patches):
   - `PLANNING_ARTIFACT` → run `go doc`/fetch docs; create `docs/artifacts/YYYY-MM-DD-{slug}.md`
   - `DEPENDENCY` → `go get {pkg}@{version}`; record exact version in task Specification
   - `TEST_NAME` → add exact function name to Acceptance Criteria
   - `SPEC_EDIT` → rewrite the named section with sourced facts
   - `TASK_SPLIT` → create new task doc at the named boundary
   - `SCOPE_REMOVE` → delete the named section
3. Write planner-fixes artifact:
   `docs/adversarial/{slug}/iter{NN}-{YYYY-MM-DD}-{HHmm}-planner-fixes.md` (SCHEMAS.md §11)
4. Return to Step 1. Next slot = incremented iteration % 3.

Hand off to ImplementerAgent only when verdict is PASSED, or CONDITIONAL with all BLOCKING fixes resolved.
```

### 6. SCHEMAS.md — Append §11

Append after the last existing section (currently §10 — Planning Artifact):

```markdown
## §11 — Planner Fixes Artifact

**Purpose:** Records the structural fixes applied by EsquissePlan after a CONDITIONAL or FAILED
adversarial review verdict. Provides the audit trail for the review cycle.

**Path:** `docs/adversarial/{plan-slug}/iter{NN}-{YYYY-MM-DD}-{HHmm}-planner-fixes.md`

Where `{NN}` is the zero-padded iteration number matching the review file this responds to. Files live under `docs/adversarial/` and are git-tracked. Distinct from reviewer reports which are gitignored under `.adversarial/{slug}/`.

### Required sections

| Section | Type | Required | Notes |
|---|---|---|---|
| Title | H1 | Yes | `# Planner Fix Summary — {plan-slug} — Iteration {N}` |
| Date | metadata line | Yes | `**Date**: {YYYY-MM-DD HH:MM}` |
| Responding to | metadata line | Yes | Markdown link to the `*-review.md` file this addresses |
| Model | metadata line | Yes | EsquissePlan model name for this session |
| Issue Resolution | table | Yes | See columns below |
| Unresolved | table | Yes | Empty row (`—`) if all issues resolved |
| Next Review | metadata line | Yes | `Slot: {slot} → {model name}` |

### Issue Resolution table columns

| Column | Notes |
|---|---|
| Fix Type | One of the six Fix Type values (GLOSSARY.md) |
| Attack | `A1`–`A7` |
| Issue | One-line description matching the reviewer's table |
| Action Taken | Exact command run or file modified |
| Resolved | `✅` or `❌` with reason |

### Unresolved table columns

| Column | Notes |
|---|---|
| Fix Type | As above |
| Issue | As above |
| Reason not addressed | Must explain why; never leave blank |
```

### 7. AGENTS.md — Common Mistakes: negation anti-pattern

Append to the Common Mistakes section:

```markdown
15. **[Prompt Engineering] Negation guardrails are less reliable than affirmative imperatives.**
    - Wrong: `"Never write code"`, `"Do not add disclaimers"`, `"Never output markdown"`
    - Right: `"Write documents only"`, `"Output raw JSON only"`, `"Affirmative verb + object"`
    - Why: LLMs trained with RLHF often process negations unreliably under generation pressure
      (arXiv:2406.05494). Affirmative imperatives engage the same constraint as an action
      rather than a prohibition, improving adherence in long-context sessions.
```

(Use the next sequential number in the existing list, not necessarily 15 — read AGENTS.md to find the current last entry number before inserting.)

## Acceptance Criteria

- `grep "Write documents only" .github/agents/EsquissePlan.agent.md` returns 1 match
- `grep "Critical Rules" .github/agents/EsquissePlan.agent.md` returns 1 match
- `grep "PLANNING_ARTIFACT" .github/agents/EsquissePlan.agent.md` returns ≥ 1 match
- `grep "planner-fixes" .github/agents/EsquissePlan.agent.md` returns 1 match
- `grep "PLANNING_ARTIFACT" skills/adversarial-review/SKILL.md` returns ≥ 1 match
- `grep "planner-fixes" SCHEMAS.md` returns ≥ 1 match
- `grep "negation" AGENTS.md` returns ≥ 1 match
- `wc -l .github/agents/EsquissePlan.agent.md` — line count decreases vs. pre-task baseline (Step 2c compression)
- Manual trigger test: after a CONDITIONAL verdict with a `PLANNING_ARTIFACT` BLOCKING issue, EsquissePlan runs `go doc {pkg}` and creates a file in `docs/artifacts/` before dispatching the next reviewer.

## Session Notes

- The four guardrail rewrites in §4 are targeted string replacements — do not rewrite the entire Guardrails section.
- The "Never" → affirmative conversions tend to be shorter; total file line count should decrease.
- SCHEMAS.md §11 is a new section — confirm §10 is the current last section before appending (verified: §10 = Planning Artifact, §11 does not yet exist as of 2026-04-30).
- AGENTS.md Common Mistakes numbering: read the file to find the current last entry number before adding.
- `upgrade.sh` §2 deploys all `.github/agents/` changes to `~/.copilot/agents/` automatically — no separate install step needed.
- **Accepted advisory (iter0)**: No Go test exists for malformed Fix Type table in reviewer output — this is a runtime behavior of the LLM reviewer, not a parseable code path. Detection is operational (planner reads the table and fails to find required columns) rather than unit-tested. Noted but not mitigated by this task.
- **arXiv citation advisory (iter1)**: `arXiv:2406.05494` cited in AGENTS.md §7 is a real, verified paper fetched during the initial planning session. No Planning Artifact required — the citation appears in AGENTS.md prose (human-read), not in a Specification where a hallucinated signature would cause implementation failure. Remove the specific arXiv ID if preferred; the underlying claim (RLHF negation unreliability) is well-documented.
- [WARN] planning_context capture skipped: no `code_ast.duckdb` at esquisse root; these are markdown-only files and have no Go symbols to capture.

### Implementation session (2026-04-30)
- All 7 changes implemented: Critical Rules box, Step 2c compression, Step 5 Fix Type workflow, 4 guardrail rewrites in EsquissePlan.agent.md; Step 5b→Step 6 condensed Fix Type section in adversarial-review/SKILL.md; §11 appended to SCHEMAS.md; entry #15 added to AGENTS.md.
- EsquissePlan.agent.md: 211 → 196 lines (15 lines removed by Step 2c compression and guardrail shortening).
- Quality review found 2 Important issues: (1) duplicate Step 5 heading — fixed by renaming new section to Step 6 and shifting Step 6→Step 7; (2) read-only constraint in Constraints section contradicted the new Fix Type action steps — fixed by narrowing constraint to apply only to reviewer agents.
- Spec criterion "Write documents only returns 1 match" was calibrated incorrectly — the task spec requires 2 occurrences (Critical Rules box + Guardrails rewrite). Both are present as designed.
- Gate check: PASSED (phase 3, 0 failures, 1 expected coverage warning for markdown-only project).
