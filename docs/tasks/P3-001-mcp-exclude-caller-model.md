# P3-001: Caller-Model Exclusion for adversarial_review in esquisse-mcp

## Status
Status: Done
Depends on: P2-006 (5-slot model pool, multi-round reviews)
Blocks: none

## Goal
Add an optional `exclude_model` parameter to the `adversarial_review` MCP tool so
that a caller agent can exclude its own exact model from the review pool,
guaranteeing that no review round uses the same model as the implementing agent.

## Background

The adversarial review pool contains models from multiple providers (e.g.
`copilot/claude-sonnet-4.6`, `copilot/gpt-4.1`, `gemini/gemini-2.5-pro-preview`).
`familyInterleaveShuffle` reduces consecutive same-family rounds, but does not
prevent the caller's own exact model from appearing in the rotation. If the
implementing agent is `copilot/claude-sonnet-4.6`, a review round assigned to
`copilot/claude-sonnet-4.6` produces zero additional independence — it is the same
reasoning process reviewing itself.

The information needed to fix this is accessible: `crush_info` is an LLM-callable
tool that returns `[model]\nlarge = {provider}/{model} ({provider})\n` in its output.
A caller agent can call `crush_info`, parse the full model ID (e.g.
`copilot/claude-sonnet-4.6`), and pass it as `exclude_model` to `adversarial_review`.

`esquisse-mcp` cannot call `crush_info` itself (it is a Crush internal tool, not
available as a subprocess). The caller agent must supply the value.

## Solution

1. Add `ExcludeModel string` to `adversarialInput` in `adversarial.go`.
2. In `newAdversarialHandler`, after `buildModelPool()`, filter the pool using
   `excludeModelFilter(pool, input.ExcludeModel)` — a new function in `models.go`.
3. `excludeModelFilter` removes the entry that exactly matches `input.ExcludeModel`
   (case-insensitive, after trimming whitespace). Edge cases:
   - Empty string or whitespace-only → trimmed to empty → no-op (pool returned unchanged).
     This is the correct behavior when the caller omits the parameter.
   - Malformed value (contains characters outside `[a-zA-Z0-9_./-]` after trimming)
     → log warning `"esquisse-mcp: exclude_model=%q is malformed, ignoring"` and
     return pool unchanged. The format check uses
     `regexp.MustCompile("^[a-zA-Z0-9_./-]+$")` compiled once at package init
     (not inside the hot path). Note: `/` is permitted because valid model IDs
     include a provider prefix (e.g. `copilot/claude-sonnet-4.6`).
   - Value not matching any pool entry → log at `%q` level, return pool unchanged
     (effectively a no-op; not an error).
   - Value matching the only pool entry (pool would become empty) → **fail-open**:
     log warning `"esquisse-mcp: exclude_model=%q would empty pool; ignoring exclusion"`
     and return the unfiltered pool unchanged.
     **Rationale for fail-open**: blocking the adversarial review entirely is a
     worse outcome than a review with reduced independence. The caller is informed
     via the log. This is consistent with the existing `ESQUISSE_ALLOWED_PROVIDERS`
     policy.
   All non-empty values are logged with Go's `%q` verb, which escapes all control
   characters, newlines, and special characters — this prevents log injection from
   caller-supplied strings.
4. Update the `adversarial_review` tool description in `tools.go` to document the
   `exclude_model` parameter and instruct the caller agent to run `crush_info`
   first to obtain the full model ID string.
5. Update the adversarial-review skill (`skills/adversarial-review/SKILL.md` in
   the parent esquisse repo) to include a Step that runs `crush_info` before
   calling `adversarial_review`, extracts the model name (from the `large = ` line, before ` (`) and the provider (inside `()`), concatenates them as `{provider}/{model}`, and passes it as `exclude_model`.

## Scope

### In Scope
- `esquisse-mcp/adversarial.go`: add `ExcludeModel string` to `adversarialInput`
- `esquisse-mcp/models.go`: add `excludeModelFilter(pool []string, exclude string) []string` with package-level `validModelRe`
- `esquisse-mcp/tools.go`: update `adversarial_review` description to document `exclude_model`
- `esquisse-mcp/models_test.go`: add `TestExcludeModelFilter` (see AC 1–13 for exact cases)
- `esquisse-mcp/AGENTS.md`: add `ExcludeModel` usage to code conventions section
- `esquisse-mcp/README.md`: document `exclude_model` parameter under adversarial_review tool section
- `skills/adversarial-review/SKILL.md`: add `crush_info` step before `adversarial_review` call

### Out of Scope
- Automatic detection of the caller model without an explicit parameter (requires
  Crush API changes not in scope)
- `exclude_provider` (provider-family filtering) — exact-model exclusion is
  stronger and more precise; provider-family exclusion is a future extension
- Changes to `gate_review` or `discover_models` tools
- Changes to `buildModelPool` env var handling
- Changes to `ESQUISSE_POOL_FALLBACK_STRICT` semantics
- Matching only the model-name portion (after `/`) without the provider prefix —
  callers must pass the full `provider/model` string as returned by `crush_info`

## Files

| File | Action | What changes |
|------|--------|--------------|
| `esquisse-mcp/adversarial.go` | Modify | Add `ExcludeModel string` to `adversarialInput`; pass to `excludeModelFilter` after `buildModelPool()` |
| `esquisse-mcp/models.go` | Modify | Add `excludeModelFilter(pool []string, exclude string) []string` with package-level `validModelRe` |
| `esquisse-mcp/models_test.go` | Modify | Add `TestExcludeModelFilter` covering all 13 AC cases |
| `esquisse-mcp/tools.go` | Modify | Update `adversarial_review` description: document `exclude_model` param, instruct caller to run `crush_info` to obtain full model ID |
| `esquisse-mcp/README.md` | Modify | Add `exclude_model` parameter row to adversarial_review tool docs; add "Caller-model independence" best-practice section |
| `esquisse-mcp/AGENTS.md` | Modify | Add `ExcludeModel` usage to code conventions section |
| `skills/adversarial-review/SKILL.md` | Modify | Add Step: run `crush_info`, extract full model ID from `large =` line, pass as `exclude_model` |

## Acceptance Criteria

1. `TestExcludeModelFilter/empty_exclude` — `excludeModelFilter(pool, "")` returns pool unchanged.
2. `TestExcludeModelFilter/removes_matching` — `excludeModelFilter(["copilot/a","gemini/b","copilot/c"], "copilot/a")` returns `["gemini/b","copilot/c"]`.
3. `TestExcludeModelFilter/case_insensitive` — `excludeModelFilter(pool, "Copilot/A")` and `excludeModelFilter(pool, "COPILOT/A")` both produce the same result as `"copilot/a"`.
4. `TestExcludeModelFilter/single_entry_fallback` — `excludeModelFilter(["copilot/a"], "copilot/a")` returns the original pool unmodified (fail-open: pool would be empty).
5. `TestExcludeModelFilter/exact_match_only` — `excludeModelFilter(["copilot/a","copilot/ab"], "copilot/a")` removes only `copilot/a`; `copilot/ab` is kept (exact match required, not prefix or substring).
6. `TestExcludeModelFilter/whitespace_only` — `excludeModelFilter(pool, "   ")` returns pool unchanged (trimmed to empty → no-op).
7. `TestExcludeModelFilter/no_match_noop` — `excludeModelFilter(["gemini/a","vertexai/b"], "copilot/x")` returns pool unchanged; no error returned.
8. `TestExcludeModelFilter/malformed_value` — `excludeModelFilter(pool, "copilot)")` returns pool unchanged (malformed format rejected; no-op).
9. Integration: when `exclude_model="copilot/a"` is passed and the pool contains only `copilot/a`, the tool completes successfully (fail-open fallback) and logs a warning containing `"would empty pool"`.
10. `go test -count=1 -race ./...` passes with no data races.
11. `adversarial_review` tool description in `tools.go` references `crush_info` as
    the source for the full model ID string (e.g. `copilot/claude-sonnet-4.6`) and
    documents that a missing, empty, or malformed value is silently ignored (no-op).
12. `skills/adversarial-review/SKILL.md` includes the `crush_info` step before the
    `adversarial_review` call, with the exact parsing pattern for the full model ID
    (extract the model name before ` (` and the provider inside `()`, then concatenate as `{provider}/{model}`,
    yielding e.g. `copilot/claude-sonnet-4.6`). If the `crush_info` output does not
    contain a `large =` line or the parse fails, the agent **must** log a visible
    note (e.g. `"[warn] could not extract model ID from crush_info; exclude_model
    omitted"`) and pass an empty string (no-op). If multiple `large =` lines appear,
    use the first one.
13. `TestExcludeModelFilter/partial_model_without_provider_noop` —
    `excludeModelFilter(["copilot/a","gemini/b"], "a")` returns pool unchanged;
    `"a"` does not exactly match `"copilot/a"` or `"gemini/b"`. This documents
    that callers must pass the full `provider/model` string, not just the model
    name portion. The function must not panic on this input.

## Session Notes

- Assumption: `excludeModelFilter` uses exact case-insensitive string comparison
  against full pool entries — no helper like `providerOf` is needed.
- Assumption: fail-open policy matches the existing `ESQUISSE_ALLOWED_PROVIDERS`
  behavior (log warning, return unfiltered pool). This is consistent and avoids
  a new flag.
- The `crush_info` output format is `large = {model} ({provider})\n`.
  The caller agent extracts the model (between `large = ` and ` (`) and the provider (inside `()`),
  and concatenates them to yield the full model ID string (e.g. `copilot/claude-sonnet-4.6`).
  The skill step documents this exact parse pattern.
- `exclude_model` is applied **after** `ESQUISSE_ALLOWED_PROVIDERS` filtering.
  If both are set, `ESQUISSE_ALLOWED_PROVIDERS` runs first (at pool construction
  time), then `exclude_model` filters the already-narrowed pool at call time.
- Input validation: `excludeModelFilter` trims whitespace; values empty after
  trimming are a no-op. Values containing characters outside `[a-zA-Z0-9_./-]`
  are rejected as malformed (no-op + warning log). The `/` character is
  explicitly allowed because valid model IDs use `provider/model` format.
  All non-empty values are logged with Go's `%q` verb to prevent log injection.
- Fail-open design rationale: returning an error when the only pool entry matches
  `exclude_model` would block the review entirely. Fail-open with a warning log
  preserves review progress over perfect independence. This is a deliberate design
  choice, consistent with `ESQUISSE_ALLOWED_PROVIDERS` policy.
- Pool entry format constraint: `buildModelPool` (P2-006) already validates that
  every pool entry contains exactly one `/` with non-empty parts on both sides.
  Malformed pool entries (no `/`) cannot enter the pool at runtime. However,
  `excludeModelFilter` must not panic on them: exact comparison with a value like
  `"copilot/claude-sonnet-4.6"` simply won't match `"noslash"`, so no filtering
  occurs and no panic results.
- `crush_info` multiple `large =` lines: the skill step specifies using the first
  occurrence. If the format changes or the line is absent, the agent must emit a
  visible warning rather than silently omitting `exclude_model`.
- Plan revised 2026-04-19: original design used `exclude_provider` (provider-family
  filtering). Changed to `exclude_model` (exact model string) per user direction —
  same-model exclusion is a stronger independence guarantee than same-family exclusion.
  Prior adversarial PASSED verdict (iter=7, exclude_provider design) is invalidated;
  requires new adversarial review before implementation.
- **Round 9 revision (2026-04-19):** Addressed finding from Adversarial-r0 FAILED verdict: (C1) Corrected the false assumption about `crush_info` output format. The plan now explicitly instructs the skill to extract the model and provider separately from `large = {model} ({provider})` and concatenate them into `{provider}/{model}` for the `exclude_model` filter.
- **Goroutine safety (iter=11 clarification):** `excludeModelFilter` is a pure function — it takes its arguments by value, allocates a new slice for results, and reads no global mutable state. It is therefore goroutine-safe and can be called from concurrent goroutines without synchronization. The compiled `validModelRe` is a `*regexp.Regexp`, which is safe for concurrent use (documented in the Go standard library).- **Round 12 FAILED rebuttal (2026-04-19):** Adversarial-r2 returned FAILED with three critical findings, all false positives:
  - C1 (crush_info format stability): Already addressed by AC12 — parse failures cause visible warning + empty string no-op. No plan change needed.
  - C2 (log injection): The plan already states `%q` is used for all non-empty values. In Go, `%q` escapes ALL control characters, newlines, and special characters; it IS the mitigation for log injection, not a vulnerability. Reviewer confused mitigation with vulnerability. No plan change needed.
  - C3 (fail-open contradiction): `ESQUISSE_ALLOWED_PROVIDERS` is itself fail-open (P2-006 design — log warning, return unfiltered pool when constraint would empty pool). There is no contradiction. No plan change needed.
  - M1 (retry logic): `esquisse-mcp` never calls `crush_info`. The caller agent calls it and passes the result. This is explicitly stated in Background and Out of Scope. Retry logic in a component that never calls the tool is nonsensical. No plan change needed.