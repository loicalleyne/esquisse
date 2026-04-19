// Package main — tool registration.
package main

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerTools(server *mcp.Server, projectRoot string) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "adversarial_review",
		Description: "Dispatch one or more adversarial review rounds for the given plan using a configurable " +
			"5-slot model pool with family-interleaved rotation. " +
			"Reads .adversarial/{plan_slug}.json for current iteration state, " +
			"runs 'rounds' review passes (default 5, max 50) against the pool, " +
			"and writes the worst-case verdict back to the state file.",
	}, newAdversarialHandler(projectRoot))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gate_review",
		Description: "Check whether all adversarial review verdicts in .adversarial/ are PASSED or CONDITIONAL.",
	}, newGateHandler(projectRoot))

	mcp.AddTool(server, &mcp.Tool{
		Name: "discover_models",
		Description: "List available crush models, optionally filtered by a substring. " +
			"Respects ESQUISSE_ALLOWED_PROVIDERS to show only permitted models. " +
			"Use this tool to verify which models are accessible before running adversarial_review.",
	}, newDiscoverHandler())
}
