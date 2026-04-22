// Command esquisse-mcp is a lightweight MCP stdio server that exposes
// adversarial_review and gate_review tools for the Esquisse framework.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	if runtime.GOOS == "windows" {
		fmt.Fprintln(os.Stderr, "esquisse-mcp does not support Windows — run from Linux, macOS, or WSL")
		os.Exit(1)
	}

	projectRoot := flag.String("project-root", "", "project root directory (default: $PWD)")
	probeFlag := flag.Bool("probe", false, "probe all crush models and print availability, then exit")
	flag.Parse()

	if *projectRoot == "" {
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("cannot determine working directory: %v", err)
		}
		*projectRoot = pwd
	}

	if *probeFlag {
		runProbeAndExit()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cachePath, _ := defaultCachePath()
	prober := newModelProber(cachePath, defaultProbeTTL())
	prober.startProbe(ctx)

	server := mcp.NewServer(&mcp.Implementation{Name: "esquisse-mcp", Version: "0.1.0"}, nil)
	registerTools(server, *projectRoot, prober)

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Printf("server exited: %v", err)
	}
}

// runProbeAndExit lists all crush models, probes each one synchronously with a
// minimal prompt, prints per-model results, writes the cache, then exits.
func runProbeAndExit() {
	crushPath, err := exec.LookPath("crush")
	if err != nil {
		fmt.Fprintf(os.Stderr, "crush not in PATH: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// List models.
	listCtx, cancelList := context.WithTimeout(ctx, 15*time.Second)
	defer cancelList()
	cmd := exec.CommandContext(listCtx, crushPath, "models")
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "crush models failed: %v\n", err)
		os.Exit(1)
	}
	var models []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "/") {
			models = append(models, line)
		}
	}
	fmt.Printf("Found %d models from `crush models`.\n\n", len(models))

	// Write a shared probe prompt file.
	tmpFile, err := os.CreateTemp("", "esquisse-probe-*.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create probe prompt: %v\n", err)
		os.Exit(1)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)
	if _, err := fmt.Fprint(tmpFile, "Reply with the single word: OK"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write probe prompt: %v\n", err)
		os.Exit(1)
	}
	_ = tmpFile.Close()

	cachePath, cacheErr := defaultCachePath()

	entries := make([]ModelEntry, 0, len(models))
	now := time.Now().UTC()

	for _, model := range models {
		fmt.Printf("  probing %-50s ", model)

		modelCtx, cancelModel := context.WithTimeout(ctx, 30*time.Second)
		res, runErr := RunCrush(modelCtx, model, tmpName)
		cancelModel()

		avail := true
		var statusMsg string
		switch {
		case runErr != nil:
			avail = false
			statusMsg = fmt.Sprintf("ERROR: %v", runErr)
		case res.ExitCode == 0:
			statusMsg = "OK"
		case isModelUnavailable(res.Output):
			avail = false
			// Extract first non-empty line of output as the reason.
			reason := firstNonEmptyLine(res.Output)
			statusMsg = fmt.Sprintf("UNAVAILABLE: %s", reason)
		default:
			// Transient failure — still considered available (fail-open).
			reason := firstNonEmptyLine(res.Output)
			statusMsg = fmt.Sprintf("TRANSIENT (exit %d): %s", res.ExitCode, reason)
		}

		if avail {
			fmt.Printf("[ OK ] %s\n", statusMsg)
		} else {
			fmt.Printf("[FAIL] %s\n", statusMsg)
		}

		entries = append(entries, ModelEntry{
			ID:        model,
			Provider:  providerOf(model),
			Available: avail,
			ProbedAt:  now,
		})
	}

	// Print summary.
	fmt.Println()
	availCount := 0
	for _, e := range entries {
		if e.Available {
			availCount++
		}
	}
	fmt.Printf("Summary: %d/%d models available.\n", availCount, len(entries))

	// Persist cache so the MCP server picks it up immediately on next start.
	if cacheErr == nil && cachePath != "" {
		prober := newModelProberWithFuncs(cachePath, defaultProbeTTL(), nil, nil)
		prober.mu.Lock()
		prober.cache = &ModelCache{
			Entries:        entries,
			CachedAt:       now,
			ProbeCompleted: true,
		}
		prober.mu.Unlock()
		prober.saveCache()
		fmt.Printf("Cache written to: %s\n", cachePath)
	} else if cacheErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not determine cache path, results not persisted: %v\n", cacheErr)
	}

	os.Exit(0)
}

// firstNonEmptyLine returns the first non-whitespace-only line of s,
// truncated to 120 characters.
func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			if len(line) > 120 {
				return line[:120] + "…"
			}
			return line
		}
	}
	return "(no output)"
}
