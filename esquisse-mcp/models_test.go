// Package main — tests for models.go.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

func TestBuildModelPool_Defaults(t *testing.T) {
	// Ensure no slot env vars are set.
	for i := 0; i < 5; i++ {
		t.Setenv(fmt.Sprintf("ESQUISSE_MODEL_SLOT%d", i), "")
	}
	t.Setenv("ESQUISSE_ALLOWED_PROVIDERS", "")
	t.Setenv("ESQUISSE_POOL_FALLBACK_STRICT", "")

	pool := buildModelPool()
	if len(pool) != 5 {
		t.Fatalf("expected 5 models, got %d: %v", len(pool), pool)
	}
	for i, m := range pool {
		if m != defaultModels[i] {
			t.Errorf("slot %d: got %q, want %q", i, m, defaultModels[i])
		}
	}
}

func TestBuildModelPool_AllowedProviders_Copilot(t *testing.T) {
	for i := 0; i < 5; i++ {
		t.Setenv(fmt.Sprintf("ESQUISSE_MODEL_SLOT%d", i), "")
	}
	t.Setenv("ESQUISSE_ALLOWED_PROVIDERS", "copilot")
	t.Setenv("ESQUISSE_POOL_FALLBACK_STRICT", "")

	pool := buildModelPool()
	// defaultModels has 3 copilot entries: slots 0, 2, 4.
	if len(pool) != 3 {
		t.Fatalf("expected 3 copilot models, got %d: %v", len(pool), pool)
	}
	for _, m := range pool {
		if !strings.HasPrefix(m, "copilot/") {
			t.Errorf("unexpected non-copilot model %q in filtered pool", m)
		}
	}
}

func TestBuildModelPool_AllowedProviders_UnknownFallback(t *testing.T) {
	for i := 0; i < 5; i++ {
		t.Setenv(fmt.Sprintf("ESQUISSE_MODEL_SLOT%d", i), "")
	}
	t.Setenv("ESQUISSE_ALLOWED_PROVIDERS", "openai")
	t.Setenv("ESQUISSE_POOL_FALLBACK_STRICT", "")

	pool := buildModelPool()
	// No defaultModels entry has "openai" prefix → fall back to all 5 defaults.
	if len(pool) != 5 {
		t.Fatalf("expected fallback to 5 default models, got %d: %v", len(pool), pool)
	}
}

func TestBuildModelPool_InvalidSlotFormat(t *testing.T) {
	t.Setenv("ESQUISSE_MODEL_SLOT0", "badformat")
	for i := 1; i < 5; i++ {
		t.Setenv(fmt.Sprintf("ESQUISSE_MODEL_SLOT%d", i), "")
	}
	t.Setenv("ESQUISSE_ALLOWED_PROVIDERS", "")
	t.Setenv("ESQUISSE_POOL_FALLBACK_STRICT", "")

	pool := buildModelPool()
	// Slot 0 falls back to default.
	if pool[0] != defaultModels[0] {
		t.Errorf("slot 0 with invalid format: got %q, want %q", pool[0], defaultModels[0])
	}
}

func TestBuildModelPool_StrictWithAllFiltered(t *testing.T) {
	for i := 0; i < 5; i++ {
		t.Setenv(fmt.Sprintf("ESQUISSE_MODEL_SLOT%d", i), "")
	}
	t.Setenv("ESQUISSE_ALLOWED_PROVIDERS", "openai")
	t.Setenv("ESQUISSE_POOL_FALLBACK_STRICT", "1")

	pool := buildModelPool()
	if pool != nil {
		t.Fatalf("expected nil pool in strict mode with all filtered, got %v", pool)
	}
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

// --- modelProber ---

func TestModelProber(t *testing.T) {
	t.Parallel()

	setupProber := func(t *testing.T, listModelsFn func(context.Context) ([]string, error), probeFn func(context.Context, string) bool) (*modelProber, string) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "model-cache.json")
		p := newModelProberWithFuncs(cachePath, time.Hour, listModelsFn, probeFn)
		return p, cachePath
	}

	t.Run("no_cache_returns_probing_state", func(t *testing.T) {
		t.Parallel()
		// Gate listModelsFn until after currentState() is called to avoid a race
		// where the goroutine completes before the read.
		gate := make(chan struct{})
		p, _ := setupProber(t, func(ctx context.Context) ([]string, error) {
			<-gate // block until state has been read
			return []string{"copilot/a"}, nil
		}, func(ctx context.Context, m string) bool { return true })

		p.startProbe(context.Background())
		entries, probing, stale, _ := p.currentState()
		close(gate) // unblock listModelsFn now that we've sampled state
		if !probing {
			t.Error("expected probing to be true")
		}
		if stale {
			t.Error("expected stale to be false")
		}
		if len(entries) != 0 {
			t.Errorf("expected 0 entries before probe completes, got %v", entries)
		}
		<-p.done // wait for probe to finish before TempDir is removed
	})

	t.Run("fresh_cache_no_rerun", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "model-cache.json")
		cacheData := ModelCache{
			Entries: []ModelEntry{{ID: "copilot/cached", Provider: "copilot", Available: true}},
			CachedAt: time.Now().Add(-10 * time.Minute),
			ProbeCompleted: true,
		}
		b, _ := json.Marshal(cacheData)
		os.WriteFile(cachePath, b, 0600)

		listCalls := 0
		p := newModelProberWithFuncs(cachePath, time.Hour, func(ctx context.Context) ([]string, error) {
			listCalls++
			return []string{"copilot/a"}, nil
		}, func(ctx context.Context, m string) bool { return true })

		entries, probing, stale, _ := p.currentState()
		if probing {
			t.Error("expected probing false for fresh cache")
		}
		if stale {
			t.Error("expected stale false")
		}
		if len(entries) != 1 || entries[0].ID != "copilot/cached" {
			t.Error("expected cached entries")
		}
		if listCalls != 0 {
			t.Error("expected listModelsFn not to be called")
		}
	})

	t.Run("stale_cache_sets_stale_flag", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "model-cache.json")
		cacheData := ModelCache{
			Entries: []ModelEntry{{ID: "copilot/cached", Provider: "copilot", Available: true}},
			CachedAt: time.Now().Add(-2 * time.Hour),
			ProbeCompleted: true,
		}
		b, _ := json.Marshal(cacheData)
		os.WriteFile(cachePath, b, 0600)

		p := newModelProberWithFuncs(cachePath, time.Hour, func(ctx context.Context) ([]string, error) {
			return []string{"copilot/new"}, nil
		}, func(ctx context.Context, m string) bool { return true })

		entries, probing, stale, _ := p.currentState()
		if probing {
			t.Error("probing shouldn't start automatically just from currentState")
		}
		if !stale {
			t.Error("expected stale true")
		}
		if len(entries) != 1 || entries[0].ID != "copilot/cached" {
			t.Error("expected cached entries to still be returned")
		}
	})

	t.Run("force_refresh_resets_state", func(t *testing.T) {
		t.Parallel()
		p, _ := setupProber(t, func(ctx context.Context) ([]string, error) {
			return []string{"copilot/new"}, nil
		}, func(ctx context.Context, m string) bool { return true })

		p.startProbe(context.Background())
		p.forceRefresh(context.Background())

		// Immediately after forceRefresh, cache is nil and new probe is starting.
		_, probing, _, _ := p.currentState()
		if !probing {
			t.Error("expected probing true after force refresh")
		}

		// Wait for probe to complete before asserting entries.
		<-p.done

		entries, _, _, _ := p.currentState()
		if len(entries) != 1 || entries[0].ID != "copilot/new" {
			t.Errorf("expected [copilot/new] after probe, got %v", entries)
		}
	})

	t.Run("probe_marks_available_true_on_zero_exit", func(t *testing.T) {
		t.Parallel()
		p, _ := setupProber(t, func(ctx context.Context) ([]string, error) {
			return []string{"copilot/a"}, nil
		}, func(ctx context.Context, m string) bool { return true })

		p.startProbe(context.Background())
		<-p.done // wait for completion

		entries, _, _, _ := p.currentState()
		if len(entries) != 1 || !entries[0].Available {
			t.Error("expected available=true")
		}
	})

	t.Run("probe_marks_available_false_on_policy_block", func(t *testing.T) {
		t.Parallel()
		p, _ := setupProber(t, func(ctx context.Context) ([]string, error) {
			return []string{"copilot/a"}, nil
		}, func(ctx context.Context, m string) bool { return false })

		p.startProbe(context.Background())
		<-p.done

		entries, _, _, _ := p.currentState()
		if len(entries) != 1 || entries[0].Available {
			t.Error("expected available=false")
		}
	})

	t.Run("corrupted_cache_starts_fresh_probe", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "model-cache.json")
		os.WriteFile(cachePath, []byte("not json"), 0600)

		p := newModelProberWithFuncs(cachePath, time.Hour, func(ctx context.Context) ([]string, error) {
			return []string{"copilot/new"}, nil
		}, func(ctx context.Context, m string) bool { return true })

		// loadCache will fail, cache is nil.
		entries, _, _, _ := p.currentState()
		if len(entries) != 0 {
			t.Error("expected no entries initially")
		}
	})

	t.Run("invalid_model_id_in_cache_skipped", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "model-cache.json")
		cacheData := ModelCache{
			Entries: []ModelEntry{
				{ID: "copilot/good", Provider: "copilot", Available: true},
				{ID: "!@#evil", Provider: "evil", Available: true},
			},
			CachedAt: time.Now(),
		}
		b, _ := json.Marshal(cacheData)
		os.WriteFile(cachePath, b, 0600)

		p := newModelProberWithFuncs(cachePath, time.Hour, nil, nil)
		entries, _, _, _ := p.currentState()
		if len(entries) != 1 || entries[0].ID != "copilot/good" {
			t.Errorf("expected 1 good entry, got %v", entries)
		}
	})

	t.Run("probe_marks_available_true_on_transient_error", func(t *testing.T) {
		t.Parallel()
		p, _ := setupProber(t, func(ctx context.Context) ([]string, error) {
			return []string{"copilot/a"}, nil
		}, func(ctx context.Context, m string) bool { return true })

		p.startProbe(context.Background())
		<-p.done

		entries, _, _, _ := p.currentState()
		if len(entries) != 1 || !entries[0].Available {
			t.Error("expected available=true for transient error (fail-open)")
		}
	})

	t.Run("structured_response_shape", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "model-cache.json")
		cacheData := ModelCache{
			Entries:        []ModelEntry{{ID: "copilot/a", Provider: "copilot", Available: true}},
			CachedAt:       time.Now(),
			ProbeCompleted: true,
		}
		b, _ := json.Marshal(cacheData)
		_ = os.WriteFile(cachePath, b, 0600)

		p := newModelProberWithFuncs(cachePath, time.Hour,
			func(ctx context.Context) ([]string, error) { return []string{"copilot/a"}, nil },
			func(ctx context.Context, m string) bool { return true })
		handler := newDiscoverHandler(p)
		result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, discoverInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Content) == 0 {
			t.Fatal("expected non-empty Content")
		}
		tc, ok := result.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatalf("expected *mcp.TextContent, got %T", result.Content[0])
		}
		var resp map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Text), &resp); err != nil {
			t.Fatalf("response is not valid JSON: %v\nraw: %s", err, tc.Text)
		}
		for _, key := range []string{"models", "cached_at", "stale", "probing"} {
			if _, ok := resp[key]; !ok {
				t.Errorf("response missing key %q: %s", key, tc.Text)
			}
		}
		if models, ok := resp["models"].([]interface{}); !ok || len(models) != 1 {
			t.Errorf("expected 1 model in response, got %v", resp["models"])
		}
		<-p.done // wait for background probe started by handler
	})

	t.Run("cache_written_atomically", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "model-cache.json")

		p := newModelProberWithFuncs(cachePath, time.Hour, func(ctx context.Context) ([]string, error) {
			return []string{"copilot/a"}, nil
		}, func(ctx context.Context, m string) bool { return true })

		p.startProbe(context.Background())
		<-p.done

		if _, err := os.Stat(cachePath); err != nil {
			t.Fatalf("cache file does not exist after probe: %v", err)
		}
		dirEntries, err := os.ReadDir(tmpDir)
		if err != nil {
			t.Fatalf("failed to read tmp dir: %v", err)
		}
		for _, e := range dirEntries {
			if strings.HasPrefix(e.Name(), "model-cache-") && strings.HasSuffix(e.Name(), ".json") {
				t.Errorf("leftover temp file found: %s", e.Name())
			}
		}
	})

}

// TestModelProberFilterAllowedProviders tests that newDiscoverHandler filters
// entries by ESQUISSE_ALLOWED_PROVIDERS. Extracted from TestModelProber because
// t.Setenv cannot be called from a subtest whose parent is parallel.
func TestModelProberFilterAllowedProviders(t *testing.T) {
	t.Setenv("ESQUISSE_ALLOWED_PROVIDERS", "copilot")
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "model-cache.json")
	cacheData := ModelCache{
		Entries: []ModelEntry{
			{ID: "copilot/a", Provider: "copilot", Available: true},
			{ID: "gemini/b", Provider: "gemini", Available: true},
		},
		CachedAt:       time.Now(),
		ProbeCompleted: true,
	}
	b, _ := json.Marshal(cacheData)
	_ = os.WriteFile(cachePath, b, 0600)

	p := newModelProberWithFuncs(cachePath, time.Hour,
		func(ctx context.Context) ([]string, error) { return []string{"copilot/a", "gemini/b"}, nil },
		func(ctx context.Context, m string) bool { return true })
	handler := newDiscoverHandler(p)
	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, discoverInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", result.Content[0])
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Text), &resp); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	models, ok := resp["models"].([]interface{})
	if !ok || len(models) != 1 {
		t.Fatalf("expected 1 copilot model, got %v", resp["models"])
	}
	m, ok := models[0].(map[string]interface{})
	if !ok || m["id"] != "copilot/a" {
		t.Errorf("unexpected model: %v", models[0])
	}
	<-p.done // wait for background probe started by handler
}

func TestModelProberConcurrentAccess(t *testing.T) {
	p := newModelProberWithFuncs("", time.Hour, func(ctx context.Context) ([]string, error) {
		time.Sleep(10 * time.Millisecond)
		return []string{"copilot/a", "gemini/b"}, nil
	}, func(ctx context.Context, m string) bool { return true })

	ctx := context.Background()
	p.startProbe(ctx)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.currentState()
		}()
	}
	wg.Wait()
	<-p.done
}

