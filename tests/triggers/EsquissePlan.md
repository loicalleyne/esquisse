# Trigger Test Checklist: EsquissePlan

Agent: `.github/agents/EsquissePlan.agent.md`
Source: `description:` frontmatter field — Verified: 2026-04-13

> **Note:** EsquissePlan is a VS Code agent mode, not an auto-triggering skill.
> It is invoked via the agent mode selector in VS Code Copilot Chat (e.g. selecting
> "EsquissePlan" from the agent picker, or typing `@EsquissePlan` in chat).
> The phrases below are example queries to send after the mode is activated —
> they do not auto-trigger the agent from the default chat mode.

## Trigger Phrases to Test

Invoked via agent mode selector, not phrase-match. The phrases below are example
queries after mode activation via `@EsquissePlan` or the agent picker.

| Invocation | Date | VS Code Copilot version | Pass/Fail |
|--------|------|------------------------|-----------|
| Select "EsquissePlan" from agent picker | | | |
| `@EsquissePlan decompose this spec into tasks` | | | |
| `@EsquissePlan plan phase 2` | | | |
| `@EsquissePlan write task documents for the export feature` | | | |
| `@EsquissePlan what tasks are needed for this feature?` | | | |

## Activation Check

| Check | How to observe | Pass/Fail |
|---|---|---|
| EsquissePlan agent mode is available in the agent picker | "EsquissePlan" appears in the VS Code Copilot Chat agent selector | |
| Agent responds with planning-mode behavior, not default chat | Response references task documents, phase structure, or adversarial review | |

## False-Positive Phrases to Test (must NOT activate EsquissePlan from default chat mode)

| Phrase | Expected behavior | Date | VS Code Copilot version | Pass/Fail |
|--------|-------------------|------|------------------------|-----------|
| "plan out the implementation" (no @EsquissePlan) | Default Copilot response | | | |
| "write tasks for this feature" (no @EsquissePlan) | Possibly new-task skill or default | | | |

## Tool-Name Compliance

For each activated session, scan the complete EsquissePlan output for these banned strings.
FAIL the session if any appear verbatim.

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
| Agent reads `AGENTS.md` at session start | `read_file` call on `AGENTS.md` appears in output | |
| Agent reads `GLOSSARY.md` | `read_file` call on `GLOSSARY.md` appears in output | |
| Task documents follow `P{n}-{nnn}-{slug}.md` naming | Created files match pattern under `docs/tasks/` | |
| Adversarial review dispatched after plan is written | `runSubagent("Adversarial-r*", ...)` appears in output | |
| Correct rotation slot used (`iteration % 3`) | Slot matches `iteration` value in `.adversarial/state.json` | |
| FAILED adversarial verdict blocks implementation dispatch | If verdict is FAILED, NO `runSubagent("ImplementerAgent", ...)` call follows | |
| ABSENT adversarial verdict (no state.json) blocks implementation | If `.adversarial/state.json` does not exist, agent does not proceed to implementation | |

## Failures

<!-- Record activation failures here (agent mode not available or not responding as expected). -->
<!-- Format: {YYYY-MM-DD} — invocation: "{method}" — VS Code version {x.y} — {symptom} -->
<!-- Debugging procedure:
  1. Verify VS Code Copilot Chat is v1.115+.
  2. Confirm .github/agents/EsquissePlan.agent.md exists and has `name:`, `model:`, and `tools:` frontmatter fields.
  3. In VS Code Chat, open the agent picker and check "EsquissePlan" is listed. If not, reload VS Code window.
  4. If the agent appears in the picker but does not follow the planning protocol, re-read the agent body for instruction errors.
  5. If the agent body is correct but adversarial rotation is wrong, check .adversarial/state.json iteration value manually.
-->
