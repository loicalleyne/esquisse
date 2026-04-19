# P2-006: Configurable Provider-Scoped Reasoning Model Rotation in esquisse-mcp

## Status
Status: Done — implemented 2026-04-18. 6-round adversarial review passed (iteration=17). All tests passing.
Depends on: P1-004 (esquisse-mcp server implemented)
Blocks: none

> **Round 1 adversarial review (Adversarial-r0 / GPT-4.1, iter 0):** FAILED verdict
> received 2026-04-18. Plan revised to address all major issues. Critical finding
> (path traversal) was a false positive — see Security Note in Specification.
> This document is the revised plan submitted for Round 2.

## Summary
`esquisse-mcp` currently hardcodes three adversarial reviewer slots that reference
OpenAI and Anthropic models (`openai/gpt-4.1`, `anthropic/claude-opus-4-5-20251101`,
`openai/gpt-4o`). Users who only have licences for GitHub Copilot, Google Gemini,
and Google Vertex AI cannot use those defaults. This task replaces the 3-slot fixed
rotation with a configurable N-slot pool that defaults to reasoning-capable models
from the user's licensed providers, adds provider filtering via
`ESQUISSE_ALLOWED_PROVIDERS`, adds a `discover_models` MCP tool backed by
`crush models` output, adds per-attempt graceful fallback so that a model
disabled by a GitHub Copilot Enterprise policy automatically tries the next slot
rather than failing the review, and runs **multiple review rounds per tool call**
with a **randomized, family-interleaved model order** (re-randomized every 5 rounds)
so that each invocation maximises reviewer diversity automatically.

## Problem
The three hardcoded default model strings in `adversarial.go`:
```go
var defaultModels = [3]string{
    "openai/gpt-4.1",
    "anthropic/claude-opus-4-5-20251101",
    "openai/gpt-4o",
}
```
reference providers the user may not have configured. The rotation is also fixed at
exactly 3 slots (`iteration % 3`), making it impossible to add more reviewers without
code changes. There is no mechanism to restrict models to a specific set of licensed
providers, and no way to discover available `provider/model` strings without running
`crush models` manually.

## Solution
1. Extend `defaultModels` to a configurable slice (default 5 slots) pointing to
   reasoning-capable models available via GitHub Copilot, Google Gemini, and
   Google Vertex AI — with verified model IDs from the user's `crush models` output.
2. Change the rotation formula from `iteration % 3` to a **per-invocation randomized
   order** built by `buildRotationOrder(pool, rounds)` — `modelForSlot` and
   `slotEnvVars` are removed.
3. Add `ESQUISSE_ALLOWED_PROVIDERS` env var (comma-separated provider IDs). When
   set, slot models whose provider prefix is not in the list are removed from the
   pool at startup; `esquisse-mcp` logs a warning per removed slot.
4. Add per-attempt enterprise fallback in `runOneRound`: when `crush run`
   exits non-zero with output matching known "model not available" patterns, try the
   next slot in the pool rather than failing the round. Record which model
   ultimately succeeded in the state file.
5. Add a `discover_models` MCP tool that shells out to `crush models` (piped,
   so Crush outputs plain `provider/model` lines), applies the provider filter, and
   returns the list to the caller.
6. Add **multi-round support**: `adversarialInput.Rounds` (default 5, minimum 1;
   values < 1 fall back to default) causes the tool to run `Rounds` consecutive
   reviews in a single call. The model order is **randomized per call** with
   family-interleaving (copilot, gemini, vertexai alternated) and a no-consecutive-
   same-model constraint. Every 5 rounds the order is re-randomized (fresh shuffle).
   The state file records the overall (worst-case) verdict across all rounds;
   `Iteration` advances by `rounds`.

## Scope
### In Scope
- `adversarial.go`: remove `modelForSlot`, `slotEnvVars`, `defaultModels [3]string`;
  add `defaultModels []string` (5 slots); add `Rounds int` to `adversarialInput`;
  replace `slot := state.Iteration % 3` with `buildRotationOrder` call;
  add multi-round loop (`runAdversarialRounds`); aggregate verdict; advance
  `Iteration` by `rounds`.
- New `models.go` in `esquisse-mcp/`: `buildModelPool()`, `isModelUnavailable()`,
  `buildRotationOrder()`, `familyInterleaveShuffle()`, `effectiveRounds()`,
  `worstVerdict()`, `runOneRound()`, `newDiscoverHandler()` MCP tool handler.
- `tools.go`: register `discover_models` tool.
- `README.md`: update model rotation table, add env var reference, add
  `discover_models` tool description, document enterprise fallback behaviour,
  document multi-round and randomized rotation.
- New `models_test.go`: tests for all new functions in `models.go`.

### Out of Scope
- Automatic `can_reason` filtering via Catwalk embedded data (adding
  `charm.land/catwalk` as a dependency — tracked as a potential P3 enhancement).
- Changes to `crush models` subcommand or Catwalk itself.
- Support for providers other than `copilot`, `gemini`, `vertexai` in the new
  default pool.
- Rotation weighting (e.g. Copilot 2× more likely) — future enhancement.
- A process-level `ESQUISSE_DEFAULT_ROUNDS` env var — rounds are per-call only.
- The `gate_review` tool and `state.go` are unchanged.

## Prerequisites
- [x] P1-004 is merged (`esquisse-mcp` server implemented and in production)
- [x] User has run `crush models` — all provider/model strings verified (see
  resolved assumptions in Session Notes).

## Specification

### New Env Vars

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `ESQUISSE_ALLOWED_PROVIDERS` | `string` | `""` (all allowed) | Comma-separated provider IDs. When non-empty, any slot model whose `provider/` prefix is not in this list is excluded from the pool. |
| `ESQUISSE_MODEL_SLOT0`–`ESQUISSE_MODEL_SLOT4` | `string` | see defaults below | Override the model for each slot. Existing pattern; extended from 3 to 5 slots. |
| `ESQUISSE_POOL_FALLBACK_STRICT` | `string` | `""` (fail-open) | Set to `"1"` to return an error instead of falling back to the full unfiltered default pool when all slots are filtered. |

No `ESQUISSE_DEFAULT_ROUNDS` env var is introduced. The round count is per-call
only (via `adversarialInput.Rounds`); process-level defaults are out of scope.

### New Default 5-Slot Pool

All model IDs verified from `crush models` output (2026-04-18). All five models
have `can_reason: true` in Catwalk. Pool order is the **static definition order**
used as the source for shuffling — individual invocations randomize the order via
`buildRotationOrder`; the slot numbers below are no longer used for sequential rotation.

| Pool index | Default `provider/model` | Provider | Family | Subject to Copilot Enterprise? |
|------------|--------------------------|----------|--------|--------------------------------|
| 0 | `copilot/claude-sonnet-4.6` | GitHub Copilot | copilot | Yes |
| 1 | `gemini/gemini-3.1-pro-preview` | Google Gemini | gemini | No |
| 2 | `copilot/gpt-4.1` | GitHub Copilot | copilot | Yes |
| 3 | `vertexai/gemini-3.1-pro-preview` | Google Vertex AI | vertexai | No |
| 4 | `copilot/gpt-4o` | GitHub Copilot | copilot | Yes |

> **ASSUMPTION-1: RESOLVED** — `crush models` confirms `copilot/claude-sonnet-4.6`
> is the correct model ID. No placeholder needed.
>
> **ASSUMPTION-2: RESOLVED** — `gemini/gemini-3.1-pro-preview` and
> `vertexai/gemini-3.1-pro-preview` are both present in the user's Catwalk data.
> Slots 1 and 3 are upgraded from `gemini-2.5-pro` to `gemini-3.1-pro-preview`.
>
> **ASSUMPTION-3 (NEW — open)**: The exact error message emitted by `crush run`
> when a GitHub Copilot Enterprise policy disables a model is unknown. The
> `isModelUnavailable()` function must match against a set of candidate substrings
> (see specification). This set must be validated by attempting to invoke a
> known-disabled model and capturing the actual `crush run` output.

### Functions / Methods

#### `adversarialInput` update (in `adversarial.go`)

```go
// adversarialInput is the input schema for the adversarial_review tool.
type adversarialInput struct {
    PlanSlug    string `json:"plan_slug"    jsonschema:"Plan slug used as state file name"`
    PlanContent string `json:"plan_content" jsonschema:"Full text of the plan to review"`
    // Rounds is the number of consecutive review rounds to run in this call.
    // Default: 5. Minimum: 1. Values < 1 fall back to the default.
    Rounds      int    `json:"rounds,omitempty" jsonschema:"Number of review rounds (default 5, min 1; <1 uses default)"`
}
```

#### `effectiveRounds(n int) int` (in `models.go`)

```go
const defaultRounds = 5
const maxRounds     = 50  // upper bound; prevents runaway resource use

// effectiveRounds returns n if 1 <= n <= maxRounds.
// Values < 1 return defaultRounds. Values > maxRounds return maxRounds
// and log: "esquisse-mcp: rounds=%d exceeds maxRounds=%d, clamping".
func effectiveRounds(n int) int
```

#### `buildModelPool() []string` (in new `models.go`)

```go
// buildModelPool constructs the ordered model pool from env vars and the
// hard-coded defaults, applying the ESQUISSE_ALLOWED_PROVIDERS filter.
//
// IMPORTANT: buildModelPool does NOT call crush models or any external binary.
// It only reads os.Getenv and the hardcoded defaultModels slice.
// The discover_models tool (newDiscoverHandler) is the only component that
// calls `crush models`; these two functions are completely independent.
//
// Precondition: none — safe to call at any time.
// Postcondition: returns a non-empty slice; falls back to the full
// default 5-slot pool if all overridden slots are filtered out.
// The returned strings are in "provider/model" format.
// This function reads os.Getenv at call time — it must be called inside
// the handler closure (at server startup) so the result is captured once
// per process lifetime, not once per request. It does NOT assign to any
// package-level variable; the pool is returned and captured in the
// newAdversarialHandler closure. This satisfies the AGENTS.md
// no-global-state invariant.
func buildModelPool() []string
```

Algorithm:
1. Build base slice from `ESQUISSE_MODEL_SLOT0`…`ESQUISSE_MODEL_SLOT4` env vars,
   falling through to the corresponding default for any unset slot.
   For each slot override: validate that it contains exactly one `/` and that neither
   the provider nor model part is empty. If invalid (e.g. `"vertexai/"`, `"/gpt-4.1"`,
   `"no-slash"`), log a warning and fall through to the default for that slot:
   ```go
   if !strings.Contains(v, "/") || strings.HasSuffix(v, "/") || strings.HasPrefix(v, "/") {
       log.Printf("esquisse-mcp: ESQUISSE_MODEL_SLOT%d=%q is invalid (expected provider/model format), using default", i, v)
       v = defaultModels[i]
   }
   ```
2. Parse `ESQUISSE_ALLOWED_PROVIDERS` into a `map[string]bool`.
   Provider IDs are compared **case-sensitively** — users must supply lowercase
   (e.g. `copilot,gemini,vertexai`), matching the `provider/model` format used
   by `crush models`.
3. If the allowed set is empty, return base slice as-is.
4. For each slot string, extract provider prefix (everything before the first `/`).
   If it is not in the allowed set, remove it from the slice and log
   `"esquisse-mcp: slot N model %q excluded (provider %q not in ESQUISSE_ALLOWED_PROVIDERS)"`.
5. **If the resulting slice is empty (all slots filtered)**:
   - Log a warning at level WARN: `"esquisse-mcp: ESQUISSE_ALLOWED_PROVIDERS filtered all slots — falling back to full default pool. Set ESQUISSE_POOL_FALLBACK_STRICT=1 to return an error instead."`
   - If `ESQUISSE_POOL_FALLBACK_STRICT=1` is set, return an empty slice; the
     caller (`newAdversarialHandler`) MUST check `len(pool) == 0` and return
     `mcpErr("all model slots excluded by ESQUISSE_ALLOWED_PROVIDERS — set ESQUISSE_POOL_FALLBACK_STRICT=0 or allow at least one provider")` before entering the fallback loop.
   - Otherwise return the full unfiltered default pool (fail-open). An empty pool
     would make adversarial review impossible; fail-open is safer than blocking
     all reviews on a misconfiguration.

> **Logging env var values:** The `log.Printf` call in `buildModelPool` logs
> invalid slot values with the `%q` verb (`log.Printf("... ESQUISSE_MODEL_SLOT%d=%q ...", i, v)`).
> The `%q` format escapes all control characters and newlines, preventing log injection.
> Env vars are server-side configuration (not user input over the wire), so exposure
> risk is confined to the server process environment. No scrubbing beyond `%q` is required.

> **`ESQUISSE_POOL_FALLBACK_STRICT` precedence:** strict mode takes precedence over
> fail-open. Evaluation order: (1) if all slots are filtered AND `ESQUISSE_POOL_FALLBACK_STRICT=1`
> is set, return empty slice → handler returns `mcpErr`. (2) if all slots are filtered and
> strict is not set, return full unfiltered default pool with warning log. There is no
> ambiguity: strict mode always wins.

> **Design Note — Fail-Open Policy:** Failing open (returning to unfiltered
> defaults when all slots are filtered) is a deliberate UX choice: a
> misconfigured `ESQUISSE_ALLOWED_PROVIDERS` should not silently disable all
> adversarial reviews. Users who require strict enforcement (e.g. no model
> outside the allowed set must ever be called) can set
> `ESQUISSE_POOL_FALLBACK_STRICT=1`. This env var and the two code paths must
> both be documented in README.md.

#### `familyInterleaveShuffle(pool []string, rng *rand.Rand) []string` (in `models.go`)

```go
// familyInterleaveShuffle returns a permutation of pool such that consecutive
// elements are from different provider families whenever possible.
//
// Algorithm (linear scan, O(n²) total — pool size ≤5 in all known uses):
//   1. Group models by provider prefix (everything before the first '/').
//   2. Shuffle within each group using rng.
//   3. Greedily pick the model from the group with the most remaining models,
//      skipping the last-picked group unless it is the only group left.
//      Tie-breaking: pick the first qualifying group in alphabetical order
//      (deterministic given the rng-shuffled within-group order).
//   4. Return the assembled permutation.
//
// Time complexity: O(n²) worst-case (linear scan per pick, n picks).
// Acceptable for pool sizes ≤ 10 (default pool is 5 models).
//
// Pool < 3 families degradation: if all pool models are from the same family
// (e.g. pool=["copilot/a", "copilot/b"]), the algorithm cannot interleave
// across families and produces any permutation — this is acceptable, documented,
// and correct. "Best-effort family alternation" means the guarantee only holds
// when the pool contains ≥2 distinct families. With 1 family, within-group
// shuffle still provides model diversity.
//
// With the default 5-model pool (copilot×3, gemini×1, vertexai×1), a typical
// result is: [copilot/X, gemini/Y, copilot/Z, vertexai/W, copilot/V].
// Precondition: pool is non-empty. rng must not be nil.
// Postcondition: returned slice has len == len(pool); each element appears exactly once.
func familyInterleaveShuffle(pool []string, rng *rand.Rand) []string
```

#### `buildRotationOrder(pool []string, rounds int) []string` (in `models.go`)

```go
// buildRotationOrder generates an ordered slice of length `rounds` for driving
// the multi-round adversarial review loop. Each call produces a fresh random order.
//
// Algorithm:
//   batchSize = 5 (the re-randomization boundary)
//   rng = rand.New(rand.NewSource(time.Now().UnixNano()))  // non-security use
//   For each batch until len(result) >= rounds:
//     batch = familyInterleaveShuffle(pool, rng)  // len(pool) items, family-interleaved
//     If pool has fewer than batchSize models, repeat the shuffled pool cyclically
//       until batchSize items are collected (avoiding consecutive same model at
//       the wrap boundary by swapping if needed).
//     Batch-boundary constraint: if result is non-empty and result[last] == batch[0],
//       swap batch[0] with batch[1] (if available).
//     Append min(batchSize, rounds-len(result)) items from batch to result.
//   Return result[:rounds].
//
// Invariants guaranteed:
//   - len(result) == rounds
//   - No two consecutive elements are identical (across batch boundaries, enforced
//     by the swap; within a batch, guaranteed by familyInterleaveShuffle when
//     pool has no duplicates)
//   - Best-effort family alternation: no consecutive same-family pair when
//     the pool composition allows it
//   - Re-randomizes every batchSize (5) rounds — fresh shuffle for each batch
//
// Precondition: pool is non-empty. rounds >= 1.
// Postcondition: len(result) == rounds.
func buildRotationOrder(pool []string, rounds int) []string
```

#### `worstVerdict(verdicts []string) string` (in `models.go`)

```go
// worstVerdict returns the most severe verdict in the slice.
// Severity order: FAILED > CONDITIONAL > PASSED > "" (unknown/empty).
// Returns "" if verdicts is empty or all entries are empty.
func worstVerdict(verdicts []string) string
```

### runOneRound(ctx, pool, targetModel, preamble, planContent, tmpDir) (usedModel, output string, err error)

```go
// runOneRound runs a single adversarial review round against targetModel,
// falling back to other pool models if targetModel is enterprise-blocked.
//
// Steps:
//   1. Create a uniquely-named temp file inside tmpDir using os.CreateTemp(tmpDir, "round-*.txt")
//      with mode 0600. Write preamble + planContent to it. Close before passing to RunCrush.
//      Delete this specific file before returning (tmpDir itself is owned by the caller).
//   2. Call RunCrush(ctx, targetModel, tmpFile). On ExitCode==0: return (targetModel, output, nil).
//   3. If isModelUnavailable(output): log and try each remaining model in pool in order.
//   4. If a fallback model succeeds: return (fallbackModel, output, nil).
//   5. If all pool models are enterprise-blocked: return ("", "", errAllModelsUnavailable).
//
// errAllModelsUnavailable actionable message:
//   "all %d models in the pool are unavailable (enterprise policy may be blocking them);
//    check ESQUISSE_ALLOWED_PROVIDERS or run discover_models to verify pool availability"
//   6. Any non-zero exit for a non-availability reason: return ("", output, fmt.Errorf(...)).
//
// Using os.CreateTemp ensures each round writes to a unique filename within tmpDir
// even though rounds are sequential — prevents any naming collision if a deletion
// fails (deferred). The caller owns tmpDir and calls os.RemoveAll at call end.
func runOneRound(ctx context.Context, pool []string, targetModel, preamble, planContent, tmpDir string) (usedModel, output string, err error)
```

#### `isModelUnavailable(output string) bool` (in `models.go`)

```go
// isModelUnavailable reports whether crush run output indicates that the
// requested model is blocked by a provider policy (e.g. GitHub Copilot
// Enterprise does not permit this model for the organisation).
// It does NOT match general API errors, auth failures, or network errors.
//
// Precondition: output is the combined stdout+stderr of a failed crush run.
// Postcondition: returns true only for policy/availability errors; returns
// false for all other failures (auth, network, etc.).
func isModelUnavailable(output string) bool
```

Matches (case-insensitive substring search) against a candidate set:
```go
var modelUnavailablePatterns = []string{
    "model is not supported",
    "model not supported",
    "not available for your organization",
    "not enabled for your organization",
    "access to this model",
    "model access denied",
    "this model is not available",
}
```

> ASSUMPTION-3 (open): the exact Copilot Enterprise error text is unverified.
> These patterns are candidates. The implementer must validate by calling a
> known-disabled model and capturing `crush run`'s actual stderr output, then
> add/adjust patterns accordingly. The test for `isModelUnavailable` must
> include the verified string.

#### `runAdversarialRounds` — multi-round handler (in `adversarial.go`)

Replaces the current single `RunCrush` call. The `pool` is built once at server
startup and captured in the closure. `modelForSlot`, `slotEnvVars`, and
`defaultModels [3]string` are **deleted** from `adversarial.go`.

> **Timeout budget note:** A single 300-second `rctx` context is created via
> `context.WithTimeout(ctx, 300*time.Second)` **before** calling `buildRotationOrder`
> or entering the round loop. Each round shares this budget. The `rctx` is passed
> to each `runOneRound` call; if the budget is exhausted mid-session, the call to
> `runCrushFn(ctx, ...)` returns immediately with `ctx.Err() == context.DeadlineExceeded`
> — there is no hung goroutine. `runAdversarialRounds` treats this as a round error
> and returns `mcpErr("round %d/%d failed: %v", ...)`, which surfaces the timeout to
> the caller. Model-unavailability errors from Copilot Enterprise are expected to
> fail in < 10s. The 300s ceiling is a documented tradeoff: sufficient for 5 rounds
> at ~60s each, acceptable for typical use.

> **State is a value copy — no partial mutation risk:** `ReadState` returns a
> `ReviewState` value (not a pointer). `state.Iteration += rounds` and the other
> field assignments below modify this local copy only. If `WriteState` fails, the
> function returns an error and the in-memory copy is discarded — no persistent
> state is written. There is no partial mutation of the on-disk state file.

> **extractVerdict:** The existing `verdictRe` regex in `adversarial.go` is reused:
> ```go
> var verdictRe = regexp.MustCompile(`(?m)^Verdict:\s*(PASSED|CONDITIONAL|FAILED)`)
> func extractVerdict(output string) string {
>     m := verdictRe.FindStringSubmatch(output)
>     if len(m) < 2 { return "" }
>     return m[1]
> }
> ```
> When `extractVerdict` returns `""` for a round (malformed output), that round is
> **logged as a warning** but does NOT by itself fail the call. `worstVerdict` treats
> `""` as the lowest priority, so valid verdicts from other rounds still contribute.
> The post-loop guard `if worstVerdict(verdicts) == ""` catches the case where ALL
> rounds produced no valid verdict line. Mixed cases (some rounds valid, some `""`) are
> correctly handled: the overall verdict is the worst of the valid rounds.
> Implementation note in the loop: `if verdict == "" { log.Printf("esquisse-mcp: round %d produced no valid Verdict: line", roundNum) }`

> **`runCrushFn` and the no-global-state invariant:** `var runCrushFn = RunCrush`
> is a package-level variable but its value is **never written in production code**
> after package initialization. In production, it is read-only after startup.
> Tests use `SetRandSource` and a similar `runCrushFn` setter to inject mocks;
> they must restore the original value after each test (test isolation). This
> pattern is consistent with the no-global-state invariant's intent (no `init()`
> side effects, no shared mutable state in concurrent code). The stdio MCP server
> processes one request at a time — no concurrent access to `runCrushFn` is possible.
> `mcp.AddTool` is the correct API for the `github.com/modelcontextprotocol/go-sdk`
> SDK — verified from the existing `tools.go` in `esquisse-mcp/`.

> **RunCrush injection for tests:** `runOneRound` calls `RunCrush` via a
> package-level variable that defaults to the real implementation:
> ```go
> var runCrushFn = RunCrush  // replaceable in tests
> ```
> Test files replace `runCrushFn` with a mock before calling `runOneRound`.
> `runOneRound` calls `runCrushFn(ctx, model, inputFile)` rather than `RunCrush`
> directly. This is documented in `models_test.go` with a `TestMain` or per-test
> restore pattern.

> **RunCrush signature** (existing function, documented here for implementer reference):
> ```go
> type RunResult struct {
>     Output   string  // combined stdout+stderr
>     ExitCode int
> }
> // RunCrush shells out to `crush run --model <model> --quiet < tmpFile`.
> // Returns (result, nil) if crush is found in PATH.
> // Returns ("", err) only if the crush binary itself cannot be launched
> // (e.g. not in PATH). A non-zero ExitCode from crush is NOT an error here —
> // the caller inspects ExitCode and result.Output directly.
> func RunCrush(ctx context.Context, model, inputFile string) (RunResult, error)
> ```

> **Deterministic rng injection for tests:** `buildRotationOrder` uses an
> internal `var newRand = func() *rand.Rand { return rand.New(rand.NewSource(time.Now().UnixNano())) }`.
> Tests override this with a package-level setter:
> ```go
> // SetRandSource replaces the rng factory used by buildRotationOrder.
> // Call in TestMain or test setup only. Not goroutine-safe.
> func SetRandSource(src rand.Source) { newRand = func() *rand.Rand { return rand.New(src) } }
> ```
> Probabilistic acceptance criteria (e.g. "100 calls produce ≥2 distinct results")
> are tested using the real time-seeded rng. Structural invariant criteria
> (len==rounds, no consecutive identical) use `SetRandSource` with a fixed seed.

> **Cyclic fill boundary swap scope:** When `len(pool) < batchSize`, the shuffled
> pool is repeated cyclically to fill `batchSize` slots. The "swap if consecutive
> identical" check applies only at each cyclic-wrap boundary (position `len(pool)-1`
> to position `len(pool)` of the expanded batch). If `pool` has only 1 unique model,
> all slots are identical — this is unavoidable and is documented as acceptable
> behaviour ("pool of size 1: single model repeats every round" in Known Risks).
> `buildModelPool` does NOT deduplicate the pool; if two `ESQUISSE_MODEL_SLOT*` vars
> are set to the same `provider/model` string, that model may appear consecutively.
> This is the user's configuration error, not a code bug.

```
rctx, cancel := context.WithTimeout(ctx, 300*time.Second)
defer cancel()

rounds   := effectiveRounds(input.Rounds)          // 5 if input.Rounds < 1

if len(pool) == 0 {  // ESQUISSE_POOL_FALLBACK_STRICT=1 with all slots filtered
    return mcpErr("all model slots excluded by ESQUISSE_ALLOWED_PROVIDERS")
}

rotOrder := buildRotationOrder(pool, rounds)        // fresh random order every call

tmpDir, err := os.MkdirTemp("", "esquisse-review-*")
if err != nil { return mcpErr("failed to create tmpDir: %v", err) }
defer func() {
    if rerr := os.RemoveAll(tmpDir); rerr != nil {
        log.Printf("esquisse-mcp: failed to remove review tmpDir %q: %v", tmpDir, rerr)
    }
}()

date := time.Now().UTC().Format("2006-01-02")
var roundOutputs []string
var usedModels   []string
var verdicts     []string

for roundIdx := 0; roundIdx < rounds; roundIdx++ {
    targetModel := rotOrder[roundIdx]
    roundNum    := roundIdx + 1

    preamble := fmt.Sprintf(
        "You are adversarial reviewer for round %d of %d. "+
        "Apply the 7-attack protocol to the plan below.\n"+
        "Write your review report to %s/.adversarial/reports/review-%s-iter%d-r%d-%s.md\n"+
        "Do NOT write the state file — the handler writes it after all rounds complete.\n"+
        "The final line of your report MUST be: Verdict: PASSED|CONDITIONAL|FAILED\n",
        roundNum, rounds,
        projectRoot, date, state.Iteration+roundIdx, roundNum, input.PlanSlug,
        // Note: state.Iteration+roundIdx uses Go int arithmetic. At current usage
        // rates (iterations in the hundreds) overflow is astronomically unlikely.
        // If it did wrap (Go int wraps at math.MaxInt64 on 64-bit), the report
        // filename would contain a negative number — harmless cosmetic issue.
    )

    usedModel, output, err := runOneRound(rctx, pool, targetModel, preamble,
                                          input.PlanContent, tmpDir)
    if err != nil {
        return mcpErr("round %d/%d failed: %v", roundNum, rounds, err)
    }

    verdict := extractVerdict(output)
    roundOutputs = append(roundOutputs,
        fmt.Sprintf("=== Round %d/%d — %s ===\n%s", roundNum, rounds, usedModel, output))
    usedModels = append(usedModels, usedModel)
    verdicts   = append(verdicts, verdict)
}

// Validate: at least one round must have produced a valid verdict.
if worstVerdict(verdicts) == "" {
    return mcpErr("no valid Verdict: line found in any round output")
}

// Write state once after all rounds complete (value copy — no partial mutation).
state.Iteration    += rounds
state.LastModel     = usedModels[len(usedModels)-1]
state.LastVerdict   = worstVerdict(verdicts)
state.LastReviewDate = date
if err := WriteState(projectRoot, state); err != nil {
    return mcpErr("failed to write state: %v", err)
}

summary := fmt.Sprintf(
    "=== Summary ===\nRounds: %d\nModels: %s\nVerdicts: %s\nOverall: %s",
    rounds,
    strings.Join(usedModels, ", "),
    strings.Join(verdicts, ", "),
    state.LastVerdict,
)
fullOutput := strings.Join(roundOutputs, "\n\n") + "\n\n" + summary
return &mcp.CallToolResult{
    Content: []mcp.Content{&mcp.TextContent{Text: fullOutput}},
}, nil, nil
```

> **Concurrency note — stdio MCP server is single-threaded:** `esquisse-mcp` uses
> the MCP Go SDK in stdio transport mode. The stdio MCP server processes one tool
> call at a time (request/response, no concurrent dispatch). `state.Iteration +=
> rounds` and `WriteState` execute in a single goroutine with no concurrent
> `adversarial_review` calls possible. No file lock or mutex is required.
> This assumption must be re-evaluated if the server is ever upgraded to a
> streamable HTTP transport.

> **`ReadState` zero-value behaviour:** If the state file is absent (first use),
> `ReadState` returns a zero-value `ReviewState` with `Iteration=0`, `LastVerdict=""`,
> etc. This is the correct behaviour for a first call: `Iteration=0` causes
> `buildRotationOrder` to generate the first batch starting from a fresh random
> shuffle. If the state file exists but is corrupt (invalid JSON), `ReadState`
> returns an error; the handler returns `mcpErr` and does not proceed.

> **No per-round state write**: the handler owns state file writing after all
> rounds complete. The preamble instructs each crush instance to write only its
> review report (to `.adversarial/reports/`), NOT the state file. This prevents
> intermediate rounds from overwriting state mid-call.

#### `newDiscoverHandler(projectRoot string)` (in `models.go`)

```go
type discoverInput struct {
    // Filter is an optional case-insensitive substring to apply to the
    // provider/model list. Empty string means no filter (return all).
    // Maximum length: 200 characters. Values longer than 200 characters
    // are rejected with IsError=true.
    // Matching uses strings.Contains(strings.ToLower(model), strings.ToLower(filter))
    // — NOT a regex. User-supplied filter values are treated as literal strings.
    Filter string `json:"filter" jsonschema:"Optional case-insensitive substring to filter provider/model strings. Empty means no filter. Max 200 chars. Literal string match, not regex."`
}

// newDiscoverHandler returns a handler for the discover_models tool.
// It shells out to crush models (piped) to list available provider/model
// strings, applies ESQUISSE_ALLOWED_PROVIDERS, and optionally filters by
// the caller-supplied substring.
//
// Security: no plan_slug is involved — this handler does not read or write
// any state file, so there is no path-traversal risk.
// Precondition: crush must be in PATH.
// Postcondition: returns a JSON array of "provider/model" strings.
// Error: if crush is not found or exits non-zero, returns IsError=true.
func newDiscoverHandler() func(context.Context, *mcp.CallToolRequest, discoverInput) (*mcp.CallToolResult, any, error)
```

Implementation:
1. Validate `input.Filter`: if `len(input.Filter) > 200`, return `mcpErr("filter too long")` immediately.
2. **Preflight check**: resolve `crush` binary path via `exec.LookPath("crush")`. If not found, return `mcpErr("crush binary not found in PATH: install crush and ensure it is in PATH")` immediately — do not proceed.
3. Run `exec.CommandContext(ctx, crushPath, "models")` with stdout piped (not a TTY
   → Crush outputs `provider/model` per line, no tree formatting). The model
   name is passed to the OS as a fixed `"models"` argument — no user input is
   interpolated into the command. No shell is involved (`exec.Command`, not
   `exec/sh -c`).
4. If `cmd.Run()` returns non-zero exit or a non-nil error, return `mcpErr("crush models failed: %v", err)`. **If `crush models` succeeds (exit 0) but outputs no lines or no valid `provider/model` lines (containing exactly one `/`), this is NOT an error** — return an empty JSON array. This is the correct behavior when no models are installed or all are filtered.
5. Split output by newlines. Apply `ESQUISSE_ALLOWED_PROVIDERS` filter (same
   logic as `buildModelPool()`; case-sensitive provider prefix). Apply
   `input.Filter` substring filter (case-insensitive).
6. Return JSON array of matching strings as a `TextContent` response.
7. **Timeout:** Use a dedicated sub-context: `discoverCtx, cancel := context.WithTimeout(ctx, 10*time.Second); defer cancel()`. Pass `discoverCtx` to `exec.CommandContext`. The 10s budget is separate from the parent ctx so that a slow `crush models` invocation does not consume the parent request budget.

> **`newDiscoverHandler` input type safety:** `discoverInput.Filter` is a `string`
> field. Invalid JSON types (e.g. integer passed for `filter`) cause MCP SDK
> unmarshaling to return an error before the handler is called — no runtime panic.
> `adversarialInput.Rounds` is an `int`; a non-integer JSON value (e.g. `"five"`)
> also causes MCP SDK unmarshaling failure before handler entry. Extreme values like
> `math.MaxInt32` are handled by `effectiveRounds` which clamps to `maxRounds=50`.

### can_reason Filtering — Design Note

`crush models` (piped) outputs only `provider/model` identifiers, not model
metadata. The `can_reason` field lives in the Catwalk model structs embedded in
the Crush binary, not in the `crush models` CLI output.

**For this task**: `can_reason` filtering is achieved **by the user's choice of
default slots** — all 5 defaults are known reasoning-capable models. No code-level
`can_reason` filter is implemented in this task. A future P3 task may add
`charm.land/catwalk` as a dependency to `esquisse-mcp` and implement
`ESQUISSE_REQUIRE_REASONING=true` filtering.

**Practical guidance for the user**: to ensure only reasoning models are used,
set `ESQUISSE_ALLOWED_PROVIDERS` to your provider IDs (e.g.
`copilot,gemini,vertexai`) and override `ESQUISSE_MODEL_SLOT*` to the specific
reasoning-capable models you have verified via `crush models`. The default pool
already satisfies this constraint.

### Configuration Changes — `README.md` and MCP `env` block

New README model pool table (no slot/iteration column — order is randomized per call):

| Pool entry | Default model | Override env var | Family |
|------------|---------------|------------------|--------|
| 0 | `copilot/claude-sonnet-4.6` | `ESQUISSE_MODEL_SLOT0` | copilot |
| 1 | `gemini/gemini-3.1-pro-preview` | `ESQUISSE_MODEL_SLOT1` | gemini |
| 2 | `copilot/gpt-4.1` | `ESQUISSE_MODEL_SLOT2` | copilot |
| 3 | `vertexai/gemini-3.1-pro-preview` | `ESQUISSE_MODEL_SLOT3` | vertexai |
| 4 | `copilot/gpt-4o` | `ESQUISSE_MODEL_SLOT4` | copilot |

New env block in the `crush.json` example:
```json
"env": {
  "ESQUISSE_ALLOWED_PROVIDERS": "copilot,gemini,vertexai",
  "ESQUISSE_MODEL_SLOT0": "copilot/claude-sonnet-4.6",
  "ESQUISSE_MODEL_SLOT1": "gemini/gemini-3.1-pro-preview",
  "ESQUISSE_MODEL_SLOT2": "copilot/gpt-4.1",
  "ESQUISSE_MODEL_SLOT3": "vertexai/gemini-3.1-pro-preview",
  "ESQUISSE_MODEL_SLOT4": "copilot/gpt-4o"
}
```

Note: Each call to `adversarial_review` randomizes the order with family-interleaving.
If a Copilot model is disabled by enterprise policy, `runOneRound` automatically
falls back to another pool model before moving to the next round.

### Security Note — Path Traversal

The `adversarial_review` handler calls `ReadState(projectRoot, input.PlanSlug)`
and `WriteState(projectRoot, input.PlanSlug, state)`. Both of these functions
already call `validateSlug(slug)` internally, which rejects any slug containing
`/`, `\`, or path traversal sequences. **No new validateSlug call is needed** —
the existing enforcement in `state.go` covers all slug → file path conversions.

The `discover_models` handler does not accept a `plan_slug` parameter at all and
does not touch the `.adversarial/` directory. There is no path-traversal surface
in this handler.

This was flagged as Critical by Adversarial-r0 (iter 0). It is a **false
positive**: the validation already exists and is inherited by all new code paths
that call `ReadState`/`WriteState`.

### Known Risks / Failure Modes

| Risk | Mitigation |
|------|-----------|
| ASSUMPTION-3: exact Copilot Enterprise "model disabled" error text is unknown | `isModelUnavailable()` uses a candidate pattern set; **hard merge gate** (see Acceptance Criteria): PR must not merge until the verified string is captured and added to both `models.go` and `models_test.go` |
| All Copilot slots (0, 2, 4) disabled by enterprise policy | `runOneRound` tries all pool models per round; slots 1 and 3 are Gemini/VertexAI — at least one will succeed |
| `ESQUISSE_ALLOWED_PROVIDERS` filters all 5 slots | `buildModelPool` falls back to unfiltered defaults with log warning (fail-open). Set `ESQUISSE_POOL_FALLBACK_STRICT=1` to convert this to an error (fail-closed). |
| `isModelUnavailable` matches a genuine auth failure and loops | Pattern set is conservative; auth failures have different error text. If patterns match an auth error, `runOneRound` exhausts all pool models and returns `errAllModelsUnavailable`, which is a hard error that stops the multi-round call. |
| `crush models` subprocess hangs | 10-second timeout on `discover_models` tool |
| Pool of size 1 | Allowed — `familyInterleaveShuffle` and `buildRotationOrder` handle it correctly; the single model is used for every round |
| Backward compatibility: existing 3-slot `ESQUISSE_MODEL_SLOT0/1/2` overrides | Preserved — slots 3 and 4 default to new defaults when not set |
| Pool exhaustion within a round — state file stale | `runAdversarialRounds` returns an error for that round (hard-fails the call). **`state.Iteration` is NOT advanced for a failed call** — this is intentional: no review was completed, so no iteration should be counted. WriteState is only called after all rounds complete successfully. |
| 300s timeout shared across all rounds | Model-unavailability errors fail fast (< 10s). Slow-failing models consume budget; worst-case 300s call produces a timeout error. Documented tradeoff. |
| Non-reproducible rotation (time-seeded rng) | By design — fresh random order per call maximises reviewer diversity. Tests use a deterministic mock rng injected via test-only constructor. |
| Cross-batch boundary duplicate model | Detected by `buildRotationOrder`'s boundary-check and corrected by swapping first two elements of the next batch. |

## Files

| Path | Action | What changes |
|------|--------|--|
| `esquisse-mcp/adversarial.go` | Modify | Delete `modelForSlot`, `slotEnvVars`, `defaultModels [3]string`, `slot := state.Iteration % 3`; add `defaultModels []string` (5 entries); add `Rounds int` to `adversarialInput`; add `runAdversarialRounds` multi-round loop; aggregate verdict via `worstVerdict`; advance `Iteration` by `rounds`; write state once after all rounds |
| `esquisse-mcp/models.go` | Create | `buildModelPool()`, `isModelUnavailable()`, `modelUnavailablePatterns`, `effectiveRounds()`, `defaultRounds`, `familyInterleaveShuffle()`, `buildRotationOrder()`, `worstVerdict()`, `runOneRound()`, `newDiscoverHandler()`, `discoverInput` |
| `esquisse-mcp/tools.go` | Modify | Register `discover_models` tool via `mcp.AddTool`; **update `adversarial_review` tool Description string** to remove the stale "iteration % 3" reference and describe the randomized multi-round rotation |
| `esquisse-mcp/README.md` | Modify | Update model pool table (5 entries, no slot/iteration column), add `ESQUISSE_ALLOWED_PROVIDERS` docs (case-sensitive, lowercase), add `ESQUISSE_POOL_FALLBACK_STRICT` docs, document multi-round default (5), document randomized family-interleaved rotation, document enterprise fallback, add `discover_models` tool docs, add migration note |
| `esquisse-mcp/models_test.go` | Create | Tests for `buildModelPool()` (with/without filter, zero-pool fallback, slot format validation); `isModelUnavailable()` (known patterns, non-matching); `effectiveRounds()` (< 1, == 1, > 1); `familyInterleaveShuffle()` (no consecutive same-family when avoidable); `buildRotationOrder()` (len == rounds, no consecutive identical, re-randomizes each batch, boundary swap, rounds > 5); `worstVerdict()`; `runOneRound()` with mock `RunCrush` |

## Acceptance Criteria

- [ ] `go test -count=1 ./...` passes in `esquisse-mcp/`
- [ ] `ESQUISSE_ALLOWED_PROVIDERS=copilot,gemini,vertexai` with no slot overrides → pool contains all 5 models; no models removed
- [ ] `ESQUISSE_ALLOWED_PROVIDERS=copilot` → gemini and vertexai models removed; pool has 3 entries; log line per removed model
- [ ] `ESQUISSE_ALLOWED_PROVIDERS=openai` → all 5 default models removed; fallback to full unfiltered pool; warning logged
- [ ] `ESQUISSE_MODEL_SLOT0=openai/gpt-4.1 ESQUISSE_ALLOWED_PROVIDERS=copilot` → slot 0 override excluded; copilot slots 2 and 4 survive
- [ ] Old 3-slot env vars (`ESQUISSE_MODEL_SLOT0/1/2`) continue to work without `ESQUISSE_ALLOWED_PROVIDERS`
- [ ] **[HARD MERGE GATE — ASSUMPTION-3]** Before merging: implementer invokes a known-disabled Copilot model, captures the exact `crush run` stderr output, adds the verified substring to `modelUnavailablePatterns` in `models.go`, and adds a test case in `models_test.go` using the verified string. The PR description must include the captured output verbatim.
- [ ] `isModelUnavailable()` returns `true` for the verified Copilot Enterprise disabled-model error string
- [ ] `isModelUnavailable()` returns `false` for a generic auth error string and a network error string
- [ ] `effectiveRounds(0)` == 5; `effectiveRounds(-1)` == 5; `effectiveRounds(1)` == 1; `effectiveRounds(3)` == 3; `effectiveRounds(51)` == 50 with log containing "rounds=51 exceeds maxRounds=50"
- [ ] `adversarial_review` with `rounds=1` runs exactly 1 crush invocation; `state.Iteration` advances by 1
- [ ] `adversarial_review` with `rounds=5` (default) runs 5 crush invocations; `state.Iteration` advances by 5
- [ ] `adversarial_review` with `rounds=0` or `rounds` omitted runs 5 invocations (default)
- [ ] `adversarial_review` with `rounds=51` clamps to 50, logs warning with actual value, runs 50 invocations
- [ ] `adversarial_review` with `rounds=7` runs 7 invocations; model order re-randomizes after the first 5
- [ ] `discover_models` Filter matching uses `strings.Contains(strings.ToLower(model), strings.ToLower(filter))` — literal string, not regex; filter value `".*"` matches only models containing the literal substring `".*"`
- [ ] `buildRotationOrder(pool, rounds)` returns a slice of length `rounds`
- [ ] `buildRotationOrder` never produces two consecutive identical model strings
- [ ] `buildRotationOrder` with default 5-model pool and `rounds=5` produces no consecutive same-family pair (copilot adjacent to copilot) unless forced by pool composition
- [ ] `buildRotationOrder` called twice with the same pool and rounds produces different orderings. Test: create two instances with different fixed seeds via `SetRandSource` and assert the results differ (avoids nanosecond-collision flakiness of the time-seeded rng). Additionally: run 20 calls with the real rng and assert ≥2 distinct orderings (acceptable probabilistic upper bound).
- [ ] `familyInterleaveShuffle` with the default 5-model pool (copilot×3, gemini×1, vertexai×1) produces at most 2 consecutive copilot slots (when the 3 copilot models must be adjacent at the end)
- [ ] `worstVerdict(["PASSED", "CONDITIONAL", "PASSED"])` == `"CONDITIONAL"`
- [ ] `worstVerdict(["PASSED", "FAILED", "CONDITIONAL"])` == `"FAILED"`
- [ ] `worstVerdict([])` == `""`
- [ ] Enterprise fallback: when primary model for a round returns model-unavailable exit, `runOneRound` retries another pool model; `LastModel` in state file records the model that actually succeeded for the final round
- [ ] Enterprise fallback: when all pool models are unavailable for a round, `runAdversarialRounds` returns `IsError=true`; `state.Iteration` is NOT advanced
- [ ] `discover_models` tool returns `provider/model` list from `crush models` output; respects `ESQUISSE_ALLOWED_PROVIDERS`; handles `crush` not in PATH (`IsError=true`)
- [ ] `discover_models` rejects `Filter` longer than 200 characters with `IsError=true`
- [ ] Combined output format: each round section starts with `=== Round N/M — provider/model ===`; a `=== Summary ===` section appears at the end listing models used, per-round verdicts, and overall verdict
- [ ] README model pool table has verified IDs: `copilot/claude-sonnet-4.6`, `gemini/gemini-3.1-pro-preview`, `copilot/gpt-4.1`, `vertexai/gemini-3.1-pro-preview`, `copilot/gpt-4o`
- [ ] README includes migration note for users of old OpenAI/Anthropic defaults
- [ ] README documents multi-round default (5), the `rounds` parameter, and randomized rotation
- [ ] `ESQUISSE_POOL_FALLBACK_STRICT=1` causes `adversarial_review` to return error (fail-closed) when all slots filtered
- [ ] `adversarial_review` where all rounds return malformed output (no `Verdict:` line) → tool returns `IsError=true`; state NOT written
- [ ] Rounds where `extractVerdict` returns `""` are logged as warnings; valid rounds in the same call still contribute to the aggregate verdict
- [ ] `buildRotationOrder` structural invariants (len==rounds, no consecutive identical) tested with `SetRandSource(rand.NewSource(42))`
- [ ] `buildRotationOrder` distinct-ordering test: call with seed=42 vs seed=99 via `SetRandSource`; assert results differ (avoids nanosecond-collision flakiness). Additionally: 20 calls with real rng assert ≥2 distinct orderings.
- [ ] `runOneRound` uses `os.CreateTemp(tmpDir, "round-*.txt")` — unique filename per round; confirmed by test that two sequential calls do not produce the same filename
- [ ] `modelForSlot`, `slotEnvVars`, and the `slot := state.Iteration % 3` pattern are absent from the final `adversarial.go` (grep confirms deletion)

## Session Notes

**2026-04-18 planning session:**
- User has active licences for: GitHub Copilot (Enterprise), Google Gemini,
  Google Vertex AI.
- Requested models: Gemini 3.1 Pro (coding-optimized), Gemini 2.5 Pro, Copilot
  Claude Sonnet 4.6, Copilot GPT-4.1, Copilot GPT-4o.
- ASSUMPTION-1: RESOLVED — `crush models` confirms `copilot/claude-sonnet-4.6`.
- ASSUMPTION-2: RESOLVED — `gemini/gemini-3.1-pro-preview` and
  `vertexai/gemini-3.1-pro-preview` are present. Slots 1 and 3 upgraded from
  `gemini-2.5-pro` to `gemini-3.1-pro-preview`.
- ASSUMPTION-3 (OPEN): Exact error text from `crush run` when a GitHub Copilot
  Enterprise policy disables a model is unknown. Implementer must invoke a
  known-disabled model and capture the actual output to finalize
  `isModelUnavailable()` patterns and their test.
- New constraint from user: not all Copilot models are enabled in their GitHub
  Copilot Enterprise account. Per-attempt fallback added to plan (slots 1 and 3
  are Gemini/Vertex and not subject to Copilot policy — guaranteed fallback path).
- Dynamic `can_reason` filtering deferred to a future P3 task (would require adding
  `charm.land/catwalk` to `esquisse-mcp` go.mod and using `embedded.GetAll()`).
  Not in scope for this task.
- `crush models` (piped, non-TTY) already outputs `provider/model` per line —
  no Crush changes required.
- Conservative choice on zero-pool fallback: fail open (use unfiltered defaults)
  rather than blocking adversarial review entirely. Strict mode available via
  `ESQUISSE_POOL_FALLBACK_STRICT=1` for orgs that need it.
- Fallback loop fails fast on hard errors (binary missing, auth failure, network
  error) and only retries on model-availability policy errors.
- **Round 1 revision (2026-04-18):** Addressed all major findings from
  Adversarial-r0 FAILED verdict: (1) validateSlug security note added (false
  positive — already enforced); (2) ASSUMPTION-3 upgraded to hard merge gate;
  (3) fail-open documented as deliberate with ESQUISSE_POOL_FALLBACK_STRICT opt-out;
  (4) shared 300s timeout budget documented; (5) state written on pool exhaustion;
  (6) pool-in-closure explicitly stated to satisfy no-global-state invariant;
  (7) discoverInput.Filter validation specified (200-char limit, case-insensitive);
  (8) modelForSlot call site count verification added to acceptance criteria.
- **Round 2 revision (2026-04-18):** Addressed 3 genuine findings from Adversarial-r2
  CONDITIONAL verdict: (1) removed LastVerdict="EXHAUSTED" from exhaustion path —
  gate.go:67 blocks any verdict that is not PASSED/CONDITIONAL, so EXHAUSTED would
  permanently block the session; now only Iteration+1 and LastModel="" are written;
  (2) changed WriteState error from silently ignored to logged; (3) added README
  migration note for users relying on old OpenAI/Anthropic defaults.
- **Scope clarification 2026-04-18 (rounds + randomization):**
  User added: (a) number of review rounds defaults to 5 but is configurable via
  `adversarialInput.Rounds`, minimum 1, values < 1 fall back to default; (b) model
  order is randomized per call with family-interleaving (copilot/gemini/vertexai
  alternated) and a no-consecutive-same-model constraint; (c) rotation order
  re-randomizes every 5 rounds when `rounds > 5`. No process-level env var for
  the rounds default — per-call only. Deterministic `slot := iteration % 3` rotation
  removed in favour of `buildRotationOrder`. `modelForSlot` and `slotEnvVars`
  deleted from `adversarial.go`. Plan status reset to Ready (needs re-review).
- **Round 4 revision (2026-04-18):** Addressed findings from Adversarial-r0 FAILED verdict: (C1) The fallback logic was verified to NOT persist a stale PASSED verdict on pool exhaustion. By returning an error early in `runAdversarialRounds`, the state file is not written at all, avoiding a false-positive gate approval. (L1) `modelForSlot` was confirmed to be completely removed from `adversarial.go` rather than refactored.
- **Round 5 revision (2026-04-18, 6-round session):** Addressed Adversarial-r2 FAILED verdict: (C1 false positive) `newDiscoverHandler` already calls `exec.LookPath` as the preflight check — made more explicit with "Preflight check:" label and actionable error message; empty `crush models` output on success is explicitly NOT an error (return empty array). (C2 false positive) Same LookPath fix. (M1) `adversarialInput.Rounds` type safety documented — MCP SDK rejects non-integer JSON before handler entry; `math.MaxInt32` is handled by `effectiveRounds` clamp. (M2) `%q` verb in log.Printf prevents log injection from env var values — documented. (M3 false positive) `ESQUISSE_POOL_FALLBACK_STRICT` precedence documented explicitly: strict mode always wins over fail-open. (M4) `errAllModelsUnavailable` now has an actionable message with recovery instructions. (M5 false positive) ASSUMPTION-3 patterns + hard merge gate already specified.
- **Round 6 revision (2026-04-18, 6-round session):** Addressed Adversarial-r2 CONDITIONAL verdict: (C1 false positive) Added explicit docstring note that `buildModelPool` does NOT call `crush models` — these are completely independent; only `newDiscoverHandler` calls the crush binary. (C2 partially valid) `runCrushFn` global var is production-immutable — documented; `mcp.AddTool` verified correct from existing tools.go. (M1) `familyInterleaveShuffle` degradation with pool < 3 families documented: falls back gracefully, within-group shuffle still provides model diversity. (M2 false positive) NOT advancing Iteration on exhaustion is INTENTIONAL and explicitly documented as such — reviewer misread the spec. (M3 false positive) 300s budget is a documented tradeoff; context.DeadlineExceeded propagation documented — no hung goroutines.
