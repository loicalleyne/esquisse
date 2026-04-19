// Package main — tests for state.go and gate.go security-relevant logic.
package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestValidateSlug verifies that validateSlug accepts valid slugs and rejects
// path traversal and separator-containing values.
func TestValidateSlug(t *testing.T) {
	t.Parallel()
	valid := []string{
		"my-plan",
		"P1-004-esquisse-mcp-server",
		"plan123",
		"a",
	}
	for _, slug := range valid {
		if err := validateSlug(slug); err != nil {
			t.Errorf("validateSlug(%q) returned unexpected error: %v", slug, err)
		}
	}

	invalid := []string{
		"../evil",
		"../../etc/passwd",
		"foo/bar",
		"foo\\bar",
		"./local",
		"sub/dir/plan",
	}
	for _, slug := range invalid {
		if err := validateSlug(slug); err == nil {
			t.Errorf("validateSlug(%q) expected error but got nil", slug)
		}
	}
}

// TestReadWriteState verifies round-trip read/write of ReviewState using a
// temporary directory.
func TestReadWriteState(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	slug := "test-plan"

	// ReadState on missing file returns zero-value with PlanSlug filled.
	s, err := ReadState(root, slug)
	if err != nil {
		t.Fatalf("ReadState on absent file: %v", err)
	}
	if s.PlanSlug != slug {
		t.Errorf("got PlanSlug=%q, want %q", s.PlanSlug, slug)
	}
	if s.Iteration != 0 {
		t.Errorf("got Iteration=%d, want 0", s.Iteration)
	}

	// WriteState creates the file.
	want := ReviewState{
		PlanSlug:       slug,
		Iteration:      3,
		LastModel:      "openai/gpt-4.1",
		LastVerdict:    "PASSED",
		LastReviewDate: "2025-01-01",
	}
	if err := WriteState(root, want); err != nil {
		t.Fatalf("WriteState: %v", err)
	}

	// ReadState returns what was written.
	got, err := ReadState(root, slug)
	if err != nil {
		t.Fatalf("ReadState after write: %v", err)
	}
	if got != want {
		t.Errorf("round-trip mismatch:\n  got  %+v\n  want %+v", got, want)
	}

	// .adversarial/ directory should be mode 0700.
	info, err := os.Stat(filepath.Join(root, ".adversarial"))
	if err != nil {
		t.Fatalf("stat .adversarial: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf(".adversarial dir perm = %o, want 0700", perm)
	}
}

// TestReadStateInvalidSlug verifies that ReadState rejects traversal slugs
// before touching the filesystem.
func TestReadStateInvalidSlug(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, err := ReadState(root, "../escape"); err == nil {
		t.Error("ReadState with traversal slug expected error, got nil")
	}
}

// TestWriteStateInvalidSlug verifies that WriteState rejects traversal slugs.
func TestWriteStateInvalidSlug(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	s := ReviewState{PlanSlug: "../../escape"}
	if err := WriteState(root, s); err == nil {
		t.Error("WriteState with traversal slug expected error, got nil")
	}
}

// TestGateHandlerNoFiles verifies non-strict and strict behavior when the
// .adversarial/ directory contains no state files.
func TestGateHandlerNoFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	handler := newGateHandler(root)

	// Non-strict: not blocked.
	res, _, err := handler(context.Background(), nil, gateInput{Strict: false})
	if err != nil {
		t.Fatalf("handler(strict=false): %v", err)
	}
	if res.IsError {
		t.Fatalf("handler(strict=false) returned IsError=true")
	}
	var out gateOutput
	if err := json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Blocked {
		t.Errorf("strict=false, no files: expected Blocked=false, got true")
	}

	// Strict: blocked.
	res, _, err = handler(context.Background(), nil, gateInput{Strict: true})
	if err != nil {
		t.Fatalf("handler(strict=true): %v", err)
	}
	if res.IsError {
		t.Fatalf("handler(strict=true) returned IsError=true")
	}
	if err := json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.Blocked {
		t.Errorf("strict=true, no files: expected Blocked=true, got false")
	}
}

// TestGateHandlerWithFiles verifies that PASSED/CONDITIONAL verdicts pass and
// FAILED (or empty) verdicts block.
func TestGateHandlerWithFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	writeState := func(slug, verdict string) {
		t.Helper()
		if err := WriteState(root, ReviewState{
			PlanSlug:    slug,
			Iteration:   1,
			LastVerdict: verdict,
		}); err != nil {
			t.Fatalf("WriteState(%q): %v", slug, err)
		}
	}

	// One PASSED plan — should not block.
	writeState("plan-a", "PASSED")
	handler := newGateHandler(root)
	res, _, err := handler(context.Background(), nil, gateInput{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	var out gateOutput
	_ = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
	if out.Blocked {
		t.Errorf("PASSED verdict: expected Blocked=false, got true; reason=%q", out.Reason)
	}

	// Add a FAILED plan — should block.
	writeState("plan-b", "FAILED")
	res, _, err = handler(context.Background(), nil, gateInput{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	_ = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
	if !out.Blocked {
		t.Errorf("FAILED verdict: expected Blocked=true, got false")
	}
	if len(out.BlockingPlans) != 1 || out.BlockingPlans[0] != "plan-b" {
		t.Errorf("BlockingPlans = %v, want [plan-b]", out.BlockingPlans)
	}

	// CONDITIONAL also passes.
	writeState("plan-b", "CONDITIONAL")
	res, _, err = handler(context.Background(), nil, gateInput{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	_ = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
	if out.Blocked {
		t.Errorf("CONDITIONAL verdict: expected Blocked=false, got true")
	}
}
