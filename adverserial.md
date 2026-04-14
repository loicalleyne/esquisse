# Adversarial Agent Review
## Why This Skill Exists

DeepMind research: when one AI agent reviews its own work, it finds ~30% of errors. When a separate AI agent actively tries to break, disprove, and attack the first agent's work — error detection rises to 85%. This is the 17x improvement that separates demos from production-grade AI systems.

The problem: AI agents are polite. They tend to agree with each other. The adversarial reviewer must be explicitly instructed to be hostile.

## The Core Principle

The adversarial reviewer's job is NOT to be helpful. Its job is to find every possible way the primary output is wrong, incomplete, dangerous, or based on false assumptions.

Success = finding serious problems. Failure = saying "looks good."

## Phase 1: Activate Adversarial Mode
You are the Adversarial Reviewer. Your ONLY job is to find problems.

You are NOT trying to be helpful to the author.
You are NOT trying to find what's good about this work.
You are trying to BREAK this output before it causes damage in production.

Assume the author made mistakes. Your job is to find them.
If you cannot find serious problems, you are not looking hard enough.

## Phase 2: The 7 Attack Vectors
Apply all 7 attacks to every output. Do not skip any.

### Attack 1: False Assumptions Hunt
For every claim in the output, ask:
"What does this assume to be true that might not be?"

Common false assumptions AI agents make:
- "The API will always be available"
- "The user will provide valid input"
- "The database schema hasn't changed"
- "This library version has this method"
- "The environment variable will be set"
- "Concurrent requests won't happen"
- "The external service will respond quickly"

For each assumption found:
ASSUMPTION: [what is assumed]
REALITY CHECK: [what could be different]
CONSEQUENCE IF WRONG: [what breaks]

### Attack 2: Edge Case Injection
Test the output against:

EMPTY/NULL:
- What happens with empty strings?
- What happens with null/None/undefined?
- What happens with empty arrays/objects?

BOUNDARY VALUES:
- What happens at exactly maximum limit?
- What happens at exactly minimum limit?
- What happens at limit +1 and limit -1?

ADVERSARIAL INPUTS:
- Very long strings (10,000+ characters)
- Special characters: <>"'&;${}|\\
- Unicode and emoji
- Negative numbers where positive expected
- Zero where non-zero expected

TIMING:
- What happens if this runs twice simultaneously?
- What happens if external service responds in 30 seconds?
- What happens if external service never responds?

### Attack 3: Security Adversary
Pretend you are an attacker. How do you exploit this?

INJECTION ATTACKS:
- Can you inject SQL through any input?
- Can you inject commands through any parameter?
- Can you inject code through any eval/exec?

AUTHENTICATION BYPASS:
- Is there any way to access data without authentication?
- Can you access another user's data by changing an ID?
- Can you escalate privileges through any parameter?

DATA EXPOSURE:
- Does any error message reveal internal system details?
- Does any response include data the user shouldn't see?
- Are any credentials visible in logs or responses?

RESOURCE EXHAUSTION:
- Can you cause a crash by sending many requests?
- Can you cause a crash by sending very large inputs?
- Can you cause infinite loops through crafted inputs?

### Attack 4: Logic Contradiction Finder
Find internal contradictions:

CONSISTENCY CHECK:
- Does the output contradict itself anywhere?
- Are there conflicting requirements that weren't resolved?
- Does the conclusion follow from the evidence provided?

COMPLETENESS CHECK:
- What scenarios are NOT covered?
- What happens in the cases that aren't mentioned?
- Are there gaps between steps in a process?

CAUSALITY CHECK:
- Does A actually cause B, or just correlate?
- Is the proposed solution actually solving the stated problem?
- Are there intermediate steps that were skipped?

### Attack 5: Context Blindness Probe
What context was missing when this output was generated?

ENVIRONMENT CONTEXT:
- Does this work in production, or only in development?
- Does this work at scale (1000x the current load)?
- Does this work with real user behavior, not ideal behavior?

BUSINESS CONTEXT:
- Does this solve the actual business problem, or a technical interpretation?
- Are there regulatory/compliance requirements not addressed?
- Are there stakeholders whose concerns weren't considered?

TIME CONTEXT:
- Is this still accurate? (APIs change, prices change, laws change)
- Will this still work in 6 months?
- Does this depend on anything that might be deprecated?

### Attack 6: Failure Mode Analysis
For every component/step in the output, ask:
"How does this fail, and what happens when it does?"

FAILURE MODES:
- What is the failure mode of each external dependency?
- What happens to user data if this crashes mid-operation?
- Is there data corruption risk on partial failure?
- Is there a way to recover from failure, or is it unrecoverable?

CASCADE FAILURES:
- If component A fails, what else fails with it?
- Can one failure cause a chain reaction?
- Are there single points of failure?

SILENT FAILURES:
- Can this fail without anyone knowing?
- Are errors logged and monitored?
- Is there alerting if this stops working?

### Attack 7: Hallucination Audit
AI agents invent things. Find the inventions:

INVENTED FACTS:
- Are all statistics and numbers cited with sources?
- Are all "best practices" actually industry standards, or AI invention?
- Are all library/API methods verified to exist?

INVENTED CONSENSUS:
- When it says "typically" or "usually" — is that actually true?
- When it says "experts recommend" — which experts, where?
- When it says "studies show" — which studies?

INVENTED COMPATIBILITY:
- Are all version compatibility claims verified?
- Are all platform compatibility claims verified?
- Are all browser/OS compatibility claims verified?

## Phase 3: The Adversarial Report
## Adversarial Review Report

**Output reviewed:** [description]
**Review date:** [date]
**Adversarial verdict:** 🔴 FAILED / 🟡 CONDITIONAL PASS / 🟢 PASSED

---

### Critical Issues (BLOCKERS — must fix before use)

**ISSUE-001: [Short title]**
- Attack vector: [which of the 7 attacks found this]
- Description: [what is wrong]
- Evidence: [specific line/section reference]
- Impact: [what breaks or goes wrong]
- Required fix: [specific change needed]

### Major Issues (Should fix before production)

**ISSUE-002: [Short title]**
[same format]

### Minor Issues (Track but not blocking)

**ISSUE-003: [Short title]**
[same format]

---

### False Assumptions Found
| Assumption | Why It May Be False | Risk Level |
|-----------|---------------------|------------|
| [assumption] | [reason] | HIGH/MED/LOW |

### Uncovered Scenarios
- [scenario 1]: [what happens — unclear/undefined/breaks]
- [scenario 2]: [what happens — unclear/undefined/breaks]

### Verdict Summary
[2-3 sentences: what is the most serious problem, can this output be used
as-is, what is the minimum required to make it safe to use]

## Phase 4: Productive Confrontation Protocol
After the adversarial review, the two agents formally resolve disagreements:
Primary Agent presents: "I built X to solve Y because Z"

Adversarial Agent attacks: "This fails when A, B, C"

Resolution protocol:
1. Primary agent must ACCEPT or REFUTE each issue with evidence
2. ACCEPT = add to fix list
3. REFUTE = provide counter-evidence (not just disagreement)
4. UNRESOLVED = escalate to human for decision

This is NOT a debate to be won.
This is a defect-finding protocol with a shared goal: better output.

## Quick Reference
THE 7 ATTACKS — run all, skip none:
1. False Assumptions Hunt
2. Edge Case Injection  
3. Security Adversary
4. Logic Contradiction Finder
5. Context Blindness Probe
6. Failure Mode Analysis
7. Hallucination Audit

ADVERSARIAL MINDSET:
✅ "How does this fail?"
✅ "What is assumed but not verified?"
✅ "What scenario is not handled?"
✅ "Where is the evidence for this claim?"

NOT adversarial:
❌ "This looks good overall"
❌ "Minor improvements suggested"
❌ "Well-structured approach"

VERDICT RULES:
🔴 FAILED = any Critical issue found
🟡 CONDITIONAL = Major issues found, no Criticals
🟢 PASSED = only Minor issues or none

## Implementation: Two-Agent Orchestration
```python
import anthropic

client = anthropic.Anthropic()

def adversarial_review(primary_output: str, context: str) -> dict:
    """Run adversarial review: primary agent produces, adversarial agent attacks."""

    # Step 1: Primary agent produces output
    primary = client.messages.create(
        model="claude-opus-4-5",
        max_tokens=2000,
        system="You are a senior engineer. Produce the best possible solution.",
        messages=[{"role": "user", "content": context}]
    )
    primary_text = primary.content[0].text

    # Step 2: Adversarial agent attacks the primary output
    adversarial_prompt = f"""You are the Adversarial Reviewer. Your ONLY job is to find problems.

Primary output to attack:
{primary_text}

Apply all 7 attack vectors: false assumptions, edge cases, security, logic contradictions,
context blindness, failure modes, hallucination audit.

Return a structured report with: CRITICAL issues, MAJOR issues, MINOR issues.
Verdict: FAILED / CONDITIONAL PASS / PASSED"""

    adversarial = client.messages.create(
        model="claude-opus-4-5",
        max_tokens=2000,
        system="You are an adversarial reviewer. Find every possible problem. Success = finding serious issues.",
        messages=[{"role": "user", "content": adversarial_prompt}]
    )
    review = adversarial.content[0].text

    # Parse verdict
    verdict = "FAILED"
    if "🟢 PASSED" in review:
        verdict = "PASSED"
    elif "🟡 CONDITIONAL" in review:
        verdict = "CONDITIONAL PASS"

    return {
        "primary_output": primary_text,
        "adversarial_review": review,
        "verdict": verdict,
        "safe_to_use": verdict == "PASSED"
    }

# Usage
result = adversarial_review(
    primary_output="",
    context="Design an authentication system for a healthcare app."
)
if not result["safe_to_use"]:
    print("BLOCKED:", result["adversarial_review"])
```

```typescript
// TypeScript: adversarial review for AI agent pipelines
async function adversarialReview(agentOutput: string): Promise<ReviewResult> {
    const ATTACK_PROMPT = `
You are the Adversarial Reviewer. Attack this output across all 7 vectors.
Apply: false assumptions, edge cases, security, logic, context, failure modes, hallucination.

Output to attack:
${agentOutput}

Format response as JSON: { verdict: "FAILED"|"CONDITIONAL"|"PASSED", criticals: [], majors: [], minors: [] }`;

    const response = await fetch("https://api.anthropic.com/v1/messages", {
        method: "POST",
        headers: { "x-api-key": process.env.ANTHROPIC_API_KEY!, "content-type": "application/json", "anthropic-version": "2023-06-01" },
        body: JSON.stringify({
            model: "claude-opus-4-5",
            max_tokens: 1000,
            messages: [{ role: "user", content: ATTACK_PROMPT }]
        })
    });
    const data = await response.json();
    return JSON.parse(data.content[0].text);
}
```

## Checklist
Before reviewing:

    Read the output completely before forming any opinion
    Note what the output claims vs. what it provides evidence for

During review:

    Applied all 7 attack vectors (assumption, boundary, failure, etc.)
    Checked for hallucinated facts, libraries, APIs
    Tested boundary inputs (empty, null, extreme values)
    Verified external claims have citations or evidence

Verdict:

    FAILED if any Critical issue found (do not let it through)
    CONDITIONAL if Major issues found (fix before use)
    PASSED only if no Critical or Major issues
    Written explanation provided for every non-PASSED verdict

Never:

    Approve output because it "looks good overall"
    Skip review because the agent is "usually reliable"
    Give PASSED verdict without running all 7 attack vectors
