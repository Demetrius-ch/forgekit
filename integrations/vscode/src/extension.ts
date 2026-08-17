import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import { spawn } from 'child_process';

interface ForgeMetadata {
  version: string;
  schema: number;
  project: string;
  language: string;
  type: string;
  created_at: string;
}

interface InstalledFeature {
  name: string;
  version: string;
  installed_at: string;
}

interface FeaturesFile {
  features: InstalledFeature[];
}

interface ValidationResult {
  status: 'valid' | 'absent' | 'invalid' | 'legacy';
  metadata?: ForgeMetadata;
  features?: InstalledFeature[];
  errors: string[];
  warnings: string[];
}

let yamlParse: (content: string) => any;

async function loadYaml() {
  if (!yamlParse) {
    const yamlModule = await import('yaml');
    yamlParse = yamlModule.parse;
  }
  return yamlParse;
}

export async function activate(context: vscode.ExtensionContext) {
  console.log('ForgeKit extension activated');

  const disposableInspect = vscode.commands.registerCommand('forgekit.inspectProject', () => {
    inspectProject();
  });

  const disposableDoctor = vscode.commands.registerCommand('forgekit.runDoctor', () => {
    runDoctor();
  });

  context.subscriptions.push(disposableInspect, disposableDoctor);

  if (vscode.workspace.workspaceFolders) {
    for (const folder of vscode.workspace.workspaceFolders) {
      validateForgeProject(folder.uri.fsPath);
    }
  }

  vscode.workspace.onDidChangeWorkspaceFolders((event) => {
    for (const folder of event.added) {
      validateForgeProject(folder.uri.fsPath);
    }
  });
}

async function validateForgeProject(workspacePath: string): Promise<ValidationResult> {
  const forgeDir = path.join(workspacePath, '.forge');
  const forgeYamlPath = path.join(forgeDir, 'forge.yaml');
  const featuresYamlPath = path.join(forgeDir, 'features.yaml');

  if (!fs.existsSync(forgeDir)) {
    return { status: 'absent', errors: ['No .forge directory found'], warnings: [] };
  }

  if (!fs.existsSync(forgeYamlPath)) {
    if (fs.existsSync(featuresYamlPath)) {
      const features = await loadFeatures(featuresYamlPath);
      return {
        status: 'legacy',
        features,
        errors: [],
        warnings: ['Legacy ForgeKit project: .forge/forge.yaml missing, only features.yaml present']
      };
    }
    return { status: 'invalid', errors: ['.forge directory exists but forge.yaml is missing'], warnings: [] };
  }

  try {
    const content = fs.readFileSync(forgeYamlPath, 'utf-8');
    const parse = await loadYaml();
    const metadata = parse(content) as ForgeMetadata;

    if (!metadata.schema || metadata.schema > 1) {
      return { status: 'invalid', errors: [`Unsupported schema version: ${metadata.schema}`], warnings: [] };
    }

    if (!metadata.version) {
      return { status: 'invalid', errors: ['ForgeKit version missing in metadata'], warnings: [] };
    }

    if (!metadata.project) {
      return { status: 'invalid', errors: ['Project name missing in metadata'], warnings: [] };
    }

    const features = await loadFeatures(featuresYamlPath);
    const warnings: string[] = [];

    if (features.length === 0) {
      warnings.push('no features registered in .forge/features.yaml');
    }

    for (const feat of features) {
      const featurePath = path.join(workspacePath, 'internal', feat.name);
      if (!fs.existsSync(featurePath)) {
        warnings.push(`feature "${feat.name}" declared but configuration missing (expected at ${featurePath})`);
      }
    }

    return { status: 'valid', metadata, features, errors: [], warnings };
  } catch (e) {
    return { status: 'invalid', errors: [`Invalid YAML: ${e instanceof Error ? e.message : String(e)}`], warnings: [] };
  }
}

async function loadFeatures(featuresPath: string): Promise<InstalledFeature[]> {
  if (!fs.existsSync(featuresPath)) {
    return [];
  }
  try {
    const content = fs.readFileSync(featuresPath, 'utf-8');
    const parse = await loadYaml();
    const parsed = parse(content) as FeaturesFile;
    return parsed.features || [];
  } catch {
    return [];
  }
}

async function inspectProject() {
  const folders = vscode.workspace.workspaceFolders;
  if (!folders || folders.length === 0) {
    vscode.window.showInformationMessage('No workspace folder open');
    return;
  }

  for (const folder of folders) {
    const result = await validateForgeProject(folder.uri.fsPath);
    showInspectResult(folder.name, result);
  }
}

function showInspectResult(projectName: string, result: ValidationResult) {
  const panel = vscode.window.createWebviewPanel(
    'forgekitInspect',
    `ForgeKit: ${projectName}`,
    vscode.ViewColumn.One,
    { enableScripts: true }
  );

  let html = '<html><body style="font-family: var(--vscode-font-family); padding: 20px; line-height: 1.6;">';
  html += '<h1 style="color: var(--vscode-terminal-ansiGreen);">ForgeKit</h1>';
  html += '<hr style="border-color: var(--vscode-panel-border);">';

  if (result.status === 'absent') {
    html += '<p><strong>Signature:</strong> <span style="color: var(--vscode-terminal-ansiRed);">ABSENTE</span></p>';
    html += '<p>Ce n\'est pas un projet ForgeKit (répertoire .forge manquant).</p>';
  } else if (result.status === 'invalid') {
    html += '<p><strong>Signature:</strong> <span style="color: var(--vscode-terminal-ansiRed);">INVALIDE</span></p>';
    html += '<ul>';
    for (const err of result.errors) {
      html += `<li style="color: var(--vscode-terminal-ansiRed);">${err}</li>`;
    }
    html += '</ul>';
    html += '<p>Exécutez <code>ForgeKit: Run Doctor</code> pour plus de détails.</p>';
  } else {
    if (result.metadata) {
      html += `<p><strong>Project:</strong> ${result.metadata.project}</p>`;
      html += `<p><strong>ForgeKit:</strong> v${result.metadata.version}</p>`;
      html += `<p><strong>Schema:</strong> ${result.metadata.schema}</p>`;
      html += `<p><strong>Language:</strong> ${result.metadata.language}</p>`;
      html += `<p><strong>Type:</strong> ${result.metadata.type}</p>`;
      if (result.metadata.created_at) {
        html += `<p><strong>Created:</strong> ${new Date(result.metadata.created_at).toLocaleString()}</p>`;
      }
    }

    if (result.status === 'legacy') {
      html += '<p style="color: var(--vscode-terminal-ansiYellow);">⚠ Projet legacy détecté (.forge/forge.yaml manquant, seules les features sont présentes)</p>';
    }

    html += '<h3>Features:</h3>';
    if (result.features && result.features.length > 0) {
      html += '<ul>';
      for (const feat of result.features) {
        html += `<li>✓ ${feat.name}  v${feat.version}</li>`;
      }
      html += '</ul>';
    } else {
      html += '<p>(aucune)</p>';
    }

    html += '<h3>Signature:</h3>';
    html += '<p style="color: var(--vscode-terminal-ansiGreen);">✓ Valide</p>';

    if (result.warnings && result.warnings.length > 0) {
      html += '<h3>Avertissements:</h3><ul>';
      for (const warn of result.warnings) {
        html += `<li style="color: var(--vscode-terminal-ansiYellow);">⚠ ${warn}</li>`;
      }
      html += '</ul>';
    }

    html += '<p style="margin-top: 20px; color: var(--vscode-descriptionForeground);">Créé avec ForgeKit</p>';
  }

  html += '</body></html>';
  panel.webview.html = html;
}

async function runDoctor() {
  const folders = vscode.workspace.workspaceFolders;
  if (!folders || folders.length === 0) {
    vscode.window.showInformationMessage('No workspace folder open');
    return;
  }

  const config = vscode.workspace.getConfiguration('forgekit');
  const cliPath = config.get<string>('cliPath') || 'forge';

  for (const folder of folders) {
    const terminal = vscode.window.createTerminal({
      name: `ForgeKit Doctor - ${folder.name}`,
      cwd: folder.uri.fsPath
    });
    terminal.show();
    terminal.sendText(`${cliPath} doctor`);
  }
}

export function deactivate() {
  console.log('ForgeKit extension deactivated');
}