---
name: adopt-project
description: >
  Use this skill to add Esquisse framework documents to an existing project that
  already has source code, build systems, and tests. Triggers on: "add esquisse to
  this project", "adopt esquisse", "add framework documents to this repo", "set up
  AGENTS.md for this project", "onboard this codebase".
  DO NOT USE when the user wants to create a brand-new project from scratch (use
  the init-project skill instead).
  DO NOT USE when the user wants to create a task document (use the new-task skill).
  DO NOT USE when the user wants to write implementation code.
---

# Skill: Adopt Esquisse in an Existing Project

## Purpose

Add Esquisse framework documents to an existing codebase by extracting real
information from the project — source files, build systems, existing tests,
config, dependencies — rather than asking the user to invent it. Follows the
incremental adoption strategy from FRAMEWORK.md §10.

The key difference from `init-project`: documents describe what already exists,
not what will be built. The glossary is archaeology (extract type names from
code), not invention. The phase label reflects the project's actual state, not P0.

## Prerequisites & Environment

- **Required MCP Servers:** None.
- **Required CLI:** `bash`; `init.sh` accessible at `./scripts/init.sh` (if already
  copied) or the Esquisse installation directory.
- **Required files:** `TEMPLATES.md` from the Esquisse installation (for language
  adapter content). The target project must already have source code.

---

## Constraints & Security

- Do NOT write task documents for work already completed. Only forward work gets tasks.
- Do NOT invent ADRs for past decisions. Only create ADRs going forward.
- Do NOT invent glossary terms. Extract only names already in the code.
- Do NOT fake the phase label. If the project is mid-P3, say P3.
- Build and test commands MUST be copy-paste runnable. Verify them before writing.
- `code_ast.duckdb` is gitignored — never commit it.

---

## Execution Steps

### Step 1: Validate Context

Determine what exists before making any changes.

**Check project state:**
- Read the project root directory listing.
- If `AGENTS.md` already exists AND contains filled sections (not just `TODO:` stubs):
  stop and tell the user: "This project already has an AGENTS.md. To update it,
  edit it directly. To re-scaffold from scratch, rename or remove the existing one first."
- If `AGENTS.md` exists but is all `TODO:` stubs: treat as **Fill-in mode** — skip
  to Step 4.
- If `AGENTS.md` does not exist: continue to Step 2.

**Locate Esquisse installation:**
- Check `./scripts/init.sh` (project already has esquisse scripts).
- Check common locations: the esquisse workspace folder if open, `/opt/esquisse/`.
- If not found: ask the user for the path.

**Locate TEMPLATES.md:**
- Find `TEMPLATES.md` in the Esquisse installation directory.
- Read the language adapters section — do not invent build/test commands.

### Step 2: Interview

Present the following prompt **in a single message**:

```
I'll add Esquisse framework documents to this existing project. A few questions:

1. **Primary language** (Go / Python / TypeScript / Rust / C++ / other)
2. **One-sentence description** — what does this project do?
3. **Architecture pattern** — e.g. "gRPC service", "Kafka consumer pipeline",
   "CLI tool", "TUI library", "REST API"
4. **Current maturity** — roughly where is this project?
   - Early (core types exist, basic tests) → ~P1
   - Feature-complete (most functionality works) → ~P2
   - Production-hardened (error handling, benchmarks done) → ~P3
   - Mature (ecosystem/plugin stage) → ~P4+
5. **The most important rule** — what is the one invariant that must never be
   violated? (default: "Tests define correctness.")
6. **Known remaining work** — any features or fixes you know are next?
   (I'll create draft task stubs for these, not full task docs.)

I'll extract everything else (dependencies, types, glossary terms, directory
layout, build commands) from the codebase directly.
```

Wait for the user's response.

### Step 3: Explore the Codebase

Before writing any documents, extract real information from the project.

**Build system detection** — check for these in order:

| File | Language | Build/Test commands |
|------|----------|---------------------|
| `Makefile` or `GNUmakefile` | any | Read targets: `grep -E '^[a-zA-Z_-]+:' Makefile` |
| `go.mod` | Go | `go build ./...`, `go test ./...`, `go vet ./...` |
| `Taskfile.yaml` or `Taskfile.yml` | any | Read tasks from YAML |
| `pyproject.toml` | Python | Look for `[tool.pytest]`, `[build-system]` |
| `package.json` | JS/TS | Read `scripts` section |
| `Cargo.toml` | Rust | `cargo build`, `cargo test` |
| `CMakeLists.txt` | C/C++ | `cmake -B build && cmake --build build` |

**If the project has a Makefile**, read it completely — it is the authoritative
source for build, test, lint, and benchmark commands. Prefer Makefile targets
over generic language commands.

**Dependency extraction:**
- Go: read `go.mod` — extract all `require` entries with comments
- Python: read `pyproject.toml` or `requirements.txt`
- JS/TS: read `package.json` `dependencies` and `devDependencies`
- Rust: read `Cargo.toml` `[dependencies]`
- C/C++: read `CMakeLists.txt` for `find_package` / `FetchContent`

**Glossary extraction:**
- If `code_ast.duckdb` exists: query for type definitions, interface names,
  exported function names using the `duckdb-code` skill.
- Otherwise: scan source files for type/class/struct definitions. For Go:
  `grep -rn 'type .* struct\|type .* interface' --include='*.go'`.
  For Python: `grep -rn 'class ' --include='*.py'`.

**Directory structure:**
- List all top-level directories and their purposes.
- Identify generated code directories (names like `gen/`, `generated/`, `build/`).
- Identify test directories.

**Existing documentation:**
- Check for existing `README.md`, `CONTRIBUTING.md`, `CLAUDE.md`, `CRUSH.md`,
  `GEMINI.md`, `.github/copilot-instructions.md`.
- If any exist, read them for project-specific conventions and invariants
  that should be carried into `AGENTS.md`.

**Verify build commands:** Run the detected build and test commands to confirm
they work. Record the results. Do not write unverified commands into AGENTS.md.

### Step 4: Run init.sh (scaffold only)

Run `init.sh` to create the directory scaffold and starter document stubs.
The script is safe to re-run — it skips files that already exist.

```sh
bash {ESQUISSE_DIR}/scripts/init.sh \
  --project-name "{PROJECT_SLUG}" \
  --target-dir "{PROJECT_ROOT}"
```

Confirm exit code 0.

### Step 5: Fill in AGENTS.md

Open `AGENTS.md`. Replace each `TODO:` placeholder using information extracted
in Step 3 — not from the user's answers alone.

| Section | Fill with | Source |
|---------|-----------|--------|
| `## THE MOST IMPORTANT RULE` | The user's answer from Q5, or the default | Interview |
| `## Project Overview` | Module path + description + architecture pattern | Interview + `go.mod` / `package.json` |
| `## Repository Layout` | Actual directory tree from Step 3 | `list_dir` / `find` |
| `## Build Commands` | Verified commands from Step 3 | Makefile / build config |
| `## Test Commands` | Verified commands from Step 3 | Makefile / build config |
| `## Benchmark Commands` | From Makefile if present | Makefile |
| `## Key Dependencies` | Table from `go.mod` / `package.json` / etc. | Dependency file |
| `## Code Conventions` | Language adapter from TEMPLATES.md + any conventions from existing docs | TEMPLATES.md + README/CONTRIBUTING |
| `## Common Mistakes to Avoid` | Seed from `// FIXME`, `// HACK`, `// TODO` comments in code. If none found, leave the stub. | `grep -rn 'FIXME\|HACK' --include='*.go'` |
| `## Invariants` | Leave stub — will be populated during first implementation task | — |
| `## Scope Exclusions` | Ask the user if anything is out of scope. Add at least one entry. | Interview |
| `## Security Boundaries` | Default template + any auth/secret patterns found in code | Code scan |

**Critical rule:** every build/test command written here must have been verified
to actually work in Step 3. Do not write aspirational commands.

### Step 6: Fill in ONBOARDING.md

| Section | Source |
|---------|--------|
| `## Mental Model` | Derive from architecture pattern + directory scan |
| `## Read-Order` | Standard Esquisse read order |
| `## Key Files Map` | Top 5-10 most important files from Step 3 exploration. For each: path, what it does, when to read it. |
| `## Data Flow` | Derive from architecture pattern. If pipeline: source → transform → sink. If API: request → handler → storage. |

Only fill the Key Files Map for files you actually found and read. Do not guess.

### Step 7: Fill in GLOSSARY.md

Populate ONLY from terms found in the codebase:
- Type names from the glossary extraction in Step 3
- Config keys from config files
- Event/message names from event types
- CLI command names

Format: `**{Term}** : {One-sentence definition. How this term is used in this codebase.}`

Minimum 5 terms. If fewer than 5 domain terms are found, the extraction was
too shallow — scan more source files.

**Rule:** use the exact names from the code. Do not clean up or rename.

### Step 8: Fill in ROADMAP.md

| Field | How to fill |
|-------|-------------|
| `## Current Phase` | `P{N} — {name}` where N matches the user's maturity answer (Q4) |
| `**Target:**` | One sentence describing the next milestone |
| Gate checklist | Standard gate from TEMPLATES.md with detected build/test/lint commands |
| Task table | One row per known remaining work item (from Q6). Status: `Draft`. These are stubs, not full task docs. |

### Step 9: Fill in NEXT_STEPS.md

```markdown
## Session Log

{YYYY-MM-DD} — Esquisse adoption — Framework documents added.
Files created: AGENTS.md, ONBOARDING.md, GLOSSARY.md, ROADMAP.md, NEXT_STEPS.md
Up next: first task under framework (use new-task skill)
```

### Step 10: Generate llms.txt

Generate a first-draft `llms.txt` from the codebase:

- Go: `go doc ./...` output, reformatted as the llms.txt schema
- Python: `pdoc` or manual scan of `__init__.py` exports
- Other: scan for exported/public symbols

This is a rough first draft. Note at the top: `<!-- First draft — refine at next phase gate -->`.

Create `llms-full.txt` as a stub with the same note.

### Step 11: Build AST Cache (optional)

If the project has a supported language (Go, Python, TypeScript, C/C++, Rust, Java):

```sh
duckdb -init /dev/null code_ast.duckdb <<'SQL'
LOAD sitting_duck;
CREATE OR REPLACE TABLE ast AS
SELECT * FROM read_ast('{glob}', ignore_errors := true, peek := 200);
SQL
```

Use the appropriate glob for the language (`**/*.go`, `**/*.py`, etc.).

Add `code_ast.duckdb` to `.gitignore` if not already present.

If `duckdb` or `sitting_duck` is not available, skip this step and note it as
a manual follow-up.

### Step 12: Confirm and Summarize

Present a summary to the user:

```
Esquisse adoption complete for {PROJECT_NAME}.

Documents created:
  AGENTS.md           — {line count} lines, {N} key dependencies documented
  ONBOARDING.md       — {N} key files mapped
  GLOSSARY.md         — {N} terms extracted from codebase
  ROADMAP.md          — Current phase: P{N}, {M} draft task stubs
  NEXT_STEPS.md       — Session log started
  llms.txt            — First draft ({K} symbols indexed)

Verified commands:
  Build: {command} ✓
  Test:  {command} ✓

Next steps:
1. Review AGENTS.md — especially Key Dependencies and Code Conventions.
   Add any constraints I missed.
2. Review GLOSSARY.md — add terms I didn't find.
3. Use the new-task skill to create your first task under the framework.
4. After the first task completes, the Completion Protocol will populate
   Common Mistakes and ADRs naturally.

Remember: from this point forward, every new feature gets a task doc BEFORE
implementation starts. This is the migration boundary.
```

---

## Key Differences from init-project

| Aspect | init-project | adopt-project |
|--------|-------------|---------------|
| Starting state | Empty directory | Existing codebase with code + tests |
| Information source | User interview (invention) | Codebase extraction (archaeology) |
| Phase label | Always P0 | Detected from maturity (P1-P4+) |
| GLOSSARY | Terms invented for the domain | Terms extracted from type/function names |
| Build commands | From TEMPLATES.md language adapter | From Makefile / build config (verified) |
| Dependencies | Empty table | Populated from go.mod / package.json |
| Common Mistakes | Empty stub | Seeded from FIXME/HACK comments |
| Task docs for past work | N/A | Explicitly forbidden (§10 rule) |
| ADRs for past decisions | N/A | Explicitly forbidden (§10 rule) |
