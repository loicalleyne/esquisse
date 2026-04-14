# Trigger Test Checklist: new-task

Skill: `skills/new-task/SKILL.md`
Source: `description:` frontmatter field — Verified: 2026-04-13

## Trigger Phrases to Test

| Phrase | Date | VS Code Copilot version | Pass/Fail |
|--------|------|------------------------|-----------|
| "create a new task" | | | |
| "add a task" | | | |
| "define task" | | | |
| "new task for phase 2" | | | |
| "I need to implement the export feature" | | | |
| "what's next?" | | | |

## False-Positive Phrases to Test (must NOT trigger new-task)

| Phrase | Expected Skill | Date | VS Code Copilot version | Pass/Fail |
|--------|----------------|------|------------------------|-----------|
| "initialize a new project" | init-project | | | |
| "implement task P1-001-trigger-tests" | implement-task | | | |
| "write a spec for the export feature" | write-spec | | | |
| "add esquisse to this project" | adopt-project | | | |

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

## Workflow Gate

| Check | How to observe | Pass/Fail |
|---|---|---|
| `scripts/new-task.sh` called via `run_in_terminal` | Copilot runs `bash scripts/new-task.sh <phase> <slug>` or equivalent | |
| Resulting file follows `P{n}-{nnn}-{slug}.md` naming | Created file path matches pattern e.g. `docs/tasks/P2-003-add-search.md` | |
| All task document schema sections present in created file | File contains: Status, Summary, Problem, Solution, Scope, Prerequisites, Specification, Acceptance Criteria, Files | |
| Skill does NOT immediately implement code | No source file is created or modified during the new-task session | |

## Failures

<!-- Record false-negative failures here (skill not triggered when it should be). -->
<!-- Format: {YYYY-MM-DD} — phrase: "{phrase}" — VS Code version {x.y} — {symptom} -->
<!-- Debugging procedure:
  1. Verify VS Code Copilot Chat is v1.115+. Older versions do not support SKILL.md auto-trigger.
  2. Open skills/new-task/SKILL.md and confirm the `description:` frontmatter field is present and not empty.
  3. Paste the exact failing phrase into a new chat. If still no trigger, try: "use the new-task skill".
  4. If the skill loads on explicit invocation but not on phrase-match, the description phrases need tuning.
  5. If the skill does not load even on explicit invocation, the YAML frontmatter may be malformed — validate with a YAML linter.
-->
