# P2-007: Planning Artifact Schema and Protocol Definition

## Status
<!-- One of: Draft | Ready | In Progress | In Review | Done | Blocked -->
Status: Done
Depends on: none
Blocks: P2-008 (implement-task skill must reference the artifact loading step defined here), P2-009 (MCP tool must produce artifacts matching the schema defined here)

## Summary
Defines the Planning Artifact document type: a structured, immutable research output
produced by EsquissePlan during planning that externalises condensed facts about
external libraries, APIs, or integration patterns. Artifacts live in `docs/artifacts/`,
are shared across multiple tasks by reference, and are loaded by the implementor agent
during task startup to prevent hallucinations on integration points not present in
training data. This task codifies the schema, updates the document hierarchy, and
wires artifact production into the EsquissePlan and FRAMEWORK.md protocols.

## Problem
EsquissePlan has no persistence mechanism for external research. Findings about library
API surfaces, SDK constraints, and integration patterns are either (a) inlined into task
Specification sections — which bloats them past the 1500-token cap and triggers
agent coherence loss — or (b) lost at planning time, forcing the implementor agent to
re-research (or hallucinate). Models like GPT-4o and Gemini 2.5 hallucinate plausible but
incorrect API signatures and parameter names when the correct ones are absent from context.

## Solution
Introduce a new document type — the Planning Artifact — with a defined schema in
`SCHEMAS.md §10`, a production protocol in `EsquissePlan.agent.md`, and a consumption
step in `FRAMEWORK.md`'s Per-Task Startup Protocol. The task schema (§4) gains an
optional `## Planning Artifacts` section (omit when empty) that links to referenced
artifacts and tells the implementor agent exactly which sections to load.

## Scope

### In Scope
- New `docs/artifacts/` directory entry in `FRAMEWORK.md §3` directory layout
- New `SCHEMAS.md §10`: Planning Artifact full schema with naming convention, required
  sections (Summary, API Surface / Key Facts, Constraints, Anti-Patterns, Minimal Examples),
  immutability rule, threshold rule (when to produce vs. inline), and `Referenced by:` field
- `SCHEMAS.md §4` task schema update: add optional `## Planning Artifacts` table section
  with "What to read from it" column; omit-when-empty rule documented
- `FRAMEWORK.md §2` Document Hierarchy: add artifacts to the read-order table
- `FRAMEWORK.md §3` Directory Layout: add `docs/artifacts/` with annotation
- `FRAMEWORK.md §9` Per-Task Startup Protocol: add artifact loading as step 3b
  (after reading the task doc, before establishing baseline)
- `EsquissePlan.agent.md` Step 2: add artifact research sub-step between "Map the work"
  and "Write task documents" — identify external integrations requiring research,
  produce artifact files, link from task Specification
- `AGENTS.md` (esquisse): add `docs/artifacts/` to Repository Layout

### Out of Scope
- `scripts/init.sh` and `scripts/upgrade.sh` directory scaffold (P2-008)
- `implement-task/SKILL.md` artifact loading step (P2-008)
- `esquisse-mcp` `write_planning_artifact` tool (P2-009)
- Any changes to gate-check.sh, gate-review.sh, or adversarial review infrastructure
- Artifact validation tooling (not needed; EsquissePlan is the sole author)

## Prerequisites
- [ ] P2-006 is done (confirmed — Status: Done)

## Specification

### Artifact File Naming Convention
```
docs/artifacts/{YYYY-MM-DD}-{slug}.md
```
- `{YYYY-MM-DD}` — UTC date the artifact was produced (signals freshness)
- `{slug}` — kebab-case topic identifier, e.g. `franz-go-consumer-api`, `arrow-go-v18-record-batch`
- No phase prefix: artifacts span phase boundaries

### Planning Artifact Schema (§10 content)
````markdown
# Artifact: {Title}

**Primary Source:** {top-level resource: URL | library module path | "AST analysis of {package}"}
**Date:** {YYYY-MM-DD}
**Produced by:** EsquissePlan
**Referenced by:** [P{n}-{nnn}-{slug}](../tasks/P{n}-{nnn}-{slug}.md), [P{n}-{nnn}-{slug}](…)

---

## Summary
2-3 sentences. What this artifact covers and why it matters.
The implementor agent reads this first to decide if the full artifact is relevant.

## API Surface / Key Facts
Each row is a verified, individually sourced fact. **A fact with no traceable source
must not be included** — omit rather than infer from training data.

Use a three-column table:

| Symbol / Field | Exact Signature or Value | Source |
|---|---|---|
| `NewClient` | `func NewClient(opts ...Opt) (*Client, error)` | `go doc github.com/twmb/franz-go/pkg/kgo.NewClient` |
| `ConsumeRecords` | `func (cl *Client) ConsumeRecords(ctx context.Context, maxPollRecords int, topics ...string) ([]*Record, error)` | `go doc github.com/twmb/franz-go/pkg/kgo.Client` |

For non-function facts (config fields, constants, enum values), the Source column
should be a file path + line number or URL anchor where the value was observed.

## Constraints
Hard limits the implementor must respect. Version requirements, thread-safety rules, quota limits.
Format: `MUST {rule}` or `MUST NOT {rule}`, followed by a parenthetical source.
Example: `MUST NOT share a Client across goroutines (source: go doc kgo.Client — "not safe for concurrent use")`

A constraint with no traceable source must not be included.

## Anti-Patterns
What the implementor must not do, with the correct alternative.
Format mirrors AGENTS.md Common Mistakes: Wrong / Right / Why.

## Minimal Examples
Copy-paste–correct code snippets for the critical integration points only.
Not tutorials. Only the exact calls the implementor needs to make.
Omit this section if no code examples are necessary.
````

**Grounding rule (applies to all sections):** EsquissePlan must perform the retrieval
(run `go doc`, fetch the URL, read the file) before writing each fact. A fact that
cannot be attributed to a specific retrieval must be omitted. This is the mechanism
that prevents artifact hallucination — generation from training data is indistinguishable
from retrieval unless the source is named.

**Platform-consistency rule:** `SCHEMAS.md §10` is the single source of truth for artifact
format. Whether EsquissePlan writes the artifact via `create_file` (VS Code / Crush),
the `write_planning_artifact` MCP tool, or any future mechanism, the output file must
produce identical section headers, metadata fields, and table structure. The tool
enforces this mechanically; the skill and agent instructions enforce it behaviourally.
The implementor agent's `read_file` call is the same in all cases — format consistency
is what makes artifacts portable across platforms.

### Threshold Rule (inline vs. artifact)
| Condition | Action |
|---|---|
| External research needed by ≥ 2 tasks | Produce artifact in `docs/artifacts/` |
| Research would exceed ~400 tokens in Specification | Produce artifact |
| Single-task, ≤ 400 tokens, no sharing | Inline in Specification — no artifact |

### Immutability Rule
Artifacts are immutable once written. If research becomes stale, EsquissePlan
creates a new artifact with a new date and slug and updates `Referenced by:` in
the new file. The old artifact is not deleted — task docs that reference it by path
remain valid.

### Task Schema §4 Addition (optional section, omit when empty)
```markdown
## Planning Artifacts
| Artifact | What to read from it |
|----------|---------------------|
| [docs/artifacts/2026-04-21-franz-go-consumer.md](…) | API Surface, Anti-Patterns |
| [docs/artifacts/2026-04-19-arrow-go-v18.md](…)      | Constraints only |
```
The "What to read from it" column is a context-budget hint for the implementor.

### EsquissePlan Step 2 Addition (after "Map the work", before "Write task documents")
New sub-step to insert:
```
2b. Produce Planning Artifacts (when applicable)

For each external library or integration point needed by ≥ 2 tasks, or whose
API surface would exceed ~400 tokens if inlined:
1. Research the library using available tools (read go.mod, run `go doc {pkg}`,
   fetch relevant documentation, read source). Perform actual retrieval —
   do not rely on training data.
2. Distill findings into a Planning Artifact at
   docs/artifacts/{YYYY-MM-DD}-{slug}.md following SCHEMAS.md §10 exactly:
   - Three-column API Surface table (Symbol | Exact Signature or Value | Source)
   - Constraints as MUST/MUST NOT with parenthetical source
   - `Referenced by:` as relative markdown links: [P{n}-{nnn}-{slug}](../tasks/P{n}-{nnn}-{slug}.md)
   When using the `write_planning_artifact` MCP tool, pass `referenced_by` as a list
   of workspace-relative task paths, e.g. `["docs/tasks/P2-007-foo.md"]` — NOT bare
   IDs like `"P2-007"`. The tool constructs the markdown link from the path.
   When writing the artifact directly via `create_file` (VS Code / Crush), construct
   the `Referenced by:` line manually using the same format.
3. Record the artifact path — link it from the `## Planning Artifacts` section
   of each task that needs it, with a "What to read from it" note.

Do not produce artifacts for internal project code. AGENTS.md, GLOSSARY.md,
llms.txt, and llms-full.txt cover internal surfaces. Artifacts are for
external dependencies and integrations only.
```

### FRAMEWORK.md Per-Task Startup Protocol Addition (step 3b)
Insert step 3b inside the `## Task Startup Protocol` code block, between step 3 and step 4.
Exact replacement in `FRAMEWORK.md`:

**Find this text:**
```
3. Read the task doc (P{n}-{nnn}-{slug}.md) completely.
4. Read the "In Scope" and "Out of Scope" lists carefully.
```
**Replace with:**
```
3. Read the task doc (P{n}-{nnn}-{slug}.md) completely.
3b. Load Planning Artifacts (if the task doc has a `## Planning Artifacts` section):
    - For each listed artifact, read only the sections named in "What to read from it".
    - If the artifact file is missing, note in Session Notes and do NOT reconstruct from training data.
    - If no `## Planning Artifacts` section is present, skip this step entirely.
4. Read the "In Scope" and "Out of Scope" lists carefully.
```

Note: The same step 3b text (adapted) is also added to `skills/implement-task/SKILL.md` by P2-008.

### FRAMEWORK.md §2 Document Hierarchy Addition
Insert a new row after `7. docs/tasks/` in the numbered list. Exact replacement in `FRAMEWORK.md`:

**Find this text:**
```
7. docs/tasks/        ← WHAT to do next (task plans)
8. skills/            ← HOW to work (reusable workflows)
```
**Replace with:**
```
7. docs/tasks/        ← WHAT to do next (task plans)
7.5. docs/artifacts/  ← Condensed external research (load referenced artifacts only)
8. skills/            ← HOW to work (reusable workflows)
```

### FRAMEWORK.md §3 Directory Layout Addition
Insert `docs/artifacts/` after the `docs/tasks/` block in the ASCII tree. Exact replacement in `FRAMEWORK.md`:

**Find this text:**
```
│   ├── tasks/                     # Granular task plans
│   │   └── P{phase}-{nnn}-{slug}.md
│   ├── planning/
```
**Replace with:**
```
│   ├── tasks/                     # Granular task plans
│   │   └── P{phase}-{nnn}-{slug}.md
│   ├── artifacts/                 # Condensed external research (EsquissePlan-produced)
│   │   └── {YYYY-MM-DD}-{slug}.md
│   ├── planning/
```

### SCHEMAS.md §4 Task Schema Addition
Insert the optional `## Planning Artifacts` section between `## Files` and `## Design Principles` in the §4 task schema template. Exact replacement in `SCHEMAS.md` (inside the ```` ````markdown ```` block under §4):

**Find this text:**
```
## Files
| File | Action | Description |
|------|--------|-------------|
| `{path}` | Create | {what it contains} |
| `{path}` | Modify | {what changes} |
| `{path}` | Delete | {why} |
| `gen/{path}` | Generated | Run `{command}` to regenerate |

## Design Principles
```
**Replace with:**
```
## Files
| File | Action | Description |
|------|--------|-------------|
| `{path}` | Create | {what it contains} |
| `{path}` | Modify | {what changes} |
| `{path}` | Delete | {why} |
| `gen/{path}` | Generated | Run `{command}` to regenerate |

## Planning Artifacts
<!-- Omit this section entirely when no external research artifacts are needed. -->
| Artifact | What to read from it |
|----------|---------------------|
| [docs/artifacts/{YYYY-MM-DD}-{slug}.md](../artifacts/{YYYY-MM-DD}-{slug}.md) | API Surface, Anti-Patterns |

## Design Principles
```

### AGENTS.md Repository Layout Addition
Insert `docs/artifacts/` after `docs/tasks/` in the ASCII tree. Exact replacement in `AGENTS.md`:

**Find this text:**
```
│   └── tasks/                    ← Granular task plans (P{n}-{nnn}-{slug}.md)
│
├── scripts/
```
**Replace with:**
```
│   ├── tasks/                    ← Granular task plans (P{n}-{nnn}-{slug}.md)
│   └── artifacts/                ← Condensed external research (EsquissePlan-produced, gitignore-optional)
│
├── scripts/
```

### Known Risks / Failure Modes
| Risk | Mitigation |
|---|---|
| EsquissePlan produces artifact for internal code | Schema says "external dependencies only"; add explicit rule to §10 |
| Implementor reads full artifact ignoring budget hint | "What to read from it" column is a budget hint, not a permission list; document in §10 |
| Stale artifact causes hallucination | Date-stamp signals freshness; per-claim Source column makes each fact independently verifiable by the implementor |
| Artifact grows too large | §10 sizing guidance: keep each section ≤ 500 tokens; split by sub-domain if needed |
| Schema §10 drifts from implement-task SKILL.md step | Add co-update rule to §10: any change to the required section list in §10 MUST be reflected in `skills/implement-task/SKILL.md` Step 2b, and vice versa |
| `docs/artifacts/` absent in manually created repos | Document in FRAMEWORK.md §3 and §10 that the `write_planning_artifact` MCP tool (P2-009) uses `os.MkdirAll` as a final guard — scripts are preferred but not required |
| Stale artifact with no replacement → implementor loads wrong facts | Immutability + new date-stamped file is the update protocol. §10 must state: when research is superseded, create a new artifact with a new date; update `Referenced by:` in the new file; link the new artifact from the task doc instead of the old one. The old file is NOT deleted — existing task references remain valid. |

## Acceptance Criteria
This task modifies documentation only — no code, no automated tests.
Observable verification:

| Check | Command / Method |
|---|---|
| §10 exists in SCHEMAS.md | `grep -n "## 10\. Planning Artifact" SCHEMAS.md` returns a result |
| §4 task schema contains `## Planning Artifacts` | `grep -n "Planning Artifacts" SCHEMAS.md` returns a result in §4 |
| `docs/artifacts/` present in FRAMEWORK.md §3 layout | `grep -n "artifacts" FRAMEWORK.md` returns a result in the directory listing |
| Artifact production step in EsquissePlan.agent.md | `grep -n "Planning Artifact" .github/agents/EsquissePlan.agent.md` returns a result |
| Per-Task Startup Protocol step 3b | `grep -n "Planning Artifact" FRAMEWORK.md` returns result near Per-Task Startup Protocol |
| `docs/artifacts/` in AGENTS.md Repository Layout | `grep -n "artifacts" AGENTS.md` returns a result |

All pre-existing `gate-check.sh` checks must pass after edits.

## Files
| File | Action | Description |
|------|--------|-------------|
| `SCHEMAS.md` | Modify | Add §10 Planning Artifact schema; add optional `## Planning Artifacts` section to §4 task schema |
| `FRAMEWORK.md` | Modify | Add `docs/artifacts/` to §3 directory layout; add to §2 document hierarchy; add step 3b to Per-Task Startup Protocol; add planning artifact sub-step to §9 EsquissePlan guidance |
| `.github/agents/EsquissePlan.agent.md` | Modify | Add Step 2b: Produce Planning Artifacts, between Map the work and Write task documents |
| `AGENTS.md` | Modify | Add `docs/artifacts/` entry to Repository Layout |

## Design Principles
1. **Artifacts are for external surfaces only.** Internal project code is documented in AGENTS.md, llms.txt, and task Specifications. Artifacts never duplicate internal docs.
2. **Immutable and date-stamped.** Once written, an artifact is not edited. Its date signals freshness to both agents and humans. Update protocol: create a new date-stamped artifact, link the new one from the task doc, leave the old file in place.
3. **Context-budget hints are mandatory.** Every `## Planning Artifacts` entry must have a "What to read from it" note. An empty note is an error — it forces the implementor to read the whole file.
4. **Threshold rule prevents over-engineering.** Single-task, small research stays inline. Only shared or large external research gets its own file.
5. **Schema and skill are co-updated.** SCHEMAS.md §10 and `skills/implement-task/SKILL.md` Step 2b must name the same set of sections. Any change to either must be accompanied by a change to the other in the same commit.

## Testing Strategy
- Manual check: re-read SCHEMAS.md and FRAMEWORK.md after editing to verify internal consistency (§10 schema must match the EsquissePlan.agent.md step instructions).
- Manual check: `grep -rn "docs/artifacts" AGENTS.md FRAMEWORK.md SCHEMAS.md .github/agents/EsquissePlan.agent.md` — all four files must contain at least one match.
- Verify no gate-check.sh failures: `bash scripts/gate-check.sh 2`

## Session Notes
<!-- Append-only. Never overwrite. -->
<!-- 2026-04-21 — EsquissePlan — Created. Design confirmed by user. -->
<!-- 2026-04-21 — EsquissePlan — Revised after Adversarial-r0 CONDITIONAL verdict (iter 0). Added co-update rule (M1), directory-guard clarification (M2), artifact update protocol (M5) to Known Risks and Design Principles. -->
<!-- 2026-04-21 — EsquissePlan — Added per-claim source grounding. Renamed "Source:" header to "Primary Source:". API Surface table gains mandatory Source column. Constraints gain parenthetical source. Added grounding rule: facts without traceable source must be omitted. -->
<!-- 2026-04-21 — EsquissePlan — Added platform-consistency rule: SCHEMAS.md §10 is the single source of truth for artifact format across VS Code, Crush, and MCP. Fixed Referenced by example to show full slug. Updated EsquissePlan Step 2b to specify full workspace-relative task paths for MCP vs. manual create_file for VS Code/Crush. -->
<!-- 2026-04-21 — ImplementerAgent — Completed. All 5 file insertions applied; §10 appended to SCHEMAS.md; EsquissePlan Step 2b inserted. -->
