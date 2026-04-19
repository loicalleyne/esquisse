// Package main — adversarial_review tool implementation.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var verdictRe = regexp.MustCompile(`(?m)^Verdict:\s*(PASSED|CONDITIONAL|FAILED)`)

func mcpErr(format string, args ...any) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf(format, args...)},
		},
	}, nil, nil
}

// extractVerdict returns the verdict string from output using verdictRe.
func extractVerdict(output string) string {
	m := verdictRe.FindStringSubmatch(output)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// adversarialInput is the input schema for the adversarial_review tool.
type adversarialInput struct {
	PlanSlug    string `json:"plan_slug"    jsonschema:"Plan slug used as state file name"`
	PlanContent string `json:"plan_content" jsonschema:"Full text of the plan to review"`
	Rounds      int    `json:"rounds,omitempty" jsonschema:"Number of review rounds (default 5, max 50)"`
}

func newAdversarialHandler(projectRoot string) func(context.Context, *mcp.CallToolRequest, adversarialInput) (*mcp.CallToolResult, any, error) {
	pool := buildModelPool()
	return func(ctx context.Context, req *mcp.CallToolRequest, input adversarialInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(input.PlanContent) == "" {
			return mcpErr("plan_content must not be empty")
		}
		if strings.TrimSpace(input.PlanSlug) == "" {
			return mcpErr("plan_slug must not be empty")
		}
		if len(pool) == 0 {
			return mcpErr("all model slots excluded by ESQUISSE_ALLOWED_PROVIDERS — set ESQUISSE_POOL_FALLBACK_STRICT=0 or allow at least one provider")
		}

		state, err := ReadState(projectRoot, input.PlanSlug)
		if err != nil {
			return mcpErr("failed to read state: %v", err)
		}

		rounds := effectiveRounds(input.Rounds)
		rotOrder := buildRotationOrder(pool, rounds)

		rctx, cancel := context.WithTimeout(ctx, 300*time.Second)
		defer cancel()

		tmpDir, err := os.MkdirTemp("", "esquisse-review-*")
		if err != nil {
			return mcpErr("failed to create tmpDir: %v", err)
		}
		defer func() {
			if rerr := os.RemoveAll(tmpDir); rerr != nil {
				log.Printf("esquisse-mcp: failed to remove review tmpDir %q: %v", tmpDir, rerr)
			}
		}()

		date := time.Now().UTC().Format("2006-01-02")
		var roundOutputs, usedModels, verdicts []string

		for roundIdx := 0; roundIdx < rounds; roundIdx++ {
			roundNum := roundIdx + 1
			preamble := fmt.Sprintf(
				"You are adversarial reviewer for round %d of %d. "+
					"Apply the 7-attack protocol to the plan below.\n"+
					"Write your review report to %s/.adversarial/reports/review-%s-iter%d-r%d-%s.md\n"+
					"Do NOT write the state file — the handler writes it after all rounds complete.\n"+
					"The final line of your report MUST be: Verdict: PASSED|CONDITIONAL|FAILED\n",
				roundNum, rounds,
				projectRoot, date, state.Iteration+roundIdx, roundNum, input.PlanSlug,
			)

			usedModel, output, err := runOneRound(rctx, pool, rotOrder[roundIdx], preamble, input.PlanContent, tmpDir)
			if err != nil {
				return mcpErr("round %d/%d failed: %v", roundNum, rounds, err)
			}

			verdict := extractVerdict(output)
			if verdict == "" {
				log.Printf("esquisse-mcp: round %d produced no valid Verdict: line", roundNum)
			}
			roundOutputs = append(roundOutputs,
				fmt.Sprintf("=== Round %d/%d — %s ===\n%s", roundNum, rounds, usedModel, output))
			usedModels = append(usedModels, usedModel)
			verdicts = append(verdicts, verdict)
		}

		if worstVerdict(verdicts) == "" {
			return mcpErr("no valid Verdict: line found in any round output")
		}

		state.Iteration += rounds
		state.LastModel = usedModels[len(usedModels)-1]
		state.LastVerdict = worstVerdict(verdicts)
		state.LastReviewDate = date
		if err := WriteState(projectRoot, state); err != nil {
			return mcpErr("failed to write state: %v", err)
		}

		summary := fmt.Sprintf("=== Summary ===\nRounds: %d\nModels: %s\nVerdicts: %s\nOverall: %s",
			rounds, strings.Join(usedModels, ", "), strings.Join(verdicts, ", "), state.LastVerdict)
		fullOutput := strings.Join(roundOutputs, "\n\n") + "\n\n" + summary
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fullOutput}},
		}, nil, nil
	}
}
