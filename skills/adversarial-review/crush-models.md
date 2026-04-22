# Adversarial Review: Crush Model Reference

When running in Crush (i.e., `runSubagent` is NOT in your tool list but `agent`
is), use the two-step bash approach below to invoke the reviewer. The rotation
slots map to these models:

| Slot | `iteration % 3` | `--model` flag value |
|------|----------------|---------------------|
| 0 | 0 | `copilot/gpt-4.1` |
| 1 | 1 | `vertexai/claude-opus-4-5-20251101` |
| 2 | 2 | `copilot/gpt-4o` |

These mirror the VS Code rotation (Adversarial-r0 = GPT-4.1, r1 = Claude
Opus, r2 = GPT-4o). The models are deliberately cross-provider to prevent
self-review bias.

Ensure `crush` is in your PATH: verify with `which crush` before running.

Model strings are pinned (not `latest`). Update this file when a model is
deprecated. Deprecation policy: when a model string returns a 404 or
"model not found" error from the provider, update this file and increment
all affected task doc Session Notes.

## Constructing the bash call — SECURITY INVARIANT

Plan content MUST NEVER appear in the shell command line. It must be
written to a file first, then passed via stdin redirect. This prevents
shell injection from plan content containing `$(...)`, backticks, or
single-quote sequences.

**Step 1** — create the temp directory and write plan content using bash:

```bash
mkdir -p .adversarial/tmp
REVIEW_FILE=".adversarial/tmp/esquisse-review-{slug}-$(date +%Y%m%dT%H%M%S).txt"
cat > "$REVIEW_FILE" << 'REVIEW_EOF'
{preamble}

--- PLAN CONTENT ---
{plan_content}
REVIEW_EOF
```

Where `{preamble}` is:
```
You are Adversarial-r{slot}. Apply the 7-attack protocol to the plan below.
Write your report to .adversarial/reports/review-{date}-iter{iteration}-r{round}-{plan-slug}.md
(round=1 for single-dispatch)
Write state to .adversarial/{plan-slug}.json with fields:
  plan_slug, iteration (={next_iteration}), last_model, last_verdict, last_review_date.
Schema: SCHEMAS.md §8. Verdict must be PASSED, CONDITIONAL, or FAILED.
```

**Step 2** — invoke the reviewer with stdin redirect (NO plan content in the command):
```bash
crush run --model {model_string} --quiet < ".adversarial/tmp/esquisse-review-{slug}-{timestamp}.txt"
```

**Step 3** — clean up:
```bash
rm -f ".adversarial/tmp/esquisse-review-{slug}-{timestamp}.txt"
```

**After Step 3** — verify the verdict file was written:

```bash
if [ ! -f ".adversarial/{plan-slug}.json" ]; then
  echo "ERROR: reviewer did not write .adversarial/{plan-slug}.json" >&2
  echo "Check .adversarial/tmp/ for leftover files and retry."
  exit 1
fi
```

## Platform detection

- **VS Code Copilot Chat**: `runSubagent` is in your tool list → use Steps 5/6 (named agent dispatch).
- **Crush**: `runSubagent` is NOT in your tool list; `agent` IS → use the two-step bash approach above.

## Environment prerequisite

The specified model providers must be configured in `~/.config/crush/crush.json`
(or the project-local `crush.json`). If the provider is not configured, the
`crush run` call will fail with "no providers configured".
