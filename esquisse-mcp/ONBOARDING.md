# ONBOARDING.md — esquisse-mcp

## Read This First

You are working in `esquisse-mcp`, a small Go MCP stdio server that connects
AI agents using the Esquisse framework to LLMs via the `crush` CLI tool.

Before writing any code, read:
1. [`AGENTS.md`](AGENTS.md) — invariants, conventions, security rules
2. [`GLOSSARY.md`](GLOSSARY.md) — exact term definitions (pool, slug, round, etc.)
3. [`README.md`](README.md) — build, configuration, env vars

---

## Mental Model

```
Crush Agent
    │
    │  MCP stdio
    ▼
esquisse-mcp (this binary)
    │
    ├─► adversarial_review tool
    │       └─► for each round: spawn `crush run --model X` subprocess
    │               └─► plan content via stdin from temp file
    │
    ├─► gate_review tool
    │       └─► scan .adversarial/*.json in project root
    │
    └─► discover_models tool
            └─► spawn `crush models` subprocess, filter, return list
```

---

## Request Lifecycle

### `adversarial_review`

1. Handler built in `newAdversarialHandler` (in `adversarial.go`) at server start.
2. Model pool captured in closure via `buildModelPool()` (`models.go`).
3. Handler called with `adversarialInput{PlanSlug, PlanContent, Rounds}`.
4. `validateSlug` in `state.go` checks `PlanSlug` for traversal sequences.
5. Existing state read from `.adversarial/{slug}.json` (or zero value if absent).
6. `buildRotationOrder(pool, iter)` determines model order for this epoch.
7. `effectiveRounds` caps rounds at `maxRounds`.
8. Round loop: `runOneRound(ctx, model, planContent, tmpDir)` for each round.
   - Plan written to `tmpDir/round-N.txt` (mode 0600)
   - `crush run --model {model}` with stdin=plan file
   - Response text parsed: last occurrence of PASSED/CONDITIONAL/FAILED
   - If model unavailable: try next model in pool; if all exhausted → error
9. `worstVerdict` aggregates per-round verdicts.
10. State written to `.adversarial/{slug}.json` with incremented `Iteration`.
11. Result returned as MCP text with verdict and individual round details.

### `gate_review`

1. `gateInput{ProjectRoot, Strict}` deserialized.
2. `filepath.Glob(.adversarial/*.json)`.
3. Each file: read `last_verdict` field.
4. Any FAILED or missing verdict → `blocked=true`.
5. In strict mode: CONDITIONAL also blocks.

### `discover_models`

1. `crush models` subprocess run with 10s timeout.
2. stdout split by newline, each line filtered:
   - If `ESQUISSE_ALLOWED_PROVIDERS` set: keep only matching prefix lines.
   - If `filter` param provided: substring match.
3. Returns filtered list as JSON array.

---

## Key Files

| File | What to read first |
|---|---|
| `models.go` | `buildModelPool`, `defaultModels`, env var overrides |
| `adversarial.go` | `newAdversarialHandler`, `runAdversarialRounds` |
| `models.go` | `familyInterleaveShuffle`, `buildRotationOrder`, `runOneRound` |
| `state.go` | `ReviewState`, `validateSlug`, `ReadState`, `WriteState` |
| `runner.go` | `RunCrush`, `RunResult` |
| `gate.go` | `newGateHandler`, `gateInput`, `gateOutput` |
| `tools.go` | All `mcp.AddTool` registrations |
| `main.go` | Server startup, Windows guard, `--project-root` flag |

---

## Data Flow Diagram

```
AGENTS.md → adversarial_review
    PlanContent ─────────────────────► temp file (mode 0600)
    PlanSlug ────► validateSlug ──────► .adversarial/{slug}.json
    Rounds ──────► effectiveRounds ──► round loop
                        │
                        ▼
                buildRotationOrder(pool, iter)
                        │
                        ▼
                runOneRound × N rounds
                        │
                        ▼
                worstVerdict(verdicts)
                        │
                        ▼
                WriteState(.adversarial/{slug}.json)
                        │
                        ▼
                MCP result text (verdict + round details)
```

---

## Where to Add Things

| What you want to add | File to edit |
|---|---|
| New MCP tool | `tools.go` (register) + new file or existing handler file |
| New env var for pool | `buildModelPool` in `models.go` |
| New verdict keyword | `worstVerdict` and `runOneRound` parsing in `models.go` |
| New state file field | `ReviewState` in `state.go` + `ReadState`/`WriteState` |
| New subprocess invocation | `runner.go` pattern, new function |
| Security validation | `state.go` (`validateSlug` or new validator) |

---

## Testing Patterns

```go
// Restore runCrushFn after test (always use defer)
orig := runCrushFn
defer func() { runCrushFn = orig }()
runCrushFn = func(...) RunResult { return RunResult{Output: "PASSED"} }

// Deterministic model order
SetRandSource(rand.NewSource(42))
// DO NOT call t.Parallel() in the same test

// Temp project root for state files
dir := t.TempDir()
os.MkdirAll(filepath.Join(dir, ".adversarial"), 0o755)
```

---

## References

- [`AGENTS.md`](AGENTS.md)
- [`GLOSSARY.md`](GLOSSARY.md)
- [`README.md`](README.md)
- [`../AGENTS.md`](../AGENTS.md) — parent esquisse project constitution
- [`../docs/tasks/P2-006-mcp-configurable-model-rotation.md`](../docs/tasks/P2-006-mcp-configurable-model-rotation.md)
