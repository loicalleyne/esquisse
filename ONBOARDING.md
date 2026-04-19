# ONBOARDING.md — esquisse

## What Is Esquisse?

Esquisse is a pure-markdown AI-agent-ready project framework. It provides the
document structure, working conventions, scripts, and reusable skill workflows
that allow AI coding agents to implement complete, correct units of work with
minimal ambiguity and minimal context waste.

It is used by cloning the repository once and then running `scripts/init.sh`
to bootstrap any new (or existing) project with Esquisse's document conventions.

Esquisse itself dogfoods the framework it provides.

---

## Agent Read Order

Read these in order before starting any task:

1. **`AGENTS.md`** — invariants, tool name rules, scope exclusions, common mistakes
2. **`GLOSSARY.md`** — canonical names for every domain concept
3. **`FRAMEWORK.md`** — full philosophy, phase-gate protocol, per-task protocols
4. **`SCHEMAS.md`** — document schemas (SKILL.md, .agent.md, task, ADR, state file)
5. **`docs/planning/ROADMAP.md`** — current phase, task status
6. **`docs/planning/NEXT_STEPS.md`** — where the last session left off
7. **Task doc** (`docs/tasks/P{n}-{nnn}-{slug}.md`) — the specific work to do

For structural questions (where is X defined, what skill calls Y), check
`code_ast.duckdb` with the `duckdb-code` skill if it exists. Fall back to
`grep_search` otherwise.

---

## Mental Model

```
Framework documents (FRAMEWORK.md, SCHEMAS.md, TEMPLATES.md)
    ↓ define
Document structure for every project using Esquisse
    ↓ includes
scripts/ → init.sh bootstraps projects; gate-check.sh validates phase gates
skills/  → SKILL.md files loaded by VS Code Copilot Chat and Crush
agents/  → .agent.md files dispatched by skills via runSubagent()
    ↓ produce
Implemented projects with AGENTS.md, task docs, ADRs, phase gates
```

The key insight: **Esquisse has no runtime**. Everything is markdown files and
bash scripts. The "output" of Esquisse is well-structured project documentation
that AI agents can consume.

---

## Key Files Map

| File | Role |
|------|------|
| `FRAMEWORK.md` | The primary reference — read this to understand the whole system |
| `SCHEMAS.md` | Document schemas — what fields each artifact type requires |
| `TEMPLATES.md` | Copy-paste starters for Go, Python, TypeScript, Rust, C/C++ |
| `scripts/init.sh` | Bootstraps a new project (creates dirs and starter docs); installs skills globally |
| `scripts/new-task.sh` | Scaffolds the next task doc in a phase (auto-sequences number) |
| `scripts/gate-check.sh` | Validates a phase gate: build, lint, tests, coverage, stubs, task status |
| `scripts/gate-review.sh` | Adversarial review enforcement; reads `.adversarial/` state |
| `scripts/upgrade.sh` | Idempotent upgrade for projects that adopted Esquisse earlier |
| `scripts/macros.sql` | Universal DuckDB/sitting_duck AST macros |
| `scripts/macros_go.sql` | Go-specific AST macros |
| `skills/adversarial-review/SKILL.md` | The multi-model plan review workflow |
| `skills/implement-task/SKILL.md` | Per-task startup + completion protocol |
| `.github/agents/EsquissePlan.agent.md` | The planning agent (EsquissePlan mode) |
| `docs/planning/ROADMAP.md` | Phase plan and task status |
| `docs/tasks/` | All active and completed task documents |
| `tests/triggers/` | Manual QA checklists for every skill |

---

## Data Flow: How a Project Gets Bootstrapped

```
User runs:
  bash /path/to/esquisse/scripts/init.sh --project-name X --target-dir /path/to/X

init.sh creates:
  X/AGENTS.md (stub)        ← to be filled by project owner
  X/ONBOARDING.md (stub)    ← to be filled
  X/GLOSSARY.md (stub)      ← to be filled
  X/docs/planning/ROADMAP.md (stub)
  X/docs/planning/NEXT_STEPS.md
  X/docs/tasks/             (empty)
  X/docs/specs/             (empty)
  X/scripts/                (copies gate-check.sh, gate-review.sh, new-task.sh)
  X/.adversarial/           (gitignored)

init.sh installs skills globally:
  ~/.copilot/skills/{skill}/   ← VS Code Copilot Chat (verbatim)
  ~/.config/crush/skills/{skill}/  ← Crush (with tool-name translation)

After P1-002 is implemented, init.sh also creates:
  X/CRUSH.md                ← Crush session context file
```

---

## Data Flow: How a Plan Gets Reviewed

```
EsquissePlan produces task docs
  → bof:adversarial-review skill orchestrates:

In VS Code:
  slot = iteration % 3
  runSubagent("Adversarial-r{slot}", full_prompt)
  → reviewer agent writes .adversarial/{slug}.json
  → EsquissePlan reads verdict

In Crush (after P1-003):
  slot = iteration % 3
  write full prompt to .adversarial/tmp/{slug}-{ts}.txt
  bash: crush run --model {model} --quiet < ".adversarial/tmp/..."
  → child process writes .adversarial/{slug}.json
  → parent reads verdict

Via MCP (after P1-004):
  tool: adversarial_review(plan_slug, plan_content)
  → esquisse-mcp resolves slot, invokes crush, reads verdict, returns result
```

---

## Invariants to Know Before Touching Anything

See `AGENTS.md §Invariants` for the full list. The two most commonly violated:

1. **No banned tool names** — `Bash`, `Task`, `Read`, `Write`, `Edit` are Claude
   Code / Cursor terms and silently fail in VS Code Copilot Chat.
2. **Adversarial review is not optional** — any EsquissePlan plan must have a
   PASSED or CONDITIONAL verdict before implementation can begin.

---

## Where to Start for Common Tasks

| Intent | Where to start |
|--------|---------------|
| Implement a task | Read the task doc → `implement-task` skill |
| Add a new skill | `write-spec` skill → `writing-plans` → `implement-task` |
| Fix a failing trigger test | `systematic-debugging` skill |
| Add a new phase of tasks | `new-task` skill |
| Review a plan adversarially | `adversarial-review` skill |
| Understand the current state | `docs/planning/ROADMAP.md` + `NEXT_STEPS.md` |
