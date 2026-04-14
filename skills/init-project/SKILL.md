---
name: init-project
description: >
  Use this skill when initializing or bootstrapping a new project with the Esquisse framework.
  Triggers on: "initialize a new project", "bootstrap a project", "set up a new repo",
  "fill in the stubs", "fill in the TODO placeholders", "create the foundation documents".
  DO NOT USE when the user asks to create a single task for an already-initialized project
  (use the new-task skill instead).
  DO NOT USE when the user asks to implement code (this skill only scaffolds and fills documents).
---

# Skill: Project Initialization

## Purpose

Interview the user, run `init.sh` to scaffold the directory structure, then fill in all
`TODO:` placeholders across `AGENTS.md`, `ONBOARDING.md`, `GLOSSARY.md`, `ROADMAP.md`,
and `NEXT_STEPS.md` — producing a project ready for a first implementation session.

## Prerequisites & Environment

- **Required MCP Servers:** None.
- **Required CLI:** `bash`; `init.sh` accessible at `./scripts/init.sh` (project already
  initialized) or `/opt/esquisse/scripts/init.sh` (standard shared installation).
- **Required files:** `TEMPLATES.md` from the Esquisse installation (for language adapter
  content). Do not invent build/test commands — read them from TEMPLATES.md.

## Overview

This skill has two modes:

| Mode | When to use |
|------|-------------|
| **Bootstrap** | No project directory exists yet. Runs `init.sh` then fills in stubs. |
| **Fill-in** | `init.sh` already ran. `AGENTS.md`, `ONBOARDING.md`, etc. exist but contain `TODO:` placeholders. |

---

## Execution Steps

### Step 1: Validate Context

Determine mode and locate required tools before making any changes.

**Determine mode:**
- Read the target directory (ask the user if unknown).
- If `AGENTS.md` exists: **Fill-in mode** — skip to Step 4.
- If `AGENTS.md` does not exist: **Bootstrap mode** — continue to Step 2.

**Locate init.sh:**
- Check `./scripts/init.sh` first (project already has a scripts dir).
- Check `/opt/esquisse/scripts/init.sh` (standard shared installation).
- If neither exists: stop and inform the user:
  `"init.sh not found. Provide the path to your Esquisse installation (e.g. /opt/esquisse)."`
  Do not proceed until resolved.

**Locate TEMPLATES.md:**
- Find `TEMPLATES.md` in the Esquisse installation directory.
- Read the language adapters section before filling any build/test commands in later steps.
- If not found: warn the user and note that build/test commands will need manual review.

### Step 2: Interview

Present the following prompt to the user **in a single message**. Do not ask one question at a time.

```
To initialize the project I need a few details:

1. **Project name** (kebab-case, e.g. `my-service`)
2. **Module path** (Go: `github.com/org/my-service` | Python: PyPI name | other: n/a)
3. **Primary language** (Go / Python / TypeScript / Rust / C++ / other)
4. **One-sentence description** — what does this project do?
5. **Architecture pattern** — e.g. "gRPC service", "Kafka consumer pipeline", "CLI tool", "REST API + PostgreSQL"
6. **Target directory** — absolute path (or `.` for current directory)
7. **Phase 0 name** — a short name for the first phase, e.g. "Foundation" or "Core Types"

Provide as a numbered list or however is convenient.
```

Wait for the user's response before proceeding.

---

### Step 3: Derive Computed Fields

From the user's answers, compute:

| Field | How to compute |
|-------|---------------|
| `PROJECT_SLUG` | kebab-case of project name |
| `LANGUAGE_ADAPTER` | one of: `go`, `python`, `ts`, `rust`, `cpp` |
| `BUILD_CMD` | look up in TEMPLATES.md language adapters |
| `TEST_CMD` | look up in TEMPLATES.md language adapters |
| `LINT_CMD` | look up in TEMPLATES.md language adapters |

---

### Step 4: Bootstrap (skip if AGENTS.md already exists)

Run `init.sh` from the esquisse installation directory:

```sh
bash /opt/esquisse/scripts/init.sh \
  --project-name "{PROJECT_SLUG}" \
  --target-dir "{TARGET_DIR}"
```

If the user is already inside the target directory:

```sh
bash /opt/esquisse/scripts/init.sh --project-name "{PROJECT_SLUG}"
```

Confirm the script exited 0 before continuing.

---

### Step 5: Fill in AGENTS.md

Open `AGENTS.md` (in the target dir). Replace each `TODO:` placeholder:

| Placeholder | Fill with |
|-------------|-----------|
| `## THE MOST IMPORTANT RULE` | "Tests define correctness. Fix code to match tests, never the reverse." (or a stronger invariant the user described) |
| `## Project Overview` | Module path + the user's one-sentence description + architecture pattern |
| `## Repository Layout` | A directory tree sketch appropriate for the language and architecture. Use the layout from TEMPLATES.md for the detected language. |
| `## Build Commands` | Commands from the detected language adapter (TEMPLATES.md) |
| `## Test Commands` | Commands from the detected language adapter |
| `## Key Dependencies` | Empty table header — leave for the user to fill in per-dependency |
| `## Code Conventions` | Paste the language adapter conventions verbatim from TEMPLATES.md |
| `## Common Mistakes to Avoid` | Leave the template comment — will be filled during implementation |
| `## Invariants` | Leave the template comment — will be filled during implementation |
| `## Scope Exclusions` | Ask: "Are there any features this project will explicitly NOT support?" Add one exclusion per answer. If the user has none, add one example stub. |
| `## Security Boundaries` | "Never hardcode credentials. All secrets from environment variables." + any domain-specific boundaries the user mentioned. |

---

### Step 6: Fill in ONBOARDING.md

Open `ONBOARDING.md`. Set:

- The project name in the title.
- `## What Does This Project Do?` — the one-sentence description + architecture pattern.
- `## Read Order` — the standard esquisse read order (AGENTS.md → ONBOARDING.md → GLOSSARY.md → llms.txt → llms-full.txt → docs/decisions/ → docs/tasks/ → skills/).
- `## Key Mental Model` — a 3-5 line description of the data flow or request lifecycle for the architecture pattern. Derive from the architecture description.
- Leave `## First Task` pointing to `docs/tasks/` (to be created).

---

### Step 7: Fill in GLOSSARY.md

Open `GLOSSARY.md`. Scan the user's description and architecture notes for domain nouns. Create one entry per noun in the format:

```markdown
**{Term}**
: {One sentence definition. How this term is used in this codebase specifically.}
```

Minimum 3 terms. If none are obvious, add placeholder terms appropriate for the architecture (e.g. for a Kafka pipeline: `Message`, `Batch`, `Partition`; for a REST API: `Resource`, `Handler`, `Middleware`).

---

### Step 8: Fill in ROADMAP.md

Open `docs/planning/ROADMAP.md`. Set:

- `## Current Phase: P0 — {PHASE_0_NAME}`
- `**Target:**` — one sentence describing what Phase 0 produces (e.g. "A compiling project skeleton with all framework documents, build/test commands verified, and the first integration test passing").
- The gate checklist — use the standard gate checklist from TEMPLATES.md, substituting `{test command}` and `{lint command}` with the language adapter commands.
- `### P0 Tasks` — a single placeholder row: `| P0-001 | Ready | Foundation skeleton |`

---

### Step 9: Fill in NEXT_STEPS.md

Open `NEXT_STEPS.md`. Set:

- Current date.
- Current phase: P0.
- `## What Exists` — "Project skeleton created by init.sh. TODO placeholders filled."
- `## Immediate Next` — "[ ] Create P0-001: {PHASE_0_NAME} foundation task (run `bash scripts/new-task.sh 0 foundation`)."
- Leave `## Blocked / Waiting` empty.
- `## Session Log` — one entry: today's date, "Project initialized from Esquisse framework."

---

### Step 10: Confirm and Offer Next Step

Report back:

```
Project initialized at {TARGET_DIR}:

  AGENTS.md         [filled]
  ONBOARDING.md     [filled]
  GLOSSARY.md       [filled] ({N} terms)
  ROADMAP.md        [filled] (Phase 0: {PHASE_0_NAME})
  NEXT_STEPS.md     [filled]

Scripts copied: new-task.sh, gate-check.sh, rebuild-ast.sh, macros.sql, macros_go.sql

To create the first task, say:
  "Create task 0 foundation" — or run: bash scripts/new-task.sh 0 foundation
```

If the user says "yes" or "create the first task", invoke the `new-task` skill immediately.

---

## Constraints & Security

- DO NOT overwrite a file section that has already been filled in (non-TODO content).
  Only replace text that is verbatim a `TODO:` placeholder or an empty template comment.
- DO NOT run `init.sh` a second time if `AGENTS.md` already contains non-TODO content.
  Go directly to Step 5.
- DO NOT invent build/test commands. Read them from TEMPLATES.md language adapters.
- ALWAYS confirm the target directory with the user before running `init.sh`.
- SECURITY: Do not execute any destructive shell commands. `init.sh` skips existing
  files — it is safe to re-run.
- SECURITY: Pass project names as `--project-name "{name}"` with explicit quoting.

## Edge Cases

| Situation | Action |
|-----------|--------|
| User does not provide a target dir | Default to current directory. Confirm before writing. |
| init.sh not found at `/opt/esquisse/` | Ask user where esquisse is installed. |
| AGENTS.md exists but has `TODO:` placeholders | Skip Step 4, proceed directly to Step 5. |
| User is not using Go | Skip Go-specific sections; apply the correct language adapter from TEMPLATES.md. |
| User mentions a framework dependency (e.g. bufarrowlib, Arrow) | Add it to Key Dependencies table immediately during Step 5. |
