# AGENTS.md — ForgeKit

## Project Overview
ForgeKit is a Go CLI tool that generates production-ready REST APIs with hexagonal architecture. It scaffolds projects with PostgreSQL, Docker, migrations, tests, and provides `doctor`, `check`, `analyze` commands for validation.

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

# Integration test (runs in CI)
./forge init ci-api --non-interactive --module github.com/forgekit/ci-api
cd ci-api && go test ./...
```

## Key Architecture

- **cmd/forge/main.go** — Entry point, delegates to `internal/cli`
- **internal/cli/commands.go** — All CLI commands (init, add, doctor, check, analyze, config, version)
- **internal/generator/** — Project generation logic (templates in `internal/template/`)
- **internal/template/api/** — Go templates for generated project structure
- **internal/feature/** — Feature registry for `forge add`
- **internal/doctor/** — Environment diagnostics
- **internal/check/** — Architecture validation
- **internal/analyze/** — Project analysis
- **internal/rules/** — Architectural rules for check/analyze

## Dependencies
- `github.com/spf13/cobra` — CLI framework
- `gopkg.in/yaml.v3` — YAML config parsing
- Go 1.22+ (per go.mod, though README says 1.25+)

## Generated Project Stack (Fixed in V0.1)
- Go stdlib `net/http` (or Chi) — no Gin/Echo/Fiber
- PostgreSQL with `database/sql` — no GORM
- Docker Compose for local dev
- Environment config via `.env` (caarlos0/env pattern)
- Table-driven tests + testcontainers-go optional

## Common Tasks

### Add a new CLI command
1. Add command in `internal/cli/commands.go`
2. Register in `internal/cli/root.go`
3. Add tests in `internal/cli/commands_test.go`

### Modify generated project templates
Edit files in `internal/template/api/` — these are Go text/template files.

### Add a new `forge add` feature
1. Register feature in `internal/feature/registry.go`
2. Implement generator logic in `internal/generator/`
3. Add template files in `internal/template/api/`

## CI Pipeline (`.github/workflows/ci.yml`)
Runs on push/PR to main/master:
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

## Known Issues
- `internal/cli/commands.go:456` has unused variable `console` — build fails until fixed
- Some internal packages lack tests (analyzer, app, config, engine, errs, prompt, template, pkg/generator)

## Environment
- Requires Docker for generated project's Docker workflow
- Uses `.env.example` as template for generated projects
- Default API port: 8080

## Useful References
- `README.md` — User-facing docs, usage examples
- `projects.md` — Product vision and roadmap
- `explication.md` — Competitive analysis and positioning
- `.github/workflows/ci.yml` — Authoritative CI commands