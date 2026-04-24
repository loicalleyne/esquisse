// Package main — tests for models.go.
package main

import (
"context"
"math/rand"
"strings"
"testing"
)

// --- effectiveRounds ---

func TestEffectiveRounds(t *testing.T) {
t.Parallel()
cases := []struct {
in   int
want int
}{
{0, defaultRounds},
{-1, defaultRounds},
{1, 1},
{3, 3},
{50, 50},
{51, maxRounds},
}
for _, tc := range cases {
got := effectiveRounds(tc.in)
if got != tc.want {
t.Errorf("effectiveRounds(%d) = %d, want %d", tc.in, got, tc.want)
}
}
}

// --- buildModelPool ---

func TestBuildModelPool_EnvVar(t *testing.T) {
t.Run("unset_returns_defaults", func(t *testing.T) {
t.Setenv("ESQUISSE_MODELS", "")
pool := buildModelPool()
if len(pool) != len(defaultModels) {
t.Fatalf("expected %d default models, got %d", len(defaultModels), len(pool))
}
for i, m := range defaultModels {
if pool[i] != m {
t.Errorf("pool[%d] = %q, want %q", i, pool[i], m)
}
}
})

t.Run("valid_entries_returned", func(t *testing.T) {
t.Setenv("ESQUISSE_MODELS", "copilot/a,gemini/b")
pool := buildModelPool()
if len(pool) != 2 || pool[0] != "copilot/a" || pool[1] != "gemini/b" {
t.Fatalf("unexpected pool: %v", pool)
}
})

t.Run("whitespace_trimmed", func(t *testing.T) {
t.Setenv("ESQUISSE_MODELS", " copilot/a , gemini/b ")
pool := buildModelPool()
if len(pool) != 2 || pool[0] != "copilot/a" || pool[1] != "gemini/b" {
t.Fatalf("unexpected pool: %v", pool)
}
})

t.Run("empty_entry_skipped", func(t *testing.T) {
t.Setenv("ESQUISSE_MODELS", "copilot/a,,gemini/b")
pool := buildModelPool()
if len(pool) != 2 {
t.Fatalf("expected 2 entries, got %d: %v", len(pool), pool)
}
})

t.Run("invalid_no_slash_skipped", func(t *testing.T) {
t.Setenv("ESQUISSE_MODELS", "badentry,copilot/a")
pool := buildModelPool()
if len(pool) != 1 || pool[0] != "copilot/a" {
t.Fatalf("unexpected pool: %v", pool)
}
})

t.Run("invalid_chars_skipped", func(t *testing.T) {
t.Setenv("ESQUISSE_MODELS", "copilot/a!,copilot/b")
pool := buildModelPool()
if len(pool) != 1 || pool[0] != "copilot/b" {
t.Fatalf("unexpected pool: %v", pool)
}
})

t.Run("all_invalid_falls_back", func(t *testing.T) {
t.Setenv("ESQUISSE_MODELS", "bad1,bad2")
pool := buildModelPool()
if len(pool) != len(defaultModels) {
t.Fatalf("expected fallback to defaults, got %d entries: %v", len(pool), pool)
}
})
}

// --- familyInterleaveShuffle ---

func TestFamilyInterleaveShuffle_NoCopilotConsecutiveMore2(t *testing.T) {
t.Parallel()
rng := rand.New(rand.NewSource(42))
result := familyInterleaveShuffle(defaultModels, rng)
if len(result) != len(defaultModels) {
t.Fatalf("len mismatch: got %d, want %d", len(result), len(defaultModels))
}
// Count consecutive copilot pairs.
consecutiveCopilot := 0
for i := 1; i < len(result); i++ {
if strings.HasPrefix(result[i], "copilot/") && strings.HasPrefix(result[i-1], "copilot/") {
consecutiveCopilot++
}
}
if consecutiveCopilot > 1 {
t.Errorf("too many consecutive copilot models (%d pairs): %v", consecutiveCopilot, result)
}
}

func TestFamilyInterleaveShuffle_LenPreserved(t *testing.T) {
t.Parallel()
rng := rand.New(rand.NewSource(99))
result := familyInterleaveShuffle(defaultModels, rng)
if len(result) != len(defaultModels) {
t.Errorf("got len=%d, want %d", len(result), len(defaultModels))
}
}

// --- buildRotationOrder ---

func TestBuildRotationOrder_LenMatchesRounds(t *testing.T) {
t.Parallel()
for _, rounds := range []int{1, 3, 5, 7, 10, 15} {
got := buildRotationOrder(defaultModels, rounds)
if len(got) != rounds {
t.Errorf("rounds=%d: got len=%d", rounds, len(got))
}
}
}

func TestBuildRotationOrder_NoConsecutiveIdentical(t *testing.T) {
SetRandSource(rand.NewSource(123))
defer SetRandSource(nil)
got := buildRotationOrder(defaultModels, 20)
for i := 1; i < len(got); i++ {
if got[i] == got[i-1] {
t.Errorf("consecutive identical at index %d: %q", i, got[i])
}
}
}

func TestBuildRotationOrder_TwoSeedsDiffer(t *testing.T) {
SetRandSource(rand.NewSource(1))
order1 := buildRotationOrder(defaultModels, 10)
SetRandSource(rand.NewSource(9999))
order2 := buildRotationOrder(defaultModels, 10)
SetRandSource(nil)

same := true
for i := range order1 {
if order1[i] != order2[i] {
same = false
break
}
}
if same {
t.Error("two different seeds produced identical rotation order")
}
}

// --- worstVerdict ---

func TestWorstVerdict(t *testing.T) {
t.Parallel()
cases := []struct {
verdicts []string
want     string
}{
{[]string{"PASSED", "CONDITIONAL"}, "CONDITIONAL"},
{[]string{"PASSED", "CONDITIONAL", "PASSED"}, "CONDITIONAL"},
{[]string{"PASSED", "FAILED", "CONDITIONAL"}, "FAILED"},
{[]string{}, ""},
{[]string{"PASSED"}, "PASSED"},
{[]string{"CONDITIONAL"}, "CONDITIONAL"},
{[]string{"FAILED"}, "FAILED"},
{[]string{"", ""}, ""},
}
for _, tc := range cases {
got := worstVerdict(tc.verdicts)
if got != tc.want {
t.Errorf("worstVerdict(%v) = %q, want %q", tc.verdicts, got, tc.want)
}
}
}

// --- isModelUnavailable ---

func TestIsModelUnavailable(t *testing.T) {
t.Parallel()
cases := []struct {
output string
want   bool
}{
{"Error: model is not supported for this organization", true},
{"Error: Model Not Supported", true},
{"not available for your organization", true},
{"not enabled for your organization", true},
{"access to this model is restricted", true},
{"model access denied by policy", true},
{"this model is not available", true},
{"auth error: invalid token", false},
{"network error: connection refused", false},
{"rate limit exceeded", false},
{"", false},
}
for _, tc := range cases {
got := isModelUnavailable(tc.output)
if got != tc.want {
t.Errorf("isModelUnavailable(%q) = %v, want %v", tc.output, got, tc.want)
}
}
}

// --- runOneRound ---

func TestRunOneRound_SuccessPath(t *testing.T) {
orig := runCrushFn
defer func() { runCrushFn = orig }()

runCrushFn = func(_ context.Context, model, _ string) (RunResult, error) {
return RunResult{Output: "Verdict: PASSED\n", ExitCode: 0}, nil
}

tmpDir := t.TempDir()
usedModel, output, err := runOneRound(
context.Background(),
defaultModels,
defaultModels[0],
"preamble",
"plan content",
tmpDir,
)
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if usedModel != defaultModels[0] {
t.Errorf("usedModel = %q, want %q", usedModel, defaultModels[0])
}
if !strings.Contains(output, "PASSED") {
t.Errorf("output missing PASSED: %q", output)
}
}

func TestRunOneRound_FallbackPath(t *testing.T) {
orig := runCrushFn
defer func() { runCrushFn = orig }()

calls := 0
primary := defaultModels[0]
fallback := defaultModels[1]

runCrushFn = func(_ context.Context, model, _ string) (RunResult, error) {
calls++
if model == primary {
return RunResult{Output: "model is not supported", ExitCode: 1}, nil
}
return RunResult{Output: "Verdict: CONDITIONAL\n", ExitCode: 0}, nil
}

tmpDir := t.TempDir()
usedModel, output, err := runOneRound(
context.Background(),
defaultModels,
primary,
"preamble",
"plan content",
tmpDir,
)
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if usedModel != fallback {
t.Errorf("usedModel = %q, want fallback %q", usedModel, fallback)
}
if !strings.Contains(output, "CONDITIONAL") {
t.Errorf("output missing CONDITIONAL: %q", output)
}
}

func TestRunOneRound_AllUnavailable(t *testing.T) {
orig := runCrushFn
defer func() { runCrushFn = orig }()

runCrushFn = func(_ context.Context, _ string, _ string) (RunResult, error) {
return RunResult{Output: "model is not supported", ExitCode: 1}, nil
}

tmpDir := t.TempDir()
_, _, err := runOneRound(
context.Background(),
defaultModels,
defaultModels[0],
"preamble",
"plan content",
tmpDir,
)
if err == nil {
t.Fatal("expected errAllModelsUnavailable, got nil")
}
if err != errAllModelsUnavailable {
t.Errorf("got error %v, want errAllModelsUnavailable", err)
}
}

// --- excludeModelFilter ---

func TestExcludeModelFilter(t *testing.T) {
t.Parallel()
samplePool := []string{"copilot/a", "gemini/b", "copilot/c"}
singleEntryPool := []string{"copilot/a"}

// AC1: empty_exclude — pool returned unchanged.
t.Run("empty_exclude", func(t *testing.T) {
t.Parallel()
got := excludeModelFilter(samplePool, "")
if len(got) != len(samplePool) {
t.Fatalf("expected %d items, got %d", len(samplePool), len(got))
}
})

// AC2: removes_matching — only exact match is removed.
t.Run("removes_matching", func(t *testing.T) {
t.Parallel()
got := excludeModelFilter(samplePool, "copilot/a")
if len(got) != 2 {
t.Fatalf("got %v, want [gemini/b copilot/c]", got)
}
for _, m := range got {
if m == "copilot/a" {
t.Errorf("copilot/a should have been excluded; got %v", got)
}
}
})

// AC3: case_insensitive — uppercase/mixed produce same result as lowercase.
t.Run("case_insensitive", func(t *testing.T) {
t.Parallel()
got1 := excludeModelFilter(samplePool, "Copilot/A")
got2 := excludeModelFilter(samplePool, "COPILOT/A")
got3 := excludeModelFilter(samplePool, "copilot/a")
if len(got1) != len(got3) || len(got2) != len(got3) {
t.Errorf("case mismatch: %v vs %v vs %v", got1, got2, got3)
}
})

// AC4: single_entry_fallback — would-empty → fail-open → original pool returned.
t.Run("single_entry_fallback", func(t *testing.T) {
t.Parallel()
got := excludeModelFilter(singleEntryPool, "copilot/a")
if len(got) != len(singleEntryPool) {
t.Errorf("fail-open: expected original pool len %d, got %d: %v", len(singleEntryPool), len(got), got)
}
})

// AC5: exact_match_only — "copilot/a" does not remove "copilot/ab".
t.Run("exact_match_only", func(t *testing.T) {
t.Parallel()
pool := []string{"copilot/a", "copilot/ab"}
got := excludeModelFilter(pool, "copilot/a")
if len(got) != 1 || got[0] != "copilot/ab" {
t.Errorf("got %v, want [copilot/ab]", got)
}
})

// AC6: whitespace_only — no-op.
t.Run("whitespace_only", func(t *testing.T) {
t.Parallel()
got := excludeModelFilter(samplePool, "   ")
if len(got) != len(samplePool) {
t.Errorf("whitespace should be no-op: got %v", got)
}
})

// AC7: no_match_noop — no entry matches → pool unchanged.
t.Run("no_match_noop", func(t *testing.T) {
t.Parallel()
pool2 := []string{"gemini/a", "vertexai/b"}
got := excludeModelFilter(pool2, "copilot/x")
if len(got) != len(pool2) {
t.Errorf("no-match should be no-op: got %v", got)
}
})

// AC8: malformed_value — invalid chars → pool unchanged.
t.Run("malformed_value", func(t *testing.T) {
t.Parallel()
got := excludeModelFilter(samplePool, "copilot)")
if len(got) != len(samplePool) {
t.Errorf("malformed should be no-op: got %v", got)
}
})

// AC9 (integration): single-entry pool where entry matches → fail-open → original returned.
t.Run("single_entry_exact_match_failopen", func(t *testing.T) {
t.Parallel()
pool := []string{"gemini/pro"}
got := excludeModelFilter(pool, "gemini/pro")
if len(got) != 1 || got[0] != "gemini/pro" {
t.Errorf("fail-open: expected original pool, got %v", got)
}
})

// AC13: partial_model_without_provider_noop — "a" does not match "copilot/a" exactly.
t.Run("partial_model_without_provider_noop", func(t *testing.T) {
t.Parallel()
pool := []string{"copilot/a", "gemini/b"}
got := excludeModelFilter(pool, "a")
if len(got) != len(pool) {
t.Errorf("partial match should be no-op: got %v", got)
}
})
}
