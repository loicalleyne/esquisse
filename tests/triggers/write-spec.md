# Trigger Test Checklist: write-spec

Skill: `skills/write-spec/SKILL.md`

## Trigger Phrases to Test

| Phrase | Date | VS Code Copilot version | Pass/Fail |
|--------|------|------------------------|-----------|
| "write a spec for the new notification system" | | | |
| "I want to add a feature for bulk CSV import" | | | |
| "spec out the authentication flow" | | | |
| "help me design the retry logic" | | | |
| "write a feature specification for the dashboard" | | | |
| "before we implement the search, let's write a spec" | | | |

## False-Positive Phrases to Test (must NOT trigger write-spec)

| Phrase | Expected Skill | Date | VS Code Copilot version | Pass/Fail |
|--------|----------------|------|------------------------|-----------|
| "implement task P1-003-add-search" | implement-task | | | |
| "create a task document for phase 2" | new-task | | | |
| "write an ADR for choosing PostgreSQL" | write-adr | | | |
| "I need to implement the Kafka ingester" | implement-task | | | |

## Failures

<!-- Record false-negative failures here (skill not triggered when it should be). -->
<!-- Format: {YYYY-MM-DD} — phrase: "{phrase}" — VS Code version {x.y} — {symptom} -->
<!-- Debugging procedure:
  1. Verify VS Code Copilot Chat is v1.115+. Older versions do not support SKILL.md auto-trigger.
  2. Open skills/write-spec/SKILL.md and confirm the `description:` frontmatter field is present and not empty.
  3. Paste the exact failing phrase into a new chat. If still no trigger, try: "use the write-spec skill".
  4. If the skill loads on explicit invocation but not on phrase-match, the description phrases need tuning.
  5. If the skill does not load even on explicit invocation, the YAML frontmatter may be malformed — validate with a YAML linter.
-->
