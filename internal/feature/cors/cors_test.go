package cors

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

	goMod := `module github.com/test/test-api

go 1.22

require github.com/go-chi/chi/v5 v5.0.12
`
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	envExample := `POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=test_api
PORT=8080
`
	if err := os.WriteFile(filepath.Join(root, ".env.example"), []byte(envExample), 0o644); err != nil {
		t.Fatal(err)
	}

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

	forgeYaml := `project: test-api
module: github.com/test/test-api
`
	if err := os.WriteFile(filepath.Join(root, "forge.yaml"), []byte(forgeYaml), 0o644); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestCorsFeatureCheck(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
		HTTPPort:  8080,
	}

	feat := CorsFeature{}

	err := feat.Check(context.Background(), project)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
}

func TestCorsFeatureCheckAlreadyInstalled(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
		HTTPPort:  8080,
	}

	feat := CorsFeature{}

	plan, err := feat.Plan(context.Background(), project)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	err = feat.Apply(context.Background(), project, plan)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	err = feat.Check(context.Background(), project)
	if err == nil {
		t.Fatal("Expected error for already installed feature")
	}
}

func TestCorsFeaturePlan(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
		HTTPPort:  8080,
	}

	feat := CorsFeature{}

	plan, err := feat.Plan(context.Background(), project)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if plan.Feature != "cors" {
		t.Fatalf("Expected feature 'cors', got %s", plan.Feature)
	}
	if plan.Version != "1.0.2" {
		t.Fatalf("Expected version '1.0.2', got %s", plan.Version)
	}
	if len(plan.Files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(plan.Files))
	}
	if len(plan.Dependencies) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(plan.Dependencies))
	}
	if plan.Dependencies[0].Module != "github.com/rs/cors" {
		t.Fatalf("Expected rs/cors dependency, got %s", plan.Dependencies[0].Module)
	}
	if len(plan.Environment) != 5 {
		t.Fatalf("Expected 5 env vars, got %d", len(plan.Environment))
	}
}

func TestCorsFeatureApply(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
		HTTPPort:  8080,
	}

	feat := CorsFeature{}

	plan, err := feat.Plan(context.Background(), project)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	err = feat.Apply(context.Background(), project, plan)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	corsGoPath := filepath.Join(root, "internal", "cors", "cors.go")

	if _, err := os.Stat(corsGoPath); err != nil {
		t.Fatalf("cors.go not created: %v", err)
	}

	featuresPath := filepath.Join(root, ".forge", "features.yaml")
	if _, err := os.Stat(featuresPath); err != nil {
		t.Fatalf("features.yaml not created: %v", err)
	}

	envPath := filepath.Join(root, ".env.example")
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read .env.example: %v", err)
	}
	envContent := string(content)
	requiredEnv := []string{
		"CORS_ENABLED",
		"CORS_ALLOWED_ORIGINS",
		"CORS_ALLOWED_METHODS",
		"CORS_ALLOWED_HEADERS",
		"CORS_ALLOW_CREDENTIALS",
	}
	for _, env := range requiredEnv {
		if !contains(envContent, env) {
			t.Fatalf("%s not found in .env.example", env)
		}
	}

	goModPath := filepath.Join(root, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}
	if !contains(string(goModContent), "github.com/rs/cors") {
		t.Fatal("CORS dependency not found in go.mod")
	}

	routerPath := filepath.Join(root, "internal", "transport", "http", "router.go")
	routerContent, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatalf("Failed to read router.go: %v", err)
	}
	if !contains(string(routerContent), "cors.Middleware") {
		t.Fatal("CORS middleware not found in router.go")
	}
	if !contains(string(routerContent), "internal/cors") {
		t.Fatal("CORS import not found in router.go")
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
