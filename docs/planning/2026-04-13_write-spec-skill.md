# Plan: Conversational Feature Specification Skill

**Date:** 2026-04-13
**Status:** Ready

---

## Goal

Add a `write-spec` conversational skill that walks a user from a raw feature idea
to a complete, approved `docs/specs/{date}-{feature-slug}-spec.md` document in a
single chat session. The spec is the required input to `EsquissePlan` / `new-task` — so
this skill is the missing first step in the Esquisse spec-first workflow.

---

## Problem Statement

The current workflow is:

```
Idea → [gap: no skill] → docs/specs/ → EsquissePlan → docs/tasks/ → adversarial-review → ImplementerAgent
```

Without a spec skill, users either:
1. Ask `EsquissePlan` to decompose a vague idea directly — producing low-quality task docs
   that fail adversarial review.
2. Write specs manually — inconsistent structure, missing invariant audits, no
   approach options.

The `write-spec` skill closes this gap by encoding the spec workflow as a
repeatable, structured interview.

---

## Workflow Design

The skill follows a strict sequence that cannot be reordered:

```
1. Constraint audit (AGENTS.md invariants + scope exclusions)
   ↓ blocks if feature violates invariants
2. Feature type detection (api / data-model / cli / background-job / pipeline / generic)
   ↓ selects question bank
3. Approach options (2–3 distinct implementations with tradeoffs)
   ↓ user selects before deep interview begins
4. Deep interview (feature-type-specific question bank)
   ↓ produces all 8 required spec sections
5. Draft spec written to docs/specs/{date}-{slug}-spec.md
   ↓ self-review pass (skill checks its own output for gaps)
6. Present spec; iterate on user feedback
   ↓ status: draft → approved
7. Handoff to EsquissePlan
```

**Key constraint:** approach options are proposed BEFORE the deep interview, not after.
This prevents the agent from over-investing in one approach and then rationalising it.

---

## Files

### New Files

| File | Purpose |
|------|---------|
| `skills/write-spec/SKILL.md` | Primary skill: Steps 1–9, frontmatter, trigger phrases |
| `skills/write-spec/references/spec-template.md` | 8-section canonical spec template with YAML front-matter |
| `skills/write-spec/references/interview-guide.md` | Feature-type heuristics + 6 question banks |
| `skills/write-spec/references/approach-options-guide.md` | What makes approaches distinct; 4 tradeoff axes; recommendation guidance |

### Modified Files

| File | Change |
|------|--------|
| `scripts/init.sh` | Add `write-spec` to skills copy loop |
| `FRAMEWORK.md §11` | Add `write-spec` row to Conversational Workflows table |

---

## Detailed File Specs

### `skills/write-spec/SKILL.md`

**Frontmatter:**

```yaml
---
name: write-spec
description: >
  Conversational feature specification. USE when a user has a feature idea that
  needs to be designed before implementation. Triggers on: "I want to add a feature",
  "write a spec for X", "spec out X", "help me design X", "before we implement X
  let's define it", "what approach should we take for X".
  DO NOT USE for planning task decomposition (use new-task or EsquissePlan instead).
  DO NOT USE for code review after implementation.
  DO NOT USE when a spec already exists and the user wants tasks created from it
  (use new-task or EsquissePlan).
---
```

**Step 1: Constraint Audit**

Before any design work, the skill reads `AGENTS.md` invariants and scope exclusions
sections. It checks whether the feature could violate any invariant or falls in the
scope exclusions list.

If a violation is found:
- Report the specific invariant(s) that would be violated.
- Suggest a modified scope that does not violate invariants.
- Ask the user: "Do you want to proceed with the restricted scope, or revisit
  the invariant first?"
- Do NOT write a spec for a feature that explicitly violates a declared invariant.

If no violations: confirm "No invariant conflicts found. Proceeding."

**Step 2: Quick Intake**

Prompt the user **in a single message**:

```
To write a good spec I need a brief overview first:

1. Feature name (kebab-case, e.g. `rate-limiting`)
2. One-sentence description — what does it add or change?
3. Who or what triggers this feature? (user action / event / scheduled / API call)
4. Any existing files you know will be involved?

(I'll ask more detailed questions after we pick an implementation approach.)
```

Wait for the reply.

**Step 3: Feature Type Detection**

From the answer, determine the feature type using heuristics from
`skills/write-spec/references/interview-guide.md`:

| Type | Signals |
|------|---------|
| `api` | "endpoint", "route", "handler", "REST", "gRPC", "request/response" |
| `data-model` | "schema", "struct", "table", "field", "migration", "type" |
| `cli` | "command", "flag", "argument", "subcommand", "CLI" |
| `background-job` | "worker", "cron", "scheduled", "retry", "queue consumer" |
| `pipeline` | "stream", "ingestion", "transform", "fan-out", "Kafka", "stage" |
| `generic` | none of the above |

Announce the detected type: "This looks like a **{type}** feature. Using that
question bank. Tell me if that's wrong and I'll switch."

**Step 4: Propose Approach Options**

Read `skills/write-spec/references/approach-options-guide.md`. Based on the feature
type and the user's one-sentence description, propose **2–3 distinct approaches**:

Format:

```
## Approach Options

**Option A — {name}**
{2-3 sentence description}
- Effort: {Low / Medium / High}
- Risk: {Low / Medium / High}
- Best when: {condition}

**Option B — {name}**
...

**Option C — {name} (optional third)**
...

Which approach fits your constraints? (Or describe a hybrid.)
```

Wait for the user's selection.

**Step 5: Deep Interview**

Load the question bank for the detected feature type from
`skills/write-spec/references/interview-guide.md`. Ask all questions for that type
**in a single message**, grouped by section (Functional, Non-Functional,
Error Handling, Integration).

Wait for the user's full answers.

**Step 6: Write Draft Spec**

Using the spec template from `skills/write-spec/references/spec-template.md`,
write the complete spec to `docs/specs/{YYYY-MM-DD}-{feature-slug}-spec.md`.

All 8 sections must be populated:

1. Front-matter (YAML: status, phase, linked-tasks, date, feature-type, approach)
2. Overview (one-sentence description + problem being solved)
3. Goals & Non-Goals
4. Constraints & Invariants (from AGENTS.md audit in Step 1)
5. Approach (the selected option, fully detailed)
6. Acceptance Criteria (specific, testable)
7. Out of Scope (explicit list)
8. Open Questions (unresolved items, empty if none)

**Step 7: Self-Review**

Before presenting the spec to the user, check:
- [ ] Every section is populated (no empty sections).
- [ ] Acceptance criteria are specific and testable (not "it works").
- [ ] Out of Scope list has at least one entry.
- [ ] Constraints reference specific AGENTS.md invariants by name.
- [ ] The chosen approach is consistent with the acceptance criteria.
- [ ] No contradictions between Goals and Out of Scope.

If any check fails: fix the spec before presenting. Do not tell the user about
trivial self-corrections. Only surface to the user if a question needs answering.

**Step 8: Present and Iterate**

Present the spec with a summary:

```
The spec is at `docs/specs/{date}-{slug}-spec.md`.

Summary:
- Feature type: {type}
- Approach: {selected approach name}
- Acceptance criteria: {n} items
- Open questions: {n or "none"}

Review the spec and let me know if anything needs changing.
When you're satisfied, say "approve" to mark it ready for planning.
```

Apply any requested changes. Repeat until the user says "approve".

**Step 9: Finalize and Hand Off**

On approval:
1. Update the spec's YAML front-matter: `status: approved`.
2. Tell the user:
   ```
   Spec approved. Next step: use the new-task skill or ask EsquissePlan to decompose
   this spec into task documents.

   Command: "Use new-task skill for phase {current_phase}, spec:
   docs/specs/{date}-{slug}-spec.md"
   ```

**Constraints:**
- Never create task documents (docs/tasks/) — that is EsquissePlan's and new-task's job.
- Never write code or implementation stubs.
- Never modify AGENTS.md based on spec content — wait for implementation.
- Always save the spec file before asking for approval (Step 8).
- One spec per skill invocation. For multiple features, invoke separately.

---

### `skills/write-spec/references/spec-template.md`

```markdown
---
status: draft
phase: P{n}
linked-tasks: []
date: {YYYY-MM-DD}
feature-type: {api | data-model | cli | background-job | pipeline | generic}
approach: {selected approach name}
---

# Spec: {Feature Name}

## 1. Overview

{One-sentence description of what this feature adds or changes.}

**Problem being solved:** {Why does this need to exist? What breaks or is missing without it?}

---

## 2. Goals & Non-Goals

### Goals
- {Specific, measurable outcome 1}
- {Specific, measurable outcome 2}

### Non-Goals
- {What this spec explicitly does not address}

---

## 3. Constraints & Invariants

Invariants from `AGENTS.md` that this feature must preserve:

| Invariant | How this feature preserves it |
|-----------|------------------------------|
| {Invariant name/number} | {short explanation} |

Scope exclusions that bound this feature:
- {Reference any relevant scope exclusion from AGENTS.md}

---

## 4. Approach

**Selected approach:** {name}

{Full description of the chosen implementation approach. Include:}
- {Architecture or structural change}
- {Key types or interfaces involved}
- {Integration points with existing code}
- {Data flow changes if any}

**Rejected alternatives:**
| Alternative | Reason rejected |
|-------------|----------------|
| {Option A name (if not selected)} | {reason} |

---

## 5. Acceptance Criteria

All criteria must be verifiable by a test or observable behaviour:

- [ ] {Specific, testable criterion 1}
- [ ] {Specific, testable criterion 2}
- [ ] {Edge case criterion}
- [ ] {Error path criterion}

---

## 6. Out of Scope

The following are explicitly excluded from this spec:

- {Excluded item 1 — if needed, create a separate spec}
- {Excluded item 2}

---

## 7. Open Questions

{Leave empty if none. Format: question | owner | due}

| Question | Owner | Due |
|----------|-------|-----|
| {Unresolved question} | {user / EsquissePlan / implementer} | {before task start / before P{n} gate} |

---

## 8. Session Notes

{Leave empty. Populated during implementation sessions.}
```

---

### `skills/write-spec/references/interview-guide.md`

The interview guide contains two components:

#### Feature Type Heuristics

A keyword-matching table to detect feature type from the user's one-sentence description.
See "Feature Type Detection" in SKILL.md Step 3.

**Priority order:** if multiple types match, use the first match in the table order
(api > data-model > cli > background-job > pipeline > generic).

#### Question Banks (one per feature type)

Each bank has 4 sections; the skill asks all sections **in a single message**.

**`api` — REST/gRPC endpoint question bank:**

```
## Functional
1. What is the HTTP method and path (REST) or RPC name (gRPC)?
2. What are the required request fields? What are optional?
3. What does a successful response look like? (status code + body shape)
4. Who calls this endpoint? (other services / UI / CLI / external)

## Non-Functional
5. Expected request volume (requests/sec at P50 and P99)?
6. Required latency SLO?
7. Idempotent? (same request twice = same result, no side effects)

## Error Handling
8. What are the 3 most likely error cases? What response does the caller receive?
9. Authentication required? What scheme?

## Integration
10. Which existing packages/handlers will this modify?
11. Any schema migrations required (DB, protobuf, config)?
```

**`data-model` — schema/struct/migration question bank:**

```
## Functional
1. What is the new type or table name?
2. List all fields: name, type, required/optional, constraints (unique, non-null, etc.)
3. What existing types reference this? What references it?
4. Is this append-only, mutable, or ephemeral?

## Non-Functional
5. Expected data volume (rows/records at 1 month and 1 year)?
6. Query patterns: what indexes are needed?
7. Any PII or sensitive fields requiring special handling?

## Error Handling
8. What happens on validation failure? Where is validation enforced?
9. What is the migration strategy for existing data?

## Integration
10. Which packages/layers read this type? Which write it?
11. Any serialisation format requirements (JSON, protobuf, YAML)?
```

**`cli` — command/subcommand question bank:**

```
## Functional
1. Full command signature: `{binary} {subcommand} [flags] [args]`
2. Required flags vs. optional flags (with defaults)?
3. What does the command do on success? What output does the user see?
4. Interactive (prompts for input) or non-interactive?

## Non-Functional
5. Expected runtime (ms / s / minutes)?
6. Should it be scriptable (machine-readable output mode, e.g. --json)?
7. Global flags it must respect (e.g. --config, --verbose)?

## Error Handling
8. What are the top 3 user error cases? What does the error message say?
9. Exit codes for each error class?

## Integration
10. Which internal packages does this command call?
11. Any config file format changes required?
```

**`background-job` — worker/cron/scheduler question bank:**

```
## Functional
1. What triggers this job? (cron schedule / event / queue message / signal)
2. What is one unit of work? (one message / one batch / one file)
3. What does a successful run produce? (side effects, outputs, state change)
4. What is the expected runtime per unit of work?

## Non-Functional
5. Required throughput (units/sec or units/min)?
6. Concurrency: how many workers can run simultaneously?
7. Retry policy on failure (max attempts, backoff)?

## Error Handling
8. What happens when one unit fails? (skip / retry / dead-letter / halt)
9. Observability: what metrics or logs are required?

## Integration
10. What upstream system provides input? What downstream system receives output?
11. Any locking or deduplication required to prevent double-processing?
```

**`pipeline` — stream/ingestion/transform question bank:**

```
## Functional
1. What is the source? (Kafka topic / file / HTTP stream / channel)
2. What is the sink? (DuckDB / S3 / Parquet / another topic / channel)
3. Describe one message/record in → one record out. What is the transform?
4. Batching: does the pipeline work per-message, per-batch, or per-window?

## Non-Functional
5. Required throughput (messages/sec or MB/s)?
6. Acceptable end-to-end latency (source to sink)?
7. Back-pressure strategy when sink is slow?

## Error Handling
8. What happens to a malformed message? (drop / dead-letter / halt)
9. What happens on partial batch failure?

## Integration
10. Which existing pipeline stages does this connect to?
11. Schema format of messages (protobuf / JSON / Arrow / raw bytes)?
```

**`generic` — fallback question bank:**

```
## Functional
1. In one paragraph: what does this feature do when it works correctly?
2. What inputs does it take? What outputs does it produce?
3. What existing behaviour does it change or replace?
4. Who or what calls it, and when?

## Non-Functional
5. Any performance requirements?
6. Any concurrency requirements?
7. Any data volume requirements?

## Error Handling
8. What are the top 3 ways this can fail?
9. What does each failure mode produce (error, fallback value, halt)?

## Integration
10. Which existing packages or files will this touch?
11. Any external dependencies (new libraries, new external services)?
```

---

### `skills/write-spec/references/approach-options-guide.md`

The approach options guide helps the skill produce genuinely distinct options —
not cosmetic restatements of the same approach.

#### What Makes Approaches Distinct

Two approaches are distinct if they differ on at least one axis:

| Axis | Example contrast |
|------|-----------------|
| **Architectural layer** | In-process vs. separate service; synchronous vs. asynchronous |
| **Data ownership** | Embed in existing type vs. new first-class type vs. external store |
| **Interface contract** | Streaming vs. batch; push vs. pull; event-driven vs. request-response |
| **Complexity/effort** | Minimal implementation using existing primitives vs. purpose-built component |

Two approaches are NOT distinct if they differ only in naming, field order, or
minor implementation detail of the same structure.

#### Tradeoff Axes to Evaluate

For each approach, assess:

1. **Effort** — implementation complexity (Low / Medium / High)
   - Low: fits in one task doc; touches ≤3 files
   - Medium: 2-4 task docs; touches 4-8 files
   - High: requires a new phase or cross-cutting changes

2. **Risk** — probability of unforeseen complexity (Low / Medium / High)
   - Low: well-understood pattern in this codebase; no new external dependencies
   - Medium: new pattern or new dependency; some unknowns
   - High: new architectural layer; external service; protocol change

3. **Reversibility** — how easy to roll back if wrong (Easy / Hard)
   - Easy: internal types only; no API or schema changes
   - Hard: API contract changes; schema migrations; shared state

4. **Compatibility** — impact on existing callers (None / Additive / Breaking)

#### Recommendation Guidance

The skill proposes, it does not prescribe. Always let the user choose. However:
- **If Effort and Risk are both Low for one option:** note it as the conservative default.
- **If the user asks for a recommendation:** name the option with the best
  Effort + Reversibility profile given the stated constraints, and state why.
- **If all options are High Risk:** surface this as an open question before proceeding.

#### Option Count Rules

- Minimum 2 options. Exception: if only one meaningful approach exists (e.g.
  "add a field to an existing struct"), propose it with an explanation of why
  alternatives are impractical.
- Maximum 3 options. More than 3 options means the feature scope is not yet
  well-understood — narrow it in conversation before continuing.

---

### `scripts/init.sh` change

Single-line change in the skills copy loop:

```bash
# Before
for skill_dir in init-project new-task adversarial-review; do

# After
for skill_dir in init-project new-task adversarial-review write-spec; do
```

---

### `FRAMEWORK.md §11` change

Add this row to the Conversational Workflows table, between `new-task` and
`adversarial-review`:

```markdown
| [`skills/write-spec/SKILL.md`](skills/write-spec/SKILL.md) | "I want to add a feature", "write a spec for X", "spec out X", "help me design X" | Constraint audit → feature-type detection → 2–3 approach options → deep interview → writes `docs/specs/YYYY-MM-DD-{slug}-spec.md` |
```

Add this note after the VS Code prerequisite note in §11:

> **Spec-first rule:** features go through `write-spec` before `new-task` or `EsquissePlan`.
> A spec in `docs/specs/` that has `status: approved` is the required input for
> planning. Skipping spec creation leads to vague task docs that fail adversarial
> review.

---

## Acceptance Criteria

- [ ] Invoking "write a spec for X" triggers the skill via description matching.
- [ ] Skill blocks on invariant violation before any design work.
- [ ] Approach options are proposed before the deep interview.
- [ ] All 6 feature type question banks are present in `interview-guide.md`.
- [ ] All 8 spec sections are written (none empty) before presenting to user.
- [ ] Self-review step (Step 7) runs before presenting the spec.
- [ ] Spec file is written to `docs/specs/` with correct front-matter.
- [ ] Approval updates `status: approved` in the front-matter.
- [ ] `init.sh` copies `skills/write-spec/` to new projects.
- [ ] FRAMEWORK.md §11 table includes the `write-spec` row.

---

## Out of Scope

- Task document creation (that is `new-task` / EsquissePlan).
- Code generation or stubs.
- Automatic adversarial review of the spec (the spec is reviewed implicitly when
  EsquissePlan creates task docs and those go through adversarial-review).
- Spec versioning or diff between spec revisions.
- Multi-feature specs (one skill invocation = one feature spec).

---

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Constraint audit BEFORE approach options | Prevents wasted design work on features that violate invariants. |
| Approach options BEFORE deep interview | Prevents the agent from rationalising the first option it thought of. |
| Feature-type detection, not manual selection | Reduces friction; user corrects if wrong. |
| Spec saved to `docs/specs/` before presenting | Ensures the spec survives a context reset between write and review. |
| Skill ends at approval — no task creation | Clean separation of concerns; spec quality gate is independent of decomposition quality gate. |
| `generic` fallback type | Ensures the skill never blocks on an unrecognised feature shape. |
| Max 3 approach options | More than 3 means the scope is unclear; the skill narrows scope instead of generating more options. |
