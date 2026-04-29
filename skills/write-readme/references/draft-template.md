# Draft Templates — write-readme

Apply each section template exactly as shown. Omit sections excluded by Step 3 conditionality or `OMIT_SECTIONS`.

---

## Opening Block

```markdown
# {Project Name} {Emoji}

<p align="center">
  <img src="assets/{logo-file}" alt="{project}-logo" width="800"/>
</p>
```
*(omit logo block if HAS_LOGO is false)*

```markdown
<p align="center">
  <a href="{badge-target-1}"><img src="{badge-img-1}" alt="{Badge 1}"></a>
  <a href="{badge-target-2}"><img src="{badge-img-2}" alt="{Badge 2}"></a>
</p>
```
*(omit badges block if language has no standard badge URLs — see Badge URL Templates in SKILL.md Quick Reference)*

```markdown
**{One-line value proposition.} {Headline performance metric.}**

{Intro paragraph}: what the library does, what you give it, what you get back,
and the headline metric. ≤ 4 sentences.
```

---

## Use Cases Table

```markdown
---

## Use cases

| Scenario | Why {Project} |
|---|---|
| {scenario 1} | {why} |
| {scenario 2} | {why} |
```

---

## Install

```markdown
---

## Install

```sh
{install command}
```
```

---

## Quick Start (one H3 per named pattern)

```markdown
---

## Quick Start

### {Pattern name — be specific, not just "Example"}

Brief one-sentence framing of when to use this pattern.

```{language}
{complete, runnable example including imports}
```
```

Each code example must:
- Include all necessary imports.
- Show the full happy-path call sequence.
- Use realistic variable names (not `foo`, `bar`).
- Have a `defer` or close/release call where applicable.

**Code example sourcing rule:** Use ONLY verbatim code blocks found in the existing
`README.md` or `llms-full.txt`. Do NOT generate examples from scratch. If neither file
contains a usable block for a named pattern, insert:
`<!-- TODO: add {pattern-name} example — none found in docs -->`
and report the gap to the user in Step 5.

---

## Features Section

One H3 per major feature area. Each subsection must have:
- A one-sentence description.
- A code block **or** a reference table (prefer tables for option sets, type maps, method lists).
- Bold the single most important fact (e.g. performance uplift, key config option).

API method tables — four columns:
```
| Method | Input | Speed | Notes |
```

Option/flag tables — three columns:
```
| Option | Default | Description |
```

---

## Performance Section

```markdown
---

## Performance

### {Test configuration context}

Brief setup: hardware, corpus size, measurement conditions.

| Method | msg/s | ns/msg | allocs/msg |
|---|---|---|---|
| ... | ... | ... | ... |

**Bold the headline result** with a human-readable interpretation.
```

Do NOT invent performance numbers. Use only numbers found in `docs/` or provided by the user.
If no data is available, omit this section entirely.

---

## Development Section

```markdown
---

## Development

### Make targets  (or: Build commands / Scripts)

| Target | Description |
|---|---|
| `{target}` | {one-line description} |
```

---

## Reference Section

```markdown
---

## Reference

- Full API: [{module-path}]({pkg-doc-url}) or `go doc ./...`
- {Sub-library}: [{path}]({path}/README.md)
- Architecture guide: [{doc}]({doc})
- LLM-optimized reference: [llms.txt](llms.txt)
```

---

## Licence Section

```markdown
---

## Licence

{Project} is released under the {License} license. See [{LICENSE-file}]({LICENSE-file})
```
