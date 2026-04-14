# P1-001: Trigger Test Suite for Esquisse Skills

## Status
Status: Done
Depends on: none
Blocks: none

## Summary

Create a `tests/triggers/` directory modeled on the bof project's manual trigger
test protocol. Each Esquisse skill gets one `.md` file documenting the exact trigger
phrase, alternative triggers, expected step-by-step behavior, and pass criteria.
These files are the canonical manual-QA specification for every skill in this repo.

## Problem

Esquisse has 7 skills and a VS Code agent mode (EsquissePlan). There is currently
no documented test protocol for verifying that skills activate correctly, follow
the right tooling conventions, or enforce workflow gates (e.g. adversarial review).
Without trigger tests, regressions in skill frontmatter or instruction bodies go
undetected until a feature breaks in production.

## Solution

Create `tests/triggers/<skill-name>.md` for each of the 7 skills and for the
EsquissePlan agent mode. Each file encodes:

- A primary trigger phrase (exact text to paste into VS Code Copilot Chat).
- Two or more alternative trigger phrases.
- Numbered expected behavior steps.
- Explicit pass/fail criteria, including any workflow-gate requirements
  (adversarial review, tool-name compliance, file-output format).

The format is identical to bof's `tests/triggers/` files. This creates a shared
mental model between the two projects.

## Scope

### In Scope

- `tests/triggers/adopt-project.md`
- `tests/triggers/adversarial-review.md`
- `tests/triggers/explore-codebase.md`
- `tests/triggers/implement-task.md`
- `tests/triggers/init-project.md`
- `tests/triggers/new-task.md`
- `tests/triggers/write-spec.md`
- `tests/triggers/EsquissePlan.md` (agent mode, not a skill)

### Out of Scope

- Trigger tests for skills that do not yet exist
  (`debug-issue`, `review-changes`, `refactor-safely`, `write-adr`,
  `update-llms`, `run-phase-gate`). Tests must track what is implemented,
  not what is planned. Create a new task when those skills are added.
- Automated execution of skills. These are manual QA checklists — no
  test runner, CI integration, or scripted evaluation.
- Modifying any existing skill SKILL.md or agent .agent.md files.
- A `tests/README.md` or any index document beyond the trigger files themselves.

## Prerequisites

- [ ] All 7 skills listed in In Scope have a `SKILL.md` with frontmatter `name:` and `description:` fields.
- [ ] `.github/agents/EsquissePlan.agent.md` and the three `Adversarial-r*.agent.md` files exist.

**Prerequisite check procedure (run before writing any trigger test file):**

For each skill in scope, verify `SKILL.md` has a non-empty `description:` field:

```
grep -A3 'description:' skills/{name}/SKILL.md
```

If `description:` is absent or empty, stop. Record the skill name in Session Notes
under "Blocked Skills". Do not write a trigger test for that skill — write a stub
file with the text: `BLOCKED: skills/{name}/SKILL.md has no description: field.`

## Specification

### Trigger Test File Schema

The schema used by this project extends bof's trigger test format with QA log
columns and false-positive tracking. The canonical schema is established by the
**three existing files** (`explore-codebase.md`, `implement-task.md`,
`write-spec.md`) in `tests/triggers/`. All new files MUST follow the same
structure. The three existing files MUST be amended to add missing sections
(the `## False-Positive Phrases to Test` table may already be present — check
before modifying).

```markdown
# Trigger Test Checklist: {skill-name}

Skill: `skills/{name}/SKILL.md`
Source: `description:` frontmatter field — Verified: {YYYY-MM-DD}

## Trigger Phrases to Test

| Phrase | Date | VS Code Copilot version | Pass/Fail |
|--------|------|------------------------|-----------|
| "{primary phrase from description: field}" | | | |
| "{second phrase from description: or triggers: list}" | | | |
| "{third phrase}" | | | |

## False-Positive Phrases to Test (must NOT trigger {skill-name})

| Phrase | Expected Skill | Date | VS Code Copilot version | Pass/Fail |
|--------|----------------|------|------------------------|-----------|
| "{phrase that looks similar but belongs to another skill}" | {other-skill} | | | |
| "{second false-positive phrase}" | {other-skill} | | | |

## Tool-Name Compliance

For each triggered run, scan the complete Copilot output for these banned strings.
FAIL the run if any appear verbatim.

| Banned string | Required replacement | Run date | Pass/Fail |
|---|---|---|---|
| `Bash(` | `run_in_terminal` | | |
| `Task(` | `runSubagent(` | | |
| `TodoWrite` | `manage_todo_list` | | |
| `AskUserQuestion` | `vscode_askQuestions` | | |
| `CLAUDE.md` | `AGENTS.md` | | |
| `superpowers:` | *(no namespace prefix)* | | |

## Workflow Gate (if applicable)

{Skill-specific gate checks — omit section for skills with no gate.}

## Failures

<!-- Record false-negative failures here (skill not triggered when it should). -->
<!-- Format: {YYYY-MM-DD} — phrase: "{phrase}" — VS Code version {x.y} — {symptom} -->
<!-- Debugging procedure:
  1. Verify VS Code Copilot Chat is v1.115+.
  2. Confirm description: frontmatter field is non-empty and not prefixed with "DO NOT USE" or "not:".
  3. Re-run phrase in a fresh chat. If still no trigger, try: "use the {name} skill".
  4. If explicit invocation works but phrase-match does not, the description phrases need tuning.
  5. If explicit invocation also fails, validate YAML frontmatter with a YAML linter.
-->
```

**Source-of-truth rule:** Trigger phrases MUST be derived verbatim from the
`description:` field and/or `triggers:` list in the skill's `SKILL.md`
frontmatter — never invented. If a skill has a `triggers:` list, those entries
take precedence. For multi-line `description:` block scalars, extract the
sentences that describe what the user would say to invoke the skill.

**EsquissePlan exception:** EsquissePlan is a VS Code agent mode, not an
auto-triggering skill. Its trigger test describes how to invoke the mode
(`@EsquissePlan` in VS Code Chat, or selecting the EsquissePlan mode from the
agent picker), not a phrase-based auto-trigger. The `## Trigger Phrases to Test`
table for `EsquissePlan.md` must note: "Invoked via agent mode selector, not
phrase-match. The 'phrases' below are example queries after mode activation."

### Skill-Specific Pass Criteria

The following workflow-gate checks are mandatory in the relevant trigger test files:

| Skill | Mandatory Pass Criterion |
|---|---|
| `adversarial-review` | Correct rotation slot selected; `runSubagent("Adversarial-r*", ...)` used; verdict written to `.adversarial/state.json`; FAILED verdict blocks further progression. |
| `implement-task` | Adversarial review verdict is checked before any `runSubagent("ImplementerAgent", ...)` dispatch. |
| `new-task` | `scripts/new-task.sh` is called; resulting file follows `P{n}-{nnn}-{slug}.md` naming. |
| `init-project` | `scripts/init.sh` is called; all `TODO:` placeholders are filled after the session. |
| `write-spec` | Output is saved to `docs/specs/{date}-{feature}.md`; no task document is created in the same step. |
| `EsquissePlan` | Plan is NOT handed off to any implementation agent if adversarial verdict is FAILED or absent. |

### Tool Name Compliance — Observable Test

Tool-name compliance is not subjective. The test is binary:

> **FAIL** if any banned string from the table below appears verbatim in the
> Copilot response during the trigger test run. **PASS** otherwise.

| Banned string (substring match) | Required replacement string |
|---|---|
| `Bash(` | `run_in_terminal` |
| `Task(` | `runSubagent(` |
| `TodoWrite` | `manage_todo_list` |
| `AskUserQuestion` | `vscode_askQuestions` |
| `CLAUDE.md` | `AGENTS.md` |
| `superpowers:` | *(no namespace prefix)* |

For each trigger test run, the tester visually scans the complete Copilot output
for any substring from the "Banned" column. This scan is the compliance check.
No automated tooling is required.

### False-Negative Debugging Procedure

If a trigger test produces no skill activation (Copilot replies with default
behavior), follow these steps before marking the test FAIL:

1. Verify the `description:` field in the skill's `SKILL.md` is non-empty and
   does not start with `DO NOT USE` or `not:`.
2. Confirm VS Code Copilot Chat is version 1.115 or newer
   (`Help > About` in VS Code).
3. Re-run the exact trigger phrase with the `@` workspace symbol prefix removed
   (trigger phrases work in chat only, not in inline completions).
4. Reload VS Code (`Developer: Reload Window`) and re-run.
5. If still failing after step 4, mark the test BLOCKED and record the VS Code
   version, Copilot Chat version, and exact trigger phrase used in Session Notes.

## Acceptance Criteria

These criteria are verified manually by pasting the trigger phrase from each
`.md` file into VS Code Copilot Chat and observing the output.

| Test File | What It Verifies |
|---|---|
| `tests/triggers/adopt-project.md` | Skill activates on "add esquisse to this project"; reads source files before writing documents; does not invent ADRs. |
| `tests/triggers/adversarial-review.md` | Rotation slot read from `.adversarial/state.json`; correct `Adversarial-r*` agent dispatched; verdict written. |
| `tests/triggers/explore-codebase.md` | Skill activates on "explore the codebase"; uses `duckdb-code` skill or `grep_search`/`read_file` tools — NOT `Bash(...)`. |
| `tests/triggers/implement-task.md` | Adversarial gate is checked before dispatching ImplementerAgent; all acceptance criteria verified before marking Done. |
| `tests/triggers/init-project.md` | `init.sh` called; all placeholders resolved; ROADMAP.md, NEXT_STEPS.md created. |
| `tests/triggers/new-task.md` | `new-task.sh` called; resulting file follows `P{n}-{nnn}-{slug}.md`; all schema sections present. |
| `tests/triggers/write-spec.md` | Output saved to `docs/specs/{date}-{feature}.md`; no task document created; EsquissePlan NOT invoked in the same session. |
| `tests/triggers/EsquissePlan.md` | Planning protocol followed (read AGENTS.md → map work → write tasks → adversarial review); FAILED verdict blocked implementation. |

All 8 files must exist and be non-empty. No existing files may be modified.

## Files

| Path | Action | What Changes |
|---|---|---|
| `tests/triggers/adopt-project.md` | Create | New trigger test for `adopt-project` skill |
| `tests/triggers/adversarial-review.md` | Create | New trigger test for `adversarial-review` skill |
| `tests/triggers/explore-codebase.md` | **Modify** | Already exists. Add `## Tool-Name Compliance` table and `## Workflow Gate` section (if missing). Preserve all existing content. |
| `tests/triggers/implement-task.md` | **Modify** | Already exists. Add `## Tool-Name Compliance` table and `## Workflow Gate` section (if missing). Preserve all existing content. |
| `tests/triggers/init-project.md` | Create | New trigger test for `init-project` skill |
| `tests/triggers/new-task.md` | Create | New trigger test for `new-task` skill |
| `tests/triggers/write-spec.md` | **Modify** | Already exists. Add `## Tool-Name Compliance` table and `## Workflow Gate` section (if missing). Preserve all existing content. |
| `tests/triggers/EsquissePlan.md` | Create | New trigger test for EsquissePlan agent mode |

Total: **5 new files, 3 modified files**. All files are in `tests/triggers/`.

**Modification rule:** For the 3 existing files, follow this procedure:

1. Read the complete current content of the file.
2. Verify compatibility: the file's existing headings must be a subset of the
   schema headings (`## Trigger Phrases to Test`, `## False-Positive Phrases
   to Test`, `## Tool-Name Compliance`, `## Workflow Gate`, `## Failures`).
   If the file uses entirely different heading names, stop. Record the
   incompatibility in Session Notes under "Incompatible Existing Files" and
   leave the file unmodified — it requires a separate task.
3. For each schema section absent from the file, append it.
4. For each schema section already present, leave it unchanged.
5. Do NOT reorder existing sections.

**Schema stability note:** The schema is derived directly from the 3 existing
production files. It is not speculative — it is already in use.

## Session Notes

**Assumptions:**

1. ASSUMPTION: Phase P1 is appropriate (P0 = foundation/scaffolding is done — skills and
   agents exist; P1 = core functionality). No ROADMAP.md exists to confirm this. If the
   project owner uses a different phase numbering, rename this file.

2. ASSUMPTION: "trigger tests" in Esquisse follow the same format as bof
   (`tests/triggers/{name}.md`) with identical section structure. The two projects share
   these conventions by design.

3. ASSUMPTION: EsquissePlan is tested as an agent mode (invoked via `@EsquissePlan` in
   VS Code Chat), not as a skill. The trigger test documents how to invoke the mode and
   what to observe.

4. ASSUMPTION: The Adversarial-r* agents are not tested independently. They are dispatched
   by the adversarial-review skill and verified indirectly through the
   `adversarial-review` trigger test.

5. ASSUMPTION: Trigger phrase extraction uses the `description:` YAML block scalar from
   each SKILL.md frontmatter as the source of truth. If a skill also has a `triggers:`
   list in frontmatter, those entries take precedence as they are explicitly designated
   trigger phrases.

**Staleness mitigation (minor issue from review):**

Each trigger test file includes a `## Source` section with a `Verified:` date.
When a SKILL.md `description:` or `triggers:` field is updated, the corresponding
trigger test file must be updated in the same commit. No automated enforcement.
This is a documentation convention, not a CI gate.
