// Package main — model pool management for adversarial review rotation.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultRounds = 5
const maxRounds = 50

// defaultModels is the default 5-slot model pool.
var defaultModels = []string{
	"copilot/claude-sonnet-4.6",
	"gemini/gemini-3.1-pro-preview",
	"copilot/gpt-4.1",
	"vertexai/gemini-3.1-pro-preview",
	"copilot/gpt-4o",
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
	Filter string `json:"filter,omitempty" jsonschema:"Optional substring filter for model names (max 200 chars)"`
}

// newDiscoverHandler returns an MCP handler that lists available crush models.
func newDiscoverHandler() func(context.Context, *mcp.CallToolRequest, discoverInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input discoverInput) (*mcp.CallToolResult, any, error) {
		if len(input.Filter) > 200 {
			return mcpErr("filter must not exceed 200 characters")
		}

		crushPath, err := exec.LookPath("crush")
		if err != nil {
			return mcpErr("crush binary not found in PATH; install crush to use discover_models: %v", err)
		}

		discoverCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(discoverCtx, crushPath, "models")
		out, err := cmd.Output()
		if err != nil {
			return mcpErr("crush models failed: %v", err)
		}

		// Parse: keep only lines containing "/".
		var models []string
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if !strings.Contains(line, "/") {
				continue
			}
			models = append(models, line)
		}

		// Apply ESQUISSE_ALLOWED_PROVIDERS filter (same logic as buildModelPool).
		if allowedRaw := os.Getenv("ESQUISSE_ALLOWED_PROVIDERS"); allowedRaw != "" {
			allowedSet := make(map[string]bool)
			for _, p := range strings.Split(allowedRaw, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					allowedSet[p] = true
				}
			}
			if len(allowedSet) > 0 {
				filtered := models[:0]
				for _, m := range models {
					if allowedSet[providerOf(m)] {
						filtered = append(filtered, m)
					}
				}
				models = filtered
			}
		}

		// Apply user filter.
		if input.Filter != "" {
			lower := strings.ToLower(input.Filter)
			out2 := models[:0]
			for _, m := range models {
				if strings.Contains(strings.ToLower(m), lower) {
					out2 = append(out2, m)
				}
			}
			models = out2
		}

		text := strings.Join(models, "\n")
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, nil, nil
	}
}
