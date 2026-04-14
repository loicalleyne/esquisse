# Trigger Test Checklist: init-project

Skill: `skills/init-project/SKILL.md`
Source: `description:` frontmatter field — Verified: 2026-04-13

## Trigger Phrases to Test

| Phrase | Date | VS Code Copilot version | Pass/Fail |
|--------|------|------------------------|-----------|
| "initialize a new project" | | | |
| "bootstrap a project" | | | |
| "set up a new repo" | | | |
| "fill in the stubs" | | | |
| "fill in the TODO placeholders" | | | |
| "create the foundation documents" | | | |

## False-Positive Phrases to Test (must NOT trigger init-project)

| Phrase | Expected Skill | Date | VS Code Copilot version | Pass/Fail |
|--------|----------------|------|------------------------|-----------|
| "add esquisse to this project" | adopt-project | | | |
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

| Check | How to observe | Pass/Fail |
|---|---|---|
| `scripts/init.sh` called via `run_in_terminal` | Copilot runs `bash scripts/init.sh` or `./scripts/init.sh` | |
| All `TODO:` placeholders filled in generated documents | After session: `grep -r "TODO:" AGENTS.md ONBOARDING.md GLOSSARY.md` returns no matches | |
| `docs/planning/ROADMAP.md` created with at least P0 tasks | File exists and contains at least one `P0-` task entry | |
| `docs/planning/NEXT_STEPS.md` created | File exists and is non-empty | |
| Skill does NOT implement code | No source file (.go, .py, .ts, etc.) is created or modified | |

## Failures

<!-- Record false-negative failures here (skill not triggered when it should be). -->
<!-- Format: {YYYY-MM-DD} — phrase: "{phrase}" — VS Code version {x.y} — {symptom} -->
<!-- Debugging procedure:
  1. Verify VS Code Copilot Chat is v1.115+. Older versions do not support SKILL.md auto-trigger.
  2. Open skills/init-project/SKILL.md and confirm the `description:` frontmatter field is present and not empty.
  3. Paste the exact failing phrase into a new chat. If still no trigger, try: "use the init-project skill".
  4. If the skill loads on explicit invocation but not on phrase-match, the description phrases need tuning.
  5. If the skill does not load even on explicit invocation, the YAML frontmatter may be malformed — validate with a YAML linter.
-->
