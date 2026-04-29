// Package main — adversarial_review tool implementation.
package main

import (
	_ "embed"

	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed embedded/task-review-protocol.md
var attackProtocol string

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
	PlanSlug    string `json:"plan_slug"               jsonschema:"Plan slug used as state file name"`
	PlanFiles   string `json:"plan_files"              jsonschema:"Newline-separated list of workspace-relative paths to task files (e.g. docs/tasks/P1-001-foo.md). The server reads files from project_root — do NOT inline file contents."`
	Rounds      int    `json:"rounds,omitempty"        jsonschema:"Number of review rounds (default 1, max 50)"`
	ExcludeModel string `json:"exclude_model,omitempty" jsonschema:"Full model ID to exclude from review pool (e.g. copilot/claude-sonnet-4.6). Obtain from crush_info tool. Empty or omitted = no exclusion."`
	ProjectRoot string `json:"project_root,omitempty"  jsonschema:"Absolute path to the project root. Overrides the --project-root flag set at server startup. Required when the server is shared across multiple projects."`
}

func newAdversarialHandler(projectRoot string) func(context.Context, *mcp.CallToolRequest, adversarialInput) (*mcp.CallToolResult, any, error) {
	pool := buildModelPool()
	return func(ctx context.Context, req *mcp.CallToolRequest, input adversarialInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(input.PlanFiles) == "" {
			return mcpErr("plan_files must not be empty: pass newline-separated workspace-relative paths to task files")
		}
		if strings.TrimSpace(input.PlanSlug) == "" {
			return mcpErr("plan_slug must not be empty")
		}
		if len(pool) == 0 {
			return mcpErr("ESQUISSE_MODELS produced an empty model pool; set ESQUISSE_MODELS to a comma-separated list of provider/model entries")
		}
		effectiveRoot := projectRoot
		if strings.TrimSpace(input.ProjectRoot) != "" {
			effectiveRoot = strings.TrimSpace(input.ProjectRoot)
		} else if effectiveRoot == "" {
			return mcpErr("project_root is required: pass the absolute path to the project being reviewed")
		}

		// Read plan files from disk. Never accept inlined content from the caller.
		var planContent strings.Builder
		for _, rel := range strings.Split(strings.TrimSpace(input.PlanFiles), "\n") {
			rel = strings.TrimSpace(rel)
			if rel == "" {
				continue
			}
			abs := filepath.Join(effectiveRoot, rel)
			data, ferr := os.ReadFile(abs)
			if ferr != nil {
				return mcpErr("failed to read plan file %q: %v", rel, ferr)
			}
			planContent.WriteString("--- " + rel + " ---\n")
			planContent.Write(data)
			planContent.WriteString("\n\n")
		}
		if planContent.Len() == 0 {
			return mcpErr("plan_files contained no readable files")
		}
		effectivePool := excludeModelFilter(pool, input.ExcludeModel)

		state, err := ReadState(effectiveRoot, input.PlanSlug)
		if err != nil {
			return mcpErr("failed to read state: %v", err)
		}

		rounds := effectiveRounds(input.Rounds)
		rotOrder := buildRotationOrder(effectivePool, rounds)

		// Ensure .adversarial/reports/ exists so the reviewer model can write report files.
		reportsDir := filepath.Join(effectiveRoot, ".adversarial", "reports")
		if err := os.MkdirAll(reportsDir, 0o700); err != nil {
			return mcpErr("failed to create reports directory %q: %v", reportsDir, err)
		}

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
			iteration := state.Iteration + roundIdx
			preamble := fmt.Sprintf(
				"You are an adversarial reviewer running via esquisse-mcp, round %d of %d.\n\n"+
					"=== REVIEW PROTOCOL ===\n"+
					"%s\n"+
					"=== END REVIEW PROTOCOL ===\n\n"+
					"Apply every attack above to the plan that follows.\n\n"+
					"Produce ONLY the report body — the file header (plan, reviewer, iteration, date) is\n"+
					"added automatically by esquisse-mcp. Do NOT write any files. Do NOT write the state file.\n"+
					"NEVER run rm, Remove-Item, or any destructive command targeting .adversarial/ or its subdirectories.\n"+
					"NEVER delete, overwrite, or move any existing report file under .adversarial/reports/.\n"+
					"Report files are the permanent audit trail — only esquisse-mcp creates them.\n\n"+
					"Your output MUST follow this exact structure:\n\n"+
					"## Attack Results\n\n"+
					"| # | Attack Vector | Result | Notes |\n"+
					"|---|---|---|---|\n"+
					"| 1 | False assumptions | PASSED\\|CONDITIONAL\\|FAILED | … |\n"+
					"| 2 | Edge cases | PASSED\\|CONDITIONAL\\|FAILED | … |\n"+
					"| 3 | Security | PASSED\\|CONDITIONAL\\|FAILED | … |\n"+
					"| 4 | Logic contradictions | PASSED\\|CONDITIONAL\\|FAILED | … |\n"+
					"| 5 | Context blindness | PASSED\\|CONDITIONAL\\|FAILED | … |\n"+
					"| 6 | Failure modes | PASSED\\|CONDITIONAL\\|FAILED | … |\n"+
					"| 7 | Hallucination | PASSED\\|CONDITIONAL\\|FAILED | … |\n\n"+
					"---\n\n"+
					"## Critical Issues (must fix before implementation)\n\n"+
					"{one subsection per FAILED attack vector, or \"None.\"}\n\n"+
					"---\n\n"+
					"## Major Issues (should fix before proceeding)\n\n"+
					"{one subsection per CONDITIONAL attack vector, or \"None.\"}\n\n"+
					"---\n\n"+
					"## Minor Issues (track but not blocking)\n\n"+
					"{any lower-severity observations, or \"None.\"}\n\n"+
					"---\n\n"+
					"Verdict: PASSED|CONDITIONAL|FAILED\n",
				roundNum, rounds, attackProtocol,
			)

			usedModel, output, err := runOneRound(rctx, effectivePool, rotOrder[roundIdx], preamble, planContent.String(), tmpDir)
			if err != nil {
				return mcpErr("round %d/%d failed: %v", roundNum, rounds, err)
			}

			// Write the report file from Go — never instruct the model to write it,
			// as models produce inconsistent filenames.
			if werr := writeReportFile(reportsDir, date, input.PlanSlug, usedModel, iteration, roundNum, rounds, output); werr != nil {
				log.Printf("esquisse-mcp: failed to write report file for round %d: %v", roundNum, werr)
			}

			verdict := extractVerdict(output)
			if verdict == "" {
				log.Printf("esquisse-mcp: round %d produced no valid Verdict: line in output", roundNum)
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
		if err := WriteState(effectiveRoot, state); err != nil {
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

// writeReportFile writes the §9 report to
// .adversarial/reports/review-{date}-{plan-slug}-iter{N}-r{round}.md.
// Grouping by plan-slug after date keeps all reports for the same task together
// in directory listings. Model name belongs in the header, not the filename.
// The header metadata (plan, reviewer, iteration, timestamp) is prepended by Go
// so the model only needs to produce the body content.
func writeReportFile(reportsDir, date, planSlug, usedModel string, iteration, roundNum, rounds int, body string) error {
	fname := fmt.Sprintf("review-%s-%s-iter%d-r%d.md",
		date, planSlug, iteration, roundNum)
	now := time.Now().UTC()
	header := fmt.Sprintf(
		"# Adversarial Review Report: %s\n\n"+
			"**Plan:** %s\n"+
			"**Reviewer:** esquisse-mcp (%s)\n"+
			"**Iteration:** %d\n"+
			"**Round:** %d of %d\n"+
			"**Timestamp:** %s\n\n---\n\n",
		planSlug, planSlug, usedModel, iteration, roundNum, rounds, now.Format(time.RFC3339),
	)
	content := []byte(header + body)
	return os.WriteFile(filepath.Join(reportsDir, fname), content, 0o600)
}
