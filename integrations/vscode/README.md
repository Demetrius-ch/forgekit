# ForgeKit for VS Code

Official Visual Studio Code extension for [ForgeKit](https://github.com/Demetrius-ch/forgekit) - a Go CLI tool that generates production-ready REST APIs with hexagonal architecture.

## Features

- **Project Detection**: Automatically detects ForgeKit projects in your workspace
- **Status Bar Indicator**: Shows ForgeKit project status at a glance (valid ✓, legacy ⚠, invalid ✗)
- **Project Inspection**: `ForgeKit: Inspect Project` command to view project metadata and features
- **Doctor Integration**: `ForgeKit: Run Doctor` command to run `forge doctor` in the terminal
- **Legacy Support**: Recognizes older ForgeKit projects with only `features.yaml`
- **Non-Intrusive**: Never modifies your icon theme or workspace settings

## Installation

### From VSIX (Manual)

```bash
# Build the extension
cd integrations/vscode
npm install
npm run compile
vsce package

# Install in VS Code
code --install-extension forgekit-vscode-0.2.5.vsix
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

## Status Bar Indicator

The extension adds a ForgeKit status item in the VS Code status bar (right side):

| Status | Indicator | Meaning |
|--------|-----------|---------|
| **Valid** | `$(check) ForgeKit ✓` | `.forge/forge.yaml` exists and is valid |
| **Legacy** | `$(warning) ForgeKit ⚠` | Only `.forge/features.yaml` exists (legacy project) |
| **Invalid** | `$(error) ForgeKit ✗` | `.forge/` exists but `forge.yaml` is missing/invalid |
| **Absent** | *(hidden)* | No `.forge/` directory in workspace |

Click the status bar item to run **ForgeKit: Inspect Project**.

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

| State | Criteria | Status Bar |
|-------|----------|------------|
| **Valid** | `.forge/forge.yaml` exists, valid YAML, supported schema | `ForgeKit ✓` |
| **Legacy** | Only `.forge/features.yaml` exists (no `forge.yaml`) | `ForgeKit ⚠` |
| **Invalid** | `.forge/` exists but `forge.yaml` is missing/invalid | `ForgeKit ✗` |
| **Absent** | No `.forge/` directory | *(hidden)* |

## Why No File Icon Theme?

The ForgeKit extension **does not** contribute a File Icon Theme and will **never** automatically change your `workbench.iconTheme` setting. This is by design:

- **VS Code API limitation**: Extensions cannot inject a single folder icon into another extension's active icon theme
- **Respect user preferences**: Your chosen icon theme (Material Icon Theme, vscode-icons, Seti, etc.) remains intact
- **No conflicts**: The extension works alongside any icon theme without interference

The `.forge/` folder icon in the Explorer is determined entirely by your active icon theme:
- **Material Icon Theme**: Shows ForgeKit icon (added via upstream PR)
- **Other themes**: Shows the theme's default folder icon
- **No theme**: Shows VS Code's built-in folder icon

ForgeKit's visual identity is provided through UI elements the extension fully controls:
- **Activity Bar** — ForgeKit logo and project view
- **Status Bar** — Real-time project status indicator
- **Commands** — `Inspect Project` and `Run Doctor`
- **WebView** — Rich project inspection panel

## Requirements

- VS Code 1.85+
- ForgeKit CLI installed and in PATH (or configured via `forgekit.cliPath`)

## Architecture

```
ForgeKit CLI          .forge/              ForgeKit VS Code
─────────────────     ─────────────        ─────────────────
creates .forge    →   forge.yaml        →  detects, displays
validates .forge    features.yaml        status, inspects
manages features                                  calls CLI for complex ops
runs doctor/inspect
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