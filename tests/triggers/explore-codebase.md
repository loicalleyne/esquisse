# Trigger Test Checklist: explore-codebase

Skill: `skills/explore-codebase/SKILL.md`

## Trigger Phrases to Test

| Phrase | Date | VS Code Copilot version | Pass/Fail |
|--------|------|------------------------|-----------|
| "explore the codebase" | | | |
| "orient me to this project" | | | |
| "where is the Kafka ingester defined?" | | | |
| "how does the transcoder work?" | | | |
| "map the architecture for me" | | | |
| "what calls the AppendRaw function?" | | | |
| "help me understand this project structure" | | | |

## False-Positive Phrases to Test (must NOT trigger explore-codebase)

| Phrase | Expected Skill | Date | VS Code Copilot version | Pass/Fail |
|--------|----------------|------|------------------------|-----------|
| "implement task P1-002-add-ingester" | implement-task | | | |
| "debug why the parser crashes on nil input" | debug-issue | | | |
| "write a spec for the export feature" | write-spec | | | |

## Failures

<!-- Record false-negative failures here (skill not triggered when it should be). -->
<!-- Format: {YYYY-MM-DD} — phrase: "{phrase}" — VS Code version {x.y} — {symptom} -->
<!-- Debugging procedure:
  1. Verify VS Code Copilot Chat is v1.115+. Older versions do not support SKILL.md auto-trigger.
  2. Open skills/explore-codebase/SKILL.md and confirm the `description:` frontmatter field is present and not empty.
  3. Paste the exact failing phrase into a new chat. If still no trigger, try: "use the explore-codebase skill".
  4. If the skill loads on explicit invocation but not on phrase-match, the description phrases need tuning.
  5. If the skill does not load even on explicit invocation, the YAML frontmatter may be malformed — validate with a YAML linter.
-->
