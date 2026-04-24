# P3-006: Replace Model Probing with ESQUISSE_MODELS Env Var

## Status
Status: Done
Depends on: P3-005 (model availability cache — being replaced)
Blocks: none

## Goal
Remove all model-probing and background-cache machinery from `esquisse-mcp` and
replace the pool configuration mechanism with a single `ESQUISSE_MODELS` env var
(comma-separated `provider/model` list). Users provide the pool explicitly in their
MCP server client configuration; no background goroutines, no disk cache.

## Background

P3-005 added a background probe (`modelProber`) that tried to determine which models
were actually available by sending them a `/dev/null` prompt. This was broken in two
ways:

1. Empty prompt → crush exited 0 without making an API call → policy-blocked models
   were wrongly marked available.
2. Even after fixing the probe prompt, `adversarialHandler` consumed a static pool
   built at construction time and never consulted the prober — so `force_refresh` had
   no effect on review rotation.

The fundamental problem is that crush's enterprise-policy rejection only surfaces at
real API call time. Probing is inherently unreliable because "available to list" and
"available to call" are different. The simplest correct design is to let the user
supply the pool explicitly in the MCP server environment configuration.

## In Scope

- Remove `modelProber` struct and all methods (`loadCache`, `saveCache`, `startProbe`,
  `currentState`, `forceRefresh`, `newModelProber`, `newModelProberWithFuncs`)
- Remove `ModelEntry` and `ModelCache` types
- Remove `defaultCachePath` and `defaultProbeTTL` functions
- Remove `filterAvailableModels` function
- Remove `discoverInput` type and `newDiscoverHandler` function
- Remove `discover_models` tool registration from `registerTools`
- Remove `--probe` CLI flag and `runProbeAndExit` function from `main.go`
- Replace `buildModelPool()`: read `ESQUISSE_MODELS` env var (comma-separated
  `provider/model`); fall back to `defaultModels` if unset or all entries invalid
- Drop `*modelProber` parameter from `newAdversarialHandler` and `registerTools`
- Remove probing-only imports from `models.go`: `encoding/json`, `os/exec`, `sync`, `strconv`,
  `filepath`, and `github.com/modelcontextprotocol/go-sdk/mcp` (used only by
  `discoverInput`/`newDiscoverHandler`, both deleted); keep `time` (used by `newRand`)
  and `strings` (used by surviving functions)
- Remove probing-only imports from `models_test.go`: `encoding/json`, `path/filepath`,
  `sync`, `time`, `fmt`, `os`, and `github.com/modelcontextprotocol/go-sdk/mcp`
  (all used exclusively by tests being deleted); keep `context`, `math/rand`, `strings`, `testing`
- Update `errAllModelsUnavailable` error text in `models.go`: remove references to
  `ESQUISSE_ALLOWED_PROVIDERS` and `discover_models` (both are gone after this task).
  New text: `"all models in the pool are unavailable (enterprise policy may be blocking them); set ESQUISSE_MODELS to a list of models known to be accessible"`
- Update the empty-pool error in `newAdversarialHandler` in `adversarial.go`: replace
  `"all model slots excluded by ESQUISSE_ALLOWED_PROVIDERS — set ESQUISSE_POOL_FALLBACK_STRICT=0 or allow at least one provider"`
  with `"ESQUISSE_MODELS produced an empty model pool; set ESQUISSE_MODELS to a comma-separated list of provider/model entries"`
- Update `models_test.go`: remove all `TestModelProber*` tests; replace
  `TestBuildModelPool_*` tests with `ESQUISSE_MODELS`-based tests
- Update `ROADMAP.md`: add P3-006 row

## Out of Scope

- Changes to any skill SKILL.md files
- Changes to `adversarial-review` skill step descriptions
- Changes to `crush.json` or MCP client configuration documentation
- Changes to `runner.go`, `gate.go`, `state.go`, `artifact.go`
- Adding a `list_models` tool or any discovery replacement
- Deprecation notices or migration guides

## Files

| File | Action | What Changes |
|------|--------|--------------|
| `esquisse-mcp/models.go` | Modify | Remove probing machinery, `ModelEntry`, `ModelCache`, `filterAvailableModels`, `newDiscoverHandler`, `discoverInput`; rewrite `buildModelPool` to read `ESQUISSE_MODELS`; remove now-unused imports |
| `esquisse-mcp/adversarial.go` | Modify | Drop `prober *modelProber` param from `newAdversarialHandler`; remove `filterAvailableModels` call; replace empty-pool error message (remove stale `ESQUISSE_ALLOWED_PROVIDERS`/`ESQUISSE_POOL_FALLBACK_STRICT` references) |
| `esquisse-mcp/tools.go` | Modify | Drop `prober *modelProber` param from `registerTools`; remove `discover_models` `mcp.AddTool` block |
| `esquisse-mcp/main.go` | Modify | Remove `--probe` flag, `runProbeAndExit`, `firstNonEmptyLine`; remove prober construction and `startProbe`; remove now-unused imports (`os/exec`, `strings`, `time`) |
| `esquisse-mcp/models_test.go` | Modify | Remove all `TestModelProber*` and `TestBuildModelPool_*` tests; add `TestBuildModelPool_EnvVar` test group |
| `docs/planning/ROADMAP.md` | Modify | Add P3-006 row under P3 tasks |

## Specification

### `ESQUISSE_MODELS` parsing (new `buildModelPool`)

```
ESQUISSE_MODELS=copilot/claude-sonnet-4.6,copilot/gpt-4.1,gemini/gemini-2.0-flash
```

Rules:
- Split on `,`, trim each entry.
- Skip empty entries.
- Validate each entry: must match `validModelRe` (`^[a-zA-Z0-9_./-]+$`) AND contain
  at least one `/` with non-empty parts on both sides (validated via `SplitN(m, "/", 2)`).
  Note: multiple slashes are allowed (e.g. `"provider/path/model"`) — `SplitN` with `n=2`
  accepts these and passes the non-empty-parts check. This is intentional.
  Log and skip invalid entries.
- If `ESQUISSE_MODELS` is unset or empty: return a copy of `defaultModels` (unchanged
  default pool — no behaviour change for users who don't set the var).
- If `ESQUISSE_MODELS` is set but produces zero valid entries after validation: log a
  warning and fall back to `defaultModels` (fail-open).

```go
// buildModelPool returns the model pool from ESQUISSE_MODELS env var.
// Falls back to defaultModels if the var is unset or all entries are invalid.
func buildModelPool() []string {
    raw := os.Getenv("ESQUISSE_MODELS")
    if raw == "" {
        return append([]string(nil), defaultModels...)
    }
    var pool []string
    for _, m := range strings.Split(raw, ",") {
        m = strings.TrimSpace(m)
        if m == "" {
            continue
        }
        if !validModelRe.MatchString(m) {
            log.Printf("esquisse-mcp: ESQUISSE_MODELS entry %q contains invalid characters, skipping", m)
            continue
        }
        parts := strings.SplitN(m, "/", 2)
        if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
            log.Printf("esquisse-mcp: ESQUISSE_MODELS entry %q must be provider/model, skipping", m)
            continue
        }
        pool = append(pool, m)
    }
    if len(pool) == 0 {
        log.Printf("esquisse-mcp: ESQUISSE_MODELS produced no valid entries, falling back to defaults")
        return append([]string(nil), defaultModels...)
    }
    return pool
}
```

### `exclude_model` best-effort behaviour (no change to `excludeModelFilter`)

`excludeModelFilter` already has fail-open: if the exclusion would empty the pool,
it returns the original pool unchanged. This satisfies the requirement that if the
caller's model is the only one registered, it is used rather than blocking.

No change to `excludeModelFilter` logic or tests.

### Signature changes

```go
// Before:
func newAdversarialHandler(projectRoot string, prober *modelProber) func(...)
func registerTools(server *mcp.Server, projectRoot string, prober *modelProber)

// After:
func newAdversarialHandler(projectRoot string) func(...)
func registerTools(server *mcp.Server, projectRoot string)
```

`adversarialHandler` builds the pool once at construction:
```go
func newAdversarialHandler(projectRoot string) func(...) {
    pool := buildModelPool()
    // Note: buildModelPool never returns empty (always falls back to defaultModels),
    // so the len==0 guard below is a defensive safety net only.
    return func(ctx context.Context, req *mcp.CallToolRequest, input adversarialInput) (...) {
        ...
        if len(pool) == 0 {
            return mcpErr("ESQUISSE_MODELS produced an empty model pool; set ESQUISSE_MODELS to a comma-separated list of provider/model entries")
        }
        effectivePool := excludeModelFilter(pool, input.ExcludeModel)
        ...
    }
}
```

### Updated `errAllModelsUnavailable` text

The current text references `ESQUISSE_ALLOWED_PROVIDERS` and `discover_models` — both removed by this task. Replace with:
```go
var errAllModelsUnavailable = errors.New("all models in the pool are unavailable " +
    "(enterprise policy may be blocking them); " +
    "set ESQUISSE_MODELS to a list of models known to be accessible")
```

### Removed from `main.go`

The prober wiring (`defaultCachePath`, `defaultProbeTTL`, `newModelProber`,
`prober.startProbe`) and the `--probe` flag + `runProbeAndExit` / `firstNonEmptyLine`
functions are all removed. `main.go` only sets up `projectRoot`, creates the server,
calls `registerTools`, and runs.

Imports removed from `main.go`: `os/exec`, `strings`, `time`.

### Tests to remove

All tests that exercise deleted types and functions — remove entirely from `models_test.go`:

`TestModelProber` sub-tests (the entire `TestModelProber` top-level function and all its sub-tests):
- `no_cache_returns_probing_state`
- `fresh_cache_no_rerun`
- `stale_cache_sets_stale_flag`
- `force_refresh_resets_state`
- `probe_marks_available_true_on_zero_exit`
- `probe_marks_available_false_on_policy_block`
- `corrupted_cache_starts_fresh_probe`
- `invalid_model_id_in_cache_skipped`
- `probe_marks_available_true_on_transient_error`
- `structured_response_shape`
- `cache_written_atomically`

Top-level test functions to remove:
- `TestModelProberFilterAllowedProviders` (uses `newDiscoverHandler`, `discoverInput`, `ModelCache`, `ModelEntry`, `mcp.CallToolRequest`)
- `TestModelProberConcurrentAccess` (uses `newModelProberWithFuncs`, `sync.WaitGroup`)

Old `buildModelPool` tests to remove (slot env vars no longer exist):
- `TestBuildModelPool_Defaults`
- `TestBuildModelPool_AllowedProviders_Copilot`
- `TestBuildModelPool_AllowedProviders_UnknownFallback`
- `TestBuildModelPool_InvalidSlotFormat`
- `TestBuildModelPool_StrictWithAllFiltered`

### Tests to add: `TestBuildModelPool_EnvVar`

Acceptance criteria names map 1:1 to `t.Run` sub-tests:

| Sub-test | Assertion |
|----------|-----------|
| `unset_returns_defaults` | `ESQUISSE_MODELS=""` → returns copy of `defaultModels`, len 5 |
| `valid_entries_returned` | `ESQUISSE_MODELS="copilot/a,gemini/b"` → pool = `["copilot/a","gemini/b"]` |
| `whitespace_trimmed` | `ESQUISSE_MODELS=" copilot/a , gemini/b "` → same as above |
| `empty_entry_skipped` | `ESQUISSE_MODELS="copilot/a,,gemini/b"` → len 2 |
| `invalid_no_slash_skipped` | `ESQUISSE_MODELS="badentry,copilot/a"` → pool = `["copilot/a"]` |
| `invalid_chars_skipped` | `ESQUISSE_MODELS="copilot/a!,copilot/b"` → pool = `["copilot/b"]` |
| `all_invalid_falls_back` | `ESQUISSE_MODELS="bad1,bad2"` → returns `defaultModels` |

## Acceptance Criteria

1. `go build ./...` succeeds with no errors.
2. `go test -count=1 ./...` passes (all surviving tests green).
3. `go vet ./...` clean.
4. `TestBuildModelPool_EnvVar` sub-tests all pass.
5. All `TestExcludeModelFilter` sub-tests still pass (no change to that function).
6. All `TestRunOneRound_*`, `TestFamilyInterleaveShuffle_*`, `TestBuildRotationOrder_*`,
   `TestWorstVerdict`, `TestIsModelUnavailable`, `TestEffectiveRounds` still pass.
7. `grep -n "modelProber\|ModelCache\|ModelEntry\|filterAvailableModels\|discoverInput\|newDiscoverHandler\|discover_models\|defaultCachePath\|defaultProbeTTL\|runProbeAndExit\|firstNonEmptyLine" esquisse-mcp/*.go` — zero matches.
8. `grep -n "ESQUISSE_MODEL_SLOT\|ESQUISSE_ALLOWED_PROVIDERS\|ESQUISSE_POOL_FALLBACK_STRICT\|ESQUISSE_MODEL_CACHE_TTL" esquisse-mcp/*.go` — zero matches.
9. `grep -n "produced no valid entries" esquisse-mcp/models.go` — exactly 1 match (fallback log line is present).

## Session Notes

- P3-005 (`mcp-model-availability-cache`) is effectively being reverted and replaced.
  Its `discover_models` tool is removed; users who relied on it should use
  `ESQUISSE_MODELS` in their MCP env config instead.
- `defaultModels` is kept as the unchanged fallback for users who set no env var.
- `exclude_model` fail-open behaviour (single-entry pool → use it anyway) was already
  correct; no change required.
- The `runCrushFn` package-level var, `RunCrush`, and `isModelUnavailable` are all
  retained — they are used by `runOneRound` which is unchanged.

**2026-04-23 — Completed.** Removed all probing machinery (`modelProber`, `ModelCache`, `ModelEntry`, `filterAvailableModels`, `discoverInput`, `newDiscoverHandler`, `--probe` flag, `runProbeAndExit`, `firstNonEmptyLine`). Replaced `buildModelPool()` to read `ESQUISSE_MODELS` env var with fallback to `defaultModels`. All 7 `TestBuildModelPool_EnvVar` sub-tests pass. All pre-existing tests pass. `go vet` clean.
