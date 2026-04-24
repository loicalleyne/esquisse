// Package main — model pool management for adversarial review rotation.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	defaultRounds = 1
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
	"set ESQUISSE_MODELS to a list of models known to be accessible")

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
