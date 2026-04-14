# Adversarial Review Report

**Plan:** {plan title or file path}
**Reviewer:** Adversarial-r{slot} ({model name})
**Iteration:** {N}
**Date:** {YYYY-MM-DD}

---

## Attack Results

| # | Attack Vector | Result | Notes |
|---|---|---|---|
| 1 | False assumptions | PASSED/CONDITIONAL/FAILED | {detail or "None found"} |
| 2 | Edge cases | PASSED/CONDITIONAL/FAILED | {detail or "None found"} |
| 3 | Security | PASSED/CONDITIONAL/FAILED | {detail or "None found"} |
| 4 | Logic contradictions | PASSED/CONDITIONAL/FAILED | {detail or "None found"} |
| 5 | Context blindness | PASSED/CONDITIONAL/FAILED | {detail or "None found"} |
| 6 | Failure modes | PASSED/CONDITIONAL/FAILED | {detail or "None found"} |
| 7 | Hallucination | PASSED/CONDITIONAL/FAILED | {detail or "None found"} |

---

## Critical Issues (must fix before implementation — each causes FAILED verdict)

<!-- Omit this section if none. -->

**ISSUE-C{N}: {short title}**
- Attack vector: {which of the 7 attacks found this}
- Description: {what is wrong}
- Evidence: {specific section/line/task reference in the plan}
- Impact: {what breaks or fails during implementation}
- Required fix: {specific change needed to the plan}

---

## Major Issues (must fix before proceeding — CONDITIONAL verdict)

<!-- Omit this section if none. -->

**ISSUE-M{N}: {short title}**
- Attack vector: {which attack}
- Description: {what is wrong}
- Evidence: {specific reference}
- Impact: {risk level and what breaks}
- Mitigation: {specific change or condition needed before implementation}

---

## Minor Issues (track but not blocking)

<!-- Omit this section if none. -->

**ISSUE-L{N}: {short title}**
- Attack vector: {which attack}
- Description: {what is suboptimal}
- Suggested fix: {optional change}

---

## False Assumptions Found

| Assumption | Why It May Be False | Risk |
|---|---|---|
| {assumption text} | {reason it may not hold} | HIGH/MED/LOW |

*If none found, write: None identified.*

---

## Uncovered Scenarios

- {scenario}: {what happens — unclear/undefined/breaks}

*If none found, write: None identified.*

---

## Verdict Summary

{2–3 sentences: what is the most serious problem found, whether the plan can
be implemented as written, and the minimum required to make it safe to proceed.}

Verdict: {PASSED|CONDITIONAL|FAILED}
