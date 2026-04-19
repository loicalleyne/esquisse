# P3-005: Model Availability Cache for discover_models

## Status
Status: Completed
Depends on: P2-006 (5-slot model pool, discover_models tool), P3-001 (exclude_model)
Blocks: none

## Goal
Add a background availability-probe goroutine and disk cache to `esquisse-mcp` so
that `discover_models` returns structured per-model availability data near-instantly
and agents can skip models blocked by enterprise policy before wasting review rounds.

## Background

The current `discover_models` implementation runs `crush models` synchronously on
every call (10-second timeout) and returns a flat plain-text list. Two problems:

1. **No availability data.** The list includes models that are blocked by enterprise
   policy; callers only discover this when a review round fails mid-run in
   `runOneRound` → `isModelUnavailable`. This burns a round and confuses the
   rotation order.

2. **Synchronous probe.** Every call to `discover_models` re-runs `crush models`,
   adding 2–10 seconds of latency. Repeated calls during a multi-task session stall
   the agent unnecessarily.

The bof-mcp design (bof `docs/specs/2026-04-19-bof-crush-compatibility-design.md`)
proposes a background probe at server startup + a disk cache with TTL. This task
ports that pattern into `esquisse-mcp` with the following adaptations:

- Cache path: `os.UserConfigDir()/esquisse-mcp/model-cache.json`
- Injectable probe functions on the `modelProber` struct (not package-level vars)
  — allows `t.Parallel()` in tests and avoids package-level global state
- `discover_models` response changes from plain text to JSON (breaking change;
  documented in README and tools.go description)
- `buildModelPool` is NOT modified in this task (see Out of Scope)

## Solution

### New types (models.go)

```go
// ModelEntry represents one probed model in the availability cache.
type ModelEntry struct {
    ID        string    `json:"id"`
    Provider  string    `json:"provider"`
    Available bool      `json:"available"`
    ProbedAt  time.Time `json:"probed_at"`
}

// ModelCache is the JSON schema for ~/.config/esquisse-mcp/model-cache.json.
type ModelCache struct {
    Entries        []ModelEntry `json:"entries"`
    CachedAt       time.Time   `json:"cached_at"`
    ProbeCompleted bool        `json:"probe_completed"`
}

// modelProber manages the background probe goroutine and disk cache.
// All exported-accessible state is protected by mu.
// Holds injectable listModelsFn and probeFn for test substitution.
type modelProber struct {
    mu           sync.RWMutex
    cache        *ModelCache
    probing      bool
    done         chan struct{} // closed when current probe completes
    cachePath    string
    ttl          time.Duration
    // Injectable for testing — set via newModelProberWithFuncs.
    listModelsFn func(ctx context.Context) ([]string, error)
    probeFn      func(ctx context.Context, model string) bool
}
```

### Probe logic

1. `startProbe(ctx)` — starts a goroutine that:
   a. Calls `listModelsFn` with a 10-second timeout context to get the raw model list from `crush models`
   b. Immediately populates the in-memory cache with the returned models (setting `Available: true` as a fail-open default) so `discover_models` returns a usable list right away.
   c. For each model in the list, calls `probeFn` (15-second timeout per model)
      - `probeFn` returns `true` if crush exits 0 (available) or exits non-zero
        with output that does NOT match `isModelUnavailable` (transient — treat
        as available, fail-open)
      - `probeFn` returns `false` only if exit is non-zero AND output matches
        `isModelUnavailable` (definitively blocked by policy)
      - Updates the in-memory cache entry for that model after each probe completes.
   d. Writes the completed `ModelCache` to disk (atomic: temp file + `os.Rename`)
   e. Closes `done` channel

2. `currentState()` — returns `(entries []ModelEntry, probing bool, stale bool, cachedAt time.Time)`
   under read lock

3. `forceRefresh(ctx)` — cancels in-progress probe, resets state, calls `startProbe`

### `discover_models` response (JSON string in TextContent)

```json
{
  "models": [
    {"id": "copilot/claude-sonnet-4.6", "provider": "copilot", "available": true},
    {"id": "gemini/gemini-3.1-pro-preview", "provider": "gemini", "available": false}
  ],
  "cached_at": "2026-04-19T14:00:00Z",
  "stale": false,
  "probing": false
}
```

States:
- No cache, probe in progress: `{"models": [], "probing": true, "stale": false}`
- Cache fresh: full list, `"probing": false`, `"stale": false`
- Cache stale: full cached list, `"stale": true`, `"probing": true`
- `force_refresh: true`: resets to `{"models": [], "probing": true, "stale": false}`
- `Filter`: The `input.Filter` string (case-insensitive substring) is matched against `ModelEntry.ID` in `newDiscoverHandler` before building the returned JSON array.

### Server startup (main.go)

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

cachePath := defaultCachePath()
prober := newModelProber(cachePath, defaultProbeTTL())
prober.startProbe(ctx)

server := mcp.NewServer(...)
registerTools(server, *projectRoot, prober)
server.Run(ctx, &mcp.StdioTransport{})
```

When `server.Run` returns (stdin closes), `ctx` is cancelled and the probe
goroutine exits at the next `ctx.Done()` check.

## In Scope

- `esquisse-mcp/models.go`: Add `ModelEntry`, `ModelCache`, `modelProber`,
  `newModelProber`, `newModelProberWithFuncs` (test constructor), `startProbe`,
  `currentState`, `forceRefresh`, `loadCache`, `saveCache`, `defaultCachePath`,
  `defaultProbeTTL`; revise `discoverInput` (add `ForceRefresh bool`); revise
  `newDiscoverHandler` to accept and use a `*modelProber`
- `esquisse-mcp/models_test.go`: Tests for all new types and functions
  (`TestModelProber/*`); test constructor uses `newModelProberWithFuncs` with
  injected stubs — all tests may use `t.Parallel()`
- `esquisse-mcp/main.go`: Thread server context through; create `modelProber`;
  call `prober.startProbe(ctx)`; pass `prober` to `registerTools`
- `esquisse-mcp/tools.go`: Update `registerTools` signature to accept `prober
  *modelProber`; update `discover_models` description to document JSON response
  format, `force_refresh` param, and cache TTL
- `esquisse-mcp/README.md`: Document new `discover_models` JSON response schema,
  `ESQUISSE_MODEL_CACHE_TTL_DAYS`, cache path, breaking change from plain text
- `esquisse-mcp/AGENTS.md`: Document `ModelEntry`/`ModelCache`/`modelProber` in
  repository layout; add `ESQUISSE_MODEL_CACHE_TTL_DAYS` to env var table; update
  concurrency note (probe goroutine → `sync.RWMutex` required)

## Out of Scope

- `buildModelPool` pre-filtering of `available: false` models from the review pool
  (requires `ESQUISSE_SKIP_UNAVAILABLE` env var design — deferred to P3-006)
- Per-model TTL / individual entry expiry
- Probing models not returned by `crush models` (only discovered models are probed)
- HTTP transport or concurrent MCP handler support
- Auth token caching

## Files

| Path | Action | What changes |
|------|--------|-------------|
| `esquisse-mcp/models.go` | Modify | Add `ModelEntry`, `ModelCache`, `modelProber` types; probe goroutine; cache I/O; revise `discoverInput`+`newDiscoverHandler` |
| `esquisse-mcp/models_test.go` | Modify | Add `TestModelProber` suite (≥10 subtests) |
| `esquisse-mcp/main.go` | Modify | Server context with cancel; create prober; `startProbe`; pass prober to `registerTools` |
| `esquisse-mcp/tools.go` | Modify | `registerTools` signature; `discover_models` description updated |
| `esquisse-mcp/README.md` | Modify | New response format, env vars, breaking change note |
| `esquisse-mcp/AGENTS.md` | Modify | New types in layout; env var table; concurrency note |

## Acceptance Criteria

1. `TestModelProber/no_cache_returns_probing_state` — when no cache file exists and
   probe is in progress, `discover_models` returns JSON with `"probing": true` and
   `"models": []`.

2. `TestModelProber/fresh_cache_no_rerun` — when cache file is within TTL,
   `discover_models` returns cached entries without calling `listModelsFn`.
   Verified by counting `listModelsFn` invocations (must be 0 after load).

3. `TestModelProber/stale_cache_sets_stale_flag` — when cache file is older than
   TTL, response contains `"stale": true` and `"probing": true`; cached entries
   are still returned.

4. `TestModelProber/force_refresh_resets_state` — `discover_models` with
   `force_refresh: true` returns `{"probing": true, "models": []}` regardless of
   cache age; a new probe is started.

5. `TestModelProber/structured_response_shape` — response JSON contains `"models"`
   (array of objects with `"id"`, `"provider"`, `"available"` fields), `"cached_at"`
   (RFC3339 string or empty when no cache), `"stale"` (bool), `"probing"` (bool).

6. `TestModelProber/probe_marks_available_true_on_zero_exit` — `probeFn` stub
   returning `(exitCode=0, output="ok")` causes the model entry to have
   `"available": true`.

7. `TestModelProber/probe_marks_available_false_on_policy_block` — `probeFn` stub
   returning `(exitCode=1, output="model is not supported")` causes the model entry
   to have `"available": false`.

8. `TestModelProber/probe_marks_available_true_on_transient_error` — `probeFn` stub
   returning `(exitCode=1, output="connection refused")` (non-policy error) causes
   the model entry to have `"available": true` (fail-open).

9. `TestModelProber/cache_written_atomically` — after probe completes, cache file
   exists at `cachePath`; no temp file remains in the same directory.

10. `TestModelProber/filter_respects_allowed_providers` — when
    `ESQUISSE_ALLOWED_PROVIDERS=copilot`, `discover_models` response contains only
    models with `"provider": "copilot"`.

11. `TestModelProber/corrupted_cache_starts_fresh_probe` — when the cache file
    contains invalid JSON (e.g. `"not json"`), `loadCache` returns an error, the
    prober starts a fresh probe, and `discover_models` returns
    `{"probing": true, "models": []}`.

12. `TestModelProber/invalid_model_id_in_cache_skipped` — when the cache file
    contains a `ModelEntry` whose `ID` fails `validModelRe` (e.g. `"../evil"`),
    that entry is dropped from the returned `models` list; valid entries are kept.

13. `go test -count=1 -race ./...` passes — no data races in `modelProber` under
    concurrent reads and probe writes.

## Session Notes

- **Breaking change:** `discover_models` response changes from a plain-text newline-
  separated list to a JSON string. Existing callers that parse the text format will
  break. This is acceptable: the only callers are agents in this workflow, and the
  README and tool description are updated.

- **Injectable functions pattern:** `modelProber` holds `listModelsFn` and `probeFn`
  as struct fields (not package-level vars). The production constructor
  `newModelProber` wires in the real functions; `newModelProberWithFuncs` is the
  test constructor. This avoids package-level state mutation and allows
  `t.Parallel()` in all `TestModelProber` subtests.

- **Probe availability logic (fail-open):** `probeFn` returns `true` (available) on
  zero exit OR on non-zero exit where output does NOT match `isModelUnavailable`.
  Only definitively policy-blocked models get `available: false`. This mirrors
  `runOneRound`'s fallback behavior and avoids false negatives from transient
  network errors.

- **Concurrency invariant update:** AGENTS.md currently states "stdio MCP server is
  single-threaded." This remains true for MCP request handling. However, the
  background probe goroutine introduces a writer concurrent with MCP handler reads.
  `modelProber.mu` (a `sync.RWMutex`) is the only lock required. The update to
  AGENTS.md must document this explicitly to prevent future confusion.

- **Atomic cache write:** write to `cachePath + ".tmp"`, then `os.Rename`. This
  prevents a half-written JSON file from being read by a concurrent
  `discover_models` call during an in-progress probe flush.

- **`defaultCachePath`:** uses `os.UserConfigDir()` (returns `~/.config` on Linux,
  `~/Library/Application Support` on macOS, `%AppData%` on Windows — but Windows is
  already blocked at `main.go` startup, so only the Linux/macOS paths are relevant).
  Directory is created with `os.MkdirAll(dir, 0o700)` if absent.

- **TTL:** `ESQUISSE_MODEL_CACHE_TTL_DAYS` env var, parsed as integer days, default
  3. Zero or negative → use default. Invalid (non-integer) → log warning + use
  default.

- **Context threading:** `main.go` currently calls `server.Run(context.Background(), ...)`.
  Change to `ctx, cancel := context.WithCancel(context.Background()); defer cancel()`.
  Pass `ctx` to `prober.startProbe(ctx)`. When `server.Run` returns, `cancel()` runs,
  the probe goroutine exits at its next `ctx.Done()` check (between per-model probe
  calls). This is best-effort cancellation — a model probe in-flight will complete
  its 15-second timeout before the goroutine exits.

- **`discoverInput` backward compatibility:** The existing `Filter string` field is
  preserved. `ForceRefresh bool` is added with JSON tag `force_refresh`.

- **Cache file validation (security):** `loadCache` validates each `ModelEntry.ID`
  against `validModelRe` after JSON unmarshal. Entries whose `ID` fails validation
  are silently dropped (log warning with `%q`). This prevents a crafted or corrupted
  cache file from injecting arbitrary strings into logs or agent output. An unmarshal
  error (malformed JSON) causes `loadCache` to return an error; the prober treats
  this identically to a missing cache file and starts a fresh probe.

- **Log injection:** All model IDs logged by probe code use `%q` (same convention
  as `excludeModelFilter`). This is enforced by AGENTS.md and must be followed in
  all new `log.Printf` calls in `modelProber`.

- **`os.UserConfigDir()` failure:** `defaultCachePath()` returns `(string, error)`.
  If it fails, `newModelProber` sets `cachePath = ""` (no-disk mode): `loadCache`
  is a no-op, `saveCache` is a no-op, the prober still runs in-memory and
  `discover_models` still returns structured data. A warning is logged at startup.

- **Cache directory creation failure / atomic rename failure:** In both cases,
  log a warning with `%q` on the path and continue — the in-memory state is still
  updated and served by subsequent `discover_models` calls. The absence of a
  persisted cache only means the next server restart re-probes from scratch.

- **Cache file permissions:** cache directory created with `os.MkdirAll(dir, 0o700)`.
  The atomic write sequence follows the `runOneRound` temp file pattern exactly:
  `os.CreateTemp(dir, "model-cache-*.json")` → `tmp.Chmod(0o600)` → write JSON
  → `tmp.Close()` → `os.Rename(tmp.Name(), cachePath)`. Temp file is created in
  the same directory as the target so `os.Rename` is always same-filesystem (no
  cross-device issue). AC9 must verify no temp files remain.

- **`listModelsFn` error (complete probe failure):** if `listModelsFn` returns an
  error (e.g. `crush models` exits non-zero or binary not found), the probe
  goroutine logs the error, sets `probing: false` under write lock, closes the
  `done` channel, and returns. The in-memory cache is left unchanged (loaded-from-
  disk entries, or empty if no cache existed). No retry — caller uses
  `force_refresh: true` to trigger a new probe.

- **Subprocess cancellation:** `probeFn` in production wraps `RunCrush`, which uses
  `exec.CommandContext(ctx, ...)`. When `forceRefresh(ctx)` cancels the probe's
  context, Go's `exec.CommandContext` sends SIGKILL to the subprocess. No additional
  termination logic is required.

- **Plan revised 2026-04-19 (Iteration 4):** Addressed Major issues from Adversarial-r0 CONDITIONAL verdict: (M1) In-memory cache is immediately populated with a fail-open (Available: true) state following listModelsFn, so the tool returns a usable list instantly rather than waiting minutes for the per-model probe. (M2) Filter logic is strictly defined to apply the substring match against ModelEntry.ID. (M3) listModelsFn is now constrained by a 10-second timeout context to prevent a crush subprocess hang from locking the probe goroutine indefinitely.
