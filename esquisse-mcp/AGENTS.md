# AGENTS.md — esquisse-mcp

## The Most Important Rules

1. **Tests define correctness.** Fix code to match tests, never the reverse.

2. **No global state.** The model pool is captured in the `newAdversarialHandler`
   closure at server startup — not in a package-level variable. `runCrushFn` is
   a package-level var but is **production-immutable** (written only by
   `SetRandSource` / the analogous test-only setter, never in production code).

3. **All errors returned, never panicked.** `mcpErr` wraps errors as MCP
   `IsError=true` responses. Hard panics are reserved only for programmer-contract
   violations in constructors — none exist today.

4. **Security invariant — no shell interpolation.** Plan content is never passed
   as a CLI argument. It is written to a temp file (mode 0600, `os.MkdirTemp`
   directory) and fed to `crush run` via stdin. `validateSlug` in `state.go`
   prevents path traversal from `plan_slug` input.

5. **stdio MCP server is single-threaded for tool handlers.** The MCP Go SDK in
   stdio transport processes one handler at a time. State files and `runCrushFn`
   have no concurrent tool-handler access. There are no background goroutines.
   If the server is ever ported to a concurrent transport, a per-slug file lock
   is required for state files.

---

## Project Overview

`esquisse-mcp` is a Go MCP stdio server that exposes three tools to AI agents
working in Esquisse-managed projects:

- **`adversarial_review`** — runs N rounds of cross-model adversarial review via
  `crush run --model`, using a family-interleaved randomized model order, with
  enterprise-policy fallback. Writes verdict to `.adversarial/{slug}.json`.
- **`gate_review`** — reads all `.adversarial/*.json` files and returns
  `blocked=true` if any plan has a FAILED or missing verdict.
- **`write_planning_artifact`** — writes a Planning Artifact file to
  `docs/artifacts/{date}-{slug}.md` per SCHEMAS.md §10.

**Module:** `github.com/loicalleyne/esquisse-mcp`  
**Go version:** 1.25+  
**Dependency:** `github.com/modelcontextprotocol/go-sdk v1.5.0`

---

## Repository Layout

```
esquisse-mcp/
├── AGENTS.md               ← This file — project constitution
├── GLOSSARY.md             ← Domain vocabulary
├── ONBOARDING.md           ← Agent orientation
├── README.md               ← Human-facing: build, install, configuration
│
├── main.go                 ← Entry point: --project-root flag, server startup
├── tools.go                ← Tool registration (mcp.AddTool)
├── adversarial.go          ← adversarial_review handler, adversarialInput, newAdversarialHandler
├── models.go               ← Model pool management:
│                              buildModelPool (reads ESQUISSE_MODELS env var),
│                              familyInterleaveShuffle, buildRotationOrder,
│                              runOneRound, isModelUnavailable, worstVerdict,
│                              effectiveRounds, excludeModelFilter, providerOf
├── runner.go               ← RunCrush, RunResult — crush subprocess management
├── gate.go                 ← gate_review handler, gateInput, gateOutput
├── state.go                ← ReadState, WriteState, ReviewState, validateSlug
│
├── models_test.go          ← Tests for all functions in models.go
├── state_test.go           ← Tests for state.go
│
├── go.mod
└── go.sum
```

---

## Build Commands

```sh
# Build binary
go build -o esquisse-mcp .

# Run tests (all)
go test -count=1 ./...

# Run tests with race detector
go test -count=1 -race ./...

# Install to GOPATH/bin
go install .
```

---

## Key Dependencies

| Dependency | Role | Notes |
|---|---|---|
| `github.com/modelcontextprotocol/go-sdk v1.5.0` | MCP server SDK | Supports stdio transport only — do not bypass with raw HTTP |
| `crush` (external binary) | LLM runner | Must be in PATH; used by `RunCrush` |

---

## Code Conventions

- **Package:** `package main` — single package, no subpackages.
- **Error handling:** `mcpErr(format, args...)` for tool-layer errors; `fmt.Errorf` with `%w` for internal wrapping; `log.Fatal` only in `main()`.
- **Logging:** `log.Printf` for warnings and operational events. Use `%q` when logging any value derived from env vars or user input to prevent log injection.
- **Temp files:** Always `os.CreateTemp(tmpDir, "round-*.txt")` with mode 0600. Never construct temp file paths manually. Always delete temp files before returning from `runOneRound`.
- **`ExcludeModel` is per-call, not per-server.** The base pool is built once in the `newAdversarialHandler` closure (`buildModelPool()`). `excludeModelFilter` is called inside the handler on every request with `input.ExcludeModel`. Never move `excludeModelFilter` to the closure-construction time.
- **Context:** Pass `ctx` through all subprocess calls. Use `context.WithTimeout` for sub-operations (`adversarial_review` uses a 300s top-level context).
- **Tests:** Use `t.Setenv()` for env var tests. Restore `runCrushFn` with `defer func() { runCrushFn = orig }()`. Do NOT call `t.Parallel()` in tests that call `SetRandSource` (global state mutation).

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `ESQUISSE_MODELS` | `""` (use defaults in `models.go`) | Comma-separated `provider/model` list. Overrides the entire pool. Invalid entries are skipped with a log warning; all-invalid falls back to `defaultModels`. |

---

## Security Boundaries

- **No shell interpolation.** `crush run` is invoked via `exec.Command`, not `exec/sh -c`. Plan content goes via stdin only.
- **Path traversal prevention.** `validateSlug` rejects slugs containing `/`, `\`, or traversal sequences before any file path is constructed.
- **Log injection prevention.** Env var values are logged with `%q`; plan content and slugs are never logged at runtime.
- **Temp file isolation.** Each `adversarial_review` call gets its own `os.MkdirTemp` directory, removed by deferred `os.RemoveAll`. Per-round files use `os.CreateTemp(tmpDir, "round-*.txt")`.
- **crush binary path.** `exec.LookPath("crush")` is called at handler entry — never hardcoded.

---

## Common Mistakes to Avoid

1. **Calling `t.Parallel()` in tests that call `SetRandSource` or mutate `runCrushFn`.**
   - Wrong: `t.Parallel()` + `SetRandSource(rand.NewSource(42))` in the same test
   - Right: omit `t.Parallel()`; restore with `defer SetRandSource(nil)` or `defer func() { runCrushFn = orig }()`
   - Why: these are package-level variables; concurrent writes cause data races.

2. **Passing plan content as a CLI argument.**
   - Wrong: `exec.Command("crush", "run", "--model", model, planContent)`
   - Right: write to temp file, pass via `cmd.Stdin = f`
   - Why: shell injection and argument length limits.

3. **Writing state file before all rounds complete.**
   - Wrong: `WriteState` inside the round loop
   - Right: `WriteState` once after the round loop, with `state.Iteration += rounds`
   - Why: partial writes leave a stale verdict in the state file if a later round fails.

4. **Not advancing `state.Iteration` when `runAdversarialRounds` errors mid-call.**
   - This is INTENTIONAL. If the call fails, no rounds were fully completed; `Iteration` must not advance. `WriteState` is only called on success.

5. **Adding a global model pool variable.**
   - Wrong: `var pool = buildModelPool()` at package level
   - Right: `pool := buildModelPool()` inside `newAdversarialHandler`, captured in closure
   - Why: `buildModelPool` reads `os.Getenv` — a package-level call runs at `init` time, before the process environment is fully configured in some hosting scenarios.

6. **Setting `ESQUISSE_MODEL_SLOT{0-4}` or `ESQUISSE_ALLOWED_PROVIDERS` in crush.json.**
   - These env vars were removed in P3-006. Use `ESQUISSE_MODELS` instead.
   - Wrong: `"ESQUISSE_MODEL_SLOT0": "copilot/claude-sonnet-4.6"`
   - Right: `"ESQUISSE_MODELS": "copilot/claude-sonnet-4.6,copilot/gpt-4.1,gemini/gemini-2.0-flash"`

---

## References

- [GLOSSARY.md](GLOSSARY.md)
- [ONBOARDING.md](ONBOARDING.md)
- [README.md](README.md) — build, install, crush.json configuration
- [../AGENTS.md](../AGENTS.md) — parent esquisse project constitution
- [../docs/tasks/P1-004-esquisse-mcp-server.md](../docs/tasks/P1-004-esquisse-mcp-server.md) — original implementation task
- [../docs/tasks/P2-006-mcp-configurable-model-rotation.md](../docs/tasks/P2-006-mcp-configurable-model-rotation.md) — model pool + multi-round task
- [../docs/tasks/P3-006-mcp-env-model-pool.md](../docs/tasks/P3-006-mcp-env-model-pool.md) — replaced probing with ESQUISSE_MODELS env var
