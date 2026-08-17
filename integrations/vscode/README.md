# ForgeKit for VS Code

Official Visual Studio Code extension for [ForgeKit](https://github.com/Demetrius-ch/forgekit) - a Go CLI tool that generates production-ready REST APIs with hexagonal architecture.

## Features

-  **Visual Identity**: Custom icon for `.forge/` folders in the file explorer
-  **Project Detection**: Automatically detects ForgeKit projects in your workspace
-  **Project Inspection**: `ForgeKit: Inspect Project` command to view project metadata and features
-  **Doctor Integration**: `ForgeKit: Run Doctor` command to run `forge doctor` in the terminal
-  **Legacy Support**: Recognizes older ForgeKit projects with only `features.yaml`

## Installation

### From VSIX (Manual)

```bash
# Build the extension
cd integrations/vscode
npm install
npm run compile
vsce package

# Install in VS Code
code --install-extension forgekit-vscode-0.2.0.vsix
```

### From Source (Development)

```bash
cd integrations/vscode
npm install
npm run compile
# Press F5 in VS Code to launch Extension Development Host
```

## Commands

| Command | Description |
|---------|-------------|
| `ForgeKit: Inspect Project` | Display project metadata, features, and signature status |
| `ForgeKit: Run Doctor` | Execute `forge doctor` in a terminal |

## Configuration

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `forgekit.cliPath` | string | `forge` | Path to the ForgeKit CLI executable |
| `forgekit.autoDetect` | boolean | `true` | Automatically detect ForgeKit projects in workspace |

## Project Detection

The extension validates ForgeKit projects by checking:

1. **`.forge/`** directory exists
2. **`.forge/forge.yaml`** exists and contains valid YAML with:
   - `version`: ForgeKit version (e.g., `0.2.0`)
   - `schema`: Schema version (currently `1`)
   - `project`: Project name
   - `language`: Language (e.g., `go`)
   - `type`: Project type (e.g., `backend-api`)
3. **`.forge/features.yaml`** (optional) contains installed features

### Project States

| State | Criteria | Indicator |
|-------|----------|-----------|
| **Valid** | `.forge/forge.yaml` exists, valid YAML, supported schema | ✓ ForgeKit Project |
| **Legacy** | Only `.forge/features.yaml` exists (no `forge.yaml`) | ⚠ Legacy ForgeKit Project |
| **Invalid** | `.forge/` exists but `forge.yaml` is missing/invalid | ⚠ Invalid ForgeKit Configuration |
| **Absent** | No `.forge/` directory | Not a ForgeKit Project |

## Folder Icon (`.forge`)

The `.forge/` folder icon is provided by **Material Icon Theme** (PR submitted upstream).  
If you use Material Icon Theme, the ForgeKit icon appears automatically — no configuration needed.

The ForgeKit extension **does not** contribute its own File Icon Theme and will never prompt you to select one. It focuses on project detection, inspection, and doctor integration.

If you prefer a different icon theme, the extension works normally — only the `.forge/` folder icon falls back to VS Code's default folder icon.

## Requirements

- VS Code 1.85+
- ForgeKit CLI installed and in PATH (or configured via `forgekit.cliPath`)

## Architecture

```
ForgeKit CLI          .forge/              ForgeKit VS Code
─────────────────     ─────────────        ─────────────────
creates .forge    →   forge.yaml        →  detects, displays
validates .forge    features.yaml        icon, inspects
manages features
runs doctor/inspect                      calls CLI for complex ops
```

The extension **never** reimplements ForgeKit business logic. It:
- Reads `.forge/` for identity and display
- Calls `forge` CLI for operations like `doctor`

## Development

### Structure

```
integrations/vscode/
├── package.json          # Extension manifest
├── tsconfig.json         # TypeScript config
├── src/
│   └── extension.ts      # Main entry point
├── icons/
│   └── forgekit-icon.svg # Activity bar / README logo
├── media/
│   └── forgekit-activity.svg  # Activity bar icon
├── README.md
└── LICENSE
```

### Building

```bash
cd integrations/vscode
npm install
npm run compile       # Compile TypeScript
npm run lint          # Lint
vsce package          # Create .vsix
```

### Testing

```bash
npm run compile
npm test
```

## License

MIT License - see [LICENSE](LICENSE) file.