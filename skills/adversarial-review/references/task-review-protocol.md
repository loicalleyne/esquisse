# Task Review Protocol — 7-Attack Adversarial Plan Review

This protocol is used by `Adversarial-r{slot}` to review plans and task
documents before implementation begins. Apply **all 7 attacks** without
exception. If you cannot find serious problems, you are not looking hard enough.

---

## Adversarial Mindset

Your job is NOT to be helpful to the planner.
Your job is NOT to find what is good about the plan.
Your job is to BREAK this plan before it causes damage in implementation.

Assume the planner made mistakes. Find them.

Correct mindset:
- "How does this fail?"
- "What is assumed but not verified?"
- "What scenario is not handled?"
- "Where is the evidence for this claim?"

Wrong mindset:
- "This looks good overall."
- "Minor improvements suggested."
- "Well-structured approach."

---

## The 7 Attacks

Apply every attack to the plan under review. Record findings using the report
template. Do not skip an attack because it seems unlikely.

---

### Attack 1: False Assumptions Hunt

For every claim, step, or precondition in the plan, ask:
> "What does this assume to be true that might not be?"

Common false assumptions in implementation plans:
- "The function/type/file already exists"
- "The dependency has this API method"
- "The library version supports this feature"
- "The test framework is already configured"
- "The build passes before this task starts"
- "Concurrent goroutines / requests won't be an issue"
- "The environment variable / config key will be set"
- "The previous task's output is in the expected shape"

For each assumption found, record:
```
ASSUMPTION: [what is assumed]
REALITY CHECK: [what could be different in this codebase]
CONSEQUENCE IF WRONG: [what breaks during implementation]
```

---

### Attack 2: Edge Case Injection

Test the plan against pathological inputs and states:

**Empty / zero / null:**
- What happens when the input to each new function is empty, nil, or zero?
- Are nil-receiver methods safe?
- Are zero-length slices handled?

**Boundary values:**
- What happens at exactly the maximum limit documented in the spec?
- What happens at limit +1?
- What happens with negative values where positive is expected?

**Concurrency:**
- Does the plan assume single-goroutine access to shared state?
- Are all shared types documented as goroutine-safe or explicitly not?
- Does the plan create new goroutines without documenting their lifecycle?

**Error paths:**
- What is the behavior if each external call (DB, network, FS) fails?
- Are partial-write / partial-update states recoverable?
- Does the plan cover cleanup on failure?

---

### Attack 3: Security Adversary

Pretend you are an attacker. How do you exploit this plan?

**Injection:**
- Does any new function accept user-supplied strings that flow into SQL, shell commands, or file paths?
- Does the plan use string concatenation to build queries rather than parameterized statements?
- Does the plan use `exec.Command` or `os/exec` with user-supplied arguments?

**Path traversal:**
- Does any file path come from external input?
- Is the path validated to stay within the project root before use?

**Privilege / access boundary:**
- Does the plan grant new capabilities (file write, network, subprocess) that were not present before?
- Are those capabilities gated by authentication or authorization?

**Secrets:**
- Does the plan log, return in errors, or include in responses any value that could be a credential?

---

### Attack 4: Logic Contradiction Finder

Find internal contradictions within the plan itself:

**Consistency:**
- Does any task's "Out of Scope" contradict another task's "In Scope"?
- Do two tasks modify the same function/type in incompatible ways?
- Do the acceptance criteria contradict the stated goal?

**Ordering:**
- Does Task N depend on output from Task M that comes after it?
- Are there circular dependencies between tasks?
- Does the plan assume a type exists before the task that creates it?

**Completeness:**
- Are there steps in the process that are described but never assigned to a task?
- Are there types/functions referenced in task bodies that have no corresponding creation task?
- Does the plan account for updating all call sites when a signature changes?

---

### Attack 5: Context Blindness Probe

What relevant constraints does the plan ignore?

**AGENTS.md constraints:**
- Does the plan violate any invariant stated in AGENTS.md?
- Does any new type violate the goroutine-safety or allocation rules in AGENTS.md?
- Does the plan add global state that AGENTS.md forbids?

**Codebase reality:**
- Does the plan reference files, packages, or functions that do not exist in the codebase?
- Does the plan assume a dependency exists that is not in go.mod (or equivalent)?
- Does the plan conflict with existing tests that already cover the area being changed?

**Scale and environment:**
- Is this plan tested only against the happy path?
- Does it work correctly at 10x or 100x the expected load?
- Does it work correctly in CI (no local-only assumptions)?

---

### Attack 6: Failure Mode Analysis

For each external dependency and each new component, ask:
> "How does this fail, and what happens when it does?"

**Each external call:**
- What is the failure mode? (timeout, error response, unavailable)
- What state is the system left in on failure?
- Is the failure surfaced to the caller or swallowed?

**Each new goroutine:**
- What stops it? Under what conditions does it exit?
- What happens to in-flight work if it is stopped mid-operation?
- Is the goroutine leaked if the parent context is cancelled?

**Data integrity:**
- If the process crashes mid-task, is the on-disk/on-db state consistent?
- Are writes atomic or can partial writes produce corrupt state?

**Recovery:**
- After a failure, can the task be retried safely (is it idempotent)?
- Is the failure logged with enough detail to diagnose it?

---

### Attack 7: Hallucination Audit

AI-generated plans invent things. Find the inventions.

**Invented identifiers:**
- Does the plan reference functions, methods, types, or constants by name?
  - Do those names actually exist in the codebase?
  - Do those names exist in the stated dependencies?
  - Are the signatures correct?

**Invented library features:**
- Does the plan rely on a method or option that may not exist in the
  installed version of the dependency?

**Invented assumptions about frameworks:**
- Does the plan assume a framework behaviour (e.g., "arrow builder auto-resets")
  that is not documented?

**Invented compatibility:**
- Does the plan assume two components are compatible (e.g., Arrow v18 + DuckDB v1)
  without citing a specific tested combination?

---

## Verdict Criteria

After applying all 7 attacks, assign one verdict based on the most severe
finding across all attacks.

| Verdict | Criteria |
|---|---|
| **FAILED** | One or more Critical issues — plan cannot be safely implemented as written |
| **CONDITIONAL** | One or more Major issues, no Critical — plan can proceed only after stated mitigations are applied |
| **PASSED** | Minor issues only, or none — plan is safe to implement |

### Issue severity definitions

**Critical (→ FAILED):**
- A logic contradiction that would cause data corruption or an unrecoverable state
- A security flaw that introduces a new injection, traversal, or privilege escalation vector
- A missing dependency or non-existent function that makes the plan unimplementable
- A violated AGENTS.md invariant

**Major (→ CONDITIONAL):**
- A false assumption that will likely fail in the target environment
- Missing error handling on a failure mode that corrupts shared state
- A hallucinated API that requires rework before implementation starts
- An ordering violation that will cause a build failure mid-plan

**Minor (→ no verdict change):**
- An edge case that is unlikely in practice and does not cause data corruption
- A style inconsistency with existing code
- A missing test for a secondary code path

---

## Output Instructions

After completing all 7 attacks:

1. Write the report to `.adversarial/reports/review-{YYYY-MM-DD}-iter{N}.md`
   using the report template at
   `skills/adversarial-review/references/report-template.md`.

2. Write the plan-specific state file `.adversarial/{plan-slug}.json`
   (path and slug provided in the dispatch instruction) using the schema
   in **SCHEMAS.md §8** (canonical source of truth).
   Required fields — names are exact, do not rename:
   ```json
   {
     "plan_slug":        "{plan-slug}",
     "iteration":        N+1,
     "last_model":       "{model name you are running on}",
     "last_verdict":     "PASSED|CONDITIONAL|FAILED",
     "last_review_date": "YYYY-MM-DD"
   }
   ```
   If `.adversarial/` does not exist, create it first.

3. The final non-empty line of the report file MUST be:
   ```
   Verdict: PASSED
   ```
   (or CONDITIONAL or FAILED). This line is machine-read by `gate-review.sh`.

4. After writing files, present the verdict and issue summary to the user.

5. Handoffs:
   - `FAILED` → offer "Revise Plan" → dispatch `@EsquissePlan`
   - `PASSED` or `CONDITIONAL` → offer "Accept and Proceed" → return to caller
