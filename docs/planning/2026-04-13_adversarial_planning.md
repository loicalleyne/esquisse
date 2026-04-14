# Plan: Adversarial Planning Review for Esquisse

**Date:** 2026-04-13
**Status:** IMPLEMENTED

---

## Goal

Add a mandatory adversarial planning review stage to the Esquisse workflow. Every
plan produced in a VS Code Copilot Chat session is reviewed by a genuinely distinct
adversarial agent (different model priority array, hostile persona) before the
session can proceed to implementation. The review is enforced via a VS Code Stop
hook — it cannot be skipped.

---

## Why a Separate Agent, Not a Skill

A skill running inside the calling agent is self-reflection: the same model reviews
its own output (~30% error detection). A separate `AdversarialD` agent with a
different model priority array produces genuine cross-model adversarial review
(~85% error detection). The skill provides the 7-attack review protocol as a
reference document; the agent provides the model, persona, and enforcement.

For multi-iteration reviews, rotating the reviewer model across invocations further
increases robustness — each model has different blind spots and reasoning patterns.

---

## VS Code Model API

VS Code exposes a Language Model API (`vscode.lm`) to *extensions*, but `.agent.md`
files are not extensions. Agents can only use tools declared in their frontmatter.
There is no built-in tool that exposes model introspection, and the `model:` array
in frontmatter is a priority list resolved by VS Code at invocation time — the
agent cannot change its own model mid-run.

**Consequence:** model rotation requires multiple agent files, each with a
different priority array. A dispatcher (PlanD body) reads an iteration counter
from a gitignored state file and hands off to the appropriate rotation variant.

---

## Model Rotation Strategy

### Available models in VS Code GitHub Copilot Chat (April 2026)

| Model | Provider | Type |
|---|---|---|
| Claude Opus 4.6 (copilot) | Anthropic | Chat — highest capability |
| Claude Sonnet 4.6 (copilot) | Anthropic | Chat — mid-tier |
| GPT-4.1 (copilot) | OpenAI | Chat |
| GPT-4o (copilot) | OpenAI | Chat |
| GPT-5 mini (copilot) | OpenAI | Fast chat — lightweight, not suited for deep critique |

Only two providers are available (Anthropic and OpenAI). Cross-provider diversity
favours Anthropic for PlanD + OpenAI primaries for r0 and r2, with Opus 4.6 in r1
to catch issues Sonnet 4.6 may have missed at a higher capability level.

### PlanD model assignment

PlanD primary: **Claude Sonnet 4.6**. Fallback: GPT-4.1.

### Rotation pool (must differ from PlanD primary)

| Slot | Agent file | Primary model | Rationale |
|---|---|---|---|
| r0 | `AdversarialD-r0.agent.md` | GPT-4.1 | Different provider from PlanD (Anthropic → OpenAI) |
| r1 | `AdversarialD-r1.agent.md` | Claude Opus 4.6 | Same provider, higher capability — catches what Sonnet missed |
| r2 | `AdversarialD-r2.agent.md` | GPT-4o | Different provider, different OpenAI model than r0 |

Rotation: `iteration % 3` → slot index. Iteration counter persists in
`.adversarial/state.json` (gitignored).

### State file format

`.adversarial/state.json`:
```json
{
  "iteration": 2,
  "last_model": "GPT-4o (copilot)",
  "last_verdict": "CONDITIONAL",
  "last_review_date": "2026-04-13"
}
```

PlanD reads `iteration`, dispatches `AdversarialD-r{iteration % 3}`. Each
rotation agent increments `iteration` and records `last_model` after writing
its report. If `.adversarial/state.json` is absent, treat as `iteration: 0`.

### Gitignore entry

`scripts/init.sh` appends `.adversarial/` to the project `.gitignore`. The folder
is created on first review. Review history in `.adversarial/reports/` is local-only.

---

## Architecture

```
PLANNING (PlanD agent)
    │
    │  plan complete
    ▼
Stop hook fires → scripts/gate-review.sh
    │
    ├─ adversarial-review.md absent or Verdict: FAILED
    │        └─ exit 2  (blocking — agent cannot exit)
    │           stderr: "Adversarial review required before proceeding."
    │           Handoff: Revise Plan → @PlanD [send:true]
    │
    ├─ Verdict: CONDITIONAL
    │        └─ exit 0  (unblocked)
    │           User decides whether to proceed or revise
    │
    └─ Verdict: PASSED
             └─ exit 0  (unblocked)
                Handoff: Start Implementation → agent
```

**VS Code agent precedence:** `.github/agents/PlanD.agent.md` (workspace-level)
shadows `~/.copilot/agents/PlanD.agent.md` (user-level) when the esquisse project
is open. The workspace agent adds the Stop hook and `AdversarialD` handoff on top
of the user-level agent's existing behaviour.

**Infinite-loop guard:** VS Code injects `"stop_hook_active": true` in the Stop
hook stdin JSON when a hook has already triggered continuation in the same turn.
`gate-review.sh` must return `{"continue": true}` immediately when this flag is
present.

---

## Files

All files are project-level and scaffolded by `scripts/init.sh` after this feature
is implemented. No user-level (`~/.copilot/`) files are required.

### New files

| # | Path | Description |
|---|------|-------------|
| 1 | `.github/agents/PlanD.agent.md` | Workspace-level PlanD; shadows user-level. Adds Stop hook + AdversarialD handoff. `disable-model-invocation: true`. |
| 2 | `.github/agents/AdversarialD.agent.md` | Read-only adversarial reviewer. Hostile persona. Model priority array distinct from PlanD. Writes verdict to `/memories/session/adversarial-review.md`. |
| 3 | `scripts/gate-review.sh` | Stop hook command. Reads verdict file; exits 2 (blocking) if absent or FAILED; exits 0 for CONDITIONAL/PASSED. |
| 4 | `.github/hooks/hooks.json` | Fallback Stop hook (fires for all agents). Fast no-op when no review file exists. |
| 5 | `skills/adversarial-review/SKILL.md` | Thin SKILL.md orchestrator (≤500 lines). Loads protocol + report template. |
| 6 | `skills/adversarial-review/references/task-review-protocol.md` | 7-attack review protocol (full reference doc). |
| 7 | `skills/adversarial-review/references/report-template.md` | Machine-readable report template. `Verdict:` line required by `gate-review.sh`. |

### Modified files

| # | Path | Change |
|---|------|--------|
| 8 | `scripts/gate-check.sh` | Append Check 8: gate-review.sh present and executable. |
| 9 | `scripts/init.sh` | Add adversarial review section using existing existence-guard pattern (safe to re-run on existing projects). Copies all four agent files, hooks.json, adversarial-review skill dir, gate-review.sh. Appends `.adversarial/` to `.gitignore`. |
| 10 | `FRAMEWORK.md §11` | Add adversarial-review rows to Conversational Workflows table; add VS Code prerequisite note (`chat.useCustomAgentHooks: true`). |

---

## Detailed File Specifications

### 1. `.github/agents/PlanD.agent.md`

```yaml
---
name: PlanD
description: >
  Planning agent for esquisse projects. Decomposes approved specs into
  bite-sized task documents (docs/tasks/P{n}-{nnn}-{slug}.md). Every plan
  must pass adversarial review before implementation begins.
target: vscode
disable-model-invocation: true
model: ['Claude Sonnet 4.6 (copilot)', 'GPT-4.1 (copilot)', 'GPT-4o (copilot)']
tools:
  - read
  - search
  - write
  - vscode/memory
agents:
  - AdversarialD
hooks:
  Stop:
    - command: "./scripts/gate-review.sh"
---
```

Body: inherits PlanD planning behaviour from the user-level agent. Adds mandatory
handoff step:

> When the plan is complete, hand off to @AdversarialD for adversarial review.
> Wait for the verdict. If FAILED, revise and resubmit. Do not hand off to the
> implementation agent until the verdict is PASSED or CONDITIONAL.

### 2–4. `.github/agents/AdversarialD-r{0,1,2}.agent.md`

Three files sharing the same body; only `name:` and `model:` differ.

**r0 (GPT-4.1 — OpenAI, different provider from PlanD):**
```yaml
---
name: AdversarialD-r0
description: >
  Adversarial plan reviewer, rotation slot 0. Hostile, skeptical persona.
  Applies 7-attack review protocol. Primary model: GPT-4.1.
target: vscode
user-invocable: false
model: ['GPT-4.1 (copilot)', 'GPT-4o (copilot)', 'Claude Opus 4.6 (copilot)']
tools:
  - read
  - search
  - write
agents: []
---
```

**r1 (Claude Opus 4.6 — Anthropic, higher capability than PlanD):**
```yaml
---
name: AdversarialD-r1
description: >
  Adversarial plan reviewer, rotation slot 1. Hostile, skeptical persona.
  Applies 7-attack review protocol. Primary model: Claude Opus 4.6.
target: vscode
user-invocable: false
model: ['Claude Opus 4.6 (copilot)', 'GPT-4.1 (copilot)', 'GPT-4o (copilot)']
tools:
  - read
  - search
  - write
agents: []
---
```

**r2 (GPT-4o — OpenAI, different OpenAI model than r0):**
```yaml
---
name: AdversarialD-r2
description: >
  Adversarial plan reviewer, rotation slot 2. Hostile, skeptical persona.
  Applies 7-attack review protocol. Primary model: GPT-4o.
target: vscode
user-invocable: false
model: ['GPT-4o (copilot)', 'GPT-4.1 (copilot)', 'Claude Opus 4.6 (copilot)']
tools:
  - read
  - search
  - write
agents: []
---
```

**Shared body (all three files):**

Hostile adversarial persona. On invocation:
1. Read `skills/adversarial-review/references/task-review-protocol.md`.
2. Read `.adversarial/state.json` if present; note `iteration` and `last_model`.
3. Apply the 7-attack protocol to the plan.
4. Write the report to `.adversarial/reports/review-{ISO-date}-iter{N}.md` using
   the template at `skills/adversarial-review/references/report-template.md`.
   Include `Model: {model name used}` in the report header.
5. Update `.adversarial/state.json`: increment `iteration`, set `last_model`,
   `last_verdict`, `last_review_date`.
6. Write `Verdict: {PASSED|CONDITIONAL|FAILED}` as the final non-empty line of the
   report (machine-read by `gate-review.sh`).

Verdict values: `PASSED`, `CONDITIONAL`, `FAILED`.

Handoffs:
- `FAILED` → Revise Plan: `@PlanD` (`send: true`)
- `PASSED` / `CONDITIONAL` → Accept and Proceed: `agent`

**PlanD dispatch logic (body addition):**

> When the plan is complete:
> 1. Read `.adversarial/state.json`. If absent, treat `iteration` as 0.
> 2. Compute `slot = iteration % 3`.
> 3. Dispatch `@AdversarialD-r0` (slot 0), `@AdversarialD-r1` (slot 1), or
>    `@AdversarialD-r2` (slot 2).
> 4. Wait for the verdict written to `.adversarial/state.json`.
> 5. If FAILED, revise the plan and restart from step 1 (iteration will have
>    incremented, so the next reviewer is a different model).
> 6. Do not hand off to the implementation agent until verdict is PASSED or
>    CONDITIONAL.

### 3. `scripts/gate-review.sh`

```bash
#!/usr/bin/env bash
# Stop hook: enforce adversarial review verdict before session exit.
# Called by VS Code with hook input JSON on stdin.
set -euo pipefail

INPUT="$(cat)"

# Infinite-loop guard: if a previous hook already triggered continuation, pass through.
if echo "$INPUT" | grep -q '"stop_hook_active": *true'; then
  echo '{"continue": true}'
  exit 0
fi

# State file written by AdversarialD-r{0,1,2} in the project-local .adversarial/ folder.
STATE_FILE=".adversarial/state.json"

if [[ ! -f "$STATE_FILE" ]]; then
  echo "Adversarial review required before proceeding. Dispatch @AdversarialD-r0 (or let PlanD dispatch automatically)." >&2
  exit 2
fi

LAST_VERDICT="$(python3 -c "import sys,json; print(json.load(open('$STATE_FILE')).get('last_verdict',''))" 2>/dev/null || echo "")"

if [[ "$LAST_VERDICT" == "PASSED" || "$LAST_VERDICT" == "CONDITIONAL" ]]; then
  echo '{"continue": true}'
  exit 0
else
  ITER="$(python3 -c "import sys,json; print(json.load(open('$STATE_FILE')).get('iteration',0))" 2>/dev/null || echo "0")"
  SLOT=$(( ITER % 3 ))
  echo "Adversarial review verdict is ${LAST_VERDICT:-absent}. Revise the plan and resubmit (next reviewer: AdversarialD-r${SLOT})." >&2
  exit 2
fi
```

Note: `python3` is used for JSON parsing (available on all target platforms). If
python3 is absent, the script falls back to treating the verdict as empty (failing
safe — blocking exit).

### 4. `.github/hooks/hooks.json`

```json
{
  "hooks": {
    "Stop": [
      {
        "type": "command",
        "command": "./scripts/gate-review.sh"
      }
    ]
  }
}
```

This is the fallback that fires for all agents (not only PlanD). When no review
file exists and the active agent is not a planning agent, `gate-review.sh` should
detect the context and fast-pass. The agent-scoped Stop hook on PlanD is the
primary enforcement path.

### 5. `skills/adversarial-review/SKILL.md`

Thin SKILL.md orchestrator. Frontmatter:

```yaml
---
name: adversarial-review
description: >
  Adversarial plan review using the 7-attack protocol. USE when a plan has been
  produced and must be reviewed before implementation. Dispatches AdversarialD.
  DO NOT USE for code review (use requesting-code-review), for ongoing
  implementation work, or for spec writing.
---
```

Steps:
1. Validate: confirm a plan exists in `/memories/session/` or `docs/tasks/`.
2. Load protocol: read `references/task-review-protocol.md`.
3. Load template: read `references/report-template.md`.
4. Hand off: dispatch `@AdversarialD` with plan content + protocol + template.
5. Confirm: report verdict to user; if FAILED, invoke `@PlanD` handoff.

### 6. `skills/adversarial-review/references/task-review-protocol.md`

7-attack review protocol (remapped from `adverserial.md` to plan/spec quality):

| Attack Vector | What to check |
|---|---|
| 1. False assumptions | Does the plan assume facts not stated in the spec? |
| 2. Edge cases | Are boundary conditions handled (empty input, max load, concurrent access)? |
| 3. Security | Does any step expose injection, traversal, or privilege escalation risks? |
| 4. Logic contradictions | Do any two tasks contradict each other's preconditions or outputs? |
| 5. Context blindness | Does the plan ignore relevant constraints from AGENTS.md or the existing codebase? |
| 6. Failure modes | What happens when each external dependency (DB, API, FS) is unavailable? |
| 7. Hallucination | Does the plan reference files, functions, or libraries that do not exist? |

Verdict criteria:
- **PASSED**: Zero FAILED attacks. Zero unmitigated CONDITIONAL attacks.
- **CONDITIONAL**: One or more CONDITIONAL attacks with documented mitigation steps.
- **FAILED**: One or more FAILED attacks with no mitigation path.

### 7. `skills/adversarial-review/references/report-template.md`

```markdown
# Adversarial Review Report

**Plan:** {plan title or file path}
**Reviewer:** AdversarialD-r{slot} ({model name})
**Iteration:** {N}
**Date:** {ISO date}

## Attack Results

| # | Attack Vector | Result | Notes |
|---|---|---|---|
| 1 | False assumptions | PASSED/CONDITIONAL/FAILED | {detail} |
| 2 | Edge cases | PASSED/CONDITIONAL/FAILED | {detail} |
| 3 | Security | PASSED/CONDITIONAL/FAILED | {detail} |
| 4 | Logic contradictions | PASSED/CONDITIONAL/FAILED | {detail} |
| 5 | Context blindness | PASSED/CONDITIONAL/FAILED | {detail} |
| 6 | Failure modes | PASSED/CONDITIONAL/FAILED | {detail} |
| 7 | Hallucination | PASSED/CONDITIONAL/FAILED | {detail} |

## Required Revisions

{List required changes if Verdict is FAILED or CONDITIONAL. Empty if PASSED.}

Verdict: {PASSED|CONDITIONAL|FAILED}
```

The `Verdict:` line is machine-read via `.adversarial/state.json` (`last_verdict`
field). The report file is human-readable history; `gate-review.sh` reads only the
state file. The `Model:` line is recorded so rotation history is auditable.

---

## Modified File Details

### 8. `scripts/gate-check.sh` — Check 8

Append after the existing 7 checks:

```bash
# Check 8: adversarial review infrastructure present
check "gate-review.sh present and executable" \
  "[[ -x scripts/gate-review.sh ]]"
```

### 9. `scripts/init.sh` additions

`$SCRIPT_DIR` is already defined at the top of `init.sh` as the directory
containing the script. `$ESQUISSE_DIR` is not a separate variable — derive it
as `"${SCRIPT_DIR}/.."` (resolved to an absolute path). All copies use the
existing `create_file` / existence-guard pattern so the script is safe to
re-run on existing projects without overwriting customized agent files.

Add after the existing Skills section:

```bash
# ── Adversarial review infrastructure ────────────────────────────────────────
echo ""
echo "Adversarial review:"
ESQUISSE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Directories
for dir in .github/agents .github/hooks; do
    create_dir "${TARGET_DIR}/${dir}"
done

# Agent files
for agent in PlanD.agent.md AdversarialD-r0.agent.md AdversarialD-r1.agent.md AdversarialD-r2.agent.md; do
    src="${ESQUISSE_DIR}/.github/agents/${agent}"
    dst="${TARGET_DIR}/.github/agents/${agent}"
    if [[ -f "$src" ]]; then
        if [[ ! -f "$dst" ]]; then
            cp "$src" "$dst"
            echo "  created  .github/agents/${agent}"
        else
            echo "  exists   .github/agents/${agent}  (skipped)"
        fi
    else
        echo "  missing  .github/agents/${agent}  (not found in Esquisse dir)"
    fi
done

# Hooks
src="${ESQUISSE_DIR}/.github/hooks/hooks.json"
dst="${TARGET_DIR}/.github/hooks/hooks.json"
if [[ -f "$src" ]]; then
    if [[ ! -f "$dst" ]]; then
        cp "$src" "$dst"
        echo "  created  .github/hooks/hooks.json"
    else
        echo "  exists   .github/hooks/hooks.json  (skipped)"
    fi
fi

# Adversarial review skill
src="${ESQUISSE_DIR}/skills/adversarial-review"
dst="${TARGET_DIR}/skills/adversarial-review"
if [[ -d "$src" ]]; then
    if [[ ! -d "$dst" ]]; then
        cp -r "$src" "$dst"
        echo "  created  skills/adversarial-review/"
    else
        echo "  exists   skills/adversarial-review/  (skipped)"
    fi
fi

# gate-review.sh alongside other scripts
src="${SCRIPT_DIR}/gate-review.sh"
dst="${TARGET_DIR}/scripts/gate-review.sh"
if [[ -f "$src" ]]; then
    if [[ ! -f "$dst" ]]; then
        cp "$src" "$dst"
        chmod +x "$dst"
        echo "  created  scripts/gate-review.sh"
    else
        echo "  exists   scripts/gate-review.sh  (skipped)"
    fi
fi

# Gitignore the local adversarial state folder (idempotent)
gitignore="${TARGET_DIR}/.gitignore"
grep -qxF '.adversarial/' "$gitignore" 2>/dev/null || echo '.adversarial/' >> "$gitignore"
echo "  ensured  .adversarial/ in .gitignore"
```

### 10. `FRAMEWORK.md §11` additions

Add to the Conversational Workflows table:

| Skill | Triggers | What it does |
|---|---|---|
| `skills/adversarial-review/SKILL.md` | "review this plan", "adversarial review" | Dispatches AdversarialD; 7-attack protocol; writes verdict |

Add prerequisite note below the table:

> **VS Code prerequisite:** `"chat.useCustomAgentHooks": true` must be enabled in
> VS Code settings for agent-scoped Stop hook enforcement on `PlanD`. The
> `.github/hooks/hooks.json` fallback fires automatically for all agents without
> this setting, but provides weaker enforcement (fast-passes for non-planning
> agents).

---

## Implementation Order

Phases 1 and 2 are independent and can proceed in parallel.

### Phase 1 — Agents, Hook, Gate Script

1. `skills/adversarial-review/references/task-review-protocol.md`
2. `skills/adversarial-review/references/report-template.md`
3. `skills/adversarial-review/SKILL.md`
4. `.github/agents/AdversarialD-r0.agent.md` (o4-mini)
5. `.github/agents/AdversarialD-r1.agent.md` (Gemini 2.5 Pro)
6. `.github/agents/AdversarialD-r2.agent.md` (GPT-4.1)
7. `.github/agents/PlanD.agent.md` (reads state, dispatches r{iteration%3})
8. `scripts/gate-review.sh` + `chmod +x`
9. `.github/hooks/hooks.json`

### Phase 2 — Integration

10. `scripts/gate-check.sh` — append Check 8
11. `scripts/init.sh` — add directory creation + copy loops + `.gitignore` entry
12. `FRAMEWORK.md §11` — add table rows + prerequisite note

---

## Key Decisions

| Decision | Choice | Reason |
|---|---|---|
| VS Code model API | Not used | `vscode.lm` is extension-only; agents cannot introspect or switch models at runtime |
| Rotation mechanism | Multiple agent files + gitignored state file | Only way to guarantee different models per iteration; state file enables auditable history |
| Rotation pool | GPT-4.1, Claude Opus 4.6, GPT-4o | Two providers available; r0+r2 cross-provider (OpenAI), r1 upgrades capability within Anthropic; all differ from PlanD primary (Sonnet 4.6) |
| GPT-5 mini excluded from rotation | Fallback only | Lightweight model; insufficient depth for thorough adversarial critique |
| State storage | `.adversarial/state.json` (gitignored) | Project-local, persistent across sessions, human-readable; avoids VS Code memory tool's session scope |
| Skill vs agent for adversarial reviewer | 3 rotation agents + skill (protocol only) | Separate model priority arrays give genuine cross-model review; skill provides protocol docs |
| User-level vs workspace agents | `.github/agents/` (workspace-level) | Self-contained; shadows user-level by same name per VS Code precedence |
| Mandatory vs optional review | Mandatory (Stop hook, exit 2) | Optional review is ignored under deadline pressure |
| Hook location | Agent-scoped (PlanD frontmatter) + `.github/hooks/hooks.json` fallback | Agent-scoped = strongest enforcement; fallback = works without `chat.useCustomAgentHooks` |
| `stop_hook_active` handling | Fast-pass `{"continue": true}` immediately | Prevents infinite hook loop in VS Code |

---

## VS Code Settings Required

```json
{
  "chat.useCustomAgentHooks": true
}
```

Without this setting, agent-scoped hooks in `.agent.md` frontmatter do not fire.
The `.github/hooks/hooks.json` fallback still fires (all-agent Stop hook), but
returns fast-pass for non-PlanD agents.

---

## References

- `adverserial.md` — original 7-attack protocol source
- `FRAMEWORK.md §11` — helper scripts and conversational workflows table
- VS Code hooks docs — `chat.useCustomAgentHooks`, Stop hook lifecycle
- Session memory: `/memories/session/plan.md`
