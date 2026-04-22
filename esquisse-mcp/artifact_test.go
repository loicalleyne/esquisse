// Package main — tests for write_planning_artifact tool.
package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fullArtifactInput returns a valid, fully-populated artifactInput for tests.
func fullArtifactInput() artifactInput {
return artifactInput{
Title:           "Test Artifact",
Slug:            "test-artifact",
Source:          "https://example.com",
ReferencedBy:    []string{"docs/tasks/P2-007-planning-artifact-schema.md"},
Summary:         "Test summary.",
APISurface:      "| Sym | Val | Src |\n|---|---|---|\n| Foo | bar | baz |",
Constraints:     "MUST do X (source: test)",
AntiPatterns:    "Wrong: A. Right: B. Why: C.",
MinimalExamples: "```go\nfoo()\n```",
}
}

func decodeArtifactOutput(t *testing.T, result *mcp.CallToolResult) artifactOutput {
t.Helper()
tc, ok := result.Content[0].(*mcp.TextContent)
if !ok {
t.Fatalf("expected *mcp.TextContent, got %T", result.Content[0])
}
var out artifactOutput
if err := json.Unmarshal([]byte(tc.Text), &out); err != nil {
t.Fatalf("failed to decode artifactOutput: %v (text: %s)", err, tc.Text)
}
return out
}

func TestWritePlanningArtifact(t *testing.T) {
root := t.TempDir()
handler := newArtifactHandler(root)

result, _, err := handler(context.Background(), nil, fullArtifactInput())
if err != nil {
t.Fatalf("unexpected Go error: %v", err)
}
if result.IsError {
tc, _ := result.Content[0].(*mcp.TextContent)
t.Fatalf("expected success, got tool error: %s", tc.Text)
}

out := decodeArtifactOutput(t, result)

if out.Path == "" {
t.Fatal("output Path is empty")
}
if !strings.HasPrefix(out.Path, "docs/artifacts/") {
t.Fatalf("expected Path to start with docs/artifacts/, got %q", out.Path)
}
if out.WordCount <= 0 {
t.Fatalf("expected WordCount > 0, got %d", out.WordCount)
}

absPath := filepath.Join(root, filepath.FromSlash(out.Path))
data, err := os.ReadFile(absPath)
if err != nil {
t.Fatalf("file not found at %s: %v", absPath, err)
}
fileContent := string(data)
expectedWordCount := len(strings.Fields(fileContent))
if out.WordCount != expectedWordCount {
t.Fatalf("WordCount mismatch: got %d, want %d", out.WordCount, expectedWordCount)
}
if !strings.Contains(fileContent, "[P2-007-planning-artifact-schema](../tasks/P2-007-planning-artifact-schema.md)") {
t.Error("file content missing expected referenced_by link")
}
if !strings.Contains(fileContent, "## Minimal Examples") {
t.Error("file content missing ## Minimal Examples section")
}
}

func TestWritePlanningArtifact_SlugValidation(t *testing.T) {
root := t.TempDir()
handler := newArtifactHandler(root)

inp := fullArtifactInput()
inp.Slug = "../evil"
result, _, err := handler(context.Background(), nil, inp)
if err != nil {
t.Fatalf("unexpected Go error: %v", err)
}
if !result.IsError {
t.Fatal("expected tool error for invalid slug, got success")
}
entries, _ := filepath.Glob(filepath.Join(root, "docs", "artifacts", "*.md"))
if len(entries) > 0 {
t.Fatal("file was written despite invalid slug")
}
}

func TestWritePlanningArtifact_MinimalExamplesOmitted(t *testing.T) {
root := t.TempDir()
handler := newArtifactHandler(root)

inp := fullArtifactInput()
inp.MinimalExamples = ""
result, _, err := handler(context.Background(), nil, inp)
if err != nil {
t.Fatalf("unexpected Go error: %v", err)
}
if result.IsError {
tc, _ := result.Content[0].(*mcp.TextContent)
t.Fatalf("unexpected tool error: %s", tc.Text)
}

out := decodeArtifactOutput(t, result)
data, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(out.Path)))
if strings.Contains(string(data), "## Minimal Examples") {
t.Error("file should not contain ## Minimal Examples when field is empty")
}
}

func TestWritePlanningArtifact_MinimalExamplesWhitespaceOmitted(t *testing.T) {
root := t.TempDir()
handler := newArtifactHandler(root)

inp := fullArtifactInput()
inp.MinimalExamples = "   \n  "
result, _, err := handler(context.Background(), nil, inp)
if err != nil {
t.Fatalf("unexpected Go error: %v", err)
}
if result.IsError {
tc, _ := result.Content[0].(*mcp.TextContent)
t.Fatalf("unexpected tool error: %s", tc.Text)
}

out := decodeArtifactOutput(t, result)
data, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(out.Path)))
if strings.Contains(string(data), "## Minimal Examples") {
t.Error("file should not contain ## Minimal Examples for whitespace-only field")
}
}

func TestWritePlanningArtifact_MissingRequiredField(t *testing.T) {
	cases := []struct {
		field  string
		mutate func(*artifactInput)
	}{
		{"title", func(a *artifactInput) { a.Title = "" }},
		{"source", func(a *artifactInput) { a.Source = "" }},
		{"summary", func(a *artifactInput) { a.Summary = "" }},
		{"api_surface", func(a *artifactInput) { a.APISurface = "" }},
		{"constraints", func(a *artifactInput) { a.Constraints = "" }},
		{"anti_patterns", func(a *artifactInput) { a.AntiPatterns = "" }},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.field, func(t *testing.T) {
			root := t.TempDir()
			handler := newArtifactHandler(root)
			inp := artifactInput{
				Title:        "Test",
				Slug:         "test-artifact",
				Source:       "https://example.com",
				ReferencedBy: []string{},
				Summary:      "Test summary.",
				APISurface:   "| S | V | Src |\n|---|---|---|\n| F | v | s |",
				Constraints:  "MUST X (source: test)",
				AntiPatterns: "Wrong: A. Right: B. Why: C.",
			}
			tc.mutate(&inp)
			result, _, err := handler(context.Background(), nil, inp)
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			if !result.IsError {
				t.Fatalf("expected tool error for missing %q, got success", tc.field)
			}
			content, _ := result.Content[0].(*mcp.TextContent)
			if !strings.Contains(content.Text, tc.field) {
				t.Errorf("error message should mention %q, got: %s", tc.field, content.Text)
			}
		})
	}
}

func TestWritePlanningArtifact_ReferencedByLinks(t *testing.T) {
root := t.TempDir()
handler := newArtifactHandler(root)

inp := fullArtifactInput()
inp.ReferencedBy = []string{"docs/tasks/P2-007-foo.md"}
result, _, err := handler(context.Background(), nil, inp)
if err != nil {
t.Fatalf("unexpected Go error: %v", err)
}
if result.IsError {
tc, _ := result.Content[0].(*mcp.TextContent)
t.Fatalf("unexpected tool error: %s", tc.Text)
}

out := decodeArtifactOutput(t, result)
data, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(out.Path)))
if !strings.Contains(string(data), "[P2-007-foo](../tasks/P2-007-foo.md)") {
t.Errorf("expected referenced_by link not found in file content")
}
}

func TestWritePlanningArtifact_ReferencedByTraversal(t *testing.T) {
root := t.TempDir()
handler := newArtifactHandler(root)

inp := fullArtifactInput()
inp.ReferencedBy = []string{"../../../etc/passwd"}
result, _, err := handler(context.Background(), nil, inp)
if err != nil {
t.Fatalf("unexpected Go error: %v", err)
}
if !result.IsError {
t.Fatal("expected tool error for path traversal in referenced_by")
}
entries, _ := filepath.Glob(filepath.Join(root, "docs", "artifacts", "*.md"))
if len(entries) > 0 {
t.Fatal("file was written despite traversal path")
}
}

func TestWritePlanningArtifact_ReferencedByEmpty(t *testing.T) {
root := t.TempDir()
handler := newArtifactHandler(root)

inp := fullArtifactInput()
inp.ReferencedBy = []string{}
result, _, err := handler(context.Background(), nil, inp)
if err != nil {
t.Fatalf("unexpected Go error: %v", err)
}
if result.IsError {
tc, _ := result.Content[0].(*mcp.TextContent)
t.Fatalf("unexpected tool error: %s", tc.Text)
}

out := decodeArtifactOutput(t, result)
data, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(out.Path)))
if !strings.Contains(string(data), "**Referenced by:**") {
t.Error("file should contain **Referenced by:** line even when slice is empty")
}
}

// --- P2-011: injectPrerequisite unit tests ---

func TestInjectPrerequisite_InsertsAfterTitle(t *testing.T) {
	root := t.TempDir()
	taskPath := "docs/tasks/P2-010-some-task.md"
	absTask := filepath.Join(root, taskPath)
	if err := os.MkdirAll(filepath.Dir(absTask), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(absTask, []byte("# P2-010: Some Task\n\n## Status\nStatus: Ready\n"), 0o644); err != nil {
		t.Fatalf("write task: %v", err)
	}

	artifactRel := "docs/artifacts/2026-04-21-foo.md"
	result, err := injectPrerequisite(root, artifactRel, taskPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != injectionResultInjected {
		t.Fatalf("expected injectionResultInjected, got %v", result)
	}

	data, _ := os.ReadFile(absTask)
	content := string(data)
	marker := "> **Prerequisite:** Read `" + artifactRel + "` before writing any code in this phase."
	if !strings.Contains(content, marker) {
		t.Errorf("expected blockquote not found in file:\n%s", content)
	}
	markerIdx := strings.Index(content, marker)
	statusIdx := strings.Index(content, "## Status")
	if markerIdx > statusIdx {
		t.Errorf("blockquote must appear before ## Status (markerIdx=%d, statusIdx=%d)", markerIdx, statusIdx)
	}
}

func TestInjectPrerequisite_AlreadyPresent(t *testing.T) {
	root := t.TempDir()
	taskPath := "docs/tasks/P2-010-some-task.md"
	absTask := filepath.Join(root, taskPath)
	if err := os.MkdirAll(filepath.Dir(absTask), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(absTask, []byte("# P2-010: Some Task\n\n## Status\nStatus: Ready\n"), 0o644); err != nil {
		t.Fatalf("write task: %v", err)
	}

	artifactRel := "docs/artifacts/2026-04-21-foo.md"
	r1, err := injectPrerequisite(root, artifactRel, taskPath)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if r1 != injectionResultInjected {
		t.Fatalf("first call: expected injectionResultInjected, got %v", r1)
	}

	r2, err := injectPrerequisite(root, artifactRel, taskPath)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if r2 != injectionResultAlreadyPresent {
		t.Fatalf("second call: expected injectionResultAlreadyPresent, got %v", r2)
	}

	data, _ := os.ReadFile(absTask)
	marker := "> **Prerequisite:** Read `" + artifactRel + "`"
	count := strings.Count(string(data), marker)
	if count != 1 {
		t.Errorf("expected exactly 1 occurrence of marker, got %d", count)
	}
}

func TestInjectPrerequisite_NotFound(t *testing.T) {
	root := t.TempDir()
	result, err := injectPrerequisite(root, "docs/artifacts/2026-04-21-foo.md", "docs/tasks/nonexistent.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != injectionResultNotFound {
		t.Fatalf("expected injectionResultNotFound, got %v", result)
	}
}

func TestInjectPrerequisite_MalformedTitle(t *testing.T) {
	root := t.TempDir()
	taskPath := "docs/tasks/P2-010-bad.md"
	absTask := filepath.Join(root, taskPath)
	if err := os.MkdirAll(filepath.Dir(absTask), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	originalContent := "Status: Ready\n\n## Status\nStatus: Ready\n"
	if err := os.WriteFile(absTask, []byte(originalContent), 0o644); err != nil {
		t.Fatalf("write task: %v", err)
	}

	_, err := injectPrerequisite(root, "docs/artifacts/2026-04-21-foo.md", taskPath)
	if err == nil {
		t.Fatal("expected non-nil error for malformed title, got nil")
	}

	data, _ := os.ReadFile(absTask)
	if string(data) != originalContent {
		t.Errorf("file content must be unchanged on error;\ngot:  %q\nwant: %q", string(data), originalContent)
	}
}

// --- P2-011: handler-level injection tests ---

func TestWritePlanningArtifact_InjectsPrerequisite(t *testing.T) {
	root := t.TempDir()
	taskPath := "docs/tasks/P2-007-planning-artifact-schema.md"
	absTask := filepath.Join(root, taskPath)
	if err := os.MkdirAll(filepath.Dir(absTask), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(absTask, []byte("# P2-007: Planning Artifact Schema\n\n## Status\nStatus: Ready\n"), 0o644); err != nil {
		t.Fatalf("write task: %v", err)
	}

	inp := fullArtifactInput()
	inp.ReferencedBy = []string{taskPath}
	handler := newArtifactHandler(root)
	result, _, err := handler(context.Background(), nil, inp)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.IsError {
		tc, _ := result.Content[0].(*mcp.TextContent)
		t.Fatalf("unexpected tool error: %s", tc.Text)
	}

	out := decodeArtifactOutput(t, result)
	if !contains(out.InjectedInto, taskPath) {
		t.Errorf("expected InjectedInto to contain %q, got %v", taskPath, out.InjectedInto)
	}

	data, _ := os.ReadFile(absTask)
	marker := "> **Prerequisite:** Read `" + out.Path + "` before writing any code in this phase."
	if !strings.Contains(string(data), marker) {
		t.Errorf("expected blockquote in task file:\n%s", string(data))
	}
}

func TestWritePlanningArtifact_InjectIdempotent(t *testing.T) {
	root := t.TempDir()
	taskPath := "docs/tasks/P2-007-planning-artifact-schema.md"
	absTask := filepath.Join(root, taskPath)
	if err := os.MkdirAll(filepath.Dir(absTask), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(absTask, []byte("# P2-007: Planning Artifact Schema\n\n## Status\nStatus: Ready\n"), 0o644); err != nil {
		t.Fatalf("write task: %v", err)
	}

	inp := fullArtifactInput()
	inp.ReferencedBy = []string{taskPath}
	handler := newArtifactHandler(root)
	result, _, err := handler(context.Background(), nil, inp)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	out := decodeArtifactOutput(t, result)

	// Second injection via direct call (idempotency).
	r2, err := injectPrerequisite(root, out.Path, taskPath)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if r2 != injectionResultAlreadyPresent {
		t.Fatalf("expected injectionResultAlreadyPresent on second call, got %v", r2)
	}

	data, _ := os.ReadFile(absTask)
	marker := "> **Prerequisite:** Read `" + out.Path + "`"
	if strings.Count(string(data), marker) != 1 {
		t.Errorf("expected exactly 1 occurrence of marker, got %d", strings.Count(string(data), marker))
	}
}

func TestWritePlanningArtifact_InjectNotFound(t *testing.T) {
	root := t.TempDir()
	inp := fullArtifactInput()
	inp.ReferencedBy = []string{"docs/tasks/nonexistent.md"}
	handler := newArtifactHandler(root)
	result, _, err := handler(context.Background(), nil, inp)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.IsError {
		tc, _ := result.Content[0].(*mcp.TextContent)
		t.Fatalf("unexpected tool error: %s", tc.Text)
	}

	out := decodeArtifactOutput(t, result)
	if !contains(out.NotFound, "docs/tasks/nonexistent.md") {
		t.Errorf("expected NotFound to contain nonexistent.md, got %v", out.NotFound)
	}
	if len(out.InjectedInto) != 0 {
		t.Errorf("expected InjectedInto to be empty, got %v", out.InjectedInto)
	}
}

func TestWritePlanningArtifact_InjectMultipleArtifacts(t *testing.T) {
	root := t.TempDir()
	taskPath := "docs/tasks/P2-010-shared.md"
	absTask := filepath.Join(root, taskPath)
	if err := os.MkdirAll(filepath.Dir(absTask), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(absTask, []byte("# P2-010: Shared Task\n\n## Status\nStatus: Ready\n"), 0o644); err != nil {
		t.Fatalf("write task: %v", err)
	}

	handler := newArtifactHandler(root)

	inpAlpha := fullArtifactInput()
	inpAlpha.Slug = "alpha"
	inpAlpha.ReferencedBy = []string{taskPath}
	r1, _, err := handler(context.Background(), nil, inpAlpha)
	if err != nil {
		t.Fatalf("alpha call error: %v", err)
	}
	out1 := decodeArtifactOutput(t, r1)
	if !contains(out1.InjectedInto, taskPath) {
		t.Errorf("alpha: expected InjectedInto to contain task, got %v", out1.InjectedInto)
	}

	inpBeta := fullArtifactInput()
	inpBeta.Slug = "beta"
	inpBeta.ReferencedBy = []string{taskPath}
	r2, _, err := handler(context.Background(), nil, inpBeta)
	if err != nil {
		t.Fatalf("beta call error: %v", err)
	}
	out2 := decodeArtifactOutput(t, r2)
	if !contains(out2.InjectedInto, taskPath) {
		t.Errorf("beta: expected InjectedInto to contain task, got %v", out2.InjectedInto)
	}

	data, _ := os.ReadFile(absTask)
	content := string(data)
	markerAlpha := "> **Prerequisite:** Read `" + out1.Path + "`"
	markerBeta := "> **Prerequisite:** Read `" + out2.Path + "`"
	if !strings.Contains(content, markerAlpha) {
		t.Errorf("task file missing alpha blockquote")
	}
	if !strings.Contains(content, markerBeta) {
		t.Errorf("task file missing beta blockquote")
	}
}

func TestWritePlanningArtifact_ReferencedByEmpty_NoInjection(t *testing.T) {
	root := t.TempDir()
	inp := fullArtifactInput()
	inp.ReferencedBy = []string{}
	handler := newArtifactHandler(root)
	result, _, err := handler(context.Background(), nil, inp)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.IsError {
		tc, _ := result.Content[0].(*mcp.TextContent)
		t.Fatalf("unexpected tool error: %s", tc.Text)
	}

	out := decodeArtifactOutput(t, result)
	if len(out.InjectedInto) != 0 {
		t.Errorf("expected InjectedInto to be nil/empty, got %v", out.InjectedInto)
	}
	if len(out.NotFound) != 0 {
		t.Errorf("expected NotFound to be nil/empty, got %v", out.NotFound)
	}
	if len(out.AlreadyPresent) != 0 {
		t.Errorf("expected AlreadyPresent to be nil/empty, got %v", out.AlreadyPresent)
	}
}

// contains is a helper for slice membership checks.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
