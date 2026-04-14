# Trigger Test Checklist: adopt-project

Skill: `skills/adopt-project/SKILL.md`
Source: `description:` frontmatter field — Verified: 2026-04-13

## Trigger Phrases to Test

| Phrase | Date | VS Code Copilot version | Pass/Fail |
|--------|------|------------------------|-----------|
| "add esquisse to this project" | | | |
| "adopt esquisse" | | | |
| "add framework documents to this repo" | | | |
| "set up AGENTS.md for this project" | | | |
| "onboard this codebase" | | | |

## False-Positive Phrases to Test (must NOT trigger adopt-project)

| Phrase | Expected Skill | Date | VS Code Copilot version | Pass/Fail |
|--------|----------------|------|------------------------|-----------|
| "initialize a new project" | init-project | | | |
| "create a new task for phase 2" | new-task | | | |
| "implement task P1-001-trigger-tests" | implement-task | | | |
| "write a spec for the export feature" | write-spec | | | |

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

No automated gate. Verify these constraints manually during the triggered run:

| Check | How to observe | Pass/Fail |
|---|---|---|
| Skill reads source files before writing any documents | Copilot uses `read_file`, `grep_search`, or `file_search` on project source before writing `AGENTS.md` | |
| Skill does NOT invent ADRs for past decisions | No `docs/decisions/ADR-*.md` file is created in the same session | |
| Phase label reflects actual project state, not P0 | Copilot asks about or infers the current phase from existing code — does not default to P0 | |
| Build and test commands verified before writing | Copilot reads existing `Makefile`, `go.mod`, `package.json`, or similar before writing build commands | |

## Failures

<!-- Record false-negative failures here (skill not triggered when it should be). -->
<!-- Format: {YYYY-MM-DD} — phrase: "{phrase}" — VS Code version {x.y} — {symptom} -->
<!-- Debugging procedure:
  1. Verify VS Code Copilot Chat is v1.115+. Older versions do not support SKILL.md auto-trigger.
  2. Open skills/adopt-project/SKILL.md and confirm the `description:` frontmatter field is present and not empty.
  3. Paste the exact failing phrase into a new chat. If still no trigger, try: "use the adopt-project skill".
  4. If the skill loads on explicit invocation but not on phrase-match, the description phrases need tuning.
  5. If the skill does not load even on explicit invocation, the YAML frontmatter may be malformed — validate with a YAML linter.
-->
