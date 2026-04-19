# P1-004 — esquisse-mcp: Adversarial Review MCP Server

**Phase:** P1  
**Status:** Done  
**Created:** 2026-04-17  
**Requires adversarial review:** minimum 3 rounds before implementation

---

## Goal

Create `esquisse-mcp/` — a lightweight MCP stdio server written in Go that
exposes `adversarial_review` and `gate_review` tools. Any MCP-capable client
(Crush, VS Code with MCP configured, CI scripts) can invoke multi-model
adversarial review with a single tool call. The server shells out to
`crush run --model {provider/model}` for cross-model dispatch, making the
adversarial review workflow platform-transparent.

---

## Background

P1-003 provides a Crush-compatible adversarial review flow using `bash: crush
run --model ...`. That approach requires the agent to manually:
1. Read the rotation state file
2. Compute the slot
3. Construct the prompt heredoc
4. Parse the verdict from stdout

The MCP server abstracts all of this into a single tool call. It provides:
- **Platform transparency**: the same `adversarial_review` tool call works
  identically in Crush and VS Code (with MCP configured)
- **Shell injection safety**: plan content is passed via stdin, not shell args
- **State management**: the server reads and writes `.adversarial/{slug}.json`
- **Retry resilience**: the server can retry a failed `crush run` invocation

The P1-003 bash approach remains the fallback; P1-004 is the production solution.

---

## In Scope

### 1 — Directory structure

```
esquisse-mcp/
├── main.go             MCP server entry point (stdio transport)
├── tools.go            Tool registration and handler dispatch
├── adversarial.go      adversarial_review tool implementation
├── gate.go             gate_review tool implementation
├── state.go            .adversarial/ state file read/write
├── runner.go           crush run subprocess management
├── go.mod              module: github.com/loicalleyne/esquisse-mcp
├── go.sum
└── README.md
```

### 2 — MCP server: `adversarial_review` tool

**Schema:**
```json
{
  "name": "adversarial_review",
  "description": "Dispatch an adversarial reviewer for the given plan using cross-model rotation. Reads .adversarial/{plan_slug}.json for current iteration, picks the model for slot = iteration % 3, invokes crush run --model, writes the verdict back to the state file.",
  "inputSchema": {
    "type": "object",
    "required": ["plan_slug", "plan_content"],
    "properties": {
      "plan_slug": {
        "type": "string",
        "description": "Slug for the plan being reviewed. Used as the state file name: .adversarial/{plan_slug}.json"
      },
      "plan_content": {
        "type": "string",
        "description": "Full text of the plan to be reviewed. Passed to the reviewer via stdin."
      }
    }
  }
}
```

**Implementation in `adversarial.go`:**
1. Read `{project_root}/.adversarial/{plan_slug}.json` (or create with iteration=0 if absent).
2. Compute `slot = iteration % 3`.
3. Look up model string from environment: `ESQUISSE_MODEL_SLOT{slot}`.
   Default values if env var absent:
   - Slot 0: `openai/gpt-4.1`
   - Slot 1: `anthropic/claude-opus-4-5-20251101`
   - Slot 2: `openai/gpt-4o`
4. Write the full review prompt to a temp file under `os.TempDir()`
   using `os.CreateTemp` (mode 0600). This is the ONLY safe way to pass
   plan content — it must NEVER appear as a shell argument or flag value.
   Prompt template (read from `adversarial-review/references/task-review-protocol.md`
   in the skill directory, or embed it):
   ```
   You are Adversarial-r{slot}. Apply the 7-attack protocol to the plan below.
   Write your report to {project_root}/.adversarial/reports/review-{date}-iter{n}-{slug}.md
   Write state to {project_root}/.adversarial/{slug}.json with exact schema fields.
   Verdict must be PASSED, CONDITIONAL, or FAILED on the final line.

   --- PLAN CONTENT ---
   {plan_content}
   ```
   If `os.CreateTemp` fails (disk full, permission denied): return MCP error
   `{error: "failed to create temp file for review prompt: {err}"}` immediately.
   Do not proceed.
5. Open the temp file for reading and assign it to `cmd.Stdin`. Do NOT use
   shell redirection syntax (`< {temp_file}`) — Go's `exec` API does not
   interpret shell metacharacters. Invoke `crush run` via `exec.CommandContext`:
   ```go
   f, err := os.Open(tempFile)
   if err != nil { return mcpError("cannot open temp file: %v", err) }
   defer f.Close()
   cmd := exec.CommandContext(ctx, crushPath, "run", "--model", model, "--quiet")
   cmd.Stdin = f
   out, err := cmd.CombinedOutput()
   ```
   - Capture stdout and stderr combined (`CombinedOutput`).
   - Timeout: 300 seconds (adversarial review is a long operation).
   - Delete temp file in a separate `defer` (unconditional; runs even on error).
6. If `crush run` exits non-zero: return MCP error `{error: "crush run exited {code}: {stderr}"}` and do not write state file.
7. Read `.adversarial/{plan_slug}.json` to confirm verdict was written.
   If absent or malformed, extract verdict from stdout using the pattern
   `^Verdict: (PASSED|CONDITIONAL|FAILED)` as a fallback, and rewrite the state file
   with the extracted verdict using `state.Write` before returning.
8. Return the full review text as the tool result.

**Security:** temp file is created with `os.CreateTemp` (mode 0600). The temp
file path is never passed as a shell argument. `cmd.Stdin` is used for stdin
injection-safe plan content delivery. `exec.CommandContext` is used (not
`exec.Command`) for timeout enforcement.

### 3 — MCP server: `gate_review` tool

**Schema:**
```json
{
  "name": "gate_review",
  "description": "Check whether all adversarial review verdicts in .adversarial/ are PASSED or CONDITIONAL. Returns a structured result indicating whether the session may proceed.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "strict": {
        "type": "boolean",
        "default": false,
        "description": "If true, block if no state files exist (planning gate). If false, pass if no state files exist (global hook mode)."
      }
    }
  }
}
```

**Implementation in `gate.go`:**
1. Glob `.adversarial/*.json` (exclude `reports/` subdirectory).
2. If no files and `strict=false`: return `{blocked: false, reason: "no reviews in progress"}`.
3. If no files and `strict=true`: return `{blocked: true, reason: "adversarial review required before completing this session"}`.
4. For each file: parse `last_verdict`. Collect FAILED or empty verdicts.
5. Return: `{blocked: bool, reason: string, blocking_plans: []string}`.

### 4 — State file read/write (`state.go`)

Implements the canonical schema from `SCHEMAS.md §8`:
```go
type ReviewState struct {
    PlanSlug       string `json:"plan_slug"`
    Iteration      int    `json:"iteration"`
    LastModel      string `json:"last_model"`
    LastVerdict    string `json:"last_verdict"`
    LastReviewDate string `json:"last_review_date"`
}
```

- `Read(projectRoot, planSlug string) (*ReviewState, error)` — returns zero-value struct (iteration=0) if file absent (not an error).
- `Write(projectRoot string, state ReviewState) error` — atomic write via temp file + rename. Creates `.adversarial/` if absent.

### 5 — Subprocess runner (`runner.go`)

```go
type RunResult struct {
    Stdout   string
    ExitCode int
}

func RunCrush(ctx context.Context, model, promptFile string) (RunResult, error)
```

- Locates `crush` binary via `exec.LookPath`.
- Returns a descriptive error if `crush` is not found (not a fatal panic).
- Uses `exec.CommandContext` with the provided context for timeout.
- Reads prompt from `promptFile` via stdin redirection.

### 6 — MCP server entry point (`main.go`)

- **Platform guard** (first thing in `main()`): `if runtime.GOOS == "windows" { fmt.Fprintln(os.Stderr, "esquisse-mcp does not support Windows — run from Linux, macOS, or WSL"); os.Exit(1) }`
- Reads `project_root` from `--project-root` flag (defaults to `$PWD`).
- Uses `github.com/modelcontextprotocol/go-sdk` v1 (or the latest stable) for
  stdio MCP transport.
- Registers `adversarial_review` and `gate_review` tools.
- Does not write any files except `.adversarial/` under `project_root`.

### 7 — `go.mod`

Module path: `github.com/loicalleyne/esquisse-mcp`

Key dependencies:
- `github.com/modelcontextprotocol/go-sdk` — MCP server SDK
- stdlib only for everything else (no heavy dependencies)

### 8 — Crush configuration fragment in README

Document the `crush.json` fragment that registers the server:

```json
{
  "mcp": {
    "esquisse": {
      "type": "stdio",
      "command": "esquisse-mcp",
      "args": ["--project-root", "."],
      "env": {
        "ESQUISSE_MODEL_SLOT0": "openai/gpt-4.1",
        "ESQUISSE_MODEL_SLOT1": "anthropic/claude-opus-4-5-20251101",
        "ESQUISSE_MODEL_SLOT2": "openai/gpt-4o"
      }
    }
  }
}
```

### 9 — Update `skills/adversarial-review/SKILL.md` to add Step 4c

Once the MCP server exists, add Step 4c to the skill. This is safe to add
only in P1-004 because the tool does not exist before the server is built
and registered — adding it earlier wastes a tool-lookup on every review.

Add immediately after Step 4b in `skills/adversarial-review/SKILL.md`:

```markdown
### Step 4c: MCP server shortcut (preferred when available)

If the `adversarial_review` MCP tool is registered (via the `esquisse-mcp`
server in your `crush.json`), use it instead of the manual bash approach:

```
adversarial_review(
  plan_slug: "{slug}",
  plan_content: "{full plan text}"
)
```

The MCP tool handles model selection, rotation state, subprocess management,
and verdict writing. Proceed to Step 6 after it returns.
If the tool is not available, continue with Step 4b (bash approach).
```

### 10 — `scripts/init.sh` does NOT auto-add MCP config

Adding MCP configuration to `crush.json` is opt-in. `init.sh` prints an
informational message at the end of the Crush section:

```
NOTE: To enable multi-model adversarial review in Crush, build and install
      esquisse-mcp and add the MCP entry to your crush.json.
      See esquisse-mcp/README.md.
```

---

## Out of Scope

- VS Code MCP configuration for esquisse-mcp (VS Code users use named agents)
- A web/HTTP transport (stdio only)
- Authentication or access control (local tool, trusted environment)
- Embedding the full adversarial review prompt — it references
  `skills/adversarial-review/references/task-review-protocol.md`, which is
  installed by `scripts/init.sh`
- Running the MCP server as a background daemon (managed by Crush as stdio)
- Windows support (Crush is Linux/WSL; esquisse-mcp follows the same constraint)
- Supporting MCP clients other than Crush and VS Code
- An `update_reviews` tool (out of scope for P1)

---

## Files

| Path | Action | What Changes |
|------|--------|-------------|
| `esquisse-mcp/main.go` | Create | MCP server entry point, flag parsing, tool registration |
| `esquisse-mcp/tools.go` | Create | Tool schema definitions and dispatch |
| `esquisse-mcp/adversarial.go` | Create | `adversarial_review` tool implementation |
| `esquisse-mcp/gate.go` | Create | `gate_review` tool implementation |
| `esquisse-mcp/state.go` | Create | State file read/write with atomic rename |
| `esquisse-mcp/runner.go` | Create | `crush run` subprocess management |
| `esquisse-mcp/go.mod` | Create | Module declaration |
| `esquisse-mcp/README.md` | Create | Usage, crush.json fragment, build instructions |
| `skills/adversarial-review/SKILL.md` | Modify | Add Step 4c (MCP shortcut, after Step 4b added by P1-003) |
| `scripts/init.sh` | Modify | Add informational note about esquisse-mcp (print only) |

---

## Acceptance Criteria

1. `go build ./...` from `esquisse-mcp/` succeeds with CGO_ENABLED=0.
2. `adversarial_review` tool: given `plan_slug="test-plan"` and any non-empty
   `plan_content`, the tool invokes `crush run --model {model}`, the child
   process writes `.adversarial/test-plan.json`, and the tool returns the
   review text.
3. `gate_review` tool with `strict=false` and no `.adversarial/*.json` files
   returns `{blocked: false}`.
4. `gate_review` tool with a state file containing `last_verdict: "FAILED"`
   returns `{blocked: true, blocking_plans: ["that-slug"]}`.
5. State file write is atomic: a failed write does not corrupt an existing
   state file.
6. The `crush` binary not being in PATH returns a clear error message
   (not a panic).
7. Plan content containing shell metacharacters (`'`, `"`, `$`, backtick,
   newlines) does not cause `crush run` to fail or misbehave (stdin approach).
8. If temp file creation fails, the tool returns an MCP error and the original
   state file is unmodified.
9. Running `esquisse-mcp` on Windows prints
   `esquisse-mcp does not support Windows — run from Linux, macOS, or WSL`
   and exits 1.
10. Calling `adversarial_review` with an empty `plan_content` returns an MCP
    error `{error: "plan_content must not be empty"}` without invoking `crush run`.

---

## Session Notes

- This module depends on `crush` being installed and configured with the
  relevant LLM providers. It is a developer tool, not a production service.
- The MCP server is a thin shell around `crush run`. It does not implement
  any LLM logic itself — the adversarial review intelligence lives in the
  skill prompt and the child Crush process.
- Each `adversarial_review` call creates a new Crush session. Sessions
  accumulate in the Crush database (`~/.local/share/crush/` or
  `~/.config/crush/`). This is acceptable for developer tooling.
- `SCHEMAS.md §8` is the canonical state file schema. `state.go` must
  implement exactly that schema. Any field name discrepancy breaks
  `gate-review.sh`.
