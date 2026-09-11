# AGENTS.md — ForgeKit

## Project Overview
ForgeKit is a Go CLI that generates production-ready REST APIs with hexagonal architecture. Commands: `init`, `add`, `remove`, `doctor`, `check`, `analyze`, `inspect`, `config`, `version`.

## Build & Test Commands

```bash
# Build CLI (binary named 'forgekit')
go build -o forgekit ./cmd/forge

# Full CI pipeline (order matters)
go vet ./...
go test ./...
go build ./...

# Package tests
go test ./internal/generator/...
go test ./internal/feature/...
go test ./internal/cli/...
go test ./internal/output/...
go test ./internal/report/...
go test ./internal/rules/...
go test ./internal/forge/...
go test ./internal/dbinspect/...
go test ./internal/ports/...

# E2E tests (requires Docker, build tag)
go test -tags=e2e ./internal/e2e/...

# Integration test (run in CI)
./forgekit init ci-api --non-interactive --module github.com/forgekit/ci-api
cd ci-api && go test ./...
```

## Makefile Targets
```bash
make build          # Build dev binary (outputs 'forge')
make test           # Run all tests
make vet            # Run go vet
make lint           # Run gofmt -w .
make build-linux    # Static Linux binary (CGO_ENABLED=0)
make package        # Release artifacts via GoReleaser (snapshot)
make clean          # Remove build artifacts
make ci             # Full pipeline: lint -> vet -> test -> build
```

**Note**: CI workflow runs `go mod download -> vet -> test -> build -> integration test` (no lint, Go 1.22). Release workflow uses Go 1.26.

## Key Architecture

| Path | Role |
|------|------|
| `cmd/forge/main.go` | Entry point, delegates to `internal/cli` |
| `internal/cli/commands.go` | All CLI commands |
| `internal/cli/root.go` | Root command, global flags, branding |
| `internal/generator/` | Project generation logic |
| `internal/template/api/` | Go text/templates for generated projects |
| `internal/feature/` | Feature registry for `forge add`/`remove` |
| `internal/feature/auth/` | JWT auth feature |
| `internal/feature/cors/` | CORS middleware feature |
| `internal/feature/logging/` | Structured logging feature |
| `internal/feature/swagger/` | OpenAPI/Swagger feature |
| `internal/rules/` | Architectural rules (security, arch, quality, env, docker, config) |
| `internal/analyzer/project.go` | Project analysis logic |
| `internal/app/` | App metadata (name, version, slogan) |
| `internal/config/` | User config (~/.forgekit/config.yaml) & project config (forge.yaml) |
| `internal/output/` | Console utils (spinner, colored output, JSON) |
| `internal/engine/` | Template execution engine with variables |
| `internal/report/` | Scoring & reporting for analyze |
| `internal/forge/` | Project signature validation, metadata, feature tracking (.forge/) |
| `internal/ports/` | Port availability checking |
| `internal/dbinspect/` | DB inspection (migrations, schema) |
| `pkg/generator/` | Shared generator types |

**Empty directories**: `internal/doctor`, `internal/analyze`, `internal/arch`, `internal/project` — logic lives in `cli/commands.go` and `rules/`. `internal/check` does not exist.

## Dependencies
- `github.com/spf13/cobra` — CLI framework
- `gopkg.in/yaml.v3` — YAML parsing
- `github.com/jackc/pgx/v5` — PostgreSQL driver (for generated projects)
- Go **1.26** (go.mod), CI uses **1.22** (`.github/workflows/ci.yml`)

## VS Code Extension
- Located in `integrations/vscode/` — TypeScript extension for ForgeKit
- Build: `cd integrations/vscode && npm install && npm run compile && npm run lint`
- Package: `vsce package`

## Generated Project Stack (frozen since v0.1)
- Go stdlib `net/http` + **Chi router** (`github.com/go-chi/chi/v5`)
- PostgreSQL with `database/sql` — **no GORM**
- Docker Compose for local dev
- Env config via `.env` (caarlos0/env pattern)
- Table-driven tests + optional testcontainers-go

## Feature Dependency Graph
```
auth (no deps)
  └── cors (depends on auth)
  └── logging (depends on auth)
      └── swagger (depends on cors)
```

## Feature Integration Internals
- `internal/feature/integration.go` — Safe router.go/main.go modification for feature installation/removal
  - `IntegrateRouterGo` / `RemoveRouterGo` — Adds/removes imports and middleware calls in `internal/transport/http/router.go`
  - `IntegrateMainGo` / `RemoveMainGo` — Adds/removes imports and code in `cmd/server/main.go`
  - Handles multiple features by merging imports and preserving other features' integrations

## Common Tasks

### Add new CLI command
1. Add command in `internal/cli/commands.go`
2. Register in `internal/cli/root.go` (`NewRootCommand`)
3. Add tests in `internal/cli/commands_test.go`

### Modify generated project templates
Edit files in `internal/template/api/` — Go text/template files.

### Add new `forge add`/`remove` feature
1. Implement `Feature` interface in `internal/feature/<name>/`
2. Add templates in `internal/template/api/internal/<name>/`
3. Register in `internal/cli/commands.go` in `newAddCommand()` and `newRemoveCommand()` (see `auth.AuthFeature{}`, `cors.CorsFeature{}`, `logging.LoggingFeature{}`, `swagger.SwaggerFeature{}`)

## CI Pipeline (`.github/workflows/ci.yml`)
Runs on push/PR to main/master (Go 1.22):
1. `go mod download`
2. `go vet ./...`
3. `go test ./...`
4. `go build -o forge ./cmd/forge`
5. Integration test: `forge init` + `go test ./...` in generated project

## Release Pipeline (`.github/workflows/release.yml`)
Triggered on `v*` tags (Go 1.26):
1. Checkout with fetch-depth: 0
2. `go mod download`
3. `go test ./...`
4. `go vet ./...`
5. GoReleaser builds & publishes to GitHub Releases

## Snap Publishing (`.github/workflows/snap.yml`)
Triggered on release publish (or manual dispatch):
1. `go mod download`
2. `go test ./...`
3. `go vet ./...`
4. Build binary with `CGO_ENABLED=0` and version ldflag
5. Build Snap via `snapcraft --use-lxd` (confinement: strict)
6. Publish to Snap Store (stable releases only)
7. Upload Snap artifact

**Snap Config** (`snap/snapcraft.yaml`):
- Name: `forgekit`, version from git tag
- Confinement: **strict** (not classic)
- Base: `core22`
- Apps: `forgekit` (main), `forge` (alias symlink)
- Plugs: `home`, `removable-media`, `network`, `network-bind`

## Pre-commit / Contribution Checks
```bash
gofmt -w .
go test ./...
go vet ./...
go build ./...
```

## Packages Without Tests
These internal packages have no test files:
- `cmd/forge` (entry point, normal)
- `internal/analyzer`
- `internal/app`
- `internal/config`
- `internal/engine`
- `internal/errs`
- `internal/prompt`
- `internal/template`
- `pkg/generator`

## Environment
- Docker required for generated project Docker workflow
- Uses `.env.example` as template for generated projects
- Default API port: 8080
- User config: `~/.forgekit/config.yaml`
- Project config: `forge.yaml` in generated projects

## Release Process
- GoReleaser (`.goreleaser.yaml`)
- Builds for linux/amd64 and linux/arm64
- Creates `.tar.gz`, `.deb`, SHA256 checksums
- Version via ldflags: `github.com/Demetrius-ch/forgekit/internal/app.Version`
- Triggered by git tags (`v*`)
- Binary name: **`forgekit`** (release), `forge` (dev build alias)

## Key References
- `README.md` — User docs, usage examples
- `projects.md` — Product vision & roadmap
- `explication.md` — Competitive analysis & positioning
- `CONTRIBUTING.md` — Contribution guidelines & commit format
- `.github/workflows/ci.yml` — Authoritative CI commands
- `Makefile` — Dev commands
- `.goreleaser.yaml` — Release config
- `integrations/vscode/` — VS Code extension source