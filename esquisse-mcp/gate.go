// Package main — gate_review tool implementation.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// gateInput is the input schema for the gate_review tool.
type gateInput struct {
	Strict bool `json:"strict" jsonschema:"If true, block when no state files exist"`
}

// gateOutput is the structured response for gate_review.
type gateOutput struct {
	Blocked       bool     `json:"blocked"`
	Reason        string   `json:"reason"`
	BlockingPlans []string `json:"blocking_plans,omitempty"`
}

func newGateHandler(projectRoot string) func(context.Context, *mcp.CallToolRequest, gateInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input gateInput) (*mcp.CallToolResult, any, error) {
		rawDir := stateDir(projectRoot)
		dir, err := filepath.Abs(filepath.Clean(rawDir))
		if err != nil {
			dir = filepath.Clean(rawDir)
		}
		entries, err := filepath.Glob(filepath.Join(dir, "*.json"))
		if err != nil {
			entries = nil
		}
		// Filter to files directly in dir (not in subdirectories like reports/).
		var files []string
		for _, e := range entries {
			if filepath.Dir(e) == dir {
				files = append(files, e)
			}
		}

		if len(files) == 0 {
			if input.Strict {
				return gateResult(gateOutput{
					Blocked: true,
					Reason:  "adversarial review required before completing this session",
				})
			}
			return gateResult(gateOutput{
				Blocked: false,
				Reason:  "no reviews in progress",
			})
		}

		var blocking []string
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			var s ReviewState
			if err := json.Unmarshal(data, &s); err != nil {
				continue
			}
			v := strings.ToUpper(strings.TrimSpace(s.LastVerdict))
			if v != "PASSED" && v != "CONDITIONAL" {
				slug := strings.TrimSuffix(filepath.Base(f), ".json")
				blocking = append(blocking, slug)
			}
		}

		if len(blocking) > 0 {
			return gateResult(gateOutput{
				Blocked:       true,
				Reason:        fmt.Sprintf("%d plan(s) have FAILED or missing verdicts", len(blocking)),
				BlockingPlans: blocking,
			})
		}
		return gateResult(gateOutput{
			Blocked: false,
			Reason:  "all plans have PASSED or CONDITIONAL verdicts",
		})
	}
}

func gateResult(out gateOutput) (*mcp.CallToolResult, any, error) {
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "internal error: " + err.Error()}},
		}, nil, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}
