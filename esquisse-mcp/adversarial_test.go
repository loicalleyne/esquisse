package main

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

func TestWriteReportFile_PathFormat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := writeReportFile(dir, "2026-04-30", "my-slug", "gpt-4.1", 3, 1, 1, "body"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}
	name := entries[0].Name()
	pattern := regexp.MustCompile(`^iter03-2026-04-30-\d{4}-review\.md$`)
	if !pattern.MatchString(name) {
		t.Errorf("filename %q does not match pattern %s", name, pattern)
	}
}

func TestWriteReportFile_CreatesDir(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	reportsDir := filepath.Join(base, "subdir", "reports")
	if err := writeReportFile(reportsDir, "2026-04-30", "my-slug", "gpt-4.1", 1, 1, 1, "body"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		t.Errorf("expected directory %q to exist after writeReportFile", reportsDir)
	}
}

func TestWriteReportFile_ContentHeader(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := writeReportFile(dir, "2026-04-30", "my-slug", "gpt-4.1", 1, 1, 1, "body"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !regexp.MustCompile(`# Adversarial Review Report: my-slug`).Match(data) {
		t.Errorf("content does not contain expected header; got:\n%s", data)
	}
}

func TestWriteReportFile_ZeroPadsIteration(t *testing.T) {
	t.Parallel()
	dir1 := t.TempDir()
	if err := writeReportFile(dir1, "2026-04-30", "slug", "model", 1, 1, 1, "body"); err != nil {
		t.Fatalf("iteration=1: unexpected error: %v", err)
	}
	entries1, err := os.ReadDir(dir1)
	if err != nil {
		t.Fatalf("iteration=1: ReadDir: %v", err)
	}
	if len(entries1) != 1 {
		t.Fatalf("iteration=1: expected 1 file, got %d", len(entries1))
	}
	if !regexp.MustCompile(`^iter01-`).MatchString(entries1[0].Name()) {
		t.Errorf("iteration=1: filename %q does not start with iter01-", entries1[0].Name())
	}

	dir10 := t.TempDir()
	if err := writeReportFile(dir10, "2026-04-30", "slug", "model", 10, 1, 1, "body"); err != nil {
		t.Fatalf("iteration=10: unexpected error: %v", err)
	}
	entries10, err := os.ReadDir(dir10)
	if err != nil {
		t.Fatalf("iteration=10: ReadDir: %v", err)
	}
	if len(entries10) != 1 {
		t.Fatalf("iteration=10: expected 1 file, got %d", len(entries10))
	}
	if !regexp.MustCompile(`^iter10-`).MatchString(entries10[0].Name()) {
		t.Errorf("iteration=10: filename %q does not start with iter10-", entries10[0].Name())
	}
}

func TestWriteReportFile_NonWritableDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-writable dir test not supported on Windows")
	}
	t.Parallel()
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if err := os.Chmod(dir, 0o444); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	err := writeReportFile(dir, "2026-04-30", "slug", "model", 1, 1, 1, "body")
	if err == nil {
		t.Error("expected error writing to non-writable directory, got nil")
	}
}
