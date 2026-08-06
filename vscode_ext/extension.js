const vscode = require('vscode');
const os = require('os');
const path = require('path');
const { mkdir, writeFile } = require('fs/promises');

async function exportAvailableModels() {
  const allModels = await vscode.lm.selectChatModels();

  const payload = {
    generated_at: new Date().toISOString(),
    source: 'vscode.lm.selectChatModels',
    model_count: allModels.length,
    models: allModels.map((m) => ({
      id: m.id,
      name: m.name,
      vendor: m.vendor,
      family: m.family,
      version: m.version,
      maxInputTokens: m.maxInputTokens
    }))
  };

  const snapshotDir = path.join(os.homedir(), '.esquisse');
  const savePath = path.join(snapshotDir, 'models.available.json');

  await mkdir(snapshotDir, { recursive: true });

  const json = JSON.stringify(payload, null, 2) + '\n';
  await writeFile(savePath, json, 'utf8');

  vscode.window.showInformationMessage(
    `Exported ${payload.model_count} models to ${savePath}`
  );
}

function activate(context) {
  const disposable = vscode.commands.registerCommand(
    'esquisse.exportAvailableModels',
    async () => {
      try {
        await exportAvailableModels();
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage(`Failed to export models: ${message}`);
      }
    }
  );

  context.subscriptions.push(disposable);
}

function deactivate() {}

module.exports = {
  activate,
  deactivate
};
