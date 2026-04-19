# P1-000 — Trigger Test Suite for Esquisse Skills

**Phase:** P1 (pre-sequence — completed before task numbering was formalised)
**Status:** Done
**Created:** 2026-04-13
**Depends on:** none
**Blocks:** none

> **Renumbering note:** This file was originally `P1-001-trigger-tests.md`,
> created before `P1-001-crush-vscode-compatibility.md` was written. It has
> been renumbered to `P1-000` to restore sequential ordering. The original
> file now redirects here.

---

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
```

**Source-of-truth rule:** Trigger phrases MUST be derived verbatim from the
`description:` field and/or `triggers:` list in the skill's `SKILL.md`
frontmatter — never invented.

**EsquissePlan exception:** EsquissePlan is a VS Code agent mode, not an
auto-triggering skill. Its trigger test describes how to invoke the mode
(`@EsquissePlan` in VS Code Chat), not a phrase-based auto-trigger.

### Skill-Specific Pass Criteria

| Skill | Mandatory Pass Criterion |
|---|---|
| `adversarial-review` | Correct rotation slot selected; `runSubagent("Adversarial-r*", ...)` used; verdict written to `.adversarial/state.json`; FAILED verdict blocks further progression. |
| `implement-task` | Adversarial review verdict is checked before any `runSubagent("ImplementerAgent", ...)` dispatch. |
| `new-task` | `scripts/new-task.sh` is called; resulting file follows `P{n}-{nnn}-{slug}.md` naming. |
| `init-project` | `scripts/init.sh` is called; all `TODO:` placeholders are filled after the session. |
| `write-spec` | Output is saved to `docs/specs/{date}-{feature}.md`; no task document is created in the same step. |
| `EsquissePlan` | Plan is NOT handed off to any implementation agent if adversarial verdict is FAILED or absent. |

## Acceptance Criteria

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
| `tests/triggers/explore-codebase.md` | Modify | Add `## Tool-Name Compliance` and `## Workflow Gate` sections if missing |
| `tests/triggers/implement-task.md` | Modify | Add `## Tool-Name Compliance` and `## Workflow Gate` sections if missing |
| `tests/triggers/init-project.md` | Create | New trigger test for `init-project` skill |
| `tests/triggers/new-task.md` | Create | New trigger test for `new-task` skill |
| `tests/triggers/write-spec.md` | Modify | Add `## Tool-Name Compliance` and `## Workflow Gate` sections if missing |
| `tests/triggers/EsquissePlan.md` | Create | New trigger test for EsquissePlan agent mode |

Total: **5 new files, 3 modified files**. All files are in `tests/triggers/`.

## Session Notes

1. ASSUMPTION: Phase P1 is appropriate (P0 = foundation/scaffolding is done — skills and
   agents exist; P1 = core functionality).
2. ASSUMPTION: "trigger tests" in Esquisse follow the same format as bof
   (`tests/triggers/{name}.md`) with identical section structure.
3. ASSUMPTION: EsquissePlan is tested as an agent mode (invoked via `@EsquissePlan` in
   VS Code Chat), not as a skill.
4. ASSUMPTION: The Adversarial-r* agents are not tested independently. They are dispatched
   by the adversarial-review skill and verified indirectly through the
   `adversarial-review` trigger test.
