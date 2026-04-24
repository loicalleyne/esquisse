# esquisse-mcp

A lightweight MCP stdio server for the [Esquisse](https://github.com/loicalleyne/esquisse) framework.

Exposes three tools:
- `adversarial_review` — run N rounds of cross-model adversarial review via `crush run --model`, with family-interleaved randomized model order and enterprise-policy fallback
- `gate_review` — check all `.adversarial/` verdicts before completing a planning session
- `write_planning_artifact` — write a Planning Artifact file to `docs/artifacts/{date}-{slug}.md`

## Requirements

- Linux, macOS, or WSL (Windows is not supported)
- Go 1.25+
- [`crush`](https://github.com/charmbracelet/crush) installed and in PATH
- LLM providers configured in `~/.config/crush/crush.json`

## Build

```sh
cd esquisse-mcp
go build -o esquisse-mcp .
```

## Install

```sh
go install github.com/loicalleyne/esquisse-mcp@latest
```

## Crush Configuration

Add to your `crush.json` (project-local or `~/.config/crush/crush.json`):

```json
{
  "mcp": {
    "esquisse": {
      "type": "stdio",
      "command": "esquisse-mcp",
      "args": ["--project-root", "."],
      "env": {
        "ESQUISSE_MODELS": "copilot/claude-sonnet-4.6,copilot/gpt-4.1,copilot/gpt-4o,gemini/gemini-2.0-flash,vertexai/gemini-2.0-flash"
      }
    }
  }
}
```

## Model Pool

The pool defaults to 5 reasoning-capable models. Model order is **randomized per call** with family-interleaving (copilot/gemini/vertexai alternated) and a no-consecutive-same-model constraint. Every 5 rounds the order re-randomizes.

Default pool (used when `ESQUISSE_MODELS` is unset):

| Model | Family |
|-------|--------|
| `copilot/claude-sonnet-4.6` | copilot |
| `gemini/gemini-3.1-pro-preview` | gemini |
| `copilot/gpt-4.1` | copilot |
| `vertexai/gemini-3.1-pro-preview` | vertexai |
| `copilot/gpt-4o` | copilot |

Set `ESQUISSE_MODELS` to override the entire pool:

```
ESQUISSE_MODELS=copilot/claude-sonnet-4.6,copilot/gpt-4.1,gemini/gemini-2.0-flash
```

Rules: comma-separated `provider/model` entries; whitespace trimmed; invalid entries skipped (logged); all-invalid falls back to defaults.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ESQUISSE_MODELS` | `""` (use defaults) | Comma-separated `provider/model` list. Overrides the entire pool. Invalid entries are skipped with a log warning; all-invalid falls back to the built-in defaults. |

## Multi-Round Reviews

`adversarial_review` accepts an optional `rounds` parameter (default: 1, max: 50). Each call runs that many consecutive review rounds, aggregates the verdicts (worst-case), and writes the state file once after all rounds complete.

## Caller-Model Independence

Pass `exclude_model` with your own full model ID to ensure adversarial reviewers are drawn from a different model than your implementing agent.

To find your model ID, call the `crush_info` tool and parse the `large = {model} ({provider})` line:
- Extract the model name: text between `large = ` and the first ` (` → e.g. `claude-sonnet-4.6`
- Extract the provider: text inside `()` → e.g. `copilot`
- Concatenate as `{provider}/{model}` → e.g. `copilot/claude-sonnet-4.6`

For example:
- `large = claude-sonnet-4.6 (copilot)` → model ID is `copilot/claude-sonnet-4.6`
- `large = gemini-1.5-pro (gemini)` → model ID is `gemini/gemini-1.5-pro`

```
adversarial_review(
  plan_slug: "my-plan",
  plan_content: "...",
  exclude_model: "copilot/claude-sonnet-4.6"
)
```

If `exclude_model` is empty, malformed (characters other than alphanumeric, `-`, `_`, `.`, `/`), or would empty the pool entirely, it is silently ignored (no-op — fail-open to avoid blocking the review).

## Enterprise Fallback

If a GitHub Copilot Enterprise policy disables a model, `esquisse-mcp` automatically tries the next model in the pool for that round rather than failing the review. If all pool models are unavailable, the tool returns `IsError=true` with an actionable error message.

## Security

Plan content is never passed as a shell argument. It is written to a temp file (mode 0600) inside a per-call `os.MkdirTemp` directory and passed to `crush run` via stdin redirect. This prevents shell injection from plan content containing `$(...)`, backticks, or quote sequences. The temp directory is removed after all rounds complete.

State file path traversal is prevented by `validateSlug()` in `state.go`, which rejects any slug containing `/`, `\\`, or path traversal sequences before constructing the file path.
