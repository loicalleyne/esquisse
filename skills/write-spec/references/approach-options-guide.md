# Approach Options Guide: write-spec

## What Makes Approaches Distinct

Two approaches are only distinct if they differ on **at least one** of these axes:

| Axis | Example contrast |
|------|-----------------|
| **Architectural layer** | In-process vs. separate service; synchronous vs. asynchronous |
| **Data ownership** | Embed in existing type vs. new first-class type vs. external store |
| **Interface contract** | Streaming vs. batch; push vs. pull; event-driven vs. request-response |
| **Complexity/effort** | Minimal implementation using existing primitives vs. purpose-built component |

Two approaches that differ only in naming, field order, or minor implementation
detail of the same structure are **not distinct** — collapse them into one.

---

## The Four Tradeoff Axes

For every approach, assess all four:

### 1. Effort — implementation complexity

| Rating | Criteria |
|--------|----------|
| **Low** | Fits in one task doc; touches ≤3 files; no new patterns |
| **Medium** | 2–4 task docs; touches 4–8 files; one new external dependency |
| **High** | Requires a new phase or cross-cutting changes; multiple new dependencies |

### 2. Risk — probability of unforeseen complexity

| Rating | Criteria |
|--------|----------|
| **Low** | Well-understood pattern already in codebase; no new external services |
| **Medium** | New pattern or new dependency; some unknowns remain |
| **High** | New architectural layer; external service; protocol change; novel algorithm |

### 3. Reversibility — how easy to roll back

| Rating | Criteria |
|--------|----------|
| **Easy** | Internal types only; no API or schema change; rollback = revert commit |
| **Hard** | API contract change; schema migration; shared external state; data already written |

### 4. Compatibility — impact on existing callers

| Rating | Criteria |
|--------|----------|
| **None** | No existing code touched |
| **Additive** | New function/field alongside existing — old callers unaffected |
| **Breaking** | Existing callers must be updated; major version bump required |

---

## Option Count Rules

- **Minimum 2 options.** Exception: if only one meaningful approach exists
  (e.g. "add a field to an existing struct with no alternatives"), propose it
  with a one-sentence explanation of why alternatives are impractical.
- **Maximum 3 options.** More than 3 means the feature scope is not yet
  well-understood. Narrow the scope in conversation before continuing.

---

## Recommendation Guidance

The skill proposes options — it does not prescribe. Always let the user choose.

However, apply these rules when asked for a recommendation:

- **If Effort and Risk are both Low for one option:** note it as the
  conservative default.
- **If the user asks for a recommendation:** name the option with the best
  Effort + Reversibility profile given the user's stated constraints, and
  state the reason in one sentence.
- **If all options are High Risk:** surface this before proceeding:
  "All approaches have High Risk because {reason}. This may indicate the
  feature scope is larger than expected. Do you want to narrow scope,
  proceed with awareness of the risk, or revisit the feature definition?"

---

## Option Format Template

```
**Option {A/B/C} — {short name}**
{2-3 sentence description of the approach. Name the key abstractions.}
- Effort: {Low / Medium / High}
- Risk: {Low / Medium / High}
- Reversibility: {Easy / Hard}
- Compatibility: {None / Additive / Breaking}
- Best when: {one condition that makes this the right choice}
```
