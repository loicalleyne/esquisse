# GLOSSARY.md — esquisse-mcp

Domain vocabulary for the `esquisse-mcp` MCP server. Use these exact terms in
code, comments, and task documents. Never invent alternative names.

---

## A

**adversarial review**  
A structured critique process where an LLM is prompted to attack a plan using
the 7-attack protocol. In `esquisse-mcp`, one *round* of adversarial review =
one `crush run --model {model}` subprocess call with the review prompt.

---

## B

**buildModelPool**  
The function in `models.go` that reads `ESQUISSE_MODELS`, validates each
`provider/model` entry, and returns the effective pool as a `[]string`.
Falls back to `defaultModels` if the env var is unset or all entries are invalid.
Called once per server startup inside the `newAdversarialHandler` closure.

---

## C

**crush**  
The external binary (`github.com/charmbracelet/crush`) that `esquisse-mcp`
invokes as a subprocess to run LLM prompts. Also the LLM client with
`crush models` and `crush run --model` subcommands.

---

## D

---

## F

**fail-open** (default)  
`buildModelPool` falls back to the full default pool if `ESQUISSE_MODELS` is
unset, empty, or all entries are invalid.

**family-interleave shuffle**  
The randomization strategy in `familyInterleaveShuffle` that interleaves models
from different provider families (copilot, gemini, vertexai) while ensuring no
two consecutive rounds use the same family. Re-randomizes every `len(pool)` rounds.

---

## G

**gate_review**  
The MCP tool that scans `.adversarial/*.json` in the project root and returns
`blocked=true` if any plan's `last_verdict` is FAILED or absent.

---

## I

**iteration**  
The zero-based count of completed adversarial review rounds for a given plan.
Stored as `iteration` in `.adversarial/{slug}.json`. Advances by `rounds` after
a successful `adversarial_review` call.

---

## L

**last_verdict**  
The most recent verdict from an adversarial review run. One of: `PASSED`,
`CONDITIONAL`, or `FAILED`. Stored in the state file alongside `iteration`.

---

## M

**mcpErr**  
A helper function in `adversarial.go` that returns an `mcp.CallToolResult`
with `IsError=true` and a formatted message.

**MODELS** (`ESQUISSE_MODELS`)  
A comma-separated list of `provider/model` strings that defines the full model
pool. Replaces the former `ESQUISSE_MODEL_SLOT{0-4}` + `ESQUISSE_ALLOWED_PROVIDERS`
mechanism (removed in P3-006). Invalid entries are skipped with a log warning;
all-invalid falls back to `defaultModels`.

**model pool**  
The ordered list of `provider/model` strings from which rotation order is
derived. Built by `buildModelPool` from `ESQUISSE_MODELS`. Default pool =
`defaultModels` in `models.go`.

---

## P

**plan_slug**  
A URL-safe identifier for a plan document, used as the state file name:
`.adversarial/{slug}.json`. Validated by `validateSlug` to reject traversal
sequences. Example: `P2-006-mcp-configurable-model-rotation`.

**provider**  
The prefix before `/` in a `provider/model` string (e.g. `copilot`, `gemini`,
`vertexai`, `openai`, `anthropic`). Used by `ALLOWED_PROVIDERS` filtering and
family-interleave logic.

---

## R

**rotation order**  
The permutation of pool indices for a given epoch, produced by
`buildRotationOrder`. Used to select which model to call in round N within an
epoch.

**round**  
A single LLM call in the adversarial review loop. One round = one
`runOneRound(ctx, model, planContent, tmpDir)` call.

**rounds** (parameter)  
The number of review rounds requested in a single `adversarial_review` call.
Defaults to `defaultRounds` (5), capped at `maxRounds` (50).

**RunResult**  
A struct in `runner.go` holding the stdout text and error (if any) from a
`crush run` subprocess call.

---

## S

**SetRandSource**  
A test-only exported function in `models.go` that swaps the random source for
`familyInterleaveShuffle`, enabling deterministic test output.

**slug** — see *plan_slug*

**state file**  
The JSON file at `.adversarial/{slug}.json` holding `last_verdict`, `iteration`,
and `plan_slug`. Schema defined in `ReviewState` in `state.go`.

---

## V

**validateSlug**  
A function in `state.go` that rejects plan slugs containing `/`, `\`, `..`, or
NUL bytes. Called at the start of every `ReadState` and `WriteState` call.

**verdict**  
The outcome of an adversarial review round. One of: `PASSED`, `CONDITIONAL`,
`FAILED`. `worstVerdict` reduces a slice of verdicts to the worst-case value.
