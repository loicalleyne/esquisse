# NEXT_STEPS.md — esquisse

## Current Status

- **Phase:** P2 — Robustness & Tooling
- **Last updated:** 2026-07-16

## Last Session

**P2-013: AST Planning Context** — ✅ Done

Added `planning_context` DuckDB table DDL and macros so EsquissePlan can capture
symbol snapshots at planning time and implement-task can detect drift at
implementation time.

Changes:
- `scripts/macros.sql` — `CREATE TABLE IF NOT EXISTS planning_context` (9 columns) + `planning_drift(task_id)` macro
- `scripts/macros_go.sql` — `capture_planning_context(task_id, role, pattern, name_like)` TABLE macro
- `.github/agents/EsquissePlan.agent.md` — Step 2c "Capture Planning Context" inserted after Step 2b
- `skills/implement-task/SKILL.md` — Step 3b drift check + Step 4a orientation via planning_context
- `AGENTS.md` — planning_context table documented in §Available Tools & Services
- `SCHEMAS.md` — Planning Artifacts note in §4
- `GLOSSARY.md` — 3 new terms: `planning_context`, `symbol snapshot`, `drift detection`

Adversarial review: PASSED (iter 10).
SpecReviewerAgent: ✅ COMPLIANT (all 11 acceptance grep checks passed).
CodeQualityReviewerAgent: 1 Important issue fixed (empty `Step 4 — Orient` heading had no body — added transition sentence). 4 Minor issues deferred.

## Previous Session

**P2-012: Eliminate Bare python3 Calls from Bash Scripts** — ✅ Done

Replaced all bare `python3 -c` invocations in `scripts/gate-review.sh` (2 calls)
and `scripts/upgrade.sh` (4 calls) with `jq -r` expressions (reads) and `printf`
heredoc (deprecation stub write). The `jq --arg` merge handles the `verdict →
last_verdict` rename for the migration write. Grep/sed fallback preserved for
environments without `jq`. Zero Python remains in either script.

Adversarial review: PASSED (iter 1, Adversarial-r0 / GPT-4.1).
Code quality: APPROVED (minor: dead `RAW_SLUG` variable is pre-existing).

Key gotcha documented by CodeQualityReviewer: in the jq expression
`del(.verdict) + {last_verdict: .verdict}`, both operands are evaluated against
the **same original input** — `del` does not mutate for the purpose of the
right-hand side. The expression is correct.

## Previous Session

Created the missing framework self-governance documents:
- `AGENTS.md` — project constitution
- `ONBOARDING.md` — agent orientation
- `GLOSSARY.md` — domain vocabulary
- `docs/planning/ROADMAP.md` — phase plan and task status

Also completed second adversarial review cycle (3 rounds) on the P1-002/P1-003/P1-004 plan:
- Round 1 (iter3, Adversarial-r0 / GPT-4.1): **PASSED** — minor: AGENTS.md stub reference
- Round 2 (iter4, Adversarial-r1 / GPT-4o): **CONDITIONAL** — fixed: empty `plan_content` guard added to P1-004 AC
- Round 3 (iter5, Adversarial-r2 / GPT-4o): **CONDITIONAL** — fixed: unquoted bash redirect path in P1-003 crush-models.md

All fixes applied in-cycle. Plan cleared for implementation handoff.

## Open Items

- [x] **P1-001 naming conflict** — resolved: `P1-001-trigger-tests.md` renumbered
      to `P1-000-trigger-tests.md`; original redirects to canonical file; ROADMAP.md
      and AGENTS.md updated.
- [x] **llms.txt and llms-full.txt** — created 2026-04-17.
- [x] **P1 tasks need implementation** — all four tasks completed (P1-001, P1-002,
      P1-003, P1-004). All passed SpecReview + CodeQualityReview.
- [x] **P1 Gate Checklist** — `bash scripts/gate-check.sh 1` passes (0 failures, 1 warning).

## Blockers

None.

## Session Log

| Date | Work Done |
|------|-----------|
| 2026-04-13 | P0 complete: FRAMEWORK.md, SCHEMAS.md, TEMPLATES.md, scripts/, skills/, agents/, tests/triggers/ |
| 2026-04-13 | Adversarial review planning design (adverserial.md) |
| 2026-04-17 | P1 spec written: docs/specs/2026-04-17-crush-vscode-compatibility.md |
| 2026-04-17 | P1 tasks written: P1-001 compat, P1-002, P1-003, P1-004 |
| 2026-04-17 | First adversarial review cycle (3 rounds): FAILED → CONDITIONAL → PASSED; plan revised |
| 2026-04-17 | P1-003 Step 4c removed (MCP server doesn't exist at that point) |
| 2026-04-17 | Second adversarial review cycle (3 rounds): PASSED, CONDITIONAL, CONDITIONAL; two fixes applied |
| 2026-04-17 | Self-governance documents created: AGENTS.md, ONBOARDING.md, GLOSSARY.md, ROADMAP.md, NEXT_STEPS.md |
| 2026-04-17 | P1-001 implemented: init.sh VS Code + Crush skill installation with WSL-aware path resolution |
| 2026-04-17 | P1-002 implemented: init.sh generates CRUSH.md context file in project root |
| 2026-04-17 | P1-003 implemented: adversarial-review skill Step 4b (Crush dispatch via crush run) + crush-models.md |
| 2026-04-17 | P1-004 implemented: esquisse-mcp/ Go MCP server (adversarial_review + gate_review tools); path traversal fix + tests added during CodeQualityReview |
| 2026-04-17 | gate-check.sh fixed: pure-markdown no-Go-package detection (checks 2+3); Status regex accepts **Status:** Done format (check 7) |
| 2026-04-17 | P1 gate PASSED — 0 failures (1 warning: coverage skipped, no Go packages at root) |
| 2026-04-18 | P2-006 spec + adversarial review: 6 rounds over 17 iterations (Adversarial-r0/r1/r2 rotation); final verdict PASSED at iter=17 |
| 2026-04-18 | P2-006 implemented: models.go created (buildModelPool, familyInterleaveShuffle, buildRotationOrder, worstVerdict, runOneRound, newDiscoverHandler, SetRandSource); adversarial.go refactored to multi-round; discover_models tool registered in tools.go |
| 2026-04-18 | P2-006 post-impl reviews: SpecReviewerAgent APPROVED; CodeQualityReviewerAgent found data race (t.Parallel+SetRandSource) — fixed; final APPROVED |
| 2026-04-19 | esquisse-mcp/ Esquisse documents created: AGENTS.md, GLOSSARY.md, ONBOARDING.md; README.md updated (5-slot pool, multi-round, discover_models, migration note) |
| 2026-04-19 | ROADMAP.md: P2-006 added as Done; NEXT_STEPS.md: phase updated to P2, session log extended |
| 2026-04-19 | P3-001 task written: exclude_provider param for adversarial_review; 6-round adversarial review (CONDITIONAL→FAILED→PASSED→CONDITIONAL→CONDITIONAL→PASSED at iter=6); task status: Ready |
| 2026-04-19 | P3-001 revised: exclude_provider → exclude_model (exact match, regex `^[a-zA-Z0-9_./-]+$`, fail-open); 3 adversarial rounds (iter 10→13, CONDITIONAL→FAILED→PASSED); implemented via ImplementerAgent; SpecReviewerAgent COMPLIANT; CodeQualityReviewerAgent APPROVED (2 minor style notes); task Done |
| 2026-04-19 | P3-005 task written: background model availability probe + disk cache for discover_models; structured JSON response (available/probing/stale); 3 adversarial rounds (iter 0→3, CONDITIONAL→CONDITIONAL→PASSED); plan cleared for implementation |
| 2026-04-19 | P3-005 implemented: ModelEntry/ModelCache/modelProber in models.go; atomic cache write (CreateTemp+Rename); newModelProberWithFuncs for test injection; main.go wired with context cancel; tools.go registerTools updated; AGENTS.md updated (3 fixes from SpecReviewerAgent); TestModelProber suite (13 ACs) + TestModelProberFilterAllowedProviders + TestModelProberConcurrentAccess added; 2 race fixes applied (gate channel in no_cache_returns_probing_state; deferred entries check in force_refresh_resets_state); SpecReviewerAgent COMPLIANT; CodeQualityReviewerAgent APPROVED (1 deferred minor: t.Parallel on TestModelProberConcurrentAccess); task Done |
