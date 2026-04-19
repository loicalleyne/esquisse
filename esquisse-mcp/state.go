// Package main — .adversarial/ state file read/write.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReviewState is the canonical schema from SCHEMAS.md §8.
// Field names must match gate-review.sh exactly.
type ReviewState struct {
	PlanSlug       string `json:"plan_slug"`
	Iteration      int    `json:"iteration"`
	LastModel      string `json:"last_model"`
	LastVerdict    string `json:"last_verdict"`
	LastReviewDate string `json:"last_review_date"`
}

// stateDir returns the .adversarial/ directory path.
func stateDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".adversarial")
}

// validateSlug ensures planSlug contains no path separators or traversal
// sequences that could escape the .adversarial/ directory.
func validateSlug(planSlug string) error {
	clean := filepath.Clean(planSlug)
	if clean != planSlug || strings.ContainsAny(planSlug, "/\\") {
		return fmt.Errorf("invalid plan_slug %q: must not contain path separators", planSlug)
	}
	return nil
}

// statePath returns the full path to the state file for the given plan slug.
func statePath(projectRoot, planSlug string) string {
	return filepath.Join(stateDir(projectRoot), planSlug+".json")
}

// ReadState reads the state file for the given plan slug.
// Returns a zero-value ReviewState (iteration=0) if the file does not exist.
func ReadState(projectRoot, planSlug string) (ReviewState, error) {
	if err := validateSlug(planSlug); err != nil {
		return ReviewState{}, err
	}
	path := statePath(projectRoot, planSlug)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ReviewState{PlanSlug: planSlug}, nil
		}
		return ReviewState{}, err
	}
	var s ReviewState
	if err := json.Unmarshal(data, &s); err != nil {
		return ReviewState{}, err
	}
	return s, nil
}

// WriteState atomically writes the state file via temp file + rename.
// Creates .adversarial/ if absent.
func WriteState(projectRoot string, s ReviewState) error {
	if err := validateSlug(s.PlanSlug); err != nil {
		return err
	}
	dir := stateDir(projectRoot)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".state-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // clean up on error
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, statePath(projectRoot, s.PlanSlug))
}
