package auth

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

	// Create router.go
	routerDir := filepath.Join(root, "internal", "transport", "http")
	if err := os.MkdirAll(routerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	routerContent := `package http

import (
	"net/http"
)

func NewRouter() http.Handler {
	return http.NewServeMux()
}
`
	if err := os.WriteFile(filepath.Join(routerDir, "router.go"), []byte(routerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestAuthFeatureCheck(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
	}

	feat := AuthFeature{}

	// First check should pass
	err := feat.Check(context.Background(), project)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
}

func TestAuthFeatureCheckAlreadyInstalled(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
	}

	feat := AuthFeature{}

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

func TestAuthFeaturePlan(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
	}

	feat := AuthFeature{}

	plan, err := feat.Plan(context.Background(), project)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if plan.Feature != "auth" {
		t.Fatalf("Expected feature 'auth', got %s", plan.Feature)
	}
	if plan.Version != "1.0.0" {
		t.Fatalf("Expected version '1.0.0', got %s", plan.Version)
	}
	if len(plan.Files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(plan.Files))
	}
	if len(plan.Dependencies) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(plan.Dependencies))
	}
	if plan.Dependencies[0].Module != "github.com/golang-jwt/jwt/v5" {
		t.Fatalf("Expected jwt dependency, got %s", plan.Dependencies[0].Module)
	}
	if len(plan.Environment) != 1 {
		t.Fatalf("Expected 1 env var, got %d", len(plan.Environment))
	}
	if plan.Environment[0] != "JWT_SECRET=your-secret-key-change-in-production" {
		t.Fatalf("Unexpected env var: %s", plan.Environment[0])
	}
}

func TestAuthFeatureApply(t *testing.T) {
	root := createTestProject(t)

	project := feature.ProjectContext{
		Root:      root,
		Module:    "github.com/test/test-api",
		GoVersion: "1.22",
	}

	feat := AuthFeature{}

	plan, err := feat.Plan(context.Background(), project)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	err = feat.Apply(context.Background(), project, plan)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify files were created
	jwtPath := filepath.Join(root, "internal", "auth", "jwt.go")
	middlewarePath := filepath.Join(root, "internal", "auth", "middleware.go")

	if _, err := os.Stat(jwtPath); err != nil {
		t.Fatalf("jwt.go not created: %v", err)
	}
	if _, err := os.Stat(middlewarePath); err != nil {
		t.Fatalf("middleware.go not created: %v", err)
	}

	// Verify .forge/features.yaml was created
	featuresPath := filepath.Join(root, ".forge", "features.yaml")
	if _, err := os.Stat(featuresPath); err != nil {
		t.Fatalf("features.yaml not created: %v", err)
	}

	// Verify JWT_SECRET was added to .env.example
	envPath := filepath.Join(root, ".env.example")
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read .env.example: %v", err)
	}
	if !contains(string(content), "JWT_SECRET") {
		t.Fatal("JWT_SECRET not found in .env.example")
	}

	// Verify go.mod has the dependency
	goModPath := filepath.Join(root, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}
	if !contains(string(goModContent), "github.com/golang-jwt/jwt/v5") {
		t.Fatal("JWT dependency not found in go.mod")
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
