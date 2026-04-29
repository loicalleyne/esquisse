// Package main — write_planning_artifact MCP tool.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// artifactInput is the input schema for the write_planning_artifact tool.
type artifactInput struct {
	Title           string   `json:"title"`
	Slug            string   `json:"slug"`
	Source          string   `json:"source"`
	ReferencedBy    []string `json:"referenced_by"`
	Summary         string   `json:"summary"`
	APISurface      string   `json:"api_surface"`
	Constraints     string   `json:"constraints"`
	AntiPatterns    string   `json:"anti_patterns"`
	MinimalExamples string   `json:"minimal_examples,omitempty"`
	ProjectRoot     string   `json:"project_root,omitempty" jsonschema:"Absolute path to the project root. Overrides the --project-root flag set at server startup. Required when the server is shared across multiple projects."`
}

// artifactOutput is the structured response for write_planning_artifact.
type artifactOutput struct {
	Path           string   `json:"path"`
	WordCount      int      `json:"word_count"`
	InjectedInto   []string `json:"injected_into,omitempty"`
	NotFound       []string `json:"not_found,omitempty"`
	AlreadyPresent []string `json:"already_present,omitempty"`
}

// injectionResult is returned by injectPrerequisite to indicate the outcome.
type injectionResult int

const (
	injectionResultInjected       injectionResult = iota // blockquote was written
	injectionResultAlreadyPresent                         // blockquote already present — skip
	injectionResultNotFound                               // task file does not exist on disk yet
)

// injectPrerequisite inserts the prerequisite blockquote immediately after the
// title line of taskPath (resolved relative to projectRoot). It is idempotent:
// calling it twice for the same artifact returns injectionResultAlreadyPresent.
// If taskPath does not exist on disk, returns injectionResultNotFound (not an error).
// Returns a Go error only for I/O failures or a malformed title line (first line
// does not start with "# ").
func injectPrerequisite(projectRoot, artifactRelPath, taskPath string) (injectionResult, error) {
	absTask := filepath.Join(projectRoot, taskPath)
	data, err := os.ReadFile(absTask)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return injectionResultNotFound, nil
		}
		return 0, err
	}
	content := string(data)

	// Idempotency marker.
	marker := "> **Prerequisite:** Read `" + artifactRelPath + "`"
	if strings.Contains(content, marker) {
		return injectionResultAlreadyPresent, nil
	}

	// Require title line as first line.
	lines := strings.SplitN(content, "\n", 2)
	if !strings.HasPrefix(lines[0], "# ") {
		return 0, fmt.Errorf("injectPrerequisite: %s: first line is not a markdown title", taskPath)
	}

	blockquote := "\n" + marker + " before writing any code in this phase.\n"
	var newContent string
	if len(lines) == 2 {
		newContent = lines[0] + blockquote + lines[1]
	} else {
		newContent = lines[0] + blockquote
	}

	if err := os.WriteFile(absTask, []byte(newContent), 0o644); err != nil {
		return 0, err
	}
	return injectionResultInjected, nil
}

// newArtifactHandler returns the handler for the write_planning_artifact tool.
func newArtifactHandler(projectRoot string) func(context.Context, *mcp.CallToolRequest, artifactInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input artifactInput) (*mcp.CallToolResult, any, error) {
		effectiveRoot := projectRoot
		if strings.TrimSpace(input.ProjectRoot) != "" {
			effectiveRoot = strings.TrimSpace(input.ProjectRoot)
		} else if effectiveRoot == "" {
			return mcpErr("project_root is required: pass the absolute path to the project")
		}
		// 1. Slug validation.
		if err := validateSlug(input.Slug); err != nil {
			return mcpErr("%v", err)
		}

		// 2. Required fields.
		required := []struct {
			name  string
			value string
		}{
			{"title", input.Title},
			{"source", input.Source},
			{"summary", input.Summary},
			{"api_surface", input.APISurface},
			{"constraints", input.Constraints},
			{"anti_patterns", input.AntiPatterns},
		}
		for _, f := range required {
			if strings.TrimSpace(f.value) == "" {
				return mcpErr("required field %q is empty", f.name)
			}
		}

		// 3. ReferencedBy validation (fail-fast).
		for _, entry := range input.ReferencedBy {
			cleaned := filepath.Clean(entry)
			if strings.Contains(cleaned, "..") {
				return mcpErr("invalid referenced_by path: %s", entry)
			}
		}

		// 4. Build ReferencedBy links.
		var links []string
		for _, entry := range input.ReferencedBy {
			stem := strings.TrimSuffix(filepath.Base(entry), ".md")
			base := filepath.Base(entry)
			links = append(links, "["+stem+"](../tasks/"+base+")")
		}
		referencedByLinks := strings.Join(links, ", ")

		// 5. Build file date.
		today := time.Now().UTC().Format("2006-01-02")

		// 6. Build file path.
		filename := today + "-" + input.Slug + ".md"
		dir := filepath.Join(effectiveRoot, "docs", "artifacts")
		absPath := filepath.Join(dir, filename)

		// 7. Create directory.
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, nil, err
		}

		// 8. Assemble file content.
		var sb strings.Builder
		fmt.Fprintf(&sb, "# Artifact: %s\n", input.Title)
		sb.WriteString("\n")
		fmt.Fprintf(&sb, "**Primary Source:** %s\n", input.Source)
		fmt.Fprintf(&sb, "**Date:** %s\n", today)
		sb.WriteString("**Produced by:** EsquissePlan\n")
		fmt.Fprintf(&sb, "**Referenced by:** %s\n", referencedByLinks)
		sb.WriteString("\n---\n")
		sb.WriteString("\n## Summary\n")
		fmt.Fprintf(&sb, "%s\n", input.Summary)
		sb.WriteString("\n## API Surface / Key Facts\n")
		fmt.Fprintf(&sb, "%s\n", input.APISurface)
		sb.WriteString("\n## Constraints\n")
		fmt.Fprintf(&sb, "%s\n", input.Constraints)
		sb.WriteString("\n## Anti-Patterns\n")
		fmt.Fprintf(&sb, "%s\n", input.AntiPatterns)
		if strings.TrimSpace(input.MinimalExamples) != "" {
			sb.WriteString("\n## Minimal Examples\n")
			fmt.Fprintf(&sb, "%s\n", input.MinimalExamples)
		}
		content := sb.String()

		// 9. Write file.
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			return nil, nil, err
		}

		// 10. Compute relative path.
		relPath, err := filepath.Rel(effectiveRoot, absPath)
		if err != nil {
			return nil, nil, err
		}
		relPath = filepath.ToSlash(relPath)

		// 10b. Inject prerequisite blockquote into existing referenced task files.
		var injectedInto, notFound, alreadyPresent []string
		for _, entry := range input.ReferencedBy {
			result, injErr := injectPrerequisite(effectiveRoot, relPath, entry)
			if injErr != nil {
				return nil, nil, injErr
			}
			switch result {
			case injectionResultInjected:
				injectedInto = append(injectedInto, entry)
			case injectionResultAlreadyPresent:
				alreadyPresent = append(alreadyPresent, entry)
			case injectionResultNotFound:
				notFound = append(notFound, entry)
			}
		}

		// 11. Compute word count.
		wordCount := len(strings.Fields(content))

		// 12. Return success.
		out := artifactOutput{
			Path:           relPath,
			WordCount:      wordCount,
			InjectedInto:   injectedInto,
			NotFound:       notFound,
			AlreadyPresent: alreadyPresent,
		}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	}
}
