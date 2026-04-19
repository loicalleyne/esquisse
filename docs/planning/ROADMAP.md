# Esquisse Roadmap

## Current Phase: P1 — Self-Governance & Dual-Platform Compatibility

**Target:** Esquisse dogfoods its own framework (AGENTS.md, ONBOARDING.md,
GLOSSARY.md, trigger tests, ROADMAP.md) and the framework tools work
correctly in both VS Code Copilot Chat and Crush.

**Gate criteria:** [P1 Gate Checklist](#p1-gate-checklist)

### P1 Tasks

| Task | Status | Summary |
|------|--------|---------|
| [P1-000-trigger-tests](../tasks/P1-000-trigger-tests.md) | ✅ Done | Trigger test suite for all 7 skills + EsquissePlan agent |
| [P1-001-crush-vscode-compatibility](../tasks/P1-001-crush-vscode-compatibility.md) | ✅ Done | `scripts/init.sh` installs skills globally for VS Code and Crush with tool-name translation |
| [P1-002-crush-bootstrap-template](../tasks/P1-002-crush-bootstrap-template.md) | ✅ Done | `scripts/init.sh` generates `CRUSH.md` context file in project root |
| [P1-003-adversarial-review-crush-adaptation](../tasks/P1-003-adversarial-review-crush-adaptation.md) | ✅ Done | `adversarial-review` skill gains Step 4b for Crush dispatch via `crush run` |
| [P1-004-esquisse-mcp-server](../tasks/P1-004-esquisse-mcp-server.md) | ✅ Done | `esquisse-mcp/` Go MCP stdio server exposing `adversarial_review` and `gate_review` tools |

> **Note:** `P1-001-trigger-tests.md` was completed before the naming sequence
> was formalised and has been renumbered to `P1-000-trigger-tests.md`.
> The original file redirects to the canonical version.

### P1 Gate Checklist

- [x] All P1 tasks above have `Status: Done`
- [ ] `grep -rn "description:" skills/*/SKILL.md` — all 7 skills have non-empty `description:` field
- [ ] `for f in .github/agents/*.agent.md; do grep -q "^name:" "$f" || echo "MISSING: $f"; done` — 0 errors
- [ ] `for f in .github/agents/*.agent.md; do grep -q "^model:" "$f" || echo "MISSING: $f"; done` — 0 errors
- [x] `bash scripts/gate-check.sh 1` passes (stubs, annotation counts, task status)
- [ ] Trigger tests in `tests/triggers/` cover all P1-introduced behaviors (Crush path in adversarial-review)
- [x] `AGENTS.md` updated with any new gotchas from P1 implementation
- [ ] `ROADMAP.md` updated: P1 marked complete, P2 tasks reviewed
- [x] No `TODO`/`FIXME`/`ASSUMPTION` annotations left unresolved in skill files or scripts

---

## Upcoming: P2 — Skill Coverage Expansion

**Target:** Close the skill coverage gaps identified in
`docs/planning/2026-04-13_skill-coverage-gaps.md`. Add `debug-issue`,
`write-adr`, `update-llms`, and `run-phase-gate` skills. Improve the
`gate-check.sh` script for pure-markdown (no-build) projects.

### Planned P2 Tasks

| Task | Status | Summary |
|------|--------|---------|
| P2-001-debug-issue-skill | ⬜ Not Started | New skill: systematic debugging workflow |
| P2-002-write-adr-skill | ⬜ Not Started | New skill: Architecture Decision Record authoring |
| P2-003-update-llms-skill | ⬜ Not Started | New skill: update llms.txt and llms-full.txt after API changes |
| P2-004-run-phase-gate-skill | ⬜ Not Started | New skill: run the phase gate checklist interactively |
| P2-005-markdown-gate-adapter | ✅ Done | `gate-check.sh` no-build adapter for pure-markdown projects |
| [P2-006-mcp-configurable-model-rotation](../tasks/P2-006-mcp-configurable-model-rotation.md) | ✅ Done | 5-slot configurable model pool, multi-round reviews, family-interleaved randomization, enterprise policy fallback, `discover_models` tool |

---

## Upcoming: P3 — Production Hardening

**Target:** CI integration, `upgrade.sh` robustness, ADR library, CHANGELOG
conventions. Caller-model exclusion for adversarial reviews.

| Task | Status | Summary |
|------|--------|----------|
| [P3-001-mcp-exclude-caller-model](../tasks/P3-001-mcp-exclude-caller-model.md) | ✅ Done | `exclude_model` param on `adversarial_review`; exact case-insensitive match; regex allows `/`; fail-open; crush_info step in adversarial-review skill |
| P3-002-ci-skill-lint | ⬜ Not Started | GitHub Actions workflow that validates all SKILL.md frontmatter |
| P3-003-upgrade-sh | ⬜ Not Started | `scripts/upgrade.sh` idempotent upgrade for adopted projects |
| P3-004-adr-library | ⬜ Not Started | Seed `docs/adr/` with foundational ADRs for Esquisse design decisions |
| [P3-005-mcp-model-availability-cache](../tasks/P3-005-mcp-model-availability-cache.md) | ✅ Done | Background availability-probe goroutine + disk cache for `discover_models`; structured JSON response with `available`, `probing`, `stale`; `force_refresh` param; `ESQUISSE_MODEL_CACHE_TTL_DAYS` |

---

## Completed

### P0 — Framework Foundation ✅

Completed 2026-04-13. Key outcomes:

- `FRAMEWORK.md` — philosophy, phase gates, per-task protocols, guardrails
- `SCHEMAS.md` — document schemas for all artifact types
- `TEMPLATES.md` — language-adapter starters (Go, Python, TypeScript, Rust, C/C++)
- `scripts/init.sh` — project bootstrap
- `scripts/new-task.sh` — task scaffolder
- `scripts/gate-check.sh` / `gate-review.sh` — phase gate validation and adversarial enforcement
- `scripts/macros.sql` / `macros_go.sql` — DuckDB/sitting_duck AST navigation macros
- 7 skills: `adopt-project`, `adversarial-review`, `explore-codebase`, `implement-task`, `init-project`, `new-task`, `write-spec`
- 4 VS Code agents: `EsquissePlan`, `Adversarial-r0`, `Adversarial-r1`, `Adversarial-r2`
- Adversarial review planning and stop-hook enforcement design (`adverserial.md`)
- `tests/triggers/` — manual trigger tests for all 7 skills + EsquissePlan

### P0 Gate Outcome

All P0 deliverables present. Framework validated against bof and peddler projects.
