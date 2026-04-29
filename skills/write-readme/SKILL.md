---
name: write-readme
description: >
  Use when a project needs a new README.md or the existing README needs to be
  updated in the bufarrow style. Triggers on: "write a README", "update the
  README", "generate README", "write a readme in bufarrow style", "create a
  project readme", "write me a README", "update the readme to match bufarrow
  style". DO NOT USE when: documenting a single function or type (use inline
  godoc); writing an ADR (use write-adr); writing a feature spec (use write-spec).
---

# Write-README

## Overview

Produces a README.md in the bufarrow library style: punchy opening with a
measurable value proposition, a use-cases decision table before any API details,
named quick-start patterns, feature tables, concrete performance numbers, and a
curated reference section.

Works for any language. Adapts badge and install sections based on language
detected from the project manifest.

**Announce at start:** "I'm using the write-readme skill."

---

## When to Use

**Use when:**
- The project has no `README.md` yet
- The existing README has no bold value-proposition line
- Use-cases appear after the API docs (wrong order for bufarrow style)
- Quick-start examples are labeled "Example" or "Usage" instead of named patterns
- Performance section contains prose ("fast", "efficient") without benchmark numbers
- User explicitly asks for bufarrow-style README

**Do NOT use when:**
- Documenting a single function or type — write inline godoc instead
- Writing an Architecture Decision Record — use `write-adr`
- Writing a feature specification — use `write-spec`
- Generating documentation for a website (this produces `README.md` only)

---

## Prerequisites

- [ ] The project has at least `AGENTS.md` or a manifest file (`go.mod`,
      `package.json`, `pyproject.toml`, `Cargo.toml`).
- [ ] The user can describe the library/tool in one sentence.

**Tools (VS Code / Crush):** `read_file`/`view`, `file_search`/`glob`,
`grep_search`/`grep`, `create_file`/`write`, `replace_string_in_file`/`edit`

---

## Workflow

### Step 1: Reconnaissance

Read the following files **in parallel** to extract project metadata.
Stop reading once the target field is found — do not read entire files
unless necessary.

| File | Extract |
|---|---|
| `AGENTS.md` | language/runtime, module path (from "Module path:" line), architecture pattern, key types, public API names, performance numbers mentioned |
| `go.mod` / `package.json` / `pyproject.toml` | exact module/package name and version, dependencies |
| `llms.txt` | concise public API surface (types, functions, config) |
| `llms-full.txt` | detailed signatures and docstrings |
| `ONBOARDING.md` | mental model, data-flow description |
| `README.md` (existing) | sections to preserve verbatim; performance tables already written |
| `docs/benchmark-results.md` or `docs/*.txt` | concrete benchmark numbers |
| `assets/` directory listing | check for logo files |

Record findings:
- `MODULE_PATH` — for install command and badge URLs
- `LANGUAGE` — `go` / `python` / `typescript` / `rust` / `c++` / `other`
- `PROJECT_TYPE` — classify as one of:
  - `library` — importable package with a public API (go.mod package name, `pip install`, npm module)
  - `cli-tool` — standalone binary / command-line tool with no importable packages
  - `service` — long-running process / server / daemon
  - `config` — configuration file, template, or schema repo
  - `other` — none of the above

  **Detection (check in order; first rule that fires exclusively wins):**
  1. Does the project have importable packages (non-`main` Go packages, a pip-installable module, an npm package)? → `library`.
     *Note: the presence of a `cmd/` subdirectory alongside importable packages does NOT override this — a library that ships a playground or CLI wrapper is still a `library`.*
  2. Does the module consist solely of `package main` files with no importable packages? → `cli-tool`.
  3. Does a `Dockerfile`, `docker-compose.yml`, or `*.service` file exist at project root? → `service`.
  4. Neither of the above → `config`.

  **Ambiguous case (rules 1 and 2 both appear to fire):** Ask the user:
  ```
  I see both importable packages and a cmd/ directory. Is this project primarily
  an importable library (with a CLI companion), or a standalone CLI tool?
  Reply "library" or "cli-tool".
  ```

  **Discovery sub-step (run after reading AGENTS.md):** If AGENTS.md does not document
  the project layout, use `file_search` to check:
  - `cmd/` directory presence (cli-tool signal)
  - `Dockerfile` or `*.service` at project root (service signal)
- `KEY_TYPES` — primary public types / classes
- `KEY_METHODS` — primary public methods / functions
- `PERF_NUMBERS` — any benchmark results found
- `HAS_LOGO` — true/false (did `assets/` contain a logo image?)
- `HAS_BINDINGS` — true/false (does the project have bindings for other languages?)
- `HAS_SUB_LIBS` — list of standalone sub-libraries (e.g. `proto/pbpath`)
- `HAS_TYPE_MAP` — true/false (is there a type conversion/mapping table in docs?)
- `HAS_BENCHMARKS` — true/false
- `EXISTING_README` — true/false (does `README.md` already exist at project root?)
- `EXISTING_VALUE_PROP` — verbatim bold opening line from existing README if found, else empty
- `EXISTING_QUICK_START_PATTERNS` — list of H3 titles found under any `## Quick Start` section, else empty
- `EXISTING_LICENSE` — license name from existing `## Licen*` section, else empty

**Halt condition:** If both `MODULE_PATH` and `KEY_TYPES` are empty after all reads,
stop and report to the user:
```
I couldn't extract enough project metadata to write a useful README.
Missing: {list the files not found}.
Please ensure at least one of these exists: AGENTS.md (with Module path: line),
go.mod / package.json / pyproject.toml, or llms.txt.
```
Do not proceed to Step 1b or Step 2.

### Step 1b: Compliance Audit (update mode only)

**Run only if `EXISTING_README` is true.** Skip for new READMEs.

**Reuse content already loaded in Step 1** — do not re-read `README.md`.

**H2 name aliases:** Before classifying, map these common aliases to their bufarrow equivalents:

| Alias found in README | Treated as |
|---|---|
| `## Getting Started`, `## Usage` | `## Quick Start` |
| `## Capabilities`, `## What it does` | `## Features` |
| `## Benchmark`, `## Benchmarks` | `## Performance` |
| `## Contributing`, `## Build`, `## Setup` | `## Development` |
| `## License`, `## Licence` | `## Licen*` (both spellings are compliant) |

Classify each bufarrow section as ✅ compliant, ⚠️ present-but-wrong, or ❌ missing:

| Section | Status criteria |
|---|---|
| Bold value-prop line | ✅ = first `**...**` bold line found in the body after any logo/badge HTML blocks is ≤ 2 sentences; ⚠️ = exists but not bold, or > 2 sentences; ❌ = no bold line found before `## Use cases` |
| Use-cases table | ✅ = appears before `## Install`; ⚠️ = exists but after Install/Quick Start; ❌ = absent |
| `## Install` | ✅ = present; ❌ = absent |
| Quick Start (named H3s) | ✅ = all H3 titles are specific patterns; ⚠️ = any titled "Example", "Usage", or a single generic word; ❌ = absent |
| Features section | ✅ = present with ≥2 H3 **or** H4 subsections; ⚠️ = present but only one subsection or no subsections; ❌ = absent |
| Performance (if benchmarks exist) | ✅ = has a numbers table; ⚠️ = prose only; ❌ = absent; N/A = no benchmark data |
| Development / Make targets | ✅ = present; ❌ = absent; N/A = no Makefile/Taskfile |
| `## Reference` | ✅ = present; ❌ = absent |
| `## Licen*` | ✅ = present (either `## License` or `## Licence`); ❌ = absent |

Also record:
- `SECTIONS_TO_UPDATE` — all ⚠️ and ❌ sections
- `SECTIONS_TO_PRESERVE` — all ✅ sections
- `CUSTOM_SECTIONS` — H2 sections present in the existing README that are NOT in the bufarrow template above

Report to the user before Step 2:
```
I've audited the existing README. Here's what I found:

✅ Compliant — will keep unchanged: {SECTIONS_TO_PRESERVE}
⚠️ Needs updating: {list with one-line reason each}
❌ Missing entirely: {list}
📋 Custom sections to preserve verbatim: {CUSTOM_SECTIONS}

I'll ask about the gaps only.
```

### Step 2: User Interview

**For updates (EXISTING_README = true):** Pre-fill from `EXISTING_VALUE_PROP`,
`EXISTING_QUICK_START_PATTERNS`, and `EXISTING_LICENSE`. Only ask about items in
`SECTIONS_TO_UPDATE` or entirely missing. Do NOT re-ask for ✅ compliant items.

Prompt the user **in a single message** (wait for reply before continuing).
Omit numbered questions whose answer was already extracted from the existing README.
**Renumber the visible questions consecutively** (1, 2, 3 … for however many remain — do not leave gaps like "1. … 3. …"):

```
I've scanned the project. Here are the gaps I need your input on:

{ONLY if value-prop missing or non-bold:}
1. **One-line value proposition** — what does {PROJECT} do and for whom?
   Example: "Protobuf → Apache Arrow. Raw bytes in, RecordBatches out. No codegen."
   {If EXISTING_VALUE_PROP found but non-compliant:} I found: "{EXISTING_VALUE_PROP}" — update it or keep it?

{ONLY if no headline perf number found anywhere:}
2. **Headline performance metric** (optional) — the single most impressive number.

{ONLY if use-cases missing or misplaced:}
3. **Top 3-5 use cases** table.

{ONLY if Quick Start missing or patterns named generically:}
4. **Quick-start pattern names** — 2-4 specific titles.
   {If EXISTING_QUICK_START_PATTERNS non-empty:} I found: {list}. Rename, keep, or add?

{ONLY if license unknown:}
5. **License** — which license?

(I'll fill in code examples from existing docs — not generated from scratch.)
```

**For new READMEs:** Ask all five questions with full examples as written above. Also ask:

```
{ONLY if PROJECT_TYPE is NOT 'library':}
6. **Section scope** — bufarrow style includes a Use-cases table, a Features section,
   and (if benchmarks exist) a Performance section. For a {PROJECT_TYPE}, some of these
   may not apply. Which sections would you like to skip?
   (Reply "all" to keep everything, or list sections to omit.)
```

Record the answer as `OMIT_SECTIONS` — a list of section names the user explicitly does not want.
For `PROJECT_TYPE = library`, set `OMIT_SECTIONS = []` (all bufarrow sections apply by default).

### Step 3: Synthesize README Structure

**For update mode (`EXISTING_README = true`):** Only synthesize sections that appear in `SECTIONS_TO_UPDATE` or are identified as missing (❌). Sections in `SECTIONS_TO_PRESERVE` are already compliant — do not re-synthesize or redraft them.

**Step 3 conditionality still applies in update mode.** Before drafting a section from `SECTIONS_TO_UPDATE`, verify it passes the Include-when condition in the table below. Remove it from the synthesis list if its condition is not met (e.g., `## Features` is in `SECTIONS_TO_UPDATE` because Step 1b classified it ❌, but `PROJECT_TYPE = cli-tool` — do not draft it).

**For new READMEs:** Use all criteria below.

**Default values when not set:** If `OMIT_SECTIONS` was never populated (update mode or `PROJECT_TYPE = library`), treat `OMIT_SECTIONS = []`.

Using reconnaissance findings and user answers, determine which sections apply.

**`OMIT_SECTIONS` overrides all rules below** — if the user named a section in Step 2
question 6, skip it regardless of any "Always" or conditional below.

| Section | Include when |
|---|---|
| Logo HTML block | `HAS_LOGO` is true |
| Badges | LANGUAGE is `go` (pkg.go.dev + goreportcard) or `python` (PyPI) or `typescript` (npmjs) |
| Bold value prop line | Always |
| Intro paragraph | Always |
| Language bindings mention | `HAS_BINDINGS` is true |
| `## Use cases` table | `PROJECT_TYPE = library`; OR AGENTS.md / ONBOARDING.md describes ≥3 named integration scenarios with different calling environments or user types |
| `## Install` | Always |
| `## Quick Start` H3 per pattern | Always (1 per user-named pattern, or 1 generic if no patterns named) |
| `## Output modes` | More than one output mode exists |
| `## Features` H3 per major feature | `PROJECT_TYPE = library` or `service`; omit for `cli-tool` and `config` unless KEY_TYPES ≥ 3 |
| `## Performance` | `HAS_BENCHMARKS` is true OR user provided a perf number |
| Language bindings section | `HAS_BINDINGS` is true |
| Sub-library section | `HAS_SUB_LIBS` is non-empty |
| Type mapping table | `HAS_TYPE_MAP` is true |
| `## Development` | Makefile / Taskfile / scripts exist |
| `## Reference` | Always |
| `## Licence` | Always |

### Step 4: Draft

Read [`references/draft-template.md`](references/draft-template.md) for all section templates.
Apply them exactly. Key rules (also in draft-template.md):

- Omit logo block if `HAS_LOGO` is false.
- Omit badges block if language has no standard badge URLs.
- **Code example sourcing rule:** verbatim from existing `README.md` or `llms-full.txt` only.
  If no usable block found for a pattern, insert `<!-- TODO: add {pattern-name} example — none found in docs -->`
  and report the gap to the user in Step 5.
- Do NOT invent performance numbers. Omit Performance section if no data exists.

### Step 5: Present and Confirm

**For updates (EXISTING_README = true):** Open with the audit summary so the user
knows exactly what is changing before seeing the draft:

```
Audit summary:
✅ Keeping unchanged ({count} sections): {SECTIONS_TO_PRESERVE}
📝 Updating ({count} sections): {SECTIONS_TO_UPDATE with one-line reason each}
➕ Adding ({count} sections): {missing sections}
📋 Preserving custom sections: {CUSTOM_SECTIONS}
```

Then present the full draft (or only the changed sections for large READMEs — defined as > 6 H2 sections or > 300 lines).

Ask:

```
Before I write to disk:

1. **Code examples** — verbatim from docs only. Patterns marked
   `<!-- TODO: add example -->` need you to supply the code.
   Verify examples compile/run against current source.
2. Any sections to add, remove, or reorder?
3. Any performance numbers to correct or add?
{IF README.md already exists:}
4. Reply "yes replace" to replace the file entirely, or "update sections" to
   patch only the non-compliant sections (✅ sections stay untouched).

Also: I extracted code from docs — scan for sensitive content (API keys, tokens,
internal endpoints) before confirming.

Reply "looks good" / "yes replace" / "update sections", or give me specific changes.
```

**Do not write to disk until the user confirms.**

> **Note:** Step 5 approval is the ONLY confirmation. Step 6 writes immediately.

### Step 6: Write

On user confirmation:

- **New README** or user said "yes replace":
  `create_file` at `README.md` in the project root.

- **User said "update sections"** (EXISTING_README = true):
  Apply one `replace_string_in_file` call per section in `SECTIONS_TO_UPDATE` plus
  each missing section. Rules:

  **Operation order matters — follow this sequence:**
  1. Update/replace each section in `SECTIONS_TO_UPDATE` (one call each, in document order).
  2. Insert each newly added (❌) section immediately before `\n---\n\n## Licen`.
  3. After all section updates are complete, re-insert `CUSTOM_SECTIONS` verbatim
     immediately before `\n---\n\n## Licen` using one `replace_string_in_file` call per
     custom section (in their original relative order).

  **Match target per update call:** the H2 heading line (e.g. `## Quick Start`) and all
  content up to the **first** of:
  a. The next `\n---\n` — preferred; include it in the replacement to preserve the divider.
  b. The next `\n## ` heading line — fallback if `---` divider not present.
  c. End of file — last-resort fallback.

  **If none of these match:** report the failure explicitly to the user:
  ```
  Could not locate "## {SectionName}" in the README — the section may use a different
  name or format. Please update it manually.
  ```
  Stop and do not proceed to the next section replacement after a failure.

  - **Sections in `SECTIONS_TO_PRESERVE`:** do not touch.
  - **H2 anchor for CUSTOM_SECTIONS and new sections:** use `\n---\n\n## Licen` (matches
    both `## License` and `## Licence`, and requires the `---` divider to avoid partial
    matches inside section content).
  - Do NOT batch multiple sections into a single `replace_string_in_file` call.

After writing:
- Report the sections written.
- Note any sections omitted and why.
- Remind the user: "Verify all code examples compile/run against current source —
  examples were extracted from docs, not run."

---

## Quick Reference

### Style Rules

| Rule | Correct | Wrong |
|---|---|---|
| Value prop | Bold, ≤ 2 sentences, has a measurable outcome | Verbose marketing paragraph |
| Use cases table | Appears **before** Install | Buried after API reference |
| Quick start H3 titles | Named patterns: "Raw bytes → Arrow" | Generic: "Example", "Usage" |
| Performance numbers | From benchmarks, attributed | Invented or vague ("fast") |
| Tables | Use for ≥ 3 options/methods/types | Prose lists |
| Section dividers | `---` between every H2 | No dividers |
| Code examples | Complete with imports + defer | Snippet fragments |
| Performance section | Concrete numbers or omitted | Prose without numbers |

---

### Decision Points

| Situation | Action |
|---|---|
| No benchmark data anywhere | Ask user for numbers; if none available, omit Performance section |
| No assets/ logo | Skip logo HTML block; do not invent a placeholder |
| Existing README has unique sections not in template | Preserve them verbatim after `## Reference` |
| User wants a language not covered by badges | Skip badges block entirely |
| Module path not found in go.mod/manifest | Ask the user before writing the install command |
| `PROJECT_TYPE = cli-tool` | Omit `## Use cases` and `## Features` unless KEY_TYPES ≥ 3; ask user in Step 2 question 6 |
| `PROJECT_TYPE = service` | Omit `## Use cases` unless ≥3 named integration scenarios in AGENTS.md/ONBOARDING.md; `## Features` included |
| `PROJECT_TYPE = config` | Omit `## Use cases`, `## Features`, `## Performance`; focus on Install + Quick Start + Reference |
| Library ships `cmd/` subdirectory | Classify as `library` — the `cmd/` is a companion; do NOT let it trigger `cli-tool` classification |
| Service with no Dockerfile in repo | `service` detection fires only on Dockerfile/docker-compose/`*.service`; if absent, fall through to `library` or `config` |
| User sets `OMIT_SECTIONS` | Override all "Always" and conditional rules; skip named sections unconditionally |

### Badge URL Templates

| Language | Badge 1 (target → img) | Badge 2 (target → img) |
|---|---|---|
| Go | `https://pkg.go.dev/{MODULE_PATH}` → `https://pkg.go.dev/badge/{MODULE_PATH}.svg` | `https://goreportcard.com/report/{MODULE_PATH}` → `https://goreportcard.com/badge/{MODULE_PATH}` |
| Python | `https://pypi.org/project/{PACKAGE}/` → `https://img.shields.io/pypi/v/{PACKAGE}.svg` | `https://pypi.org/project/{PACKAGE}/` → `https://img.shields.io/pypi/pyversions/{PACKAGE}.svg` |
| TypeScript | `https://www.npmjs.com/package/{PACKAGE}` → `https://img.shields.io/npm/v/{PACKAGE}.svg` | — |
| Rust | `https://crates.io/crates/{PACKAGE}` → `https://img.shields.io/crates/v/{PACKAGE}.svg` | — |

---

## Common Mistakes

Read [`references/common-mistakes.md`](references/common-mistakes.md) for the full anti-pattern catalog.

Top 3 (memorize; read the file for the rest):
- **Generating code examples** — verbatim from docs only; use `<!-- TODO -->` placeholder if none found.
- **Wrong value-prop criterion** — first `**...**` bold line AFTER logo/badge HTML blocks.
- **cmd/ overrides library** — it does not; library wins when importable packages exist.

*Last updated: 2026-04-27 (iter-2: C1 value-prop criterion, M1–M5 update-mode machinery; iter-3: PROJECT_TYPE signal + tiebreaker, OMIT_SECTIONS default, discovery sub-step, deployment-scenarios definition, Step 3 conditionality in update mode)*
