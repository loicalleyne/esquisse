---
name: write-spec
description: >
  Conversational feature specification. USE when a user has a feature idea that
  needs to be designed before implementation. Triggers on: "I want to add a feature",
  "write a spec for X", "spec out X", "help me design X", "before we implement X
  let's define it", "what approach should we take for X".
  DO NOT invoke this skill when: planning task decomposition (use new-task or
  EsquissePlan instead); a spec already exists and the user wants tasks created from
  it (use new-task or EsquissePlan); reviewing code after implementation.
triggers:
  - "write a spec for"
  - "I want to add a feature"
  - "spec out"
  - "help me design"
  - "write a feature specification"
  - "before we implement"
  - not: "implement task — use implement-task skill instead"
  - not: "create a task document — use new-task skill instead"
  - not: "write an ADR — use write-adr skill instead"
tools_required:
  - read_file
  - create_file
  - replace_string_in_file
updated: 2026-04-13
---

# Write-Spec

## Overview

Walks the user from a raw feature idea to a complete, approved
`docs/specs/{YYYY-MM-DD}-{slug}-spec.md` in a single chat session.
The spec is the required input to `EsquissePlan` / `new-task`.

**Announce at start:** "I'm using the write-spec skill."

## Prerequisites

- [ ] `AGENTS.md` exists and has `## Invariants` and `## Scope Exclusions` sections.
- [ ] `docs/specs/` directory exists (create it if not).
- [ ] The user has a feature idea — even a vague one is enough to start.

## Workflow

### Step 1: Constraint Audit

Read `AGENTS.md` — focus on `## Invariants` and `## Scope Exclusions`.

Check whether the proposed feature could violate any invariant or falls under
a scope exclusion.

- **If a violation is found:** report the specific invariant(s), suggest a
  restricted scope that does not violate them, and ask: "Do you want to proceed
  with the restricted scope, or revisit the invariant first?" Do NOT write a spec
  for a feature that explicitly violates a declared invariant.
- **If no violations:** confirm "No invariant conflicts found. Proceeding."

### Step 2: Quick Intake

Prompt the user **in a single message**:

```
To write a good spec I need a brief overview first:

1. Feature name (kebab-case, e.g. `rate-limiting`)
2. One-sentence description — what does it add or change?
3. Who or what triggers this feature? (user action / event / scheduled / API call)
4. Any existing files you know will be involved?

(I'll ask more detailed questions after we pick an implementation approach.)
```

Wait for the reply before continuing.

### Step 3: Feature Type Detection

From the user's answer, determine feature type using the heuristics in
`skills/write-spec/references/interview-guide.md`:

| Type | Signals |
|------|---------|
| `api` | endpoint, route, handler, REST, gRPC, request/response |
| `data-model` | schema, struct, table, field, migration, type |
| `cli` | command, flag, argument, subcommand, CLI |
| `background-job` | worker, cron, scheduled, retry, queue consumer |
| `pipeline` | stream, ingestion, transform, fan-out, Kafka, stage |
| `generic` | none of the above |

Priority order: api > data-model > cli > background-job > pipeline > generic.

Announce: "This looks like a **{type}** feature. Using that question bank.
Tell me if that's wrong and I'll switch."

### Step 4: Propose Approach Options

Read `skills/write-spec/references/approach-options-guide.md`. Based on the
feature type and one-sentence description, propose **2–3 distinct approaches**:

```
## Approach Options

**Option A — {name}**
{2-3 sentence description}
- Effort: {Low / Medium / High}
- Risk: {Low / Medium / High}
- Best when: {condition}

**Option B — {name}**
...
```

Two approaches are only distinct if they differ on at least one axis: architectural
layer, data ownership, interface contract, or complexity/effort.

Wait for the user's selection before continuing.

### Step 5: Deep Interview

Load the question bank for the detected feature type from
`skills/write-spec/references/interview-guide.md`. Ask all sections
(Functional, Non-Functional, Error Handling, Integration) **in a single message**.

Wait for the user's full answers before writing the spec.

### Step 6: Write Draft Spec

Using `skills/write-spec/references/spec-template.md`, write the complete spec to
`docs/specs/{YYYY-MM-DD}-{feature-slug}-spec.md`.

All 8 sections must be populated:
1. Front-matter (YAML: status, phase, linked-tasks, date, feature-type, approach)
2. Overview
3. Goals & Non-Goals
4. Constraints & Invariants (from Step 1 audit)
5. Approach (selected option, fully detailed; rejected alternatives table)
6. Acceptance Criteria (specific, testable)
7. Out of Scope (at least one entry)
8. Open Questions (empty if none)

### Step 7: Self-Review

Before presenting, verify:
- [ ] Every section is populated — no placeholders remaining.
- [ ] Acceptance criteria are specific and testable (not "it works").
- [ ] Out of Scope has at least one entry.
- [ ] Constraints reference specific AGENTS.md invariants by name.
- [ ] The chosen approach is consistent with the acceptance criteria.
- [ ] No contradictions between Goals and Non-Goals.

Fix any failures silently. Only surface to the user if a question must be answered.

### Step 8: Present and Iterate

Present the spec:

```
Spec written to `docs/specs/{date}-{slug}-spec.md`.

Summary:
- Feature type: {type}
- Approach: {selected approach name}
- Acceptance criteria: {n} items
- Open questions: {n or "none"}

Review the spec and let me know if anything needs changing.
When you're satisfied, say "approve" to finalize it.
```

Apply requested changes. Repeat until the user says "approve".

### Step 9: Finalize and Hand Off

On approval:
1. Update the spec's YAML front-matter: `status: approved`.
2. Tell the user:
   ```
   Spec approved. Next step: use the new-task skill or ask EsquissePlan to decompose
   this spec into task documents.

   Command: "Use new-task skill for phase {current_phase}, spec:
   docs/specs/{date}-{slug}-spec.md"
   ```

## Decision Points

- **Feature type ambiguous** → ask the user to confirm before loading question bank.
- **Invariant conflict found** → stop; do not write spec; present conflict and options.
- **User selects a hybrid approach** → document it as "Hybrid (A+B)" in step 5 and spec.
- **All approaches are High Risk** → surface as open question before interviewing.
- **User approves without changes** → still update status to approved before handing off.

## Anti-Patterns

- **Don't** create task documents (`docs/tasks/`) — that is EsquissePlan's job.
- **Don't** write code or implementation stubs in the spec.
- **Don't** modify `AGENTS.md` based on spec content — wait for implementation.
- **Don't** save the spec after approval only — save at draft stage (Step 6) so
  the user can read it in their editor.
- **Don't** skip the approach options step — selecting an approach before the
  deep interview prevents the agent rationalising its first instinct.

## Definition of Done

- [ ] `docs/specs/{date}-{slug}-spec.md` exists with `status: approved`.
- [ ] All 8 spec sections are populated.
- [ ] Acceptance criteria are specific and testable.
- [ ] Handoff message presented to user with next-step command.
