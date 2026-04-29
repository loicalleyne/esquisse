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
			"runs 'rounds' review passes (default 1, max 50) against the pool, " +
			"and writes the worst-case verdict back to the state file.\n\n" +
			"REQUIRED: always pass 'project_root' as the absolute path to the project you are reviewing " +
			"(e.g. \"/home/user/myproject\"). This server is shared across projects; without project_root " +
			"the state file will be written to the wrong directory.\n\n" +
			"REQUIRED: pass 'plan_files' as a newline-separated list of workspace-relative paths to the " +
			"task documents to review (e.g. \"docs/tasks/P1-001-foo.md\ndocs/tasks/P1-002-bar.md\"). " +
			"The server reads the files directly from project_root — do NOT inline or summarize file contents. " +
			"Passing file contents instead of paths will waste tokens and may exceed context limits.\n\n" +
			"Optional: pass 'exclude_model' with your own full model ID (e.g. \"copilot/claude-sonnet-4.6\") to exclude it from the review pool, " +
			"ensuring reviewers come from a different model than your implementing agent. " +
			"To find your model ID, call the crush_info tool and parse the 'large = {model} ({provider})' line: " +
			"take the text before ' (' as the model name and the text inside '()' as the provider, then concatenate as '{provider}/{model}'. " +
			"If exclude_model is empty, malformed, or would empty the pool, it is silently ignored (no-op).",
	}, newAdversarialHandler(projectRoot))

	mcp.AddTool(server, &mcp.Tool{
		Name: "gate_review",
		Description: "Check whether all adversarial review verdicts in .adversarial/ are PASSED or CONDITIONAL. " +
			"REQUIRED: always pass 'project_root' as the absolute path to the project being checked " +
			"(e.g. \"/home/user/myproject\"). This server is shared across projects; without project_root " +
			"the check will read .adversarial/ from the wrong directory.",
	}, newGateHandler(projectRoot))

	mcp.AddTool(server, &mcp.Tool{
		Name: "write_planning_artifact",
		Description: "Write a Planning Artifact file to docs/artifacts/{date}-{slug}.md. " +
			"Accepts structured research content (API surface, constraints, anti-patterns, " +
			"optional minimal examples) and produces a schema-compliant artifact per SCHEMAS.md §10. " +
			"Returns the written file path (relative to project root) and word count. " +
			"Use during planning, after identifying external libraries needed by ≥ 2 tasks " +
			"or whose API surface exceeds ~400 tokens inline. " +
			"Do NOT use for internal project code — AGENTS.md and llms.txt cover those surfaces.\n\n" +
			"REQUIRED: always pass 'project_root' as the absolute path to the project " +
			"(e.g. \"/home/user/myproject\"). This server is shared across projects; without project_root " +
			"the artifact will be written to the wrong directory.",
	}, newArtifactHandler(projectRoot))
}
