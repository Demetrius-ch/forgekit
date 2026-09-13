// Package generationplan converts a validated ProjectConfig into an explicit,
// inspectable list of generation components. It deliberately contains no file
// system writes; the engine executes the resulting plan in a later step.
package generationplan

import (
	"fmt"
	"sort"

	"github.com/Demetrius-ch/forgekit/internal/projectconfig"
)

// File describes a template and the path it will produce in a project.
type File struct {
	Template    string `json:"template"`
	Destination string `json:"destination"`
	Component   string `json:"component"`
}

// Dependency describes an explicit Go module required by a component.
type Dependency struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	Reason  string `json:"reason"`
}

// Plan is the immutable result of composing project components.
type Plan struct {
	Configuration projectconfig.ProjectConfig `json:"configuration"`
	Components    []string                    `json:"components"`
	Files         []File                      `json:"files"`
	Dependencies  []Dependency                `json:"dependencies"`
}

// Component contributes templates and dependencies for a configuration. New
// architectures and capabilities can be introduced by registering another
// component instead of spreading conditional checks through the generator.
type Component interface {
	Name() string
	Contribute(projectconfig.ProjectConfig) (Contribution, error)
}

// Contribution is the output of one Component.
type Contribution struct {
	Files        []File
	Dependencies []Dependency
}

// Registry owns all component variants supported by a ForgeKit build.
type Registry struct {
	architectures  map[projectconfig.Architecture]Component
	databases      map[projectconfig.Database]Component
	docker         Component
	authentication map[projectconfig.Authentication]Component
	documentation  map[projectconfig.Documentation]Component
	tests          map[projectconfig.TestStrategy]Component
	ci             map[projectconfig.CIStrategy]Component
	base           Component
}

// DefaultRegistry returns the v0.6 Go component set. The registry is intentionally
// the only place where a configuration value is mapped to a concrete variant.
func DefaultRegistry() Registry {
	return GoRegistry()
}

// GoRegistry returns the Go-specific component set.
func GoRegistry() Registry {
	return Registry{
		base: staticComponent{
			name: "base",
			files: files("base",
				"v04/base/.env.example.tmpl", ".env.example",
				"v04/base/.gitignore.tmpl", ".gitignore",
				"v04/base/Makefile.tmpl", "Makefile",
				"v04/base/README.md.tmpl", "README.md",
				"v04/base/forge.yaml.tmpl", "forge.yaml",
				"v04/base/internal/infrastructure/config/config.go.tmpl", "internal/infrastructure/config/config.go",
			),
			dependencies: []Dependency{{Module: "github.com/go-chi/chi/v5", Version: "v5.0.12", Reason: "HTTP router"}},
		},
		architectures: map[projectconfig.Architecture]Component{
			projectconfig.ArchitectureHexagonal: architectureComponent{
				name: "architecture:hexagonal",
				files: files("architecture:hexagonal",
					"v04/architectures/hexagonal/cmd/server/main.go.tmpl", "cmd/server/main.go",
					"v04/architectures/hexagonal/internal/domain/health.go.tmpl", "internal/domain/health.go",
					"v04/architectures/hexagonal/internal/application/health.go.tmpl", "internal/application/health.go",
					"v04/architectures/hexagonal/internal/transport/http/router.go.tmpl", "internal/transport/http/router.go",
				),
			},
			projectconfig.ArchitectureClean: architectureComponent{
				name: "architecture:clean",
				files: files("architecture:clean",
					"v04/architectures/clean/cmd/server/main.go.tmpl", "cmd/server/main.go",
					"v04/architectures/clean/internal/entities/health.go.tmpl", "internal/entities/health.go",
					"v04/architectures/clean/internal/usecases/health.go.tmpl", "internal/usecases/health.go",
					"v04/architectures/clean/internal/interfaces/http/router.go.tmpl", "internal/interfaces/http/router.go",
				),
			},
			projectconfig.ArchitectureLayered: architectureComponent{
				name: "architecture:layered",
				files: files("architecture:layered",
					"v04/architectures/layered/cmd/server/main.go.tmpl", "cmd/server/main.go",
					"v04/architectures/layered/internal/models/health.go.tmpl", "internal/models/health.go",
					"v04/architectures/layered/internal/services/health.go.tmpl", "internal/services/health.go",
					"v04/architectures/layered/internal/handlers/http.go.tmpl", "internal/handlers/http.go",
				),
			},
		},
		databases: map[projectconfig.Database]Component{
			projectconfig.DatabasePostgres: databaseComponent{
				name: "database:postgres", database: projectconfig.DatabasePostgres,
				dependencies: []Dependency{{Module: "github.com/jackc/pgx/v5", Version: "v5.10.0", Reason: "PostgreSQL driver"}},
			},
			projectconfig.DatabaseMySQL: databaseComponent{
				name: "database:mysql", database: projectconfig.DatabaseMySQL,
				dependencies: []Dependency{{Module: "github.com/go-sql-driver/mysql", Version: "v1.8.1", Reason: "MySQL driver"}},
			},
			projectconfig.DatabaseSQLite: databaseComponent{
				name: "database:sqlite", database: projectconfig.DatabaseSQLite,
				dependencies: []Dependency{{Module: "modernc.org/sqlite", Version: "v1.34.5", Reason: "pure Go SQLite driver"}},
			},
			projectconfig.DatabaseNone: staticComponent{name: "database:none"},
		},
		docker: dockerComponent{},
		authentication: map[projectconfig.Authentication]Component{
			projectconfig.AuthenticationNone: staticComponent{name: "authentication:none"},
			projectconfig.AuthenticationJWT: staticComponent{
				name: "authentication:jwt",
				files: files("authentication:jwt",
					"v04/components/auth/jwt/jwt.go.tmpl", "internal/auth/jwt.go",
					"v04/components/auth/jwt/middleware.go.tmpl", "internal/auth/middleware.go",
				),
				dependencies: []Dependency{{Module: "github.com/golang-jwt/jwt/v5", Version: "v5.2.0", Reason: "JWT tokens"}},
			},
		},
		documentation: map[projectconfig.Documentation]Component{
			projectconfig.DocumentationNone: staticComponent{name: "documentation:none"},
			projectconfig.DocumentationSwagger: staticComponent{
				name: "documentation:swagger",
				files: files("documentation:swagger",
					"v04/components/documentation/swagger/openapi.yaml.tmpl", "internal/swagger/openapi.yaml",
					"v04/components/documentation/swagger/swagger.go.tmpl", "internal/swagger/swagger.go",
				),
				dependencies: []Dependency{{Module: "github.com/swaggo/http-swagger", Version: "v1.3.4", Reason: "Swagger UI"}},
			},
		},
		tests: map[projectconfig.TestStrategy]Component{
			projectconfig.TestStrategyUnit: staticComponent{
				name:  "tests:unit",
				files: files("tests:unit", "v04/components/tests/unit/health_test.go.tmpl", "tests/health_test.go"),
			},
			projectconfig.TestStrategyUnitIntegration: staticComponent{
				name: "tests:unit-integration",
				files: files("tests:unit-integration",
					"v04/components/tests/unit/health_test.go.tmpl", "tests/health_test.go",
					"v04/components/tests/integration/http_test.go.tmpl", "tests/http_integration_test.go",
				),
			},
		},
		ci: map[projectconfig.CIStrategy]Component{
			projectconfig.CIStrategyNone: staticComponent{name: "ci:none"},
			projectconfig.CIStrategyGitHub: staticComponent{
				name:  "ci:github-actions",
				files: files("ci:github-actions", "v04/components/ci/github-actions/ci.yml.tmpl", ".github/workflows/ci.yml"),
			},
		},
	}
}

// TypeScriptRegistry returns the TypeScript-specific component set.
func TypeScriptRegistry() Registry {
	return Registry{
		base: staticComponent{
			name: "base",
			files: files("base",
				"v04/languages/typescript/base/package.json.tmpl", "package.json",
				"v04/languages/typescript/base/tsconfig.json.tmpl", "tsconfig.json",
				"v04/languages/typescript/base/.env.example.tmpl", ".env.example",
				"v04/languages/typescript/base/.gitignore.tmpl", ".gitignore",
				"v04/languages/typescript/base/vitest.config.ts.tmpl", "vitest.config.ts",
			),
		},
		architectures: map[projectconfig.Architecture]Component{
			projectconfig.ArchitectureHexagonal: architectureComponent{
				name: "architecture:hexagonal",
				files: files("architecture:hexagonal",
					"v04/languages/typescript/architectures/hexagonal/src/index.ts.tmpl", "src/index.ts",
					"v04/languages/typescript/architectures/hexagonal/src/app.ts.tmpl", "src/app.ts",
					"v04/languages/typescript/architectures/hexagonal/src/domain/health.entity.ts.tmpl", "src/domain/health.entity.ts",
					"v04/languages/typescript/architectures/hexagonal/src/application/health.usecase.ts.tmpl", "src/application/health.usecase.ts",
					"v04/languages/typescript/architectures/hexagonal/src/interfaces/http/health.router.ts.tmpl", "src/interfaces/http/health.router.ts",
				),
			},
			projectconfig.ArchitectureClean: architectureComponent{
				name: "architecture:clean",
				files: files("architecture:clean",
					"v04/languages/typescript/architectures/clean/src/index.ts.tmpl", "src/index.ts",
					"v04/languages/typescript/architectures/clean/src/app.ts.tmpl", "src/app.ts",
					"v04/languages/typescript/architectures/clean/src/entities/health.entity.ts.tmpl", "src/entities/health.entity.ts",
					"v04/languages/typescript/architectures/clean/src/usecases/health.usecase.ts.tmpl", "src/usecases/health.usecase.ts",
					"v04/languages/typescript/architectures/clean/src/interfaces/http/health.router.ts.tmpl", "src/interfaces/http/health.router.ts",
				),
			},
			projectconfig.ArchitectureLayered: architectureComponent{
				name: "architecture:layered",
				files: files("architecture:layered",
					"v04/languages/typescript/architectures/layered/src/index.ts.tmpl", "src/index.ts",
					"v04/languages/typescript/architectures/layered/src/app.ts.tmpl", "src/app.ts",
					"v04/languages/typescript/architectures/layered/src/models/health.model.ts.tmpl", "src/models/health.model.ts",
					"v04/languages/typescript/architectures/layered/src/services/health.service.ts.tmpl", "src/services/health.service.ts",
					"v04/languages/typescript/architectures/layered/src/routes/health.routes.ts.tmpl", "src/routes/health.routes.ts",
				),
			},
		},
		databases: map[projectconfig.Database]Component{
			projectconfig.DatabasePostgres: staticComponent{
				name: "database:postgres",
				files: files("database:postgres",
					"v04/languages/typescript/components/database/postgres/prisma/schema.prisma.tmpl", "prisma/schema.prisma",
				),
			},
			projectconfig.DatabaseMySQL: staticComponent{
				name: "database:mysql",
				files: files("database:mysql",
					"v04/languages/typescript/components/database/mysql/prisma/schema.prisma.tmpl", "prisma/schema.prisma",
				),
			},
			projectconfig.DatabaseSQLite: staticComponent{
				name: "database:sqlite",
				files: files("database:sqlite",
					"v04/languages/typescript/components/database/sqlite/prisma/schema.prisma.tmpl", "prisma/schema.prisma",
				),
			},
			projectconfig.DatabaseNone: staticComponent{name: "database:none"},
		},
		docker: dockerComponent{},
		authentication: map[projectconfig.Authentication]Component{
			projectconfig.AuthenticationNone: staticComponent{name: "authentication:none"},
			projectconfig.AuthenticationJWT: staticComponent{
				name: "authentication:jwt",
				files: files("authentication:jwt",
					"v04/languages/typescript/components/auth/jwt/src/auth/jwt.ts.tmpl", "src/auth/jwt.ts",
					"v04/languages/typescript/components/auth/jwt/src/auth/middleware.ts.tmpl", "src/auth/middleware.ts",
				),
			},
		},
		documentation: map[projectconfig.Documentation]Component{
			projectconfig.DocumentationNone: staticComponent{name: "documentation:none"},
			projectconfig.DocumentationSwagger: staticComponent{
				name:  "documentation:swagger",
				files: files("documentation:swagger", "v04/languages/typescript/base/package.json.tmpl", "package.json"),
			},
		},
		tests: map[projectconfig.TestStrategy]Component{
			projectconfig.TestStrategyUnit: staticComponent{
				name:  "tests:unit",
				files: files("tests:unit", "v04/languages/typescript/components/tests/unit/health.test.ts.tmpl", "tests/health.test.ts"),
			},
			projectconfig.TestStrategyUnitIntegration: staticComponent{
				name:  "tests:unit-integration",
				files: files("tests:unit-integration", "v04/languages/typescript/components/tests/unit/health.test.ts.tmpl", "tests/health.test.ts"),
			},
		},
		ci: map[projectconfig.CIStrategy]Component{
			projectconfig.CIStrategyNone: staticComponent{name: "ci:none"},
			projectconfig.CIStrategyGitHub: staticComponent{
				name:  "ci:github-actions",
				files: files("ci:github-actions", "v04/components/ci/github-actions/ci.yml.tmpl", ".github/workflows/ci.yml"),
			},
		},
	}
}

// RegistryForLanguage returns the appropriate registry for the given language.
func RegistryForLanguage(language projectconfig.Language) Registry {
	switch language {
	case projectconfig.LanguageTypeScript:
		return TypeScriptRegistry()
	case projectconfig.LanguageJavaScript:
		return JavaScriptRegistry()
	case projectconfig.LanguagePython:
		return PythonRegistry()
	default:
		return GoRegistry()
	}
}

// JavaScriptRegistry returns the JavaScript-specific component set.
func JavaScriptRegistry() Registry {
	return Registry{
		base: staticComponent{
			name: "base",
			files: files("base",
				"v04/languages/javascript/base/package.json.tmpl", "package.json",
				"v04/languages/javascript/base/.env.example.tmpl", ".env.example",
				"v04/languages/javascript/base/.gitignore.tmpl", ".gitignore",
				"v04/languages/javascript/base/vitest.config.js.tmpl", "vitest.config.js",
			),
		},
		architectures: map[projectconfig.Architecture]Component{
			projectconfig.ArchitectureHexagonal: architectureComponent{
				name: "architecture:hexagonal",
				files: files("architecture:hexagonal",
					"v04/languages/javascript/architectures/hexagonal/src/index.js.tmpl", "src/index.js",
					"v04/languages/javascript/architectures/hexagonal/src/app.js.tmpl", "src/app.js",
					"v04/languages/javascript/architectures/hexagonal/src/domain/health.entity.js.tmpl", "src/domain/health.entity.js",
					"v04/languages/javascript/architectures/hexagonal/src/application/health.usecase.js.tmpl", "src/application/health.usecase.js",
					"v04/languages/javascript/architectures/hexagonal/src/interfaces/http/health.router.js.tmpl", "src/interfaces/http/health.router.js",
				),
			},
			projectconfig.ArchitectureClean: architectureComponent{
				name: "architecture:clean",
				files: files("architecture:clean",
					"v04/languages/javascript/architectures/clean/src/index.js.tmpl", "src/index.js",
					"v04/languages/javascript/architectures/clean/src/app.js.tmpl", "src/app.js",
					"v04/languages/javascript/architectures/clean/src/entities/health.entity.js.tmpl", "src/entities/health.entity.js",
					"v04/languages/javascript/architectures/clean/src/usecases/health.usecase.js.tmpl", "src/usecases/health.usecase.js",
					"v04/languages/javascript/architectures/clean/src/interfaces/http/health.router.js.tmpl", "src/interfaces/http/health.router.js",
				),
			},
			projectconfig.ArchitectureLayered: architectureComponent{
				name: "architecture:layered",
				files: files("architecture:layered",
					"v04/languages/javascript/architectures/layered/src/index.js.tmpl", "src/index.js",
					"v04/languages/javascript/architectures/layered/src/app.js.tmpl", "src/app.js",
					"v04/languages/javascript/architectures/layered/src/models/health.model.js.tmpl", "src/models/health.model.js",
					"v04/languages/javascript/architectures/layered/src/services/health.service.js.tmpl", "src/services/health.service.js",
					"v04/languages/javascript/architectures/layered/src/routes/health.routes.js.tmpl", "src/routes/health.routes.js",
				),
			},
		},
		databases: map[projectconfig.Database]Component{
			projectconfig.DatabasePostgres: staticComponent{
				name: "database:postgres",
				files: files("database:postgres",
					"v04/languages/javascript/components/database/postgres/prisma/schema.prisma.tmpl", "prisma/schema.prisma",
				),
			},
			projectconfig.DatabaseMySQL: staticComponent{
				name: "database:mysql",
				files: files("database:mysql",
					"v04/languages/javascript/components/database/mysql/prisma/schema.prisma.tmpl", "prisma/schema.prisma",
				),
			},
			projectconfig.DatabaseSQLite: staticComponent{
				name: "database:sqlite",
				files: files("database:sqlite",
					"v04/languages/javascript/components/database/sqlite/prisma/schema.prisma.tmpl", "prisma/schema.prisma",
				),
			},
			projectconfig.DatabaseNone: staticComponent{name: "database:none"},
		},
		docker: dockerComponent{},
		authentication: map[projectconfig.Authentication]Component{
			projectconfig.AuthenticationNone: staticComponent{name: "authentication:none"},
			projectconfig.AuthenticationJWT: staticComponent{
				name: "authentication:jwt",
				files: files("authentication:jwt",
					"v04/languages/javascript/components/auth/jwt/src/auth/jwt.js.tmpl", "src/auth/jwt.js",
					"v04/languages/javascript/components/auth/jwt/src/auth/middleware.js.tmpl", "src/auth/middleware.js",
				),
			},
		},
		documentation: map[projectconfig.Documentation]Component{
			projectconfig.DocumentationNone: staticComponent{name: "documentation:none"},
			projectconfig.DocumentationSwagger: staticComponent{
				name:  "documentation:swagger",
				files: files("documentation:swagger", "v04/languages/javascript/base/package.json.tmpl", "package.json"),
			},
		},
		tests: map[projectconfig.TestStrategy]Component{
			projectconfig.TestStrategyUnit: staticComponent{
				name:  "tests:unit",
				files: files("tests:unit", "v04/languages/javascript/components/tests/unit/health.test.js.tmpl", "tests/health.test.js"),
			},
			projectconfig.TestStrategyUnitIntegration: staticComponent{
				name:  "tests:unit-integration",
				files: files("tests:unit-integration", "v04/languages/javascript/components/tests/unit/health.test.js.tmpl", "tests/health.test.js"),
			},
		},
		ci: map[projectconfig.CIStrategy]Component{
			projectconfig.CIStrategyNone: staticComponent{name: "ci:none"},
			projectconfig.CIStrategyGitHub: staticComponent{
				name:  "ci:github-actions",
				files: files("ci:github-actions", "v04/components/ci/github-actions/ci.yml.tmpl", ".github/workflows/ci.yml"),
			},
		},
	}
}

// PythonRegistry returns the Python-specific component set.
func PythonRegistry() Registry {
	return Registry{
		base: staticComponent{
			name: "base",
			files: files("base",
				"v04/languages/python/base/requirements.txt.tmpl", "requirements.txt",
				"v04/languages/python/base/pyproject.toml.tmpl", "pyproject.toml",
				"v04/languages/python/base/.env.example.tmpl", ".env.example",
				"v04/languages/python/base/.gitignore.tmpl", ".gitignore",
				"v04/languages/python/base/settings.py.tmpl", "settings.py",
			),
		},
		architectures: map[projectconfig.Architecture]Component{
			projectconfig.ArchitectureHexagonal: architectureComponent{
				name: "architecture:hexagonal",
				files: files("architecture:hexagonal",
					"v04/languages/python/architectures/hexagonal/main.py.tmpl", "main.py",
					"v04/languages/python/architectures/hexagonal/app.py.tmpl", "app.py",
					"v04/languages/python/architectures/hexagonal/domain/__init__.py.tmpl", "domain/__init__.py",
					"v04/languages/python/architectures/hexagonal/domain/health_entity.py.tmpl", "domain/health_entity.py",
					"v04/languages/python/architectures/hexagonal/application/__init__.py.tmpl", "application/__init__.py",
					"v04/languages/python/architectures/hexagonal/application/health_usecase.py.tmpl", "application/health_usecase.py",
					"v04/languages/python/architectures/hexagonal/interfaces/__init__.py.tmpl", "interfaces/__init__.py",
					"v04/languages/python/architectures/hexagonal/interfaces/http/__init__.py.tmpl", "interfaces/http/__init__.py",
					"v04/languages/python/architectures/hexagonal/interfaces/http/health_router.py.tmpl", "interfaces/http/health_router.py",
				),
			},
			projectconfig.ArchitectureClean: architectureComponent{
				name: "architecture:clean",
				files: files("architecture:clean",
					"v04/languages/python/architectures/clean/main.py.tmpl", "main.py",
					"v04/languages/python/architectures/clean/app.py.tmpl", "app.py",
					"v04/languages/python/architectures/clean/entities/__init__.py.tmpl", "entities/__init__.py",
					"v04/languages/python/architectures/clean/entities/health_entity.py.tmpl", "entities/health_entity.py",
					"v04/languages/python/architectures/clean/usecases/__init__.py.tmpl", "usecases/__init__.py",
					"v04/languages/python/architectures/clean/usecases/health_usecase.py.tmpl", "usecases/health_usecase.py",
					"v04/languages/python/architectures/clean/interfaces/__init__.py.tmpl", "interfaces/__init__.py",
					"v04/languages/python/architectures/clean/interfaces/http/__init__.py.tmpl", "interfaces/http/__init__.py",
					"v04/languages/python/architectures/clean/interfaces/http/health_router.py.tmpl", "interfaces/http/health_router.py",
				),
			},
			projectconfig.ArchitectureLayered: architectureComponent{
				name: "architecture:layered",
				files: files("architecture:layered",
					"v04/languages/python/architectures/layered/main.py.tmpl", "main.py",
					"v04/languages/python/architectures/layered/app.py.tmpl", "app.py",
					"v04/languages/python/architectures/layered/models/__init__.py.tmpl", "models/__init__.py",
					"v04/languages/python/architectures/layered/models/health_model.py.tmpl", "models/health_model.py",
					"v04/languages/python/architectures/layered/services/__init__.py.tmpl", "services/__init__.py",
					"v04/languages/python/architectures/layered/services/health_service.py.tmpl", "services/health_service.py",
					"v04/languages/python/architectures/layered/routes/__init__.py.tmpl", "routes/__init__.py",
					"v04/languages/python/architectures/layered/routes/health_routes.py.tmpl", "routes/health_routes.py",
				),
			},
		},
		databases: map[projectconfig.Database]Component{
			projectconfig.DatabasePostgres: staticComponent{
				name: "database:postgres",
				files: files("database:postgres",
					"v04/languages/python/components/database/postgres/database.py.tmpl", "database.py",
				),
				dependencies: []Dependency{{Module: "asyncpg", Version: ">=0.29.0", Reason: "PostgreSQL async driver"}},
			},
			projectconfig.DatabaseMySQL: staticComponent{
				name: "database:mysql",
				files: files("database:mysql",
					"v04/languages/python/components/database/mysql/database.py.tmpl", "database.py",
				),
				dependencies: []Dependency{{Module: "aiomysql", Version: ">=0.2.0", Reason: "MySQL async driver"}},
			},
			projectconfig.DatabaseSQLite: staticComponent{
				name: "database:sqlite",
				files: files("database:sqlite",
					"v04/languages/python/components/database/sqlite/database.py.tmpl", "database.py",
				),
				dependencies: []Dependency{{Module: "aiosqlite", Version: ">=0.19.0", Reason: "SQLite async driver"}},
			},
			projectconfig.DatabaseNone: staticComponent{name: "database:none"},
		},
		docker: dockerComponent{},
		authentication: map[projectconfig.Authentication]Component{
			projectconfig.AuthenticationNone: staticComponent{name: "authentication:none"},
			projectconfig.AuthenticationJWT: staticComponent{
				name: "authentication:jwt",
				files: files("authentication:jwt",
					"v04/languages/python/components/auth/jwt/auth/__init__.py.tmpl", "auth/__init__.py",
					"v04/languages/python/components/auth/jwt/auth/jwt.py.tmpl", "auth/jwt.py",
					"v04/languages/python/components/auth/jwt/auth/middleware.py.tmpl", "auth/middleware.py",
				),
				dependencies: []Dependency{
					{Module: "python-jose[cryptography]", Version: ">=3.3.0", Reason: "JWT handling"},
					{Module: "passlib[bcrypt]", Version: ">=1.7.4", Reason: "Password hashing"},
				},
			},
		},
		documentation: map[projectconfig.Documentation]Component{
			projectconfig.DocumentationNone: staticComponent{name: "documentation:none"},
			projectconfig.DocumentationSwagger: staticComponent{
				name:  "documentation:swagger",
				files: files("documentation:swagger", "v04/languages/python/base/requirements.txt.tmpl", "requirements.txt"),
			},
		},
		tests: map[projectconfig.TestStrategy]Component{
			projectconfig.TestStrategyUnit: staticComponent{
				name:  "tests:unit",
				files: files("tests:unit",
					"v04/languages/python/components/tests/unit/conftest.py.tmpl", "tests/conftest.py",
					"v04/languages/python/components/tests/unit/test_health.py.tmpl", "tests/test_health.py",
				),
				dependencies: []Dependency{
					{Module: "pytest", Version: ">=7.4.4", Reason: "Test framework"},
					{Module: "pytest-asyncio", Version: ">=0.23.0", Reason: "Async test support"},
					{Module: "httpx", Version: ">=0.26.0", Reason: "HTTP client for testing"},
				},
			},
			projectconfig.TestStrategyUnitIntegration: staticComponent{
				name:  "tests:unit-integration",
				files: files("tests:unit-integration",
					"v04/languages/python/components/tests/unit/conftest.py.tmpl", "tests/conftest.py",
					"v04/languages/python/components/tests/unit/test_health.py.tmpl", "tests/test_health.py",
				),
				dependencies: []Dependency{
					{Module: "pytest", Version: ">=7.4.4", Reason: "Test framework"},
					{Module: "pytest-asyncio", Version: ">=0.23.0", Reason: "Async test support"},
					{Module: "httpx", Version: ">=0.26.0", Reason: "HTTP client for testing"},
				},
			},
		},
		ci: map[projectconfig.CIStrategy]Component{
			projectconfig.CIStrategyNone: staticComponent{name: "ci:none"},
			projectconfig.CIStrategyGitHub: staticComponent{
				name:  "ci:github-actions",
				files: files("ci:github-actions", "v04/components/ci/github-actions/ci.yml.tmpl", ".github/workflows/ci.yml"),
			},
		},
	}
}

// Build validates configuration, resolves each selected component once, and
// rejects duplicate destinations before anything is written to disk.
func (r Registry) Build(configuration projectconfig.ProjectConfig) (Plan, error) {
	if err := configuration.Validate(); err != nil {
		return Plan{}, err
	}

	components := []Component{
		r.base,
		r.architectures[configuration.Architecture],
		r.databases[configuration.Database],
		r.authentication[configuration.Authentication],
		r.documentation[configuration.Documentation],
		r.tests[configuration.Tests],
		r.ci[configuration.CI],
	}
	if configuration.Docker {
		components = append(components, r.docker)
	} else {
		components = append(components, staticComponent{name: "docker:none"})
	}

	plan := Plan{Configuration: configuration}
	seenFiles := make(map[string]string)
	seenDependencies := make(map[string]struct{})
	for _, component := range components {
		if component == nil {
			return Plan{}, fmt.Errorf("no component registered for selected configuration")
		}
		contribution, err := component.Contribute(configuration)
		if err != nil {
			return Plan{}, fmt.Errorf("contribute %s: %w", component.Name(), err)
		}
		plan.Components = append(plan.Components, component.Name())
		for _, file := range contribution.Files {
			if previous, exists := seenFiles[file.Destination]; exists {
				return Plan{}, fmt.Errorf("duplicate destination %q contributed by %s and %s", file.Destination, previous, component.Name())
			}
			seenFiles[file.Destination] = component.Name()
			plan.Files = append(plan.Files, file)
		}
		for _, dependency := range contribution.Dependencies {
			if _, exists := seenDependencies[dependency.Module]; exists {
				continue
			}
			seenDependencies[dependency.Module] = struct{}{}
			plan.Dependencies = append(plan.Dependencies, dependency)
		}
	}

	sort.Slice(plan.Files, func(i, j int) bool { return plan.Files[i].Destination < plan.Files[j].Destination })
	sort.Slice(plan.Dependencies, func(i, j int) bool { return plan.Dependencies[i].Module < plan.Dependencies[j].Module })
	return plan, nil
}

type staticComponent struct {
	name         string
	files        []File
	dependencies []Dependency
}

func (c staticComponent) Name() string { return c.name }

func (c staticComponent) Contribute(_ projectconfig.ProjectConfig) (Contribution, error) {
	return Contribution{Files: c.files, Dependencies: c.dependencies}, nil
}

type architectureComponent struct {
	name  string
	files []File
}

func (c architectureComponent) Name() string { return c.name }

func (c architectureComponent) Contribute(_ projectconfig.ProjectConfig) (Contribution, error) {
	return Contribution{Files: c.files}, nil
}

type databaseComponent struct {
	name         string
	database     projectconfig.Database
	dependencies []Dependency
}

func (c databaseComponent) Name() string { return c.name }

func (c databaseComponent) Contribute(_ projectconfig.ProjectConfig) (Contribution, error) {
	root := "v04/components/database/" + string(c.database)
	return Contribution{
		Files: files(c.name,
			root+"/database.go.tmpl", "internal/infrastructure/"+string(c.database)+"/database.go",
			root+"/migration.up.sql.tmpl", "migrations/000001_init.up.sql",
			root+"/migration.down.sql.tmpl", "migrations/000001_init.down.sql",
		),
		Dependencies: c.dependencies,
	}, nil
}

type dockerComponent struct{}

func (dockerComponent) Name() string { return "docker" }

func (dockerComponent) Contribute(configuration projectconfig.ProjectConfig) (Contribution, error) {
	root := "v04/components/docker/" + string(configuration.Database)
	return Contribution{Files: files("docker",
		"v04/components/docker/Dockerfile.tmpl", "docker/Dockerfile",
		root+"/docker-compose.yml.tmpl", "docker/docker-compose.yml",
	)}, nil
}

func files(component string, paths ...string) []File {
	if len(paths)%2 != 0 {
		panic("template path and destination must be paired")
	}
	result := make([]File, 0, len(paths)/2)
	for i := 0; i < len(paths); i += 2 {
		result = append(result, File{Template: paths[i], Destination: paths[i+1], Component: component})
	}
	return result
}
