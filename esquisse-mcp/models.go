// Package main — model pool management for adversarial review rotation.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultRounds = 5
	maxRounds     = 50
)

// defaultModels is the default 5-slot model pool.
var defaultModels = []string{
	"copilot/claude-sonnet-4.6",
	"gemini/gemini-3.1-pro-preview",
	"copilot/gpt-4.1",
	"vertexai/gemini-3.1-pro-preview",
	"copilot/gpt-4o",
}

// validModelRe matches valid exclude_model values: alphanumeric, hyphen, underscore, dot, slash.
// Slash is explicitly allowed because valid model IDs are "provider/model".
var validModelRe = regexp.MustCompile(`^[a-zA-Z0-9_./-]+$`)

// excludeModelFilter returns a copy of pool with all entries that exactly match
// exclude (case-insensitive) removed.
// If exclude is empty or whitespace-only: no-op (returns pool unchanged).
// If exclude is malformed (fails validModelRe): logs warning, returns pool unchanged.
// If no pool entry matches: logs info, returns pool unchanged.
// If all entries match (would empty pool): logs warning "would empty pool; ignoring exclusion",
// returns pool unchanged (fail-open — blocking review is worse than reduced independence).
// All non-empty exclude values are logged with %q to prevent log injection.
func excludeModelFilter(pool []string, exclude string) []string {
	exclude = strings.TrimSpace(exclude)
	if exclude == "" {
		return pool
	}
	log.Printf("esquisse-mcp: exclude_model=%q requested", exclude)
	if !validModelRe.MatchString(exclude) {
		log.Printf("esquisse-mcp: exclude_model=%q is malformed, ignoring", exclude)
		return pool
	}
	excludeLower := strings.ToLower(exclude)
	filtered := make([]string, 0, len(pool))
	for _, m := range pool {
		if strings.ToLower(m) != excludeLower {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) == 0 {
		log.Printf("esquisse-mcp: exclude_model=%q would empty pool; ignoring exclusion", exclude)
		return pool
	}
	if len(filtered) == len(pool) {
		log.Printf("esquisse-mcp: exclude_model=%q matched no pool entries (no-op)", exclude)
	}
	return filtered
}

// errAllModelsUnavailable is returned when every model in the pool is blocked.
var errAllModelsUnavailable = errors.New("all models in the pool are unavailable " +
	"(enterprise policy may be blocking them); " +
	"check ESQUISSE_ALLOWED_PROVIDERS or run discover_models to verify pool availability")

// modelUnavailablePatterns are case-insensitive substrings that indicate a model
// is blocked by enterprise policy rather than a transient error.
var modelUnavailablePatterns = []string{
	"model is not supported",
	"model not supported",
	"not available for your organization",
	"not enabled for your organization",
	"access to this model",
	"model access denied",
	"this model is not available",
	"not supported via",
}

// runCrushFn is the function used to invoke crush — replaceable in tests.
var runCrushFn = RunCrush

// randSource is used by buildRotationOrder; replaceable via SetRandSource.
var randSource rand.Source

// SetRandSource replaces the random source used by buildRotationOrder.
// Not goroutine-safe — intended for test use only.
func SetRandSource(src rand.Source) {
	randSource = src
}

// newRand returns a *rand.Rand seeded from randSource if set, otherwise from
// time.Now().UnixNano().
func newRand() *rand.Rand {
	if randSource != nil {
		return rand.New(randSource)
	}
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

// effectiveRounds clamps n to [1, maxRounds], defaulting to defaultRounds when n < 1.
func effectiveRounds(n int) int {
	if n < 1 {
		return defaultRounds
	}
	if n > maxRounds {
		log.Printf("esquisse-mcp: rounds=%d exceeds maxRounds=%d, clamping", n, maxRounds)
		return maxRounds
	}
	return n
}

// buildModelPool constructs the effective model pool from env vars + defaults.
// Does not invoke any external binary.
func buildModelPool() []string {
	const slots = 5
	// Step 1: build base slice from slot env vars with validation.
	base := make([]string, slots)
	for i := 0; i < slots; i++ {
		envKey := fmt.Sprintf("ESQUISSE_MODEL_SLOT%d", i)
		v := os.Getenv(envKey)
		if v == "" {
			base[i] = defaultModels[i]
			continue
		}
		// Validate: exactly one "/" with non-empty parts on both sides.
		parts := strings.SplitN(v, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			log.Printf("esquisse-mcp: ESQUISSE_MODEL_SLOT%d=%q is invalid, using default", i, v)
			base[i] = defaultModels[i]
			continue
		}
		base[i] = v
	}

	// Step 2: parse ESQUISSE_ALLOWED_PROVIDERS.
	allowedRaw := os.Getenv("ESQUISSE_ALLOWED_PROVIDERS")
	if allowedRaw == "" {
		return base
	}
	allowedSet := make(map[string]bool)
	for _, p := range strings.Split(allowedRaw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			allowedSet[p] = true
		}
	}
	if len(allowedSet) == 0 {
		return base
	}

	// Step 3: filter by allowed providers.
	filtered := make([]string, 0, len(base))
	for _, m := range base {
		provider := providerOf(m)
		if allowedSet[provider] {
			filtered = append(filtered, m)
		} else {
			log.Printf("esquisse-mcp: model %q excluded by ESQUISSE_ALLOWED_PROVIDERS", m)
		}
	}

	// Step 4: handle empty result.
	if len(filtered) == 0 {
		if os.Getenv("ESQUISSE_POOL_FALLBACK_STRICT") == "1" {
			return nil
		}
		log.Printf("esquisse-mcp: all models filtered by ESQUISSE_ALLOWED_PROVIDERS; falling back to default pool (set ESQUISSE_POOL_FALLBACK_STRICT=1 to prevent this)")
		return append([]string(nil), defaultModels...)
	}

	return filtered
}

// providerOf extracts the provider prefix (before the first "/") from a model string.
func providerOf(model string) string {
	if idx := strings.Index(model, "/"); idx >= 0 {
		return model[:idx]
	}
	return model
}

// familyInterleaveShuffle returns a permutation of pool where models from the
// same provider are spread out as evenly as possible.
// Uses rng for intra-group shuffling; O(n²), acceptable for n≤10.
func familyInterleaveShuffle(pool []string, rng *rand.Rand) []string {
	// Group by provider.
	order := make([]string, 0, len(pool))
	groups := make(map[string][]string)
	providers := make([]string, 0)
	for _, m := range pool {
		p := providerOf(m)
		if _, seen := groups[p]; !seen {
			providers = append(providers, p)
		}
		groups[p] = append(groups[p], m)
	}
	// Shuffle within each group.
	for _, p := range providers {
		g := groups[p]
		rng.Shuffle(len(g), func(i, j int) { g[i], g[j] = g[j], g[i] })
		groups[p] = g
	}
	// Greedy pick: most-remaining group, not same as last; tie-break alphabetically.
	sort.Strings(providers)
	remaining := make(map[string][]string, len(groups))
	for k, v := range groups {
		remaining[k] = append([]string(nil), v...)
	}
	last := ""
	for len(order) < len(pool) {
		best := ""
		bestCount := -1
		for _, p := range providers {
			if p == last {
				continue
			}
			if cnt := len(remaining[p]); cnt > bestCount {
				bestCount = cnt
				best = p
			}
		}
		// If every non-last provider is empty, fall back to last.
		if best == "" || bestCount == 0 {
			best = last
		}
		order = append(order, remaining[best][0])
		remaining[best] = remaining[best][1:]
		if len(remaining[best]) == 0 {
			delete(remaining, best)
		}
		last = best
	}
	return order
}

// buildRotationOrder returns a slice of model strings of length rounds, drawn
// from pool in family-interleaved batches of batchSize (5).
func buildRotationOrder(pool []string, rounds int) []string {
	const batchSize = 5
	rng := newRand()
	result := make([]string, 0, rounds)
	for len(result) < rounds {
		batch := familyInterleaveShuffle(pool, rng)
		// Cyclic fill if pool < batchSize.
		for len(batch) < batchSize {
			extra := familyInterleaveShuffle(pool, rng)
			// Swap at wrap boundary to avoid consecutive identical.
			if len(batch) > 0 && len(extra) > 0 && batch[len(batch)-1] == extra[0] {
				if len(extra) > 1 {
					extra[0], extra[1] = extra[1], extra[0]
				}
			}
			batch = append(batch, extra...)
		}
		// Batch-boundary swap: result[-1] == batch[0].
		if len(result) > 0 && batch[0] == result[len(result)-1] {
			if len(batch) > 1 {
				batch[0], batch[1] = batch[1], batch[0]
			}
		}
		take := batchSize
		if rounds-len(result) < take {
			take = rounds - len(result)
		}
		result = append(result, batch[:take]...)
	}
	return result
}

// worstVerdict returns the most severe verdict from the slice.
// Severity order: FAILED > CONDITIONAL > PASSED > "".
func worstVerdict(verdicts []string) string {
	worst := ""
	for _, v := range verdicts {
		switch v {
		case "FAILED":
			return "FAILED"
		case "CONDITIONAL":
			if worst != "FAILED" {
				worst = "CONDITIONAL"
			}
		case "PASSED":
			if worst == "" {
				worst = "PASSED"
			}
		}
	}
	return worst
}

// filterAvailableModels returns pool with any models the prober has confirmed
// unavailable removed. Fail-open: if the prober has no data or all models would
// be removed, returns pool unchanged.
func filterAvailableModels(pool []string, prober *modelProber) []string {
	if prober == nil {
		return pool
	}
	entries, _, _, _ := prober.currentState()
	if len(entries) == 0 {
		return pool
	}
	avail := make(map[string]bool, len(entries))
	for _, e := range entries {
		avail[e.ID] = e.Available
	}
	filtered := make([]string, 0, len(pool))
	for _, m := range pool {
		av, known := avail[m]
		if !known || av {
			filtered = append(filtered, m)
		} else {
			log.Printf("esquisse-mcp: model %q excluded by probe cache (marked unavailable)", m)
		}
	}
	if len(filtered) == 0 {
		log.Printf("esquisse-mcp: probe cache would empty pool; using full pool (fail-open)")
		return pool
	}
	return filtered
}

// isModelUnavailable reports whether the crush output indicates the model is
// blocked by enterprise policy.
func isModelUnavailable(output string) bool {
	lower := strings.ToLower(output)
	for _, pattern := range modelUnavailablePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// runOneRound runs the adversarial review prompt against targetModel, falling
// back to other pool models if the primary is unavailable.
func runOneRound(ctx context.Context, pool []string, targetModel, preamble, planContent, tmpDir string) (usedModel, output string, err error) {
	tmp, err := os.CreateTemp(tmpDir, "round-*.txt")
	if err != nil {
		return "", "", fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return "", "", fmt.Errorf("chmod temp file: %w", err)
	}
	if _, err := fmt.Fprint(tmp, preamble+"\n--- PLAN CONTENT ---\n"+planContent); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return "", "", fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return "", "", fmt.Errorf("close temp file: %w", err)
	}

	tryModel := func(model string) (string, bool, error) {
		res, err := runCrushFn(ctx, model, tmpName)
		if err != nil {
			return "", false, err
		}
		if res.ExitCode == 0 {
			return res.Output, false, nil
		}
		if isModelUnavailable(res.Output) {
			return res.Output, true, nil
		}
		return res.Output, false, fmt.Errorf("crush exited %d: %s", res.ExitCode, res.Output)
	}

	// Try primary model.
	out, unavailable, runErr := tryModel(targetModel)
	if runErr != nil {
		_ = os.Remove(tmpName)
		return "", out, runErr
	}
	if !unavailable {
		_ = os.Remove(tmpName)
		return targetModel, out, nil
	}

	// Primary unavailable — try remaining pool models.
	for _, m := range pool {
		if m == targetModel {
			continue
		}
		out, unavailable, runErr = tryModel(m)
		if runErr != nil {
			_ = os.Remove(tmpName)
			return "", out, runErr
		}
		if !unavailable {
			_ = os.Remove(tmpName)
			return m, out, nil
		}
	}

	_ = os.Remove(tmpName)
	return "", "", errAllModelsUnavailable
}

// discoverInput is the input schema for the discover_models tool.
type discoverInput struct {
	Filter       string `json:"filter,omitempty" jsonschema:"Optional substring filter for model names (max 200 chars)"`
	ForceRefresh bool   `json:"force_refresh,omitempty" jsonschema:"Set to true to clear the cache and trigger a new background probe"`
}

// newDiscoverHandler returns an MCP handler that lists available crush models.
func newDiscoverHandler(prober *modelProber) func(context.Context, *mcp.CallToolRequest, discoverInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input discoverInput) (*mcp.CallToolResult, any, error) {
		if len(input.Filter) > 200 {
			return mcpErr("filter must not exceed 200 characters")
		}

		if input.ForceRefresh {
			prober.forceRefresh(ctx)
		} else {
			// If cache is missing or corrupt but not probing, start probe.
			_, probing, _, _ := prober.currentState()
			if !probing {
				prober.startProbe(ctx)
			}
		}

		entries, probing, stale, cachedAt := prober.currentState()

		var models []ModelEntry

		// Apply ESQUISSE_ALLOWED_PROVIDERS filter (same logic as buildModelPool).
		allowedRaw := os.Getenv("ESQUISSE_ALLOWED_PROVIDERS")
		var allowedSet map[string]bool
		if allowedRaw != "" {
			allowedSet = make(map[string]bool)
			for _, p := range strings.Split(allowedRaw, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					allowedSet[p] = true
				}
			}
		}

		var lowerFilter string
		if input.Filter != "" {
			lowerFilter = strings.ToLower(input.Filter)
		}

		for _, m := range entries {
			// Apply allowed providers
			if allowedSet != nil && len(allowedSet) > 0 && !allowedSet[m.Provider] {
				continue
			}
			// Apply user filter
			if lowerFilter != "" && !strings.Contains(strings.ToLower(m.ID), lowerFilter) {
				continue
			}
			models = append(models, m)
		}

		if models == nil {
			models = []ModelEntry{} // ensure JSON array instead of null
		}

		cachedAtStr := ""
		if !cachedAt.IsZero() {
			cachedAtStr = cachedAt.Format(time.RFC3339)
		}

		resp := map[string]interface{}{
			"models":    models,
			"cached_at": cachedAtStr,
			"stale":     stale,
			"probing":   probing,
		}

		respBytes, err := json.Marshal(resp)
		if err != nil {
			return mcpErr("failed to encode discover_models response: %v", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(respBytes)}},
		}, nil, nil
	}
}

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
	CachedAt       time.Time    `json:"cached_at"`
	ProbeCompleted bool         `json:"probe_completed"`
}

// modelProber manages the background probe goroutine and disk cache.
// All exported-accessible state is protected by mu.
// Holds injectable listModelsFn and probeFn for test substitution.
type modelProber struct {
	mu          sync.RWMutex
	cache       *ModelCache
	probing     bool
	cancelProbe context.CancelFunc
	done        chan struct{} // closed when current probe completes
	cachePath   string
	ttl         time.Duration
	// Injectable for testing — set via newModelProberWithFuncs.
	listModelsFn func(ctx context.Context) ([]string, error)
	probeFn      func(ctx context.Context, model string) bool
}

func defaultCachePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "esquisse-mcp", "model-cache.json"), nil
}

func defaultProbeTTL() time.Duration {
	envVal := os.Getenv("ESQUISSE_MODEL_CACHE_TTL_DAYS")
	if envVal != "" {
		days, err := strconv.Atoi(envVal)
		if err != nil || days <= 0 {
			log.Printf("esquisse-mcp: invalid ESQUISSE_MODEL_CACHE_TTL_DAYS=%q, using default (3)", envVal)
		} else {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	return 3 * 24 * time.Hour
}

func newModelProber(cachePath string, ttl time.Duration) *modelProber {
	return newModelProberWithFuncs(cachePath, ttl, func(ctx context.Context) ([]string, error) {
		crushPath, err := exec.LookPath("crush")
		if err != nil {
			return nil, fmt.Errorf("crush not in PATH: %w", err)
		}
		cmd := exec.CommandContext(ctx, crushPath, "models")
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("crush models failed: %w", err)
		}
		var models []string
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if !strings.Contains(line, "/") {
				continue
			}
			models = append(models, line)
		}
		return models, nil
	}, func(ctx context.Context, model string) bool {
		// Use a minimal non-empty prompt so crush actually calls the model API.
		// An empty prompt (/dev/null) causes crush to exit 0 without making an API
		// request, so enterprise policy blocks (e.g. "not supported via Responses API")
		// are never surfaced and the model is incorrectly marked available.
		tmp, err := os.CreateTemp("", "esquisse-probe-*.txt")
		if err != nil {
			log.Printf("esquisse-mcp: probe: failed to create temp file for model %q: %v", model, err)
			return true // fail-open
		}
		tmpName := tmp.Name()
		defer os.Remove(tmpName)
		if _, err := fmt.Fprint(tmp, "Reply with the single word: OK"); err != nil {
			_ = tmp.Close()
			log.Printf("esquisse-mcp: probe: failed to write temp file for model %q: %v", model, err)
			return true // fail-open
		}
		if err := tmp.Close(); err != nil {
			log.Printf("esquisse-mcp: probe: failed to close temp file for model %q: %v", model, err)
			return true // fail-open
		}

		res, err := runCrushFn(ctx, model, tmpName)
		if err != nil {
			return false
		}
		if res.ExitCode == 0 {
			return true
		}
		if isModelUnavailable(res.Output) {
			return false
		}
		return true // transient error — fail-open
	})
}

func newModelProberWithFuncs(cachePath string, ttl time.Duration, listModelsFn func(context.Context) ([]string, error), probeFn func(context.Context, string) bool) *modelProber {
	p := &modelProber{
		cachePath:    cachePath,
		ttl:          ttl,
		listModelsFn: listModelsFn,
		probeFn:      probeFn,
	}
	if cachePath == "" {
		log.Printf("esquisse-mcp: running without model-cache.json (UserConfigDir failed)")
	} else {
		err := p.loadCache()
		if err != nil {
			log.Printf("esquisse-mcp: failed to load cache: %v", err)
		}
	}
	return p
}

func (p *modelProber) loadCache() error {
	if p.cachePath == "" {
		return nil
	}
	data, err := os.ReadFile(p.cachePath)
	if err != nil {
		return err
	}
	var cache ModelCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return err
	}
	// Drop invalid IDs
	validEntries := make([]ModelEntry, 0, len(cache.Entries))
	for _, e := range cache.Entries {
		if validModelRe.MatchString(e.ID) {
			validEntries = append(validEntries, e)
		} else {
			log.Printf("esquisse-mcp: invalid model ID in cache: %q, dropping", e.ID)
		}
	}
	cache.Entries = validEntries

	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache = &cache
	return nil
}

func (p *modelProber) saveCache() {
	if p.cachePath == "" {
		return
	}
	p.mu.RLock()
	cache := p.cache
	p.mu.RUnlock()

	if cache == nil {
		return
	}

	data, err := json.Marshal(cache)
	if err != nil {
		log.Printf("esquisse-mcp: failed to encode cache: %v", err)
		return
	}

	dir := filepath.Dir(p.cachePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Printf("esquisse-mcp: failed to create cache dir %q: %v", dir, err)
		return
	}

	tmp, err := os.CreateTemp(dir, "model-cache-*.json")
	if err != nil {
		log.Printf("esquisse-mcp: failed to create cache temp file: %v", err)
		return
	}
	tmpName := tmp.Name()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		log.Printf("esquisse-mcp: failed to chmod cache temp file: %v", err)
		return
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		log.Printf("esquisse-mcp: failed to write cache temp file: %v", err)
		return
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		log.Printf("esquisse-mcp: failed to close cache temp file: %v", err)
		return
	}
	if err := os.Rename(tmpName, p.cachePath); err != nil {
		_ = os.Remove(tmpName)
		log.Printf("esquisse-mcp: failed to rename cache temp file: %v", err)
	}
}

func (p *modelProber) startProbe(ctx context.Context) {
	p.mu.Lock()
	if p.probing {
		p.mu.Unlock()
		return
	}
	p.probing = true
	p.done = make(chan struct{})
	probeCtx, cancel := context.WithCancel(ctx)
	p.cancelProbe = cancel
	p.mu.Unlock()

	go func() {
		defer func() {
			p.mu.Lock()
			p.probing = false
			close(p.done)
			p.mu.Unlock()
			cancel()
		}()

		listCtx, cancelList := context.WithTimeout(probeCtx, 10*time.Second)
		defer cancelList()
		models, err := p.listModelsFn(listCtx)
		if err != nil {
			log.Printf("esquisse-mcp: failed to list models for probe: %v", err)
			return
		}

		now := time.Now().UTC()
		entries := make([]ModelEntry, len(models))
		for i, m := range models {
			entries[i] = ModelEntry{
				ID:        m,
				Provider:  providerOf(m),
				Available: true,
				ProbedAt:  now,
			}
		}

		p.mu.Lock()
		p.cache = &ModelCache{
			Entries:  entries,
			CachedAt: now,
		}
		p.mu.Unlock()

		for i, m := range models {
			select {
			case <-probeCtx.Done():
				return
			default:
			}

			modelCtx, cancelModel := context.WithTimeout(probeCtx, 15*time.Second)
			avail := p.probeFn(modelCtx, m)
			cancelModel()

			p.mu.Lock()
			if p.cache != nil && len(p.cache.Entries) > i {
				p.cache.Entries[i].Available = avail
				p.cache.Entries[i].ProbedAt = time.Now().UTC()
			}
			p.mu.Unlock()
		}

		p.mu.Lock()
		if p.cache != nil {
			p.cache.ProbeCompleted = true
			p.cache.CachedAt = time.Now().UTC()
		}
		p.mu.Unlock()

		p.saveCache()
	}()
}

func (p *modelProber) currentState() ([]ModelEntry, bool, bool, time.Time) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var entries []ModelEntry
	var stale bool
	var cachedAt time.Time

	if p.cache != nil {
		entries = p.cache.Entries
		cachedAt = p.cache.CachedAt
		stale = time.Since(cachedAt) > p.ttl
	}

	return entries, p.probing, stale, cachedAt
}

func (p *modelProber) forceRefresh(ctx context.Context) {
	p.mu.Lock()
	if p.probing && p.cancelProbe != nil {
		p.cancelProbe()
	}
	p.cache = nil
	p.mu.Unlock()

	p.mu.RLock()
	d := p.done
	p.mu.RUnlock()
	if d != nil {
		<-d
	}

	p.startProbe(ctx)
}
