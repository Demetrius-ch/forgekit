package swagger

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Demetrius-ch/forgekit/internal/feature"
)

func createTestProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	// Create go.mod
	goMod := `module github.com/test/test-api

go 1.22

require github.com/go-chi/chi/v5 v5.0.12
`
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create .env.example
	envExample := `POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=test_api
PORT=8080
`
	if err := os.WriteFile(filepath.Join(root, ".env.example"), []byte(envExample), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create router.go (matching ForgeKit template pattern for swagger integration)
	routerDir := filepath.Join(root, "internal", "transport", "http")
	if err := os.MkdirAll(routerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	routerContent := `package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(logger *slog.Logger, healthSvc interface{}) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", handler.NewHealth(logger, healthSvc).ServeHTTP)

	return r
}
`
	if err := os.WriteFile(filepath.Join(routerDir, "router.go"), []byte(routerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create main.go
	cmdDir := filepath.Join(root, "cmd", "server")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mainContent := `package main

import (
	"log/slog"
	"net/http"

	"github.com/test/test-api/internal/transport/http"
)

func main() {
	logger := slog.Default()
	router := http.NewRouter(logger)
	http.ListenAndServe(":8080", router)
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create forge.yaml to mark as ForgeKit project
	forgeYaml := `project: test-api
module: github.com/test/test-api
`
	if err := os.WriteFile(filepath.Join(root, "forge.yaml"), []byte(forgeYaml), 0o644); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestSwaggerFeatureCheck(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
	}

	feat := SwaggerFeature{}

	// First check should pass
	err := feat.Check(context.Background(), project)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
}

func TestSwaggerFeatureCheckAlreadyInstalled(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
	}

	feat := SwaggerFeature{}

	// Install once
	plan, err := feat.Plan(context.Background(), project)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	err = feat.Apply(context.Background(), project, plan)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Second check should fail
	err = feat.Check(context.Background(), project)
	if err == nil {
		t.Fatal("Expected error for already installed feature")
	}
}

func TestSwaggerFeaturePlan(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
	}

	feat := SwaggerFeature{}

	plan, err := feat.Plan(context.Background(), project)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if plan.Feature != "swagger" {
		t.Fatalf("Expected feature 'swagger', got %s", plan.Feature)
	}
	if plan.Version != "1.0.1" {
		t.Fatalf("Expected version '1.0.1', got %s", plan.Version)
	}
	if len(plan.Files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(plan.Files))
	}
	if len(plan.Dependencies) != 2 {
		t.Fatalf("Expected 2 dependencies, got %d", len(plan.Dependencies))
	}
	if plan.Dependencies[0].Module != "github.com/swaggo/http-swagger" {
		t.Fatalf("Expected swaggo/http-swagger dependency, got %s", plan.Dependencies[0].Module)
	}
	if plan.Dependencies[1].Module != "gopkg.in/yaml.v3" {
		t.Fatalf("Expected gopkg.in/yaml.v3 dependency, got %s", plan.Dependencies[1].Module)
	}
	if len(plan.Environment) != 1 {
		t.Fatalf("Expected 1 env var, got %d", len(plan.Environment))
	}
	if plan.Environment[0] != "SWAGGER_ENABLED=true" {
		t.Fatalf("Unexpected env var: %s", plan.Environment[0])
	}
}

func TestSwaggerFeatureApply(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
	}

	feat := SwaggerFeature{}

	plan, err := feat.Plan(context.Background(), project)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	err = feat.Apply(context.Background(), project, plan)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify files were created
	swaggerGoPath := filepath.Join(root, "internal", "swagger", "swagger.go")
	openapiPath := filepath.Join(root, "internal", "swagger", "openapi.yaml")

	if _, err := os.Stat(swaggerGoPath); err != nil {
		t.Fatalf("swagger.go not created: %v", err)
	}
	if _, err := os.Stat(openapiPath); err != nil {
		t.Fatalf("openapi.yaml not created: %v", err)
	}

	// Verify .forge/features.yaml was created
	featuresPath := filepath.Join(root, ".forge", "features.yaml")
	if _, err := os.Stat(featuresPath); err != nil {
		t.Fatalf("features.yaml not created: %v", err)
	}

	// Verify SWAGGER_ENABLED was added to .env.example
	envPath := filepath.Join(root, ".env.example")
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read .env.example: %v", err)
	}
	if !contains(string(content), "SWAGGER_ENABLED") {
		t.Fatal("SWAGGER_ENABLED not found in .env.example")
	}

	// Verify openapi.yaml contains expected endpoints
	openapiContent, err := os.ReadFile(openapiPath)
	if err != nil {
		t.Fatalf("Failed to read openapi.yaml: %v", err)
	}
	if !contains(string(openapiContent), "/health") {
		t.Fatal("/health endpoint not found in openapi.yaml")
	}
	if !contains(string(openapiContent), "/users") {
		t.Fatal("/users endpoint not found in openapi.yaml")
	}
	if !contains(string(openapiContent), "bearerAuth") {
		t.Fatal("bearerAuth security scheme not found in openapi.yaml")
	}

	// Verify go.mod has the dependency
	goModPath := filepath.Join(root, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}
	if !contains(string(goModContent), "github.com/swaggo/http-swagger") {
		t.Fatal("Swagger dependency not found in go.mod")
	}

	// Verify main.go was modified with swagger import
	mainPath := filepath.Join(root, "cmd", "server", "main.go")
	mainContent, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}
	if !contains(string(mainContent), "internal/swagger") {
		t.Fatal("Swagger import not found in main.go")
	}

	// Verify router.go was modified with swagger route
	routerPath := filepath.Join(root, "internal", "transport", "http", "router.go")
	routerContent, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatalf("Failed to read router.go: %v", err)
	}
	if !contains(string(routerContent), "swagger.RegisterRoutes") {
		t.Fatal("Swagger RegisterRoutes not found in router.go")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
