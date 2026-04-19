# esquisse-mcp

A lightweight MCP stdio server for the [Esquisse](https://github.com/loicalleyne/esquisse) framework.

Exposes three tools:
- `adversarial_review` — run N rounds of cross-model adversarial review via `crush run --model`, with family-interleaved randomized model order and enterprise-policy fallback
- `gate_review` — check all `.adversarial/` verdicts before completing a planning session
- `discover_models` — list available `provider/model` strings from `crush models`, with optional provider and substring filtering

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
        "ESQUISSE_ALLOWED_PROVIDERS": "copilot,gemini,vertexai",
        "ESQUISSE_MODEL_SLOT0": "copilot/claude-sonnet-4.6",
        "ESQUISSE_MODEL_SLOT1": "gemini/gemini-3.1-pro-preview",
        "ESQUISSE_MODEL_SLOT2": "copilot/gpt-4.1",
        "ESQUISSE_MODEL_SLOT3": "vertexai/gemini-3.1-pro-preview",
        "ESQUISSE_MODEL_SLOT4": "copilot/gpt-4o"
      }
    }
  }
}
```

## Model Pool

The pool contains 5 slots defaulting to reasoning-capable models. Model order is **randomized per call** with family-interleaving (copilot/gemini/vertexai alternated) and a no-consecutive-same-model constraint. Every 5 rounds the order re-randomizes.

| Pool entry | Default model | Override env var | Family |
|------------|---------------|------------------|--------|
| 0 | `copilot/claude-sonnet-4.6` | `ESQUISSE_MODEL_SLOT0` | copilot |
| 1 | `gemini/gemini-3.1-pro-preview` | `ESQUISSE_MODEL_SLOT1` | gemini |
| 2 | `copilot/gpt-4.1` | `ESQUISSE_MODEL_SLOT2` | copilot |
| 3 | `vertexai/gemini-3.1-pro-preview` | `ESQUISSE_MODEL_SLOT3` | vertexai |
| 4 | `copilot/gpt-4o` | `ESQUISSE_MODEL_SLOT4` | copilot |

> **Migration note:** The previous defaults (`openai/gpt-4.1`, `anthropic/claude-opus-4-5-20251101`, `openai/gpt-4o`) have been replaced. If you had `ESQUISSE_MODEL_SLOT0/1/2` set to OpenAI/Anthropic models, update your `crush.json` to use the new defaults above, or keep your overrides if you have valid licences for those providers.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ESQUISSE_MODEL_SLOT0`–`ESQUISSE_MODEL_SLOT4` | see pool table | Override a pool slot. Format: `provider/model`. Invalid format logs a warning and falls back to the default. |
| `ESQUISSE_ALLOWED_PROVIDERS` | `""` (all allowed) | Comma-separated provider IDs (case-sensitive lowercase). When set, slots whose provider prefix is not in this list are excluded from the pool. Provider IDs are the prefix before `/` in each `provider/model` string — use the `discover_models` tool (or `crush models`) to see which providers are available in your environment. |
| `ESQUISSE_POOL_FALLBACK_STRICT` | `""` (fail-open) | Set to `"1"` to return an error instead of falling back to the full default pool when all slots are filtered by `ESQUISSE_ALLOWED_PROVIDERS`. |

## Multi-Round Reviews

`adversarial_review` accepts an optional `rounds` parameter (default: 5, max: 50). Each call runs that many consecutive review rounds, aggregates the verdicts (worst-case), and writes the state file once after all rounds complete.

## Enterprise Fallback

If a GitHub Copilot Enterprise policy disables a model, `esquisse-mcp` automatically tries the next model in the pool for that round rather than failing the review. If all pool models are unavailable, the tool returns `IsError=true` with an actionable error message.

## Security

Plan content is never passed as a shell argument. It is written to a temp file (mode 0600) inside a per-call `os.MkdirTemp` directory and passed to `crush run` via stdin redirect. This prevents shell injection from plan content containing `$(...)`, backticks, or quote sequences. The temp directory is removed after all rounds complete.

State file path traversal is prevented by `validateSlug()` in `state.go`, which rejects any slug containing `/`, `\\`, or path traversal sequences before constructing the file path.
