# P2-008: Planning Artifact Consumption — implement-task Skill and Scripts

## Status
Status: Done
Depends on: P2-007 (schema and protocol defined before consumption wired up)
Blocks: none

## Summary
Wires planning artifact consumption into two places: (1) the `implement-task`
skill's startup phase, so the implementor agent loads referenced artifacts before
touching source files; (2) the directory scaffold in `scripts/init.sh` and
`scripts/upgrade.sh`, so newly bootstrapped and upgraded projects automatically
have the `docs/artifacts/` directory in place.

## Problem
Without changes to `implement-task/SKILL.md`, the artifact loading step defined in
`FRAMEWORK.md §9` (added by P2-007) will not be followed by agents using the skill —
skills are the authoritative workflow guide, not FRAMEWORK.md. Without changes to
`init.sh` and `upgrade.sh`, `docs/artifacts/` will not exist in bootstrapped or
upgraded projects, causing `create_file` calls to fail when EsquissePlan tries to
write the first artifact.

## Solution
Add a Step 2b ("Load Planning Artifacts") to the implement-task skill's Phase 1
Startup, immediately after Step 2 (Load Context) and before Step 3 (Establish
Baseline). Add `create_dir "docs/artifacts"` to the directory scaffold in `init.sh`.
Add a `mkdir -p docs/artifacts` to the relevant section of `upgrade.sh` so it
creates the directory in target projects that do not yet have it.

## Scope

### In Scope
- `skills/implement-task/SKILL.md`: add Step 2b — Load Planning Artifacts, with
  per-artifact section loading guided by "What to read from it" column; include
  missing-artifact fallback note
- `scripts/init.sh`: add `create_dir "docs/artifacts"` to directory scaffold section
- `scripts/upgrade.sh`: add `mkdir -p docs/artifacts` to ensure directory exists in
  target; add note in script header comment that `docs/artifacts/` is NOT overwritten

### Out of Scope
- SCHEMAS.md or FRAMEWORK.md changes (P2-007)
- EsquissePlan.agent.md artifact production step (P2-007)
- `write_planning_artifact` MCP tool (P2-009)
- Any changes to gate-check.sh or adversarial review infrastructure

## Prerequisites
- [ ] P2-007 is done (`## Planning Artifacts` section defined in SCHEMAS.md)

## Specification

### implement-task/SKILL.md: Step 2b (exact replacement)

Exact replacement in `skills/implement-task/SKILL.md`:

**Find this text:**
```
- Any `Depends on:` prerequisites

#### Step 3 — Establish Baseline
```
**Replace with (the full Step 2b block):**
```
- Any `Depends on:` prerequisites

#### Step 2b — Load Planning Artifacts

If the task doc contains a `## Planning Artifacts` section:

For each row in the Planning Artifacts table:
1. Read the linked artifact file from `docs/artifacts/`.
2. Read **only the sections named in "What to read from it"** — not the full file.
   Example: "API Surface, Anti-Patterns" → read those two sections, skip the rest.
3. Hold the loaded facts as working context for Steps 6+.
4. If the artifact's word count is not known: after reading the requested sections,
   estimate whether the content is unusually large (> 600 words). If so, note it in
   Session Notes and read only the subsections most directly relevant to the current task.

If an artifact file is missing (path in table but file does not exist):
- Note in Session Notes: `MISSING ARTIFACT: {path} — proceeding without it.`
- Do NOT attempt to reconstruct the artifact's content from training data.
  Hallucinated API surfaces are the failure mode this mechanism exists to prevent.

If the task doc has no `## Planning Artifacts` section, skip this step entirely.
A missing section means no external research was needed — do not search for artifacts speculatively.

#### Step 3 — Establish Baseline
```

### scripts/init.sh: directory scaffold addition (exact replacement)

Exact replacement in `scripts/init.sh`:

**Find this text:**
```sh
create_dir "docs/tasks"
create_dir "docs/adr"
```
**Replace with:**
```sh
create_dir "docs/tasks"
create_dir "docs/artifacts"
create_dir "docs/adr"
```

### scripts/upgrade.sh: target directory creation (exact replacements)

**Replacement 1** — header comment, find:
```
# What gets overwritten (Esquisse infrastructure — not user-authored):
```
Replace with:
```
# What gets overwritten (Esquisse infrastructure — not user-authored):
#   - docs/artifacts/ directory (created if absent, contents never touched)
```

**Replacement 2** — mkdir block, find:
```sh
mkdir -p scripts
for item in gate-review.sh gate-check.sh
```
Replace with:
```sh
mkdir -p scripts
mkdir -p docs/artifacts
echo "  ensured  docs/artifacts/"
for item in gate-review.sh gate-check.sh
```

### Known Risks / Failure Modes
| Risk | Mitigation |
|---|---|
| Implementor reads full artifact ignoring "What to read from it" | Step 2b explicitly says "only the sections named" — phrased as an instruction not a suggestion |
| Missing artifact file — agent hallucinates content | Step 2b's fallback says "Do NOT reconstruct" — the explicit prohibition is the mitigation |
| init.sh directory already exists on re-run | `create_dir` helper already skips existing dirs — no change needed |

## Acceptance Criteria
This task modifies documentation and shell scripts — no Go code, no automated tests.

| Check | Command / Method |
|---|---|
| Step 2b present in skill | `grep -n "Planning Artifacts" skills/implement-task/SKILL.md` returns result in Phase 1 |
| "Do NOT reconstruct" guard present | `grep -n "hallucinated\|reconstruct\|training data" skills/implement-task/SKILL.md` returns result |
| `docs/artifacts` in init.sh | `grep -n "artifacts" scripts/init.sh` returns result |
| `docs/artifacts` in upgrade.sh | `grep -n "artifacts" scripts/upgrade.sh` returns result |
| Gate check passes | `bash scripts/gate-check.sh 2` exits 0 |

## Files
| File | Action | Description |
|------|--------|-------------|
| `skills/implement-task/SKILL.md` | Modify | Insert Step 2b: Load Planning Artifacts after Step 2, before Step 3 |
| `scripts/init.sh` | Modify | Add `create_dir "docs/artifacts"` to directory scaffold |
| `scripts/upgrade.sh` | Modify | Add `mkdir -p docs/artifacts` to target directory creation; update header comment |

## Design Principles
1. **Explicit prohibition over guidance.** The missing-artifact fallback says "Do NOT reconstruct" rather than "be careful". Models respond more reliably to prohibition than to caution.
2. **Minimal footprint in init/upgrade.** Directory creation only. No new files, no template stubs. Artifacts are produced by EsquissePlan on demand, not scaffolded.

## Testing Strategy
- Manual: run `bash scripts/init.sh --target-dir /tmp/test-init` and confirm `docs/artifacts/` is created.
- Manual: run `bash scripts/upgrade.sh --target-dir /tmp/test-upgrade` (after creating a minimal project) and confirm `docs/artifacts/` appears.
- Manual: read the updated SKILL.md and trace the startup flow for a task doc that has a `## Planning Artifacts` section with two rows.

## Session Notes
<!-- Append-only. Never overwrite. -->
<!-- 2026-04-21 — EsquissePlan — Created. Depends on P2-007. -->
<!-- 2026-04-21 — EsquissePlan — Revised after Adversarial-r0 CONDITIONAL (iter 0). Added word count warning guidance (M4) and tightened "no section → skip" language with explicit "do not search speculatively" (M6). -->
<!-- 2026-04-21 — ImplementerAgent — Completed. Step 2b inserted in SKILL.md; docs/artifacts added to init.sh and upgrade.sh. -->
