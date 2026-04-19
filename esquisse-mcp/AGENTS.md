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
   have no concurrent tool-handler access. However, `modelProber` runs a
   background goroutine (`startProbe`) that writes shared cache state; all
   reads/writes to `modelProber.cache` and `modelProber.probing` MUST go through
   `modelProber.mu` (`sync.RWMutex`). If the server is ever ported to a
   concurrent transport, a per-slug file lock is also required for state files.

---

## Project Overview

`esquisse-mcp` is a Go MCP stdio server that exposes three tools to AI agents
working in Esquisse-managed projects:

- **`adversarial_review`** — runs N rounds of cross-model adversarial review via
  `crush run --model`, using a family-interleaved randomized model order, with
  enterprise-policy fallback. Writes verdict to `.adversarial/{slug}.json`.
- **`gate_review`** — reads all `.adversarial/*.json` files and returns
  `blocked=true` if any plan has a FAILED or missing verdict.
- **`discover_models`** — lists available `provider/model` strings from
  `crush models`, filtered by `ESQUISSE_ALLOWED_PROVIDERS` and an optional
  caller-supplied substring.

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
├── models.go               ← Model pool + availability cache:
│                              buildModelPool, familyInterleaveShuffle,
│                              buildRotationOrder, runOneRound, isModelUnavailable,
│                              worstVerdict, effectiveRounds, newDiscoverHandler;
│                              ModelEntry, ModelCache, modelProber,
│                              newModelProber, newModelProberWithFuncs,
│                              startProbe, currentState, forceRefresh,
│                              loadCache, saveCache, defaultCachePath, defaultProbeTTL
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
| `crush` (external binary) | LLM runner | Must be in PATH; used by `RunCrush` and `newDiscoverHandler` |

---

## Code Conventions

- **Package:** `package main` — single package, no subpackages.
- **Error handling:** `mcpErr(format, args...)` for tool-layer errors; `fmt.Errorf` with `%w` for internal wrapping; `log.Fatal` only in `main()`.
- **Logging:** `log.Printf` for warnings and operational events. Use `%q` when logging any value derived from env vars or user input to prevent log injection.
- **Temp files:** Always `os.CreateTemp(tmpDir, "round-*.txt")` with mode 0600. Never construct temp file paths manually. Always delete temp files before returning from `runOneRound`.
- **`ExcludeModel` is per-call, not per-server.** The base pool is built once in the `newAdversarialHandler` closure (`buildModelPool()`). `excludeModelFilter` is called inside the handler on every request with `input.ExcludeModel`. Never move `excludeModelFilter` to the closure-construction time.
- **Context:** Pass `ctx` through all subprocess calls. Use `context.WithTimeout` for sub-operations (`discover_models` uses a 10s sub-context; `adversarial_review` uses a 300s top-level context).
- **Tests:** Use `t.Setenv()` for env var tests. Restore `runCrushFn` with `defer func() { runCrushFn = orig }()`. Do NOT call `t.Parallel()` in tests that call `SetRandSource` (global state mutation).

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `ESQUISSE_MODEL_SLOT0`–`ESQUISSE_MODEL_SLOT4` | defaults in `models.go` | Override pool slot. Format: `provider/model`. Invalid format → warning + default. |
| `ESQUISSE_ALLOWED_PROVIDERS` | `""` (all allowed) | Comma-separated provider IDs (case-sensitive). Filters pool at startup. |
| `ESQUISSE_POOL_FALLBACK_STRICT` | `""` (fail-open) | Set `"1"` to error instead of falling back to full default pool when all slots filtered. |
| `ESQUISSE_MODEL_CACHE_TTL_DAYS` | `"3"` (3 days) | Availability cache TTL in days. Set to `"0"` to disable disk cache (in-memory only). |

---

## Security Boundaries

- **No shell interpolation.** `crush run` is invoked via `exec.Command`, not `exec/sh -c`. Plan content goes via stdin only.
- **Path traversal prevention.** `validateSlug` rejects slugs containing `/`, `\`, or traversal sequences before any file path is constructed.
- **Log injection prevention.** Env var values are logged with `%q`; plan content and slugs are never logged at runtime.
- **Temp file isolation.** Each `adversarial_review` call gets its own `os.MkdirTemp` directory, removed by deferred `os.RemoveAll`. Per-round files use `os.CreateTemp(tmpDir, "round-*.txt")`.
- **crush binary path.** `exec.LookPath("crush")` is called at handler entry — never hardcoded.

---

## Common Mistakes to Avoid

1. **Calling `t.Parallel()` in tests that call `SetRandSource`.**
   - Wrong: `t.Parallel()` + `SetRandSource(rand.NewSource(42))` in the same test
   - Right: omit `t.Parallel()` from any test that mutates `runCrushFn` or the rng factory
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

---

## References

- [GLOSSARY.md](GLOSSARY.md)
- [ONBOARDING.md](ONBOARDING.md)
- [README.md](README.md) — build, install, crush.json configuration
- [../AGENTS.md](../AGENTS.md) — parent esquisse project constitution
- [../docs/tasks/P1-004-esquisse-mcp-server.md](../docs/tasks/P1-004-esquisse-mcp-server.md) — original implementation task
- [../docs/tasks/P2-006-mcp-configurable-model-rotation.md](../docs/tasks/P2-006-mcp-configurable-model-rotation.md) — model pool + multi-round task
