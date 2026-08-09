# ForgeKit

> Generate production-ready Go REST APIs with a clean hexagonal architecture.

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
-  Automatic Go formatting
-  Architecture validation
-  Development environment diagnostics
-  Project analysis
-  Human-readable or JSON output
-  Extensible CLI architecture

---

##  Installation

### From source

Requirements:

- Go 1.25+
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

---

##  Usage

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

##  Generated project

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

---

##  Architecture

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

##  Run the generated API with Docker

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

##  Validate a generated project

ForgeKit provides several commands to help developers verify their project.

### Doctor

Check the development environment:

```bash
forge doctor
```

### Check

Validate architectural conventions:

```bash
forge check
```

### Analyze

Analyze the project:

```bash
forge analyze
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

##  CLI

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
version
```

Global options:

```text
--debug
--format
--quiet
```

JSON output can be requested with:

```bash
forge --format json doctor
```

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

The developer can then focus on the actual business logic.

---

##  Project philosophy

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

Planned:

- [ ] Authentication manifest
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

##  Contributing

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

**ForgeKit — Build the backend, not the boilerplate.**
