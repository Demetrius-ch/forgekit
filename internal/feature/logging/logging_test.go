package logging

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Demetrius-ch/forgekit/internal/feature"
)

func createTestProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	goMod := `module github.com/test/test-api

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.12
)
`
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	// Initialize the module and download dependencies
	cmd := exec.Command("go", "mod", "download")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		t.Fatalf("go mod download failed: %v", err)
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

func NewRouter(logger *slog.Logger, pool interface{}, userRepo interface{}) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	return r
}
`
	if err := os.WriteFile(filepath.Join(routerDir, "router.go"), []byte(routerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	mainDir := filepath.Join(root, "cmd", "server")
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mainContent := `package main

import (
	"log/slog"
	"net/http"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	_ = logger
	_ = http.NewServeMux()
}
`
	if err := os.WriteFile(filepath.Join(mainDir, "main.go"), []byte(mainContent), 0o644); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestLoggingFeatureCheck(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
	}

	feat := LoggingFeature{}

	err := feat.Check(context.Background(), project)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
}

func TestLoggingFeatureCheckAlreadyInstalled(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
	}

	feat := LoggingFeature{}

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

func TestLoggingFeaturePlan(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
	}

	feat := LoggingFeature{}

	plan, err := feat.Plan(context.Background(), project)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if plan.Feature != "logging" {
		t.Fatalf("Expected feature 'logging', got %s", plan.Feature)
	}
	if plan.Version != "1.0.0" {
		t.Fatalf("Expected version '1.0.0', got %s", plan.Version)
	}
	if len(plan.Files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(plan.Files))
	}
	if len(plan.Dependencies) != 0 {
		t.Fatalf("Expected 0 dependencies, got %d", len(plan.Dependencies))
	}
	if len(plan.Environment) != 2 {
		t.Fatalf("Expected 2 env vars, got %d", len(plan.Environment))
	}
	if plan.Environment[0] != "LOG_LEVEL=info" {
		t.Fatalf("Unexpected env var: %s", plan.Environment[0])
	}
	if plan.Environment[1] != "LOG_FORMAT=text" {
		t.Fatalf("Unexpected env var: %s", plan.Environment[1])
	}
}

func TestLoggingFeatureApply(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
	}

	feat := LoggingFeature{}

	plan, err := feat.Plan(context.Background(), project)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	err = feat.Apply(context.Background(), project, plan)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	loggerPath := filepath.Join(root, "internal", "logging", "logger.go")
	middlewarePath := filepath.Join(root, "internal", "logging", "middleware.go")

	if _, err := os.Stat(loggerPath); err != nil {
		t.Fatalf("logger.go not created: %v", err)
	}
	if _, err := os.Stat(middlewarePath); err != nil {
		t.Fatalf("middleware.go not created: %v", err)
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
	if !contains(string(content), "LOG_LEVEL") {
		t.Fatal("LOG_LEVEL not found in .env.example")
	}
	if !contains(string(content), "LOG_FORMAT") {
		t.Fatal("LOG_FORMAT not found in .env.example")
	}
}

func TestLoggingFeatureIdempotent(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
	}

	feat := LoggingFeature{}

	plan, err := feat.Plan(context.Background(), project)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	err = feat.Apply(context.Background(), project, plan)
	if err != nil {
		t.Fatalf("First Apply failed: %v", err)
	}

	err = feat.Apply(context.Background(), project, plan)
	if err != nil {
		t.Fatalf("Second Apply failed: %v", err)
	}

	loggerPath := filepath.Join(root, "internal", "logging", "logger.go")
	if _, err := os.Stat(loggerPath); err != nil {
		t.Fatalf("logger.go not created: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
