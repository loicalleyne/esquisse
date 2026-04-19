// Package main — crush run subprocess management.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// RunResult holds the output of a crush run invocation.
type RunResult struct {
	Output   string
	ExitCode int
}

// RunCrush invokes crush run --model {model} --quiet with the given prompt
// file passed via stdin. It does NOT pass the prompt file path as a shell
// argument — this is the security invariant for shell-injection prevention.
func RunCrush(ctx context.Context, model, promptFile string) (RunResult, error) {
	crushPath, err := exec.LookPath("crush")
	if err != nil {
		return RunResult{}, fmt.Errorf("crush binary not found in PATH: %w", err)
	}

	f, err := os.Open(promptFile)
	if err != nil {
		return RunResult{}, fmt.Errorf("cannot open prompt file: %w", err)
	}
	defer f.Close()

	cmd := exec.CommandContext(ctx, crushPath, "run", "--model", model, "--quiet")
	cmd.Stdin = f
	out, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return RunResult{}, fmt.Errorf("crush run: %w", err)
		}
	}
	return RunResult{Output: string(out), ExitCode: exitCode}, nil
}
