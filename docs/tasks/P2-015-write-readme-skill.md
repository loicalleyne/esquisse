# P2-015 — write-readme skill

## Status

Done (2026-04-27)

## Goal

Add a `write-readme` skill to esquisse that generates or updates a project's
`README.md` in the bufarrow library style: punchy value proposition, use-cases
table, named quick-start patterns, feature tables, concrete performance numbers,
and a curated reference section.

## In Scope

- `skills/write-readme/SKILL.md` — the new skill file

## Out of Scope

- Modifying the existing `readme-blueprint-generator` skill (different use case —
  that skill scans `.github/copilot` files; this one reads project source docs)
- Multi-file README generation (single `README.md` at project root only)
- CI validation trigger test for this skill (tracked in P3-002)

## Files

| File | Action | What changes |
|---|---|---|
| `skills/write-readme/SKILL.md` | Create | New skill definition |
| `docs/planning/ROADMAP.md` | Modify | Add P2-015 row |

## Acceptance Criteria

- [x] `grep -q "^name:" skills/write-readme/SKILL.md` exits 0
- [x] `grep -q "^description:" skills/write-readme/SKILL.md` exits 0 with non-empty value
- [x] `grep -q "^  Use when" skills/write-readme/SKILL.md` exits 0 (description starts with "Use when")
- [x] `grep -qE '"write a README"|"update the README"|"bufarrow style"' skills/write-readme/SKILL.md` exits 0
- [x] Steps 1–6 all present: Reconnaissance, User Interview, Synthesize, Draft, Present and Confirm, Write
- [x] `grep -qE '\bBash\b|\bTask\b|\bTodoWrite\b|\bAskUserQuestion\b' skills/write-readme/SKILL.md` exits 1 (no banned names)
- [x] `grep -q "## When to Use" skills/write-readme/SKILL.md` exits 0
- [x] `grep -q "## Common Mistakes" skills/write-readme/SKILL.md` exits 0
- [x] `grep -q "VS Code / Crush" skills/write-readme/SKILL.md` exits 0 (dual-platform tool notation)
- [x] `grep -q "## Quick Reference" skills/write-readme/SKILL.md` exits 0
- [x] Halt condition present: skill stops if MODULE_PATH and KEY_TYPES both empty after recon
- [x] Code example sourcing rule present: verbatim-only policy stated
- [x] `grep -q "PROJECT_TYPE" skills/write-readme/SKILL.md` exits 0 (project type detection in Step 1)
- [x] `grep -q "OMIT_SECTIONS" skills/write-readme/SKILL.md` exits 0 (user opt-out mechanism in Step 2 and Step 3)
- [x] `grep -q "library wins" skills/write-readme/SKILL.md` exits 0 (tiebreaker rule for library+cmd/ projects)
- [x] `grep -q "discovery sub-step" skills/write-readme/SKILL.md` exits 0 (file_search for cmd/ and Dockerfile)
- [x] Step 3 conditionality overrides SECTIONS_TO_UPDATE — inapplicable sections removed before drafting

## Session Notes

- Skill created directly in session 2026-04-27 (skill is a markdown document).
- Reference README: `c:\Users\lalleyne\go_src\etl\bufarrowlib\README.md`
- bufarrow style key elements encoded as rules: bold value prop + metric on opening,
  use-cases table BEFORE install, H3 named patterns (not "Example"), tables everywhere,
  no invented performance numbers, `---` dividers between every H2.
- **Adversarial review 2026-04-27 (iteration 1, slot 0, Adversarial-r0): CONDITIONAL.**
  1 critical (hallucinated code examples), 5 major issues.
  Mitigations applied same session:
  - Code example sourcing locked to verbatim-only (no agent generation).
  - Halt condition added when MODULE_PATH + KEY_TYPES both empty after recon.
  - Double-confirmation collapsed: Step 5 is the single overwrite confirmation.
  - Sensitive-content warning moved to Step 5 (pre-write, not post-write).
  - Acceptance criteria rewritten as grep-verifiable assertions and marked checked.
  State file: `.adversarial/p2-015-write-readme-skill.json`
- **writing-skills SKILL.md pass 2026-04-27:** Applied bof writing-skills conventions:
  - `description:` rewritten to start with "Use when..." (was summarizing workflow — banned)
  - Added `## When to Use` section with symptom-based triggers and DO NOT USE guards
  - Tools updated to dual-platform notation: `read_file`/`view`, `create_file`/`write`, etc.
  - `## Style Rules Summary` + `## Decision Points` merged under `## Quick Reference`
  - Added explicit Badge URL Templates table (fixes ISSUE-L2 from adversarial review)
  - Added `## Common Mistakes` section encoding all adversarial review findings as agent-facing rules
  - Added `*Last updated: 2026-04-27*` footer per writing-skills convention
  - ACs updated: added `Use when` check, `When to Use` check, `Common Mistakes` check, dual-platform tools check
- **Update-mode gap fix pass 2026-04-27:**
  User asked: "does this skill work for updating existing readme docs?"
  6 gaps identified and fixed:
  - Added `EXISTING_README`, `EXISTING_VALUE_PROP`, `EXISTING_QUICK_START_PATTERNS`, `EXISTING_LICENSE` metadata fields to Step 1.
  - Added Step 1b: Compliance Audit (classifies each section ✅/⚠️/❌; builds `SECTIONS_TO_UPDATE`, `SECTIONS_TO_PRESERVE`, `CUSTOM_SECTIONS`).
  - Step 2 rewritten with pre-fill logic — omits questions already answered by existing README.
  - Step 5 opens with audit summary for update mode.
  - Step 6 "update sections" path expanded with per-section `replace_string_in_file` rules.
  - 3 new Common Mistakes rows added for update-mode anti-patterns.
- **Adversarial review 2026-04-27 (iteration 2, slot 1, Adversarial-r1): CONDITIONAL.**
  1 critical, 5 major, 4 minor issues. All in update-mode machinery; new-README path sound.
  Mitigations applied same session:
  - C1: Value-prop criterion changed from "first body line is `**...**`" to "first `**...**` bold line after logo/badge HTML blocks".
  - M1: Step 3 now has explicit update-mode clause — only synthesizes `SECTIONS_TO_UPDATE` and missing sections; skips `SECTIONS_TO_PRESERVE`.
  - M2: Step 6 fallback match strategy added: `\n---\n` → `\n## ` → EOF; explicit failure report if none match.
  - M3: Features section criterion relaxed to accept H3 **or** H4 subsections.
  - M4: H2 aliases table added to Step 1b (Getting Started → Quick Start, Capabilities → Features, etc.).
  - M5: Step 6 operation ordering specified: (1) update existing, (2) insert new sections, (3) re-insert CUSTOM_SECTIONS.
  - L1: Anchor updated from `\n## Licen` to `\n---\n\n## Licen`.
  - L2: Step 2 renumbering guidance added ("renumber visible questions consecutively").
  - L3: "Large README" defined as > 6 H2 sections or > 300 lines.
  - L4: Step 1b note added to reuse content already loaded in Step 1.
  - 5 new Common Mistakes rows for iteration-2 findings.
  - C1: PROJECT_TYPE tiebreaker added — library wins when importable packages coexist with `cmd/`; ambiguous case prompts user.
  - M1: Step 3 update-mode explicitly removes inapplicable sections from SECTIONS_TO_UPDATE before drafting.
  - M2: OMIT_SECTIONS default = [] documented in Step 3 preamble.
  - M3: Discovery sub-step added to Step 1 — file_search for `cmd/` and `Dockerfile` if AGENTS.md lacks layout.
  - M4: "≥3 distinct deployment scenarios" replaced with "≥3 named integration scenarios in AGENTS.md/ONBOARDING.md".
  - M5: Task doc ACs updated (below); Decision Points updated with tiebreaker rows; 3 new Common Mistakes rows.
  State file: `.adversarial/p2-015-write-readme-skill.json` (iteration 3, CONDITIONAL)
