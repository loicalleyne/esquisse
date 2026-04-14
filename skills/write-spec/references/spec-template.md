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

{Full description of the chosen implementation approach:}
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

| Question | Owner | Due |
|----------|-------|-----|
| {Unresolved question} | {user / EsquissePlan / implementer} | {before task start / before P{n} gate} |

---

## 8. Session Notes

<!-- Leave empty. Populated during implementation sessions. -->
