# Planner Fix Summary — P3-reviewer-planner-hardening — Iteration 2

**Date**: 2026-04-30  
**Responding to**: `.adversarial/P3-reviewer-planner-hardening/iter02-2026-04-30-review.md` (Adversarial-r2 / GPT-4.1, FAILED)  
**Model**: Claude Sonnet 4.6 (copilot) — EsquissePlan

---

## Issue Resolution

| Fix Type | Attack | Issue | Action Taken | Resolved |
|---|---|---|---|---|
| `SPEC_EDIT` | A4 | B1: P3-010 invariant contradiction — Step 5 writes to `.adversarial/{slug}/` but new guardrail says "Write to `docs/` only" | Resolved by design split: planner-fixes → `docs/adversarial/{slug}/` (git-tracked, in `docs/`); reviewer reports → `.adversarial/{slug}/` (gitignored). Updated P3-010 §3 Step 4, §5 SKILL.md Step 3, §6 SCHEMAS §11 path. Guardrail change retained as-is ("Write to `docs/` only" — `docs/adversarial/` is under `docs/`). | ✅ |
| `SPEC_EDIT` | A1 | B2: `docs/adversarial/{slug}/` claimed fixed but still present in P3-010 §3 and §5 | Actually the iter01 fix changed to `.adversarial/{slug}/` which was correct for reviewer reports but wrong for EsquissePlan-written planner-fixes. B1 fix above resolves B2 by assigning the correct path to each artifact type. | ✅ |
| `SPEC_EDIT` | A2 | B3: P3-009 §2 find string `.adversarial/reports/{YYYY-MM-DD}-{HHmm}-iter{N}-{slug}.md` does not match actual path in Adversarial-r0.agent.md | Read actual Adversarial-r0.agent.md verbatim. Actual path: `.adversarial/reports/review-{YYYY-MM-DD}-iter{N}-r{round}-{plan-slug}.md`. Updated P3-009 §2 find string to match. Added note: "Verify exact current text in each agent file before editing." | ✅ |
| `SPEC_EDIT` | A1 | B4: P3-010 §4 guardrail find strings missing `- ` prefix; `replace_string_in_file` exact match would fail | Read actual EsquissePlan.agent.md guardrail section verbatim. Confirmed `- ` prefix on all four items. Updated all four find strings in P3-010 §4 table. | ✅ |
| `SPEC_EDIT` | A3 | M1: `edit/editFiles` stripped from reviewer tool list; reviewers overwrite existing state JSON | Added `edit/editFiles` to P3-009 §1 new tool list with explicit rationale comment. Updated acceptance criteria grep. | ✅ |
| `SPEC_EDIT` | A1 | M2: `## Required Changes` section does not exist in Adversarial-r0.agent.md; "Replace" instruction would silently do nothing | Changed P3-009 §3 instruction from "Locate…Replace" to "Add…immediately before `## Adversarial Constraints`". | ✅ |
| `SPEC_EDIT` | A6 | M3: SCHEMAS.md §9 documents old path; P3-008 changes path but §9 was not in scope | Added SCHEMAS.md to P3-008 Files table (Modify). Added §5 to P3-008 Specification with exact §9 changes. Added SCHEMAS.md grep to Acceptance Criteria. | ✅ |

## Unresolved

| Fix Type | Issue | Reason not addressed |
|---|---|---|
| — | — | All BLOCKING and ADVISORY issues resolved |

## Next Review

Slot: 0 → Adversarial-r0 (GPT-4.1 via copilot)  
Iteration: 3  
State file: `.adversarial/P3-reviewer-planner-hardening.json`
