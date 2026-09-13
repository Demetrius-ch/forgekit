package generationplan

import (
	"strings"
	"testing"

	"github.com/Demetrius-ch/forgekit/internal/projectconfig"
)

func TestBuildMinimalLayeredProject(t *testing.T) {
	t.Parallel()

	configuration := projectconfig.ProjectConfig{
		Name:           "minimal-api",
		ModulePath:     "github.com/example/minimal-api",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureLayered,
		Database:       projectconfig.DatabaseNone,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}

	plan, err := DefaultRegistry().Build(configuration)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	assertComponent(t, plan, "architecture:layered")
	assertComponent(t, plan, "database:none")
	assertComponent(t, plan, "docker:none")
	assertComponent(t, plan, "authentication:none")
	assertComponent(t, plan, "documentation:none")
	assertAbsentDestination(t, plan, "docker/Dockerfile")
	assertAbsentPrefix(t, plan, "migrations/")
	assertAbsentPrefix(t, plan, "internal/infrastructure/postgres/")
	assertDependency(t, plan, "github.com/go-chi/chi/v5")
	assertNoDependency(t, plan, "github.com/jackc/pgx/v5")
}

func TestBuildCleanMySQLProject(t *testing.T) {
	t.Parallel()

	configuration := projectconfig.ProjectConfig{
		Name:           "clean-api",
		ModulePath:     "github.com/example/clean-api",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureClean,
		Database:       projectconfig.DatabaseMySQL,
		Docker:         true,
		Authentication: projectconfig.AuthenticationJWT,
		Documentation:  projectconfig.DocumentationSwagger,
		Tests:          projectconfig.TestStrategyUnitIntegration,
		CI:             projectconfig.CIStrategyGitHub,
	}

	plan, err := DefaultRegistry().Build(configuration)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	for _, component := range []string{
		"architecture:clean", "database:mysql", "docker", "authentication:jwt",
		"documentation:swagger", "tests:unit-integration", "ci:github-actions",
	} {
		assertComponent(t, plan, component)
	}
	for _, destination := range []string{
		"cmd/server/main.go", "docker/Dockerfile", "docker/docker-compose.yml",
		"internal/auth/jwt.go", "internal/swagger/openapi.yaml", ".github/workflows/ci.yml",
		"migrations/000001_init.up.sql", "tests/http_integration_test.go",
	} {
		assertDestination(t, plan, destination)
	}
	assertDependency(t, plan, "github.com/go-sql-driver/mysql")
	assertDependency(t, plan, "github.com/golang-jwt/jwt/v5")
	assertDependency(t, plan, "github.com/swaggo/http-swagger")
	assertNoDependency(t, plan, "github.com/jackc/pgx/v5")
}

func TestBuildRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	configuration := projectconfig.Default("bad-api", "github.com/example/bad-api")
	configuration.Database = "mongo"

	_, err := DefaultRegistry().Build(configuration)
	if err == nil || !strings.Contains(err.Error(), "invalid database") {
		t.Fatalf("Build() error = %v, want invalid database", err)
	}
}

func assertComponent(t *testing.T, plan Plan, component string) {
	t.Helper()
	for _, current := range plan.Components {
		if current == component {
			return
		}
	}
	t.Fatalf("components %#v do not include %q", plan.Components, component)
}

func assertDestination(t *testing.T, plan Plan, destination string) {
	t.Helper()
	for _, file := range plan.Files {
		if file.Destination == destination {
			return
		}
	}
	t.Fatalf("plan does not include %q", destination)
}

func assertAbsentDestination(t *testing.T, plan Plan, destination string) {
	t.Helper()
	for _, file := range plan.Files {
		if file.Destination == destination {
			t.Fatalf("plan unexpectedly includes %q", destination)
		}
	}
}

func assertAbsentPrefix(t *testing.T, plan Plan, prefix string) {
	t.Helper()
	for _, file := range plan.Files {
		if strings.HasPrefix(file.Destination, prefix) {
			t.Fatalf("plan unexpectedly includes %q", file.Destination)
		}
	}
}

func assertDependency(t *testing.T, plan Plan, module string) {
	t.Helper()
	for _, dependency := range plan.Dependencies {
		if dependency.Module == module {
			return
		}
	}
	t.Fatalf("dependencies %#v do not include %q", plan.Dependencies, module)
}

func assertNoDependency(t *testing.T, plan Plan, module string) {
	t.Helper()
	for _, dependency := range plan.Dependencies {
		if dependency.Module == module {
			t.Fatalf("dependencies unexpectedly include %q", module)
		}
	}
}
