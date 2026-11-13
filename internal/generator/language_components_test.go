package generator_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Demetrius-ch/forgekit/internal/generator"
	"github.com/Demetrius-ch/forgekit/internal/projectconfig"
)

// generateOnly renders a project without npm/pip installs or post-processing,
// so that the generated files can be inspected quickly.
func generateOnly(t *testing.T, module string, cfg projectconfig.ProjectConfig) (string, error) {
	t.Helper()
	target := filepath.Join(t.TempDir(), cfg.Name)
	gen, err := generator.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = gen.Init(generator.InitOptions{
		ProjectName:      cfg.Name,
		ModulePath:       module,
		HTTPPort:         8080,
		PostgresHostPort: 5433,
		DatabaseName:     strings.ReplaceAll(cfg.Name, "-", "_"),
		TargetDir:        target,
		ProjectConfig:    cfg,
		SkipNpmInstall:   true,
		SkipPostprocess:  true,
	})
	return target, err
}

func readGenerated(t *testing.T, target, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(target, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

func languageConfig(name string, language projectconfig.Language) projectconfig.ProjectConfig {
	cfg := projectconfig.Default(name, "")
	cfg.Language = language
	if language == projectconfig.LanguageTypeScript || language == projectconfig.LanguageJavaScript {
		cfg.Runtime = projectconfig.RuntimeNode
	}
	return cfg
}

// Python imports jose/passlib (JWT) and pytest/httpx (tests): they must be
// installable from requirements.txt, along with the selected DB driver.
func TestPythonRequirementsMatchImports(t *testing.T) {
	cfg := languageConfig("py-req", projectconfig.LanguagePython)
	cfg.ModulePath = "py-req"
	cfg.Database = projectconfig.DatabaseMySQL
	cfg.Authentication = projectconfig.AuthenticationJWT
	cfg.Documentation = projectconfig.DocumentationSwagger

	target, err := generateOnly(t, "py-req", cfg)
	if err != nil {
		t.Fatalf("Init() error = %v (a non-Go package name and swagger must be accepted for Python)", err)
	}

	requirements := readGenerated(t, target, "requirements.txt")
	for _, dep := range []string{"python-jose[cryptography]", "passlib[bcrypt]", "pymysql", "pytest", "httpx"} {
		if !strings.Contains(requirements, dep) {
			t.Errorf("requirements.txt is missing %q:\n%s", dep, requirements)
		}
	}
	if strings.Contains(requirements, "psycopg") {
		t.Errorf("PostgreSQL driver must not be installed for MySQL:\n%s", requirements)
	}

	settings := readGenerated(t, target, "settings.py")
	if !strings.Contains(settings, "mysql+pymysql://forgekit:forgekit@localhost:5433/py_req") {
		t.Errorf("settings.py DATABASE_URL must match the installed driver and docker-compose:\n%s", settings)
	}
	compose := readGenerated(t, target, "docker/docker-compose.yml")
	if !strings.Contains(compose, `DATABASE_URL: "mysql+pymysql://forgekit:forgekit@mysql:3306/py_req"`) {
		t.Errorf("docker-compose DATABASE_URL must use the SQLAlchemy URL format:\n%s", compose)
	}
	if !strings.Contains(readGenerated(t, target, "main.py"), "port=settings.PORT") {
		t.Error("main.py must listen on the configured PORT")
	}
}

func TestNodeJWTDependencyDeclared(t *testing.T) {
	for _, language := range []projectconfig.Language{projectconfig.LanguageTypeScript, projectconfig.LanguageJavaScript} {
		t.Run(string(language), func(t *testing.T) {
			cfg := languageConfig("node-jwt", language)
			cfg.ModulePath = "node-jwt"
			cfg.Authentication = projectconfig.AuthenticationJWT

			target, err := generateOnly(t, "node-jwt", cfg)
			if err != nil {
				t.Fatalf("Init() error = %v", err)
			}

			var pkg struct {
				Dependencies    map[string]string `json:"dependencies"`
				DevDependencies map[string]string `json:"devDependencies"`
			}
			if err := json.Unmarshal([]byte(readGenerated(t, target, "package.json")), &pkg); err != nil {
				t.Fatalf("package.json is not valid JSON: %v", err)
			}
			if _, ok := pkg.Dependencies["jsonwebtoken"]; !ok {
				t.Error("jsonwebtoken is imported by src/auth but missing from dependencies")
			}
			if language == projectconfig.LanguageTypeScript {
				if _, ok := pkg.DevDependencies["@types/jsonwebtoken"]; !ok {
					t.Error("@types/jsonwebtoken is required to type-check src/auth/jwt.ts")
				}
			}
		})
	}
}

func TestNodeSwaggerRejectedWithClearError(t *testing.T) {
	cfg := languageConfig("ts-swagger", projectconfig.LanguageTypeScript)
	cfg.ModulePath = "ts-swagger"
	cfg.Documentation = projectconfig.DocumentationSwagger

	_, err := generateOnly(t, "ts-swagger", cfg)
	if err == nil || !strings.Contains(err.Error(), "swagger is not supported yet for typescript") {
		t.Fatalf("expected an explicit unsupported-swagger error, got %v", err)
	}
}

func TestCIWorkflowMatchesLanguage(t *testing.T) {
	cases := map[projectconfig.Language][]string{
		projectconfig.LanguageGo:         {"actions/setup-go", "go test ./..."},
		projectconfig.LanguagePython:     {"actions/setup-python", "pip install -r requirements.txt", "python -m pytest"},
		projectconfig.LanguageTypeScript: {"actions/setup-node", "npm test", "npm run build"},
		projectconfig.LanguageJavaScript: {"actions/setup-node", "npm test"},
	}
	for language, want := range cases {
		t.Run(string(language), func(t *testing.T) {
			cfg := languageConfig("ci-"+string(language), language)
			module := "example.com/test/ci"
			cfg.ModulePath = module
			cfg.CI = projectconfig.CIStrategyGitHub
			if language == projectconfig.LanguageGo {
				cfg.Database = projectconfig.DatabaseNone
			}

			target, err := generateOnly(t, module, cfg)
			if err != nil {
				t.Fatalf("Init() error = %v", err)
			}
			workflow := readGenerated(t, target, ".github/workflows/ci.yml")
			for _, step := range want {
				if !strings.Contains(workflow, step) {
					t.Errorf("ci.yml is missing %q:\n%s", step, workflow)
				}
			}
			if language != projectconfig.LanguageGo && strings.Contains(workflow, "go test") {
				t.Errorf("non-Go project must not run Go commands in CI:\n%s", workflow)
			}
		})
	}
}

func TestTypeScriptDockerRunsCompiledOutput(t *testing.T) {
	cfg := languageConfig("ts-docker", projectconfig.LanguageTypeScript)
	cfg.ModulePath = "ts-docker"

	target, err := generateOnly(t, "ts-docker", cfg)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	dockerfile := readGenerated(t, target, "docker/Dockerfile")
	if !strings.Contains(dockerfile, "npm run build") || !strings.Contains(dockerfile, `CMD ["node", "dist/index.js"]`) {
		t.Errorf("TypeScript image must build and run dist/index.js:\n%s", dockerfile)
	}
	compose := readGenerated(t, target, "docker/docker-compose.yml")
	if !strings.Contains(compose, `PORT: "8080"`) {
		t.Errorf("the api container must listen on the published port 8080:\n%s", compose)
	}
}
