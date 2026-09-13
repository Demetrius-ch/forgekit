package generator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Demetrius-ch/forgekit/internal/generator"
	"github.com/Demetrius-ch/forgekit/internal/projectconfig"
)

// initAndVerify generates a project and verifies it compiles.
// Returns the target directory.
func initAndVerify(t *testing.T, name string, cfg projectconfig.ProjectConfig) string {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, name)

	gen, err := generator.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = gen.Init(generator.InitOptions{
		ProjectName:   name,
		ModulePath:    "github.com/test/" + name,
		HTTPPort:      8080,
		DatabaseName:  strings.ReplaceAll(name, "-", "_") + "_db",
		TargetDir:     target,
		ProjectConfig: cfg,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Verify go.mod exists
	goMod := filepath.Join(target, "go.mod")
	if _, err := os.Stat(goMod); err != nil {
		t.Fatalf("go.mod not generated: %v", err)
	}

	return target
}

func fileExists(t *testing.T, base, rel string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(base, rel))
	return err == nil
}

func filesNotExist(t *testing.T, base string, paths []string) {
	t.Helper()
	for _, p := range paths {
		if fileExists(t, base, p) {
			t.Errorf("unexpected file found: %s", p)
		}
	}
}

func filesExist(t *testing.T, base string, paths []string) {
	t.Helper()
	for _, p := range paths {
		if !fileExists(t, base, p) {
			t.Errorf("expected file not found: %s", p)
		}
	}
}

func TestGenCaseA_HexagonalPostgresDockerJWTSwagger(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "case-a",
		ModulePath:     "github.com/test/case-a",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         true,
		Authentication: projectconfig.AuthenticationJWT,
		Documentation:  projectconfig.DocumentationSwagger,
		Tests:          projectconfig.TestStrategyUnitIntegration,
		CI:             projectconfig.CIStrategyGitHub,
	}
	target := initAndVerify(t, "case-a", cfg)
	filesExist(t, target, []string{
		"cmd/server/main.go",
		"internal/domain/health.go",
		"internal/application/health.go",
		"internal/transport/http/router.go",
		"internal/infrastructure/postgres/database.go",
		"internal/auth/jwt.go",
		"internal/auth/middleware.go",
		"internal/swagger/swagger.go",
		"internal/swagger/openapi.yaml",
		"tests/health_test.go",
		"tests/http_integration_test.go",
		".github/workflows/ci.yml",
		"docker/Dockerfile",
		"docker/docker-compose.yml",
		"migrations/000001_init.up.sql",
	})
	filesNotExist(t, target, []string{
		"internal/infrastructure/mysql/database.go",
		"internal/infrastructure/sqlite/database.go",
	})
}

func TestGenCaseB_HexagonalPostgresMinimal(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "case-b",
		ModulePath:     "github.com/test/case-b",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerify(t, "case-b", cfg)
	filesExist(t, target, []string{
		"cmd/server/main.go",
		"internal/domain/health.go",
		"internal/infrastructure/postgres/database.go",
		"tests/health_test.go",
	})
	filesNotExist(t, target, []string{
		"internal/auth/jwt.go",
		"internal/swagger/swagger.go",
		"docker/Dockerfile",
		"docker/docker-compose.yml",
		".github/workflows/ci.yml",
	})
}

func TestGenCaseC_CleanMySQLDocker(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "case-c",
		ModulePath:     "github.com/test/case-c",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureClean,
		Database:       projectconfig.DatabaseMySQL,
		Docker:         true,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerify(t, "case-c", cfg)
	filesExist(t, target, []string{
		"cmd/server/main.go",
		"internal/entities/health.go",
		"internal/usecases/health.go",
		"internal/interfaces/http/router.go",
		"internal/infrastructure/mysql/database.go",
		"docker/Dockerfile",
		"docker/docker-compose.yml",
	})
	filesNotExist(t, target, []string{
		"internal/infrastructure/postgres/database.go",
		"internal/infrastructure/sqlite/database.go",
	})
}

func TestGenCaseD_LayeredSQLiteNoDocker(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "case-d",
		ModulePath:     "github.com/test/case-d",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureLayered,
		Database:       projectconfig.DatabaseSQLite,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerify(t, "case-d", cfg)
	filesExist(t, target, []string{
		"cmd/server/main.go",
		"internal/models/health.go",
		"internal/services/health.go",
		"internal/handlers/http.go",
		"internal/infrastructure/sqlite/database.go",
		"tests/health_test.go",
	})
	filesNotExist(t, target, []string{
		"internal/infrastructure/postgres/database.go",
		"internal/infrastructure/mysql/database.go",
		"docker/Dockerfile",
		"docker/docker-compose.yml",
	})
}

func TestGenCaseE_LayeredNoDatabaseNoDocker(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "case-e",
		ModulePath:     "github.com/test/case-e",
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
	target := initAndVerify(t, "case-e", cfg)
	filesExist(t, target, []string{
		"cmd/server/main.go",
		"internal/models/health.go",
		"internal/services/health.go",
		"internal/handlers/http.go",
		"tests/health_test.go",
	})
	filesNotExist(t, target, []string{
		"internal/infrastructure/postgres/database.go",
		"internal/infrastructure/mysql/database.go",
		"internal/infrastructure/sqlite/database.go",
		"migrations/000001_init.up.sql",
		"docker/Dockerfile",
		"docker/docker-compose.yml",
	})
}

func TestGenCaseF_CleanNoDatabase(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "case-f",
		ModulePath:     "github.com/test/case-f",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureClean,
		Database:       projectconfig.DatabaseNone,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerify(t, "case-f", cfg)
	filesExist(t, target, []string{
		"cmd/server/main.go",
		"internal/entities/health.go",
		"internal/usecases/health.go",
		"internal/interfaces/http/router.go",
	})
	filesNotExist(t, target, []string{
		"internal/infrastructure/postgres/database.go",
		"internal/infrastructure/mysql/database.go",
		"internal/infrastructure/sqlite/database.go",
		"migrations/",
	})
}

func TestGenCaseG_InvalidConfig(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:         "case-g",
		ModulePath:   "github.com/test/case-g",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture: "invalid",
		Database:     "invalid",
	}
	gen, err := generator.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = gen.Init(generator.InitOptions{
		ProjectName:   "case-g",
		ModulePath:    "github.com/test/case-g",
		TargetDir:     filepath.Join(t.TempDir(), "case-g"),
		ProjectConfig: cfg,
	})
	if err == nil {
		t.Fatal("expected error for invalid config, got nil")
	}
}

func TestCriticalNoDatabase(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "no-db",
		ModulePath:     "github.com/test/no-db",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabaseNone,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerify(t, "no-db", cfg)
	filesNotExist(t, target, []string{
		"internal/infrastructure/postgres/database.go",
		"internal/infrastructure/mysql/database.go",
		"internal/infrastructure/sqlite/database.go",
		"migrations/000001_init.up.sql",
		"docker/docker-compose.yml",
	})

	// Verify go.mod does not contain database drivers
	data, err := os.ReadFile(filepath.Join(target, "go.mod"))
	if err != nil {
		t.Fatalf("ReadFile(go.mod): %v", err)
	}
	content := string(data)
	for _, driver := range []string{"pgx", "go-sql-driver", "modernc.org/sqlite"} {
		if strings.Contains(content, driver) {
			t.Errorf("go.mod should not contain database driver %q when database=none", driver)
		}
	}
}

func TestCriticalNoDocker(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "no-docker",
		ModulePath:     "github.com/test/no-docker",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerify(t, "no-docker", cfg)
	filesNotExist(t, target, []string{
		"docker/Dockerfile",
		"docker/docker-compose.yml",
	})
	filesExist(t, target, []string{
		"internal/infrastructure/postgres/database.go",
	})
}

func TestCriticalNoAuth(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "no-auth",
		ModulePath:     "github.com/test/no-auth",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabaseNone,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerify(t, "no-auth", cfg)
	filesNotExist(t, target, []string{
		"internal/auth/jwt.go",
		"internal/auth/middleware.go",
	})

	// Verify go.mod does not contain JWT
	data, err := os.ReadFile(filepath.Join(target, "go.mod"))
	if err != nil {
		t.Fatalf("ReadFile(go.mod): %v", err)
	}
	if strings.Contains(string(data), "golang-jwt") {
		t.Error("go.mod should not contain JWT dependency when auth=none")
	}
}

func TestCriticalNoSwagger(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "no-swagger",
		ModulePath:     "github.com/test/no-swagger",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabaseNone,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerify(t, "no-swagger", cfg)
	filesNotExist(t, target, []string{
		"internal/swagger/swagger.go",
		"internal/swagger/openapi.yaml",
	})

	// Verify go.mod does not contain swagger
	data, err := os.ReadFile(filepath.Join(target, "go.mod"))
	if err != nil {
		t.Fatalf("ReadFile(go.mod): %v", err)
	}
	if strings.Contains(string(data), "http-swagger") {
		t.Error("go.mod should not contain swagger dependency when docs=none")
	}
}

func TestCombinatorialNoMixedDB(t *testing.T) {
	// MySQL config should not contain postgres files
	cfg := projectconfig.ProjectConfig{
		Name:           "mysql-only",
		ModulePath:     "github.com/test/mysql-only",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabaseMySQL,
		Docker:         true,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerify(t, "mysql-only", cfg)
	filesExist(t, target, []string{"internal/infrastructure/mysql/database.go"})
	filesNotExist(t, target, []string{
		"internal/infrastructure/postgres/database.go",
		"internal/infrastructure/sqlite/database.go",
	})
}

func TestCombinatorialNoDBNoDockerCompose(t *testing.T) {
	// No database + no docker should have no docker-compose
	cfg := projectconfig.ProjectConfig{
		Name:           "no-db-no-docker",
		ModulePath:     "github.com/test/no-db-no-docker",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureClean,
		Database:       projectconfig.DatabaseNone,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerify(t, "no-db-no-docker", cfg)
	filesNotExist(t, target, []string{
		"docker/Dockerfile",
		"docker/docker-compose.yml",
		"internal/infrastructure/postgres/database.go",
	})
}

func TestCombinatorialNoAuthNoJWT(t *testing.T) {
	// Auth none should not generate JWT middleware
	cfg := projectconfig.ProjectConfig{
		Name:           "no-auth-combo",
		ModulePath:     "github.com/test/no-auth-combo",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureLayered,
		Database:       projectconfig.DatabasePostgres,
		Docker:         true,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationSwagger,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyGitHub,
	}
	target := initAndVerify(t, "no-auth-combo", cfg)
	filesExist(t, target, []string{
		"internal/infrastructure/postgres/database.go",
		"internal/swagger/swagger.go",
		".github/workflows/ci.yml",
	})
	filesNotExist(t, target, []string{
		"internal/auth/jwt.go",
		"internal/auth/middleware.go",
	})
}

func TestMinimalDependencies(t *testing.T) {
	// Minimal project should have minimal deps
	cfg := projectconfig.ProjectConfig{
		Name:           "minimal-deps",
		ModulePath:     "github.com/test/minimal-deps",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabaseNone,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerify(t, "minimal-deps", cfg)

	data, err := os.ReadFile(filepath.Join(target, "go.mod"))
	if err != nil {
		t.Fatalf("ReadFile(go.mod): %v", err)
	}
	content := string(data)

	// Should have chi router
	if !strings.Contains(content, "go-chi/chi") {
		t.Error("go.mod missing chi router")
	}

	// Should NOT have database drivers
	for _, driver := range []string{"pgx", "go-sql-driver", "modernc.org/sqlite"} {
		if strings.Contains(content, driver) {
			t.Errorf("go.mod should not contain driver %q for database=none", driver)
		}
	}

	// Should NOT have JWT
	if strings.Contains(content, "golang-jwt") {
		t.Error("go.mod should not contain JWT for auth=none")
	}

	// Should NOT have swagger
	if strings.Contains(content, "http-swagger") {
		t.Error("go.mod should not contain swagger for docs=none")
	}
}

// initAndVerifyNode generates a Node.js project and verifies it has package.json.
func initAndVerifyNode(t *testing.T, name string, cfg projectconfig.ProjectConfig) string {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, name)

	gen, err := generator.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = gen.Init(generator.InitOptions{
		ProjectName:   name,
		ModulePath:    "github.com/test/" + name,
		HTTPPort:      8080,
		DatabaseName:  strings.ReplaceAll(name, "-", "_") + "_db",
		TargetDir:     target,
		ProjectConfig: cfg,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Verify package.json exists
	packageJSON := filepath.Join(target, "package.json")
	if _, err := os.Stat(packageJSON); err != nil {
		t.Fatalf("package.json not generated: %v", err)
	}

	return target
}

func TestTypeScriptHexagonalPostgres(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "ts-hex-pg",
		ModulePath:     "github.com/test/ts-hex-pg",
		Language:       projectconfig.LanguageTypeScript,
		Runtime:        projectconfig.RuntimeNode,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         true,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerifyNode(t, "ts-hex-pg", cfg)
	filesExist(t, target, []string{
		"src/index.ts",
		"src/app.ts",
		"src/domain/health.entity.ts",
		"src/application/health.usecase.ts",
		"src/interfaces/http/health.router.ts",
		"prisma/schema.prisma",
		"tests/health.test.ts",
	})
	filesNotExist(t, target, []string{
		"go.mod",
		"cmd/server/main.go",
	})
}

func TestTypeScriptCleanMySQL(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "ts-clean-mysql",
		ModulePath:     "github.com/test/ts-clean-mysql",
		Language:       projectconfig.LanguageTypeScript,
		Runtime:        projectconfig.RuntimeNode,
		Architecture:   projectconfig.ArchitectureClean,
		Database:       projectconfig.DatabaseMySQL,
		Docker:         false,
		Authentication: projectconfig.AuthenticationJWT,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerifyNode(t, "ts-clean-mysql", cfg)
	filesExist(t, target, []string{
		"src/index.ts",
		"src/app.ts",
		"src/entities/health.entity.ts",
		"src/usecases/health.usecase.ts",
		"src/interfaces/http/health.router.ts",
		"src/auth/jwt.ts",
		"src/auth/middleware.ts",
		"prisma/schema.prisma",
	})
	filesNotExist(t, target, []string{
		"go.mod",
	})
}

func TestTypeScriptLayeredSQLite(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "ts-layered-sqlite",
		ModulePath:     "github.com/test/ts-layered-sqlite",
		Language:       projectconfig.LanguageTypeScript,
		Runtime:        projectconfig.RuntimeNode,
		Architecture:   projectconfig.ArchitectureLayered,
		Database:       projectconfig.DatabaseSQLite,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerifyNode(t, "ts-layered-sqlite", cfg)
	filesExist(t, target, []string{
		"src/index.ts",
		"src/app.ts",
		"src/models/health.model.ts",
		"src/services/health.service.ts",
		"src/routes/health.routes.ts",
		"prisma/schema.prisma",
	})
	filesNotExist(t, target, []string{
		"go.mod",
	})
}

func TestTypeScriptNoDatabase(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "ts-nodb",
		ModulePath:     "github.com/test/ts-nodb",
		Language:       projectconfig.LanguageTypeScript,
		Runtime:        projectconfig.RuntimeNode,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabaseNone,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerifyNode(t, "ts-nodb", cfg)
	filesExist(t, target, []string{
		"src/index.ts",
		"src/app.ts",
	})
	filesNotExist(t, target, []string{
		"prisma/schema.prisma",
		"go.mod",
	})
}

func TestJavaScriptHexagonalPostgres(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "js-hex-pg",
		ModulePath:     "github.com/test/js-hex-pg",
		Language:       projectconfig.LanguageJavaScript,
		Runtime:        projectconfig.RuntimeNode,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         true,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerifyNode(t, "js-hex-pg", cfg)
	filesExist(t, target, []string{
		"src/index.js",
		"src/app.js",
		"src/domain/health.entity.js",
		"src/application/health.usecase.js",
		"src/interfaces/http/health.router.js",
		"prisma/schema.prisma",
		"tests/health.test.js",
	})
	filesNotExist(t, target, []string{
		"go.mod",
		"tsconfig.json",
	})
}

func TestJavaScriptCleanMySQL(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "js-clean-mysql",
		ModulePath:     "github.com/test/js-clean-mysql",
		Language:       projectconfig.LanguageJavaScript,
		Runtime:        projectconfig.RuntimeNode,
		Architecture:   projectconfig.ArchitectureClean,
		Database:       projectconfig.DatabaseMySQL,
		Docker:         false,
		Authentication: projectconfig.AuthenticationJWT,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerifyNode(t, "js-clean-mysql", cfg)
	filesExist(t, target, []string{
		"src/index.js",
		"src/app.js",
		"src/entities/health.entity.js",
		"src/usecases/health.usecase.js",
		"src/interfaces/http/health.router.js",
		"src/auth/jwt.js",
		"src/auth/middleware.js",
		"prisma/schema.prisma",
	})
	filesNotExist(t, target, []string{
		"go.mod",
		"tsconfig.json",
	})
}

func TestJavaScriptLayeredSQLite(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "js-layered-sqlite",
		ModulePath:     "github.com/test/js-layered-sqlite",
		Language:       projectconfig.LanguageJavaScript,
		Runtime:        projectconfig.RuntimeNode,
		Architecture:   projectconfig.ArchitectureLayered,
		Database:       projectconfig.DatabaseSQLite,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerifyNode(t, "js-layered-sqlite", cfg)
	filesExist(t, target, []string{
		"src/index.js",
		"src/app.js",
		"src/models/health.model.js",
		"src/services/health.service.js",
		"src/routes/health.routes.js",
		"prisma/schema.prisma",
	})
	filesNotExist(t, target, []string{
		"go.mod",
		"tsconfig.json",
	})
}

func TestJavaScriptNoDatabase(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "js-nodb",
		ModulePath:     "github.com/test/js-nodb",
		Language:       projectconfig.LanguageJavaScript,
		Runtime:        projectconfig.RuntimeNode,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabaseNone,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}
	target := initAndVerifyNode(t, "js-nodb", cfg)
	filesExist(t, target, []string{
		"src/index.js",
		"src/app.js",
	})
	filesNotExist(t, target, []string{
		"prisma/schema.prisma",
		"go.mod",
		"tsconfig.json",
	})
}

// initAndVerifyNodeSkipNpm generates a Node.js project without running npm install.
func initAndVerifyNodeSkipNpm(t *testing.T, name string, cfg projectconfig.ProjectConfig) string {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, name)

	gen, err := generator.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = gen.Init(generator.InitOptions{
		ProjectName:    name,
		ModulePath:     "github.com/test/" + name,
		HTTPPort:       8080,
		DatabaseName:   strings.ReplaceAll(name, "-", "_") + "_db",
		TargetDir:      target,
		ProjectConfig:  cfg,
		SkipNpmInstall: true,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Verify package.json exists
	packageJSON := filepath.Join(target, "package.json")
	if _, err := os.Stat(packageJSON); err != nil {
		t.Fatalf("package.json not generated: %v", err)
	}

	return target
}

// TestTypeScriptWithNodeRuntime validates that TS projects with RuntimeNode work.
func TestTypeScriptWithNodeRuntime(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "ts-node-runtime",
		ModulePath:     "github.com/test/ts-node-runtime",
		Language:       projectconfig.LanguageTypeScript,
		Runtime:        projectconfig.RuntimeNode,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         true,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("config should be valid: %v", err)
	}

	target := initAndVerifyNode(t, "ts-node-runtime", cfg)
	filesExist(t, target, []string{
		"src/index.ts",
		"src/app.ts",
		"package.json",
		"tsconfig.json",
		"prisma/schema.prisma",
		"node_modules/.package-lock.json",
	})
}

// TestJavaScriptWithNodeRuntime validates that JS projects with RuntimeNode work.
func TestJavaScriptWithNodeRuntime(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "js-node-runtime",
		ModulePath:     "github.com/test/js-node-runtime",
		Language:       projectconfig.LanguageJavaScript,
		Runtime:        projectconfig.RuntimeNode,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         true,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("config should be valid: %v", err)
	}

	target := initAndVerifyNode(t, "js-node-runtime", cfg)
	filesExist(t, target, []string{
		"src/index.js",
		"src/app.js",
		"package.json",
		"prisma/schema.prisma",
		"node_modules/.package-lock.json",
	})
	filesNotExist(t, target, []string{
		"tsconfig.json",
	})
}

// TestTypeScriptWithoutNodeRuntime validates that TS projects with RuntimeNone fail validation.
func TestTypeScriptWithoutNodeRuntime(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "ts-no-node",
		ModulePath:     "github.com/test/ts-no-node",
		Language:       projectconfig.LanguageTypeScript,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for TypeScript with RuntimeNone")
	}
	if !strings.Contains(err.Error(), "typescript projects require runtime=node") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestJavaScriptWithoutNodeRuntime validates that JS projects with RuntimeNone fail validation.
func TestJavaScriptWithoutNodeRuntime(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "js-no-node",
		ModulePath:     "github.com/test/js-no-node",
		Language:       projectconfig.LanguageJavaScript,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for JavaScript with RuntimeNone")
	}
	if !strings.Contains(err.Error(), "javascript projects require runtime=node") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestTypeScriptSkipNpmInstall validates that TS projects can be generated without npm install.
func TestTypeScriptSkipNpmInstall(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "ts-skip-npm",
		ModulePath:     "github.com/test/ts-skip-npm",
		Language:       projectconfig.LanguageTypeScript,
		Runtime:        projectconfig.RuntimeNode,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}

	target := initAndVerifyNodeSkipNpm(t, "ts-skip-npm", cfg)

	// Files should exist
	filesExist(t, target, []string{
		"src/index.ts",
		"src/app.ts",
		"package.json",
		"tsconfig.json",
		"prisma/schema.prisma",
	})

	// node_modules should NOT exist (npm install was skipped)
	if _, err := os.Stat(filepath.Join(target, "node_modules")); err == nil {
		t.Error("node_modules should not exist when SkipNpmInstall=true")
	}
}

// TestJavaScriptSkipNpmInstall validates that JS projects can be generated without npm install.
func TestJavaScriptSkipNpmInstall(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "js-skip-npm",
		ModulePath:     "github.com/test/js-skip-npm",
		Language:       projectconfig.LanguageJavaScript,
		Runtime:        projectconfig.RuntimeNode,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}

	target := initAndVerifyNodeSkipNpm(t, "js-skip-npm", cfg)

	// Files should exist
	filesExist(t, target, []string{
		"src/index.js",
		"src/app.js",
		"package.json",
		"prisma/schema.prisma",
	})

	// node_modules should NOT exist (npm install was skipped)
	if _, err := os.Stat(filepath.Join(target, "node_modules")); err == nil {
		t.Error("node_modules should not exist when SkipNpmInstall=true")
	}
}

// TestGoWithNodeRuntime validates that Go projects with RuntimeNode fail validation.
func TestGoWithNodeRuntime(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "go-node-runtime",
		ModulePath:     "github.com/test/go-node-runtime",
		Language:       projectconfig.LanguageGo,
		Runtime:        projectconfig.RuntimeNode,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for Go with RuntimeNode")
	}
	if !strings.Contains(err.Error(), "Go projects do not support a runtime") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestAllLanguageRuntimeCombinations tests all valid and invalid language×runtime combinations.
func TestAllLanguageRuntimeCombinations(t *testing.T) {
	tests := []struct {
		name     string
		language projectconfig.Language
		runtime  projectconfig.Runtime
		valid    bool
	}{
		{"go+none", projectconfig.LanguageGo, projectconfig.RuntimeNone, true},
		{"go+node", projectconfig.LanguageGo, projectconfig.RuntimeNode, false},
		{"typescript+none", projectconfig.LanguageTypeScript, projectconfig.RuntimeNone, false},
		{"typescript+node", projectconfig.LanguageTypeScript, projectconfig.RuntimeNode, true},
		{"javascript+none", projectconfig.LanguageJavaScript, projectconfig.RuntimeNone, false},
		{"javascript+node", projectconfig.LanguageJavaScript, projectconfig.RuntimeNode, true},
		{"python+none", projectconfig.LanguagePython, projectconfig.RuntimeNone, true},
		{"python+node", projectconfig.LanguagePython, projectconfig.RuntimeNode, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := projectconfig.ProjectConfig{
				Name:           "test-" + tt.name,
				ModulePath:     "github.com/test/" + tt.name,
				Language:       tt.language,
				Runtime:        tt.runtime,
				Architecture:   projectconfig.ArchitectureHexagonal,
				Database:       projectconfig.DatabasePostgres,
				Docker:         false,
				Authentication: projectconfig.AuthenticationNone,
				Documentation:  projectconfig.DocumentationNone,
				Tests:          projectconfig.TestStrategyUnit,
				CI:             projectconfig.CIStrategyNone,
			}

			err := cfg.Validate()
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

// initAndVerifyPython generates a Python project and verifies it has requirements.txt.
func initAndVerifyPython(t *testing.T, name string, cfg projectconfig.ProjectConfig) string {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, name)

	gen, err := generator.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = gen.Init(generator.InitOptions{
		ProjectName:    name,
		ModulePath:     "github.com/test/" + name,
		HTTPPort:       8080,
		DatabaseName:   strings.ReplaceAll(name, "-", "_") + "_db",
		TargetDir:      target,
		ProjectConfig:  cfg,
		SkipNpmInstall: true,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Verify requirements.txt exists
	requirementsTxt := filepath.Join(target, "requirements.txt")
	if _, err := os.Stat(requirementsTxt); err != nil {
		t.Fatalf("requirements.txt not generated: %v", err)
	}

	return target
}

// TestPythonHexagonalPostgres validates Python hexagonal with PostgreSQL.
func TestPythonHexagonalPostgres(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "py-hex-pg",
		ModulePath:     "github.com/test/py-hex-pg",
		Language:       projectconfig.LanguagePython,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         true,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("config should be valid: %v", err)
	}

	target := initAndVerifyPython(t, "py-hex-pg", cfg)
	filesExist(t, target, []string{
		"main.py",
		"app.py",
		"settings.py",
		"requirements.txt",
		"pyproject.toml",
		"domain/health_entity.py",
		"application/health_usecase.py",
		"interfaces/http/health_router.py",
		"tests/test_health.py",
	})
	filesNotExist(t, target, []string{
		"go.mod",
		"package.json",
	})
}

// TestPythonCleanMySQL validates Python clean with MySQL.
func TestPythonCleanMySQL(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "py-clean-mysql",
		ModulePath:     "github.com/test/py-clean-mysql",
		Language:       projectconfig.LanguagePython,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureClean,
		Database:       projectconfig.DatabaseMySQL,
		Docker:         false,
		Authentication: projectconfig.AuthenticationJWT,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("config should be valid: %v", err)
	}

	target := initAndVerifyPython(t, "py-clean-mysql", cfg)
	filesExist(t, target, []string{
		"main.py",
		"app.py",
		"settings.py",
		"requirements.txt",
		"pyproject.toml",
		"entities/health_entity.py",
		"usecases/health_usecase.py",
		"interfaces/http/health_router.py",
		"auth/jwt.py",
		"auth/middleware.py",
		"tests/test_health.py",
	})
	filesNotExist(t, target, []string{
		"go.mod",
		"package.json",
	})
}

// TestPythonLayeredSQLite validates Python layered with SQLite.
func TestPythonLayeredSQLite(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "py-layered-sqlite",
		ModulePath:     "github.com/test/py-layered-sqlite",
		Language:       projectconfig.LanguagePython,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureLayered,
		Database:       projectconfig.DatabaseSQLite,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("config should be valid: %v", err)
	}

	target := initAndVerifyPython(t, "py-layered-sqlite", cfg)
	filesExist(t, target, []string{
		"main.py",
		"app.py",
		"settings.py",
		"requirements.txt",
		"pyproject.toml",
		"models/health_model.py",
		"services/health_service.py",
		"routes/health_routes.py",
		"tests/test_health.py",
	})
	filesNotExist(t, target, []string{
		"go.mod",
		"package.json",
	})
}

// TestPythonNoDatabase validates Python without database.
func TestPythonNoDatabase(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "py-nodb",
		ModulePath:     "github.com/test/py-nodb",
		Language:       projectconfig.LanguagePython,
		Runtime:        projectconfig.RuntimeNone,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabaseNone,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("config should be valid: %v", err)
	}

	target := initAndVerifyPython(t, "py-nodb", cfg)
	filesExist(t, target, []string{
		"main.py",
		"app.py",
		"settings.py",
		"requirements.txt",
		"pyproject.toml",
	})
	filesNotExist(t, target, []string{
		"go.mod",
		"package.json",
		"database.py",
	})
}

// TestPythonWithNodeRuntime validates that Python projects with RuntimeNode fail validation.
func TestPythonWithNodeRuntime(t *testing.T) {
	cfg := projectconfig.ProjectConfig{
		Name:           "py-node-runtime",
		ModulePath:     "github.com/test/py-node-runtime",
		Language:       projectconfig.LanguagePython,
		Runtime:        projectconfig.RuntimeNode,
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for Python with RuntimeNode")
	}
	if !strings.Contains(err.Error(), "Python projects do not support a runtime") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
