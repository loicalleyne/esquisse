# P2-010: Prerequisite Blockquote in Task Documents (Documentation)

## Status
Status: Done
Depends on: P2-007 (Planning Artifact schema), P2-009 (write_planning_artifact MCP tool)
Blocks: P2-011 (schema must be stable before MCP tests assert on injected format)

## Summary
Extends the task document schema and the EsquissePlan planning protocol so that every
task document whose work depends on an external library also carries a visible
`> **Prerequisite:** Read ...` blockquote immediately after its title line. This gives
implementor agents a prominent, un-missable pointer to the Planning Artifact at the
very top of the file — before they reach the `## Planning Artifacts` table lower down.

## Problem
The `## Planning Artifacts` section already links to research artifacts, but it is
buried after the Status, Summary, Problem, Solution, and Scope sections. An LLM agent
skimming the task doc for code to write may load the artifact too late or miss it
entirely. There is no schema-level convention guaranteeing the artifact path appears
at the top.

## Solution
Add a blockquote convention to SCHEMAS.md §4 (task document schema). Update
EsquissePlan.agent.md Step 2b to instruct the planner to inject the blockquote into
existing task files and to include it when writing new ones. Update FRAMEWORK.md
Per-Task Startup Protocol step 3b to mention the blockquote as the canonical
top-of-file pointer.

## Scope

### In Scope
- `SCHEMAS.md` §4 — insert blockquote line (and HTML comment) after title in the task
  doc schema template, immediately before `## Status`
- `.github/agents/EsquissePlan.agent.md` — add point 4 to Step 2b (inject into
  existing task files; include when writing new ones in Step 3)
- `FRAMEWORK.md` — append one bullet to Per-Task Startup Protocol step 3b noting the
  blockquote as a supplementary pointer

### Out of Scope
- `scripts/new-task.sh` — scaffold does not auto-inject (artifacts are not known at
  scaffold time; EsquissePlan injects during planning)
- Any Go/MCP changes (covered by P2-011)
- Changes to `skills/implement-task/SKILL.md` (already updated in P2-008; step 2b
  there already directs agents to read artifacts — no further change needed)

## Prerequisites
- [x] P2-007 is done (SCHEMAS.md §10 defines Planning Artifact format)
- [x] P2-009 is done (MCP tool exists — injection behaviour referenced in EsquissePlan)

## Specification

### SCHEMAS.md §4 — exact insertion

**File:** `SCHEMAS.md`

The current schema template (inside the `````markdown` fenced block) contains the
following text (verified at line ~228):

```
# P{n}-{nnn}: {Title}

## Status
```

Replace with:

```
# P{n}-{nnn}: {Title}

> **Prerequisite:** Read `docs/artifacts/{YYYY-MM-DD}-{slug}.md` before writing any code in this phase.
<!-- One > Prerequisite line per artifact. Omit this block entirely when no artifacts apply. -->

## Status
```

That is: insert two lines (blockquote + HTML comment) and a blank line between the
title and `## Status`. The HTML comment is rendered by markdown but invisible in
rendered output — it exists solely to instruct agents.

---

### EsquissePlan.agent.md — Step 2b point 4

**File:** `.github/agents/EsquissePlan.agent.md`

The current Step 2b ends with point 3 followed by the "Do not produce artifacts..."
sentence:

```
3. Record the artifact path — link it from the `## Planning Artifacts` section
   of each task that needs it, with a "What to read from it" note.

Do not produce artifacts for internal project code. AGENTS.md, GLOSSARY.md,
llms.txt, and llms-full.txt cover internal surfaces.
```

Insert a new point 4 between point 3 and "Do not produce artifacts...":

```
4. Inject the prerequisite blockquote into each task document that uses this artifact:
   - If using the `write_planning_artifact` MCP tool: injection is **automatic**.
     The tool writes `> **Prerequisite:** Read \`{relPath}\` before writing any code
     in this phase.` immediately after the title line of each `referenced_by` task file
     that already exists on disk (idempotent — skips if already present; records path
     under `not_found` if the task file does not exist yet).
   - If writing via `create_file`: use `replace_string_in_file` to insert the blockquote
     immediately after the title line of each affected task file.
   - When creating a **new** task in Step 3 that references this artifact, include the
     blockquote immediately after the task title line before filling in `## Status`.
```

---

### EsquissePlan.agent.md — Step 3 note

**File:** `.github/agents/EsquissePlan.agent.md`

In Step 3, find:

```
For each logical unit of work, create a task document using
`bash scripts/new-task.sh {phase} {slug}`. Fill in all fields:
```

Append after this sentence (before the table):

```
If the task references a Planning Artifact (it will have a `## Planning Artifacts`
entry), include the `> **Prerequisite:**` blockquote immediately after the title line
before `## Status`.
```

---

### FRAMEWORK.md — step 3b addendum

**File:** `FRAMEWORK.md`

The current step 3b (verified at line ~576) reads:

```
3b. Load Planning Artifacts (if the task doc has a `## Planning Artifacts` section):
    - For each listed artifact, read only the sections named in "What to read from it".
    - If the artifact file is missing, note in Session Notes and do NOT reconstruct from training data.
    - If no `## Planning Artifacts` section is present, skip this step entirely.
```

Append one bullet:

```
    - The `> **Prerequisite:** Read \`...\`` blockquote at the top of the task doc is
      the canonical shortcut to the artifact path. Use it when present; fall back to
      the `## Planning Artifacts` table when the blockquote is absent.
```

## Acceptance Criteria

All verifiable via `grep` on the modified files:

| Check | Command |
|-------|---------|
| Blockquote in SCHEMAS.md schema template | `grep -n 'Prerequisite.*Read' SCHEMAS.md` → at least 1 match inside the fenced block |
| HTML comment present | `grep -n 'One > Prerequisite line per artifact' SCHEMAS.md` → 1 match |
| Point 4 in EsquissePlan Step 2b | `grep -n 'Inject the prerequisite blockquote' .github/agents/EsquissePlan.agent.md` → 1 match |
| "automatic" injection note | `grep -n 'injection is \*\*automatic\*\*' .github/agents/EsquissePlan.agent.md` → 1 match |
| Step 3 note | `grep -n 'Prerequisite.*blockquote.*title line' .github/agents/EsquissePlan.agent.md` → 1 match |
| FRAMEWORK.md addendum | `grep -n 'canonical shortcut to the artifact path' FRAMEWORK.md` → 1 match |

## Files
| File | Action | Description |
|------|--------|-------------|
| `SCHEMAS.md` | Modify | Add blockquote + HTML comment to task doc schema template |
| `.github/agents/EsquissePlan.agent.md` | Modify | Step 2b point 4 + Step 3 note |
| `FRAMEWORK.md` | Modify | Append bullet to Per-Task Startup Protocol step 3b |

## Planning Artifacts
<!-- No external library research required — all changes are to internal docs. -->

## Design Principles
1. **One authoritative pointer.** The blockquote IS the pointer; the `## Planning Artifacts`
   table is the detail. Agents should not need to scan the full document to find the artifact path.
2. **Opt-in, not required.** The HTML comment explicitly states "Omit entirely when no artifacts apply."
   Tasks with no external research do not carry an empty blockquote.
3. **Idempotent.** Inserting the blockquote twice must not duplicate it.

## Session Notes
<!-- 2026-04-21 — EsquissePlan — Created during planning session for prerequisite-injection feature. -->
2026-04-21 — ImplementerAgent — Completed. SCHEMAS.md §4, EsquissePlan.agent.md Step 2b/Step 3, FRAMEWORK.md step 3b all updated. All 6 grep checks pass.
