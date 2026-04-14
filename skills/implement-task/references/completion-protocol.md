# Per-Task Completion Protocol

Source: FRAMEWORK.md §9

When acceptance criteria are met, execute these steps in order before ending
the session.

---

## 1. Task Document

- Set `Status: Done`.
- Append to `## Session Notes`:
  ```
  {YYYY-MM-DD} — Completed. {one sentence on approach or key decision.}
  ```
- If any "Out of Scope" item was nearly touched: note it with a reference to
  the task that should own it.

## 2. AGENTS.md

For every:
- Workaround or surprising API behaviour encountered
- Near-mistake that could trap a future agent
- New exported type or package added

Add an entry to `## Common Mistakes to Avoid` or update `## Repository Layout`.

## 3. GLOSSARY.md

For every new domain term introduced (type name, concept, event name):
add a row to `GLOSSARY.md`.

**Trigger:** if you named something during this task that isn't already in the
glossary, add it now.

## 4. ONBOARDING.md

- New package or key file created → add row to Key Files Map.
- Primary data flow changed → update Data Flow diagram.

## 5. ROADMAP.md

- Mark this task ✅ Done in the phase table.
- New tasks identified during implementation → create stub task docs
  (`Status: Draft`) and add to ROADMAP.md.

## 6. docs/decisions/ (ADRs)

**Trigger:** if the phrase "we chose X over Y because" appeared anywhere in
the session reasoning, that reasoning belongs in an ADR.

- **If `write-adr` skill available:** invoke it.
- **Otherwise:** create `docs/decisions/ADR-{nnnn}-{slug}.md` using the
  schema from SCHEMAS.md §5 directly. Set status: Accepted.

## 7. docs/planning/NEXT_STEPS.md

Append to `## Session Log`:
```
{YYYY-MM-DD} — {task-id} — {one sentence summary of what was done}
Files changed: {comma-separated list}
Up next: {next task id and slug, or "phase gate for P{n}" if this is the last task}
```

**This step is mandatory. Never skip it.**

## 8. llms.txt / llms-full.txt

Update **only** if a public API was added or changed.
Do not update mid-task — only after tests pass and the task is Done.

- **If `update-llms` skill available:** invoke it.
- **Otherwise:** run the canonical doc-gen for the project language:
  - Go: `go doc ./...` → update `llms.txt`; `go doc -all ./...` → update `llms-full.txt`
  - Python: `pydoc-markdown` or equivalent
  - Other: consult AGENTS.md `## Build Commands`

Update both `llms.txt` and `llms-full.txt` together. Never one without the other.

## 9. Phase Gate

Only when all tasks in the current phase are Done or Blocked:

Run the full phase gate. Do not mark a phase complete until every gate
criterion passes.

- **If `run-phase-gate` skill available:** invoke it.
- **Otherwise:** run `bash scripts/gate-check.sh {n}` directly and manually
  verify each check.

This step is **user-initiated**. The `implement-task` skill prompts the user
(Step 18 of SKILL.md) but does not run the gate automatically.

## 10. PR / Handoff

When the task is complete and all documentation is updated:

- Create a pull request using the PR Schema from SCHEMAS.md §7.
- The PR description must be complete enough for the reviewer to test without
  reading source code. Link to the task doc instead of duplicating it.

This step is **user-initiated**. The `implement-task` skill notifies the user
at Step 18 but does not create the PR automatically.
