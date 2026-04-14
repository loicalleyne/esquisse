# Trigger Test Checklist: implement-task

Skill: `skills/implement-task/SKILL.md`

## Trigger Phrases to Test

| Phrase | Date | VS Code Copilot version | Pass/Fail |
|--------|------|------------------------|-----------|
| "implement task P1-003-add-search" | | | |
| "start working on P2-001-kafka-consumer" | | | |
| "execute this task" | | | |
| "let's implement the retry logic" | | | |
| "do the work for task P1-004-metrics" | | | |

## False-Positive Phrases to Test (must NOT trigger implement-task)

| Phrase | Expected Skill | Date | VS Code Copilot version | Pass/Fail |
|--------|----------------|------|------------------------|-----------|
| "create a new task for phase 2" | new-task | | | |
| "write a spec for the export feature" | write-spec | | | |
| "initialize a new project" | init-project | | | |
| "debug why tests are failing" | debug-issue | | | |

## Failures

<!-- Record false-negative failures here (skill not triggered when it should be). -->
<!-- Format: {YYYY-MM-DD} — phrase: "{phrase}" — VS Code version {x.y} — {symptom} -->
<!-- Debugging procedure:
  1. Verify VS Code Copilot Chat is v1.115+. Older versions do not support SKILL.md auto-trigger.
  2. Open skills/implement-task/SKILL.md and confirm the `description:` frontmatter field is present and not empty.
  3. Paste the exact failing phrase into a new chat. If still no trigger, try: "use the implement-task skill".
  4. If the skill loads on explicit invocation but not on phrase-match, the description phrases need tuning.
  5. If the skill does not load even on explicit invocation, the YAML frontmatter may be malformed — validate with a YAML linter.
-->
