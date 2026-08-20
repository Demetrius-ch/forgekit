# AGENTS.md — ForgeKit

## Project Overview
ForgeKit is a Go CLI tool that generates production-ready REST APIs with hexagonal architecture. It scaffolds projects with PostgreSQL, Docker, migrations, tests, and provides `doctor`, `check`, `analyze`, `inspect`, `config`, `add`, `remove`, `version` commands for validation and extension.

## Build & Test Commands

```bash
# Build the CLI
go build -o forge ./cmd/forge

# Run all tests (CI order: vet -> test -> build)
go vet ./...
go test ./...
go build ./...

# Run single package tests
go test ./internal/generator/...
go test ./internal/feature/...
go test ./internal/cli/...
go test ./internal/output/...
go test ./internal/report/...
go test ./internal/rules/...
go test ./internal/forge/...
go test ./internal/dbinspect/...
go test ./internal/ports/...

# Run e2e tests (requires build tag)
go test -tags=e2e ./internal/e2e/...

# Integration test (runs in CI)
./forge init ci-api --non-interactive --module github.com/forgekit/ci-api
cd ci-api && go test ./...
```

## Key Architecture

- **cmd/forge/main.go** — Entry point, delegates to `internal/cli`
- **internal/cli/commands.go** — All CLI commands (init, add, remove, doctor, check, analyze, inspect, config, version)
- **internal/cli/root.go** — Root command setup, global flags, branding
- **internal/generator/** — Project generation logic (templates in `internal/template/`)
- **internal/template/api/** — Go text/template files for generated project structure
- **internal/feature/** — Feature registry for `forge add`/`forge remove` (interface, detector, installer, registry)
- **internal/feature/auth/** — JWT authentication feature implementation
- **internal/feature/cors/** — CORS middleware feature implementation
- **internal/feature/logging/** — Logging feature implementation
- **internal/feature/swagger/** — Swagger/OpenAPI feature implementation
- **internal/rules/** — Architectural rules for check/analyze (security, architecture, quality, environment, docker, config)
- **internal/analyzer/project.go** — Project analysis logic
- **internal/app/** — App metadata (name, version, slogan)
- **internal/config/** — User config (~/.forgekit/config.yaml) and project config (forge.yaml)
- **internal/output/** — Console output utilities (spinner, colored output, JSON)
- **internal/engine/** — Template execution engine with variables
- **internal/report/** — Scoring and reporting for analyze
- **internal/errs/** — Error types
- **internal/prompt/** — Interactive prompts
- **internal/template/** — Template rendering engine
- **internal/ports/** — Port availability checking for generated projects
- **internal/dbinspect/** — Database inspection (migrations, schema)
- **internal/forge/** — Project signature validation, metadata, feature tracking (.forge/forge.yaml, .forge/features.yaml)
- **pkg/generator/** — Shared generator types

Note: `internal/doctor`, `internal/analyze`, `internal/arch`, `internal/project` are empty directories; `internal/check` does not exist; logic lives in `cli/commands.go` and `rules/`.

## Dependencies
- `github.com/spf13/cobra` — CLI framework
- `gopkg.in/yaml.v3` — YAML config parsing
- Go 1.25 (per go.mod), CI uses 1.22 (see `.github/workflows/ci.yml`)

## Generated Project Stack (Fixed in V0.1)
- Go stdlib `net/http` with **Chi router** (`github.com/go-chi/chi/v5`)
- PostgreSQL with `database/sql` — no GORM
- Docker Compose for local dev
- Environment config via `.env` (caarlos0/env pattern)
- Table-driven tests + testcontainers-go optional

## Common Tasks

### Add a new CLI command
1. Add command in `internal/cli/commands.go`
2. Register in `internal/cli/root.go` (in `NewRootCommand`)
3. Add tests in `internal/cli/commands_test.go`

### Modify generated project templates
Edit files in `internal/template/api/` — these are Go text/template files.

### Add a new `forge add` / `forge remove` feature
1. Implement the `Feature` interface in `internal/feature/<name>/`
2. Add template files in `internal/template/api/internal/<name>/`
3. Register the feature in `internal/cli/commands.go` in `newAddCommand()` and `newRemoveCommand()` (see `auth.AuthFeature{}, cors.CorsFeature{}, logging.LoggingFeature{}, swagger.SwaggerFeature{}`)

## CI Pipeline (`.github/workflows/ci.yml`)
Runs on push/PR to main/master (uses Go 1.22):
1. `go mod download`
2. `go vet ./...`
3. `go test ./...`
4. `go build -o forge ./cmd/forge`
5. Integration test: `forge init` + `go test ./...` in generated project

## Pre-commit / Contribution Checks
From README contributing section:
```bash
gofmt -w .
go test ./...
go vet ./...
go build ./...
```

## Packages Without Tests
These internal packages currently have no test files:
- `cmd/forge` (entry point, expected)
- `internal/analyzer`
- `internal/app`
- `internal/config`
- `internal/engine`
- `internal/errs`
- `internal/prompt`
- `internal/template`
- `pkg/generator`

## Environment
- Requires Docker for generated project's Docker workflow
- Uses `.env.example` as template for generated projects
- Default API port: 8080
- User config at `~/.forgekit/config.yaml`
- Project config at `forge.yaml` in generated projects

## Useful References
- `README.md` — User-facing docs, usage examples
- `projects.md` — Product vision and roadmap
- `explication.md` — Competitive analysis and positioning
- `.github/workflows/ci.yml` — Authoritative CI commands