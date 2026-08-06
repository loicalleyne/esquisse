# Esquisse VS Code Model Sync

This folder contains:

- A VS Code extension command to export currently available chat models.
- A generator script that updates Esquisse model rotation lists in agent files and examples.

## 1) Export available models from VS Code

1. Open this folder as an extension project in VS Code.
2. Press F5 to run an Extension Development Host.
3. Run the command:
   - `Esquisse: Export Available Chat Models`
4. The command writes the snapshot to `~/.esquisse/models.available.json`.

The command uses `vscode.lm.selectChatModels()` and writes JSON like:

```json
{
  "generated_at": "...",
  "source": "vscode.lm.selectChatModels",
  "model_count": 5,
  "models": [
    {
      "id": "gpt-4o",
      "name": "GPT-4o",
      "vendor": "copilot",
      "family": "gpt-4o",
      "version": "...",
      "maxInputTokens": 64000
    }
  ]
}
```

## 2) Update Esquisse rotation files

From the root of an Esquisse-enabled repository:

```bash
/path/to/esquisse/vscode_ext/generate-rotation.sh
```

Dry run:

```bash
/path/to/esquisse/vscode_ext/generate-rotation.sh --dry-run
```

The generator reads `~/.esquisse/models.available.json` and updates the repository rooted at `.` by default, so it is meant to be run from the repository that already has Esquisse files.

## What the generator updates

- `.github/agents/EsquissePlan.agent.md`
  - frontmatter `model:` list
  - slot mapping wording in Step 4
- `.github/agents/Adversarial-r0.agent.md`
  - frontmatter `model:` list
- `.github/agents/Adversarial-r1.agent.md`
  - frontmatter `model:` list
- `.github/agents/Adversarial-r2.agent.md`
  - frontmatter `model:` list
- `skills/adversarial-review/crush-models.md`
  - slot table model strings
  - mirror sentence under the table

## Notes

- The selector is availability-based, so results vary by account, policy, and provider setup.
- The generator requires at least 3 distinct models.
- Model selection is score-based and deterministic from the exported snapshot.
