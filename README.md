# ForgeKit

<p align="center">
  <img src="assets/logo/forgekit-logo.png" alt="ForgeKit" width="520">
</p>

<h1 align="center">ForgeKit</h1>

<p align="center">
  <strong>Go Backend Generator</strong>
</p>

<p align="center">
  Build • Extend • Ship
</p>

<p align="center">
  Generate production-ready Go REST APIs with a clean hexagonal architecture.
</p>

ForgeKit is a developer CLI written in Go that helps you bootstrap backend REST APIs without manually creating the same project structure, configuration, database integration, Docker files, migrations, and tests every time.

The goal is simple:

```text
forge init my-api
```

and get a structured Go backend ready for development.

---

## ✨ Features

- Generate Go REST APIs
- Hexagonal architecture
- PostgreSQL integration
- Docker and Docker Compose
- Database migrations
- Environment configuration
- Automatic tests
- Automatic Go formatting
- Architecture validation
- Development environment diagnostics
- Project analysis with scoring
- Human-readable or JSON output
- Extensible CLI architecture
- **Feature system (`forge add` / `forge remove`) for extending generated projects**
- **JWT authentication infrastructure (`forge add auth`)**
- **CORS middleware (`forge add cors`)**
- **Structured logging (`forge add logging`)**
- **Swagger/OpenAPI documentation (`forge add swagger`)**
- **CI/CD integration with `--ci` mode**
- **Project diagnostics with `forge doctor`**
- **Feature version tracking and rollback**

---

## Installation

### Linux (Debian/Ubuntu) — Recommended (APT)

Install ForgeKit from the official APT repository:

```bash
# 1. Add the ForgeKit GPG key
curl -fsSL https://demetrius-ch.github.io/forgekit/gpg/forgekit-archive-keyring.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/forgekit.gpg

# 2. Add the APT repository
echo "deb [signed-by=/usr/share/keyrings/forgekit.gpg] https://demetrius-ch.github.io/forgekit stable main" \
  | sudo tee /etc/apt/sources.list.d/forgekit.list

# 3. Update and install
sudo apt update
sudo apt install forge

# 4. Verify installation
forge version
```

Uninstall:

```bash
sudo apt remove forge
# Optionally remove the repository and key
sudo rm /etc/apt/sources.list.d/forgekit.list /usr/share/keyrings/forgekit.gpg
sudo apt update
```

### Linux (Debian/Ubuntu) — Manual `.deb` download

Download the latest `.deb` package from [GitHub Releases](https://github.com/Demetrius-ch/forgekit/releases):

```bash
# Replace VERSION with the desired version (e.g., 0.3.0)
VERSION=0.3.0
wget https://github.com/Demetrius-ch/forgekit/releases/download/v${VERSION}/forge_${VERSION}_linux_amd64.deb
sudo dpkg -i forge_${VERSION}_linux_amd64.deb
forge version
```

Uninstall:

```bash
sudo apt remove forge
```

### From source

Requirements:

- Go 1.22+
- Git
- Docker (required for the generated project's Docker workflow)

Clone the repository:

```bash
git clone git@github.com:Demetrius-ch/forgekit.git
cd forgekit
```

Build ForgeKit:

```bash
go build -o forge ./cmd/forge
```

Install it globally:

```bash
go install ./cmd/forge
```

Verify the installation:

```bash
forge version
```

### Binary (Linux)

Download the static binary from [GitHub Releases](https://github.com/Demetrius-ch/forgekit/releases):

```bash
# Replace VERSION with the desired version (e.g., 0.3.0)
VERSION=0.3.0
wget https://github.com/Demetrius-ch/forgekit/releases/download/v${VERSION}/forge_${VERSION}_linux_amd64.tar.gz
tar -xzf forge_${VERSION}_linux_amd64.tar.gz
sudo mv forge /usr/local/bin/
forge version
```

Verify checksums (SHA256):

```bash
wget https://github.com/Demetrius-ch/forgekit/releases/download/v${VERSION}/checksums.txt
sha256sum -c checksums.txt
```

---

## Usage

### Initialize a project

```bash
forge init my-api
```

ForgeKit interactively asks for the project configuration and generates the backend.

You can also use non-interactive mode:

```bash
forge init my-api \
  --module github.com/example/my-api \
  --port 8080 \
  --db-name my_api \
  --non-interactive
```

Specify a target directory:

```bash
forge init my-api --dir /tmp/my-api
```

Preview the files without creating them:

```bash
forge init my-api --dry-run
```

---

## Generated project

A generated project follows a structure similar to:

```text
my-api/
├── cmd/
│   └── server/
│       └── main.go
│
├── internal/
│   ├── application/
│   │   ├── health/
│   │   └── user/
│   │
│   ├── domain/
│   │   ├── health.go
│   │   └── user.go
│   │
│   ├── infrastructure/
│   │   ├── config/
│   │   └── postgres/
│   │
│   └── transport/
│       └── http/
│           ├── handler/
│           └── router.go
│
├── migrations/
│   ├── 000001_init.up.sql
│   └── 000001_init.down.sql
│
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
│
├── tests/
├── .env.example
├── .gitignore
├── forge.yaml
├── Makefile
├── README.md
├── go.mod
└── go.sum
```

After installing features (e.g., `forge add auth`), the structure extends with:

```text
my-api/
├── internal/
│   └── auth/
│       ├── jwt.go
│       └── middleware.go
├── .forge/
│   └── features.yaml
└── .env.example  (extended with JWT_SECRET)
```

---

## Architecture

ForgeKit generates a backend organized around hexagonal architecture.

```text
                     ┌──────────────────────┐
                     │      HTTP API        │
                     │    Transport Layer   │
                     └──────────┬───────────┘
                                │
                                ▼
                     ┌──────────────────────┐
                     │    Application       │
                     │      Services        │
                     └──────────┬───────────┘
                                │
                                ▼
                     ┌──────────────────────┐
                     │       Domain         │
                     │   Business Logic     │
                     └──────────┬───────────┘
                                │
                                ▼
                     ┌──────────────────────┐
                     │   Infrastructure     │
                     │ PostgreSQL / Config   │
                     └──────────────────────┘
```

The objective is to keep business logic independent from infrastructure and transport concerns.

---

## Run the generated API with Docker

Enter the generated project:

```bash
cd my-api
```

Copy the environment configuration:

```bash
cp .env.example .env
```

Start the complete stack:

```bash
docker compose -f docker/docker-compose.yml up --build
```

The stack includes:

- PostgreSQL
- Database migrations
- Go API

The API is available by default on:

```text
http://localhost:8080
```

Test the health endpoint:

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{
  "service": "my-api",
  "status": "ok"
}
```

---

## Extend a generated project with `forge add`

ForgeKit introduces an extensible feature system to add capabilities to existing projects.

### List available features

```bash
forge add --list
```

Output:

```text
ForgeKit Features
────────────────────────────────
auth       1.0.0 — Infrastructure d'authentification JWT (middleware, validation de token)
cors       1.0.2 — Middleware CORS pour les requêtes cross-origin
logging    1.0.0 — Logging structuré et middleware HTTP
swagger    1.0.1 — Documentation OpenAPI/Swagger
```

Each feature shows its name, version, description, and dependencies (if any).

### Preview a feature installation (plan mode)

```bash
forge add auth --plan
```

Output:

```text
ForgeKit Plan
────────────────────────────────

Feature: auth
Version: 1.0.0

Changes:
  + internal/auth/jwt.go
  + internal/auth/middleware.go

Dependencies:
  → github.com/golang-jwt/jwt/v5 v5.2.0

Environment:
  → JWT_SECRET=your-secret-key-change-in-production

No files were modified.
```

The `--plan` (or `--dry-run`) flag shows what would be created/modified/deleted without making any changes.

### Install a feature

```bash
forge add auth
```

Output:

```text
ForgeKit Add
────────────────────────────────

✓ Projet ForgeKit détecté
✓ Feature "auth" trouvée
✓ Prérequis validés
✓ Plan validé

Installation...
✓ Fichiers installés
✓ Dépendances installées
✓ Projet validé

────────────────────────────────
✓ Feature "auth" installée avec succès
```

The `auth` feature adds:

- `internal/auth/jwt.go` — JWT token generation and validation
- `internal/auth/middleware.go` — HTTP middleware for authentication
- Dependency: `github.com/golang-jwt/jwt/v5`
- Environment variable: `JWT_SECRET` in `.env.example`
- Tracks installation in `.forge/features.yaml`

### Idempotency

Running `forge add auth` twice is safe:

```bash
$ forge add auth
✓ Feature "auth" installée avec succès

$ forge add auth
⚠ Feature "auth" déjà installée
```

### JSON output

All commands support `--format json` for machine-readable output:

```bash
forge add --list --format json
forge add auth --dry-run --format json
```

### Quiet mode

Suppress non-essential output:

```bash
forge add auth --quiet
```

### Remove a feature

```bash
forge remove auth
```

Before removal, ForgeKit checks:
1. Feature is installed
2. No other installed features depend on it
3. Detects user modifications to feature files

Output:

```text
ForgeKit Remove
────────────────────────────────

✓ Projet ForgeKit détecté
✓ Feature "auth" trouvée
✓ Aucune feature dépendante

Suppression...
  ✓ Fichiers supprimés
  ✓ Dépendances retirées
  ✓ Projet validé

────────────────────────────────
✓ Feature "auth" supprimée avec succès
```

Preview removal with `--plan`:

```bash
forge remove auth --plan
```

### JSON output

All commands support `--format json` for machine-readable output:

```bash
forge add --list --format json
forge add auth --plan --format json
forge remove auth --format json
forge doctor --format json
forge analyze --format json
```

Example JSON output:

```json
{
  "schema_version": "1",
  "tool": "forge",
  "version": "0.3.0",
  "command": "add",
  "features": [
    {"name": "auth", "version": "1.0.0", "description": "..."},
    {"name": "cors", "version": "1.0.2", "description": "..."}
  ]
}
```

### Quiet mode

Suppress non-essential output:

```bash
forge add auth --quiet
```

---

## Validate a generated project

ForgeKit provides several commands to help developers verify their project.

### Doctor

Check the development environment and project health:

```bash
forge doctor
```

Output includes:
- ForgeKit version
- `.forge` signature validation (valid, legacy, absent, invalid)
- Installed feature versions (vs registry)
- Go, Git, Docker availability
- PostgreSQL configuration
- Project structure (go.mod, .env, docker-compose.yml)
- Architecture rules, security, dependencies, documentation

**CI mode** (non-interactive, deterministic, exit codes):

```bash
forge doctor --ci
```

Exit codes: `0` = success, `1` = warnings/errors, `2` = execution error

**JSON output**:

```bash
forge doctor --format json
```

### Check

Validate architectural conventions:

```bash
forge check
```

### Analyze

Analyze the project structure and practices across categories:
- Architecture
- Tests
- Security
- Configuration
- Docker
- Documentation

```bash
forge analyze
```

Output shows category scores (0-100), global score, and recommendations.

**CI mode** (non-interactive, deterministic, exit codes):

```bash
forge analyze --ci
```

Exit codes: `0` = success, `1` = issues detected, `2` = execution error

**JSON output**:

```bash
forge analyze --format json
```

### Run tests

Inside the generated project:

```bash
go test ./...
```

### Run static analysis

```bash
go vet ./...
```

---

## CLI

Display the available commands:

```bash
forge --help
```

Available commands include:

```text
add
analyze
check
completion
config
doctor
init
inspect
remove
version
```

Global options:

```text
--debug
--format
--quiet
```

**CI modes** (non-interactive, deterministic, exit codes 0/1/2):

```bash
forge doctor --ci
forge analyze --ci
```

JSON output can be requested with:

```bash
forge --format json doctor
forge --format json analyze
forge add --list --format json
forge add auth --plan --format json
forge remove auth --format json
```

---

## Feature system architecture

ForgeKit introduces a generic feature infrastructure in `internal/feature/`:

- **Feature interface** — Defines `Name()`, `Description()`, `Version()`, `Check()`, `Plan()`, `Apply()`
- **FeatureDependencies** (optional) — Declares dependencies via `DependsOn() []string`
- **FeatureRemover** (optional) — Supports removal via `Remove()`
- **ProjectContext** — Project metadata (root, module, Go version, type)
- **Manifest** — Feature resources (dependencies, files, environment variables)
- **Plan** — Computed installation/removal plan with conflict detection
- **Registry** — Feature registration and dependency resolution (topological sort)
- **Detector** — Validates ForgeKit project structure (ForgeKit, legacy, external compatible, invalid)
- **Installer** — Applies plans with rollback support and idempotent operations
- **Installed tracking** — `.forge/features.yaml` records installed features with versions

### Feature dependencies

Features can declare dependencies on other features:

```go
func (MyFeature) DependsOn() []string {
    return []string{"auth"}  // auth must be installed first
}
```

When installing a feature with dependencies:
- Dependencies are automatically resolved and installed first
- Topological sort ensures correct installation order
- Circular dependencies are detected and rejected

Current feature dependency graph:
```
auth (no deps)
  └── cors (depends on auth)
  └── logging (depends on auth)
      └── swagger (depends on cors)
```

### .forge system

The `.forge/` directory tracks project metadata and installed features:

```
.forge/
├── forge.yaml      # Project metadata (version, schema, project name, type)
└── features.yaml   # Installed features with versions and timestamps
```

**Signature validation** (used by `doctor`, `analyze`, `inspect`):
- **Valid**: Both `forge.yaml` and `features.yaml` present, schema compatible
- **Legacy**: Only `features.yaml` (v0.1.x projects)
- **Absent**: No `.forge` directory (external projects)
- **Invalid**: Directory exists but missing required files or schema incompatible

### Adding a new feature

1. Implement the `Feature` interface in `internal/feature/<name>/`
2. Add template files in `internal/template/api/internal/<name>/`
3. Register the feature in `internal/cli/commands.go` in `newAddCommand()` and `newRemoveCommand()`

---

## Rollback System

ForgeKit includes a rollback mechanism to restore project state when feature installation fails.

### How it works

1. **Before installation**: A complete snapshot is taken of:
   - `go.mod` and `go.sum`
   - `.env.example`
   - `.forge/` directory (metadata and features.yaml)

2. **On failure**: Automatic rollback restores all captured files

3. **Verification**: Post-rollback verification ensures state matches the snapshot

### Capabilities

| Can Rollback | Cannot Rollback |
|-------------|-----------------|
| `go.mod` / `go.sum` content | Go module cache (GOMODCACHE) |
| `.env.example` file content | Docker container state |
| `.forge/forge.yaml` metadata | External databases |
| `.forge/features.yaml` installed features | Git history |
| Feature-generated files (`internal/*`) | System packages (apt/brew) |
| | gofmt whitespace changes |

### Limitations

- **Not transactional**: External operations (`go get`, `go mod tidy`, Docker) cannot be fully undone
- **Module cache**: `go get` downloads modules to GOMODCACHE which persists
- **Docker**: Container state, volumes, and networks are not managed
- **External services**: PostgreSQL databases, Redis, etc. are outside ForgeKit's control
- **Git**: Commits, tags, branches are not modified by ForgeKit

### Manual rollback

If automatic rollback fails, restore from snapshot:

```bash
# Manual restoration from backups
cp go.mod.bak go.mod
cp go.sum.bak go.sum
rm -rf .forge
# Then restore from your backups
```

---

## Limitations

Current v0.3.0 limitations:

- No feature version upgrade path (manual intervention required)
- Features must be registered in the CLI binary (no plugin system yet)
- External project support is limited to compatible Go projects with Chi router
- Rollback cannot undo external operations (Go module cache, Docker, databases)

---

## ⚡ Why ForgeKit?

Creating a backend from scratch often means repeating the same work:

```text
Create directories
       ↓
Create Go modules
       ↓
Configure HTTP server
       ↓
Configure PostgreSQL
       ↓
Create migrations
       ↓
Create Docker files
       ↓
Create tests
       ↓
Configure environment
       ↓
Check architecture
       ↓
Format code
       ↓
Run tests
```

ForgeKit turns this repetitive process into:

```bash
forge init my-api
```

And extends it with features:

```bash
forge add auth
```

The developer can then focus on the actual business logic.

---

## Project philosophy

ForgeKit is not intended to hide Go from developers.

It is intended to eliminate repetitive boilerplate while keeping the generated project:

- readable
- conventional
- testable
- maintainable
- extensible
- understandable by any Go developer

The generated project belongs to the developer. ForgeKit does not create a proprietary runtime or lock the project into a framework.

---

## 🛣 Roadmap

### v0.1.0

- [x] Go REST API generation
- [x] Hexagonal architecture
- [x] PostgreSQL
- [x] Docker
- [x] Database migrations
- [x] Project tests
- [x] `forge doctor`
- [x] `forge check`
- [x] `forge analyze`

### v0.2.0

- [x] Feature system (`forge add`)
- [x] JWT authentication (`forge add auth`)
- [x] Idempotent feature installation
- [x] Dry-run support
- [x] JSON/quiet output modes
- [ ] Redis integration
- [ ] Swagger/OpenAPI generation
- [ ] Additional database options
- [ ] More project templates
- [ ] Better project analysis
- [ ] More automated architecture rules

### Future

Potential directions:

- [ ] Plugin system
- [ ] More backend architectures
- [ ] Microservice templates
- [ ] CI/CD generation
- [ ] Cloud deployment templates
- [ ] Templates for other ecosystems

---

## Contributing

Contributions are welcome.

Fork the repository, create a branch, make your changes, and open a pull request.

Before submitting a pull request, run:

```bash
gofmt -w .
go test ./...
go vet ./...
go build ./...
```

---

## License

See the repository license for details.

---

## Support the project

If ForgeKit is useful to you, consider starring the repository on GitHub.

Issues, feature requests, documentation improvements, and pull requests are welcome.

---

## Changelog

### v0.2.0 (2026-08-10)

- Added `forge add` command with extensible feature system
- Implemented `forge add auth` for JWT authentication infrastructure
- Added `--list`, `--dry-run`, `--format json`, `--quiet` flags
- Added progress display with spinners and colored output
- Added `.forge/features.yaml` for tracking installed features
- Added rollback mechanism for failed installations
- Added comprehensive unit tests for feature system
- Updated generated project structure with `internal/auth/`

### v0.1.2

- Initial release with `forge init`, `forge doctor`, `forge check`, `forge analyze`
- Hexagonal architecture scaffolding
- PostgreSQL, Docker, migrations, tests

---

**ForgeKit — Build the backend, not the boilerplate.**
