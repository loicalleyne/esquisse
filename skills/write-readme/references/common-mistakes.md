# Common Mistakes — write-readme

| Mistake | What happens | Fix |
|---|---|---|
| Generating code examples from scratch | Hallucinated examples that don't compile | Source ONLY from existing `README.md` or `llms-full.txt`; use `<!-- TODO: add {pattern} example -->` placeholder if none found |
| Inventing performance numbers | Misleading benchmark claims | Use only numbers from `docs/` files or user-supplied; omit the Performance section entirely if no data exists |
| Proceeding when MODULE_PATH and KEY_TYPES are both empty | Skeletal README full of `{placeholder}` text | Halt after Step 1 and report exactly which files were missing |
| Asking for overwrite confirmation in Step 6 | Double-prompt confuses user | Step 5 is the ONLY confirmation; Step 6 writes immediately on approval |
| Moving sensitive-content warning to after writing | API keys published before user sees warning | Surface the warning in Step 5 pre-confirm, never post-write |
| Skipping Step 1b for existing READMEs | Overwrites compliant sections unnecessarily; misses ordering issues | Always run compliance audit when `EXISTING_README = true` |
| Asking all 5 interview questions when updating | Re-asking what's already in the README wastes user time | In update mode, omit any question answered by `EXISTING_VALUE_PROP`, `EXISTING_QUICK_START_PATTERNS`, or `EXISTING_LICENSE` |
| Batching multiple sections into one `replace_string_in_file` | Large context match fails or corrupts adjacent sections | One call per section; match H2 heading + content up to next `\n---\n` or next `\n## ` |
| Treating "first body line" as the value-prop test for bufarrow READMEs | Every compliant README (logo + badges before bold line) classified as ❌ | Use "first `**...**` bold line after any logo/badge HTML blocks" as the criterion |
| Not following the Step 6 operation order for update mode | CUSTOM_SECTIONS insertion anchor may be shifted by prior edits | Always: (1) update existing sections, (2) insert new sections, (3) re-insert CUSTOM_SECTIONS |
| Using `\n## Licen` as anchor without `---` prefix | Partial matches inside section content | Use `\n---\n\n## Licen` — requires the section divider prefix |
| Treating `## Getting Started` or `## Capabilities` as unrelated custom sections | Duplicate feature/quick-start sections in output | Apply H2 aliases table in Step 1b before classification |
| Forcing `## Use cases` and `## Features` on non-library projects | Bloated README for a CLI tool or config repo | Check `PROJECT_TYPE`; apply Step 3 conditionality; ask user in question 6 if not `library` |
| Letting `cmd/` subdirectory override `library` classification | bufarrowlib and similar projects misclassified as `cli-tool`; Use-cases table omitted | Library wins when importable packages exist alongside `cmd/`; ask user only when module is solely `package main` |
| Skipping the Step 1 discovery sub-step for `cmd/` and Dockerfile | `cli-tool` and `service` signals never fire; all projects fall through to `library` or `config` | Add `file_search` for `cmd/` and `Dockerfile` if AGENTS.md doesn't document project layout |
| Drafting `## Features` for a cli-tool because Step 1b put it in SECTIONS_TO_UPDATE | Section added despite PROJECT_TYPE saying omit | Step 3 conditionality removes inapplicable sections from SECTIONS_TO_UPDATE; check both lists before drafting |
| Writing the `description:` of this skill to summarize the workflow | Auto-trigger false-positive or mis-trigger | `description:` must start "Use when..." and name triggering conditions, not steps |
