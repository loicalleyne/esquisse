# Trigger Test Checklist: adversarial-review

Skill: `skills/adversarial-review/SKILL.md`
Source: `description:` frontmatter field — Verified: 2026-04-13

## Trigger Phrases to Test

| Phrase | Date | VS Code Copilot version | Pass/Fail |
|--------|------|------------------------|-----------|
| "run adversarial review on this plan" | | | |
| "send this plan through adversarial review" | | | |
| "review this plan before implementation begins" | | | |
| "apply the 7-attack protocol to this plan" | | | |
| "adversarial plan review" | | | |

## False-Positive Phrases to Test (must NOT trigger adversarial-review)

| Phrase | Expected Skill | Date | VS Code Copilot version | Pass/Fail |
|--------|----------------|------|------------------------|-----------|
| "review my code after the implementation" | *(no skill — default response)* | | | |
| "write a spec for the new feature" | write-spec | | | |
| "create a new task for phase 2" | new-task | | | |
| "implement task P1-002-add-ingester" | implement-task | | | |

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
| Rotation slot read from `.adversarial/state.json` | Copilot reads the state file; uses `iteration % 3` to select slot | |
| Correct `Adversarial-r*` agent dispatched for slot | slot 0 → `Adversarial-r0`; slot 1 → `Adversarial-r1`; slot 2 → `Adversarial-r2` | |
| `runSubagent("Adversarial-r*", ...)` used (not a direct skill call) | `runSubagent(` appears in output with correct agent name | |
| Verdict written to `.adversarial/state.json` | File is updated with `"verdict":` field after review completes | |
| `iteration` counter incremented after review | `state.json` `"iteration"` value is one higher than before the run | |
| FAILED verdict blocks any subsequent implementation dispatch | If verdict is FAILED, Copilot does NOT call `runSubagent("ImplementerAgent", ...)` or equivalent | |

## Failures

<!-- Record false-negative failures here (skill not triggered when it should be). -->
<!-- Format: {YYYY-MM-DD} — phrase: "{phrase}" — VS Code version {x.y} — {symptom} -->
<!-- Debugging procedure:
  1. Verify VS Code Copilot Chat is v1.115+. Older versions do not support SKILL.md auto-trigger.
  2. Open skills/adversarial-review/SKILL.md and confirm the `description:` frontmatter field is present and not empty.
  3. Paste the exact failing phrase into a new chat. If still no trigger, try: "use the adversarial-review skill".
  4. If the skill loads on explicit invocation but not on phrase-match, the description phrases need tuning.
  5. If the skill does not load even on explicit invocation, the YAML frontmatter may be malformed — validate with a YAML linter.
-->
