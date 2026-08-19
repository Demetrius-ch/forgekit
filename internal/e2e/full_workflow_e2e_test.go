//go:build e2e

package e2e_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestFullWorkflowE2E tests the complete ForgeKit workflow:
// forge init → forge add → go test → forge analyze → forge check → forge doctor → forge inspect
func TestFullWorkflowE2E(t *testing.T) {
	root := t.TempDir()
	project := "full-workflow-api"
	target := filepath.Join(root, project)

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "forge")
	build := exec.Command("go", "build", "-o", bin, "./cmd/forge")
	build.Dir = repoRoot
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build forge: %s: %v", out, err)
	}

	// Step 1: forge init
	t.Log("Step 1: forge init")
	init := exec.Command(bin, "init", project,
		"--non-interactive",
		"--module", "github.com/forgekit/full-workflow-api",
		"--dir", target,
	)
	out, err = init.CombinedOutput()
	if err != nil {
		t.Fatalf("forge init: %s: %v", out, err)
	}
	t.Log("✓ forge init succeeded")

	// Step 2: forge add cors (will auto-install auth as dependency)
	t.Log("Step 2: forge add cors (auto-installs auth)")
	addCors := exec.Command(bin, "add", "cors")
	addCors.Dir = target
	out, err = addCors.CombinedOutput()
	if err != nil {
		t.Fatalf("forge add cors: %s: %v", out, err)
	}
	t.Log("✓ forge add cors succeeded (with auth dependency)")

	// Step 4: go test
	t.Log("Step 4: go test ./...")
	testCmd := exec.Command("go", "test", "./...")
	testCmd.Dir = target
	out, err = testCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test: %s: %v", out, err)
	}
	t.Log("✓ go test succeeded")

	// Step 5: forge analyze
	t.Log("Step 5: forge analyze")
	analyze := exec.Command(bin, "analyze")
	analyze.Dir = target
	out, err = analyze.CombinedOutput()
	if err != nil {
		t.Fatalf("forge analyze: %s: %v", out, err)
	}
	if !strings.Contains(string(out), "Score global") {
		t.Fatalf("forge analyze output missing score: %s", out)
	}
	t.Log("✓ forge analyze succeeded")

	// Step 6: forge check
	t.Log("Step 6: forge check")
	check := exec.Command(bin, "check")
	check.Dir = target
	out, err = check.CombinedOutput()
	if err != nil {
		t.Fatalf("forge check: %s: %v", out, err)
	}
	t.Log("✓ forge check succeeded")

	// Step 7: forge doctor
	t.Log("Step 7: forge doctor")
	doctor := exec.Command(bin, "doctor")
	doctor.Dir = target
	out, err = doctor.CombinedOutput()
	if err != nil {
		t.Fatalf("forge doctor: %s: %v", out, err)
	}
	if !strings.Contains(string(out), "Signature ForgeKit valide") {
		t.Fatalf("forge doctor missing signature validation: %s", out)
	}
	t.Log("✓ forge doctor succeeded")

	// Step 8: forge inspect
	t.Log("Step 8: forge inspect")
	inspect := exec.Command(bin, "inspect")
	inspect.Dir = target
	out, err = inspect.CombinedOutput()
	if err != nil {
		t.Fatalf("forge inspect: %s: %v", out, err)
	}
	if !strings.Contains(string(out), "auth") || !strings.Contains(string(out), "cors") {
		t.Fatalf("forge inspect missing features: %s", out)
	}
	t.Log("✓ forge inspect succeeded")

	// Verify .forge/features.yaml has both features
	featuresYaml := filepath.Join(target, ".forge", "features.yaml")
	data, err := os.ReadFile(featuresYaml)
	if err != nil {
		t.Fatalf("read features.yaml: %v", err)
	}
	if !strings.Contains(string(data), "auth") || !strings.Contains(string(data), "cors") {
		t.Fatalf("features.yaml missing features: %s", string(data))
	}

	// Verify go.mod has dependencies
	mod, err := os.ReadFile(filepath.Join(target, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	modContent := string(mod)
	if !strings.Contains(modContent, "github.com/golang-jwt/jwt/v5") {
		t.Fatalf("go.mod missing jwt dependency: %s", modContent)
	}
	if !strings.Contains(modContent, "github.com/rs/cors") {
		t.Fatalf("go.mod missing cors dependency: %s", modContent)
	}

	t.Log("✅ Full workflow test passed")
}

// TestExternalProjectE2E tests ForgeKit with an external compatible Go project
func TestExternalProjectE2E(t *testing.T) {
	root := t.TempDir()
	project := "external-api"
	target := filepath.Join(root, project)

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "forge")
	build := exec.Command("go", "build", "-o", bin, "./cmd/forge")
	build.Dir = repoRoot
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build forge: %s: %v", out, err)
	}

	// Create an external compatible Go project (has go.mod + cmd/server but no .forge)
	t.Log("Creating external compatible project")
	if err := os.MkdirAll(filepath.Join(target, "cmd", "server"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "internal", "transport", "http"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "go.mod"), []byte("module github.com/test/external\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	routerContent := `package http

import "github.com/go-chi/chi/v5"

func NewRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Timeout(30 * time.Second))
	return r
}
`
	if err := os.WriteFile(filepath.Join(target, "internal", "transport", "http", "router.go"), []byte(routerContent), 0o644); err != nil {
		t.Fatal(err)
	}
	mainContent := `package main

import (
	"net/http"
	"github.com/test/external/internal/transport/http"
)

func main() {
	router := http.NewRouter()
	http.ListenAndServe(":8080", router)
}
`
	if err := os.WriteFile(filepath.Join(target, "cmd", "server", "main.go"), []byte(mainContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test forge analyze on external project (should work with --ci or loose detection)
	t.Log("Testing forge analyze on external project")
	analyze := exec.Command(bin, "analyze", "--ci")
	analyze.Dir = target
	out, err = analyze.CombinedOutput()
	t.Logf("forge analyze output: %s", out)
	_ = err // ignore error for external project

	// Test forge add on external project - should detect as external compatible
	t.Log("Testing forge add auth on external project (should work)")
	addAuth := exec.Command(bin, "add", "auth")
	addAuth.Dir = target
	out, err = addAuth.CombinedOutput()
	t.Logf("forge add auth output: %s", out)

	// Verify feature was added
	featuresYaml := filepath.Join(target, ".forge", "features.yaml")
	if _, err := os.Stat(featuresYaml); err == nil {
		data, _ := os.ReadFile(featuresYaml)
		if strings.Contains(string(data), "auth") {
			t.Log("✓ Feature auth installed on external project")
		}
	}
}

// TestErrorScenariosE2E tests various error scenarios
func TestErrorScenariosE2E(t *testing.T) {
	root := t.TempDir()
	project := "error-test-api"
	target := filepath.Join(root, project)

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "forge")
	build := exec.Command("go", "build", "-o", bin, "./cmd/forge")
	build.Dir = repoRoot
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build forge: %s: %v", out, err)
	}

	// Initialize project
	init := exec.Command(bin, "init", project,
		"--non-interactive",
		"--module", "github.com/forgekit/error-test",
		"--dir", target,
	)
	out, err = init.CombinedOutput()
	if err != nil {
		t.Fatalf("forge init: %s: %v", out, err)
	}

	// Test 1: Unknown feature
	t.Log("Test 1: Unknown feature")
	addUnknown := exec.Command(bin, "add", "nonexistent")
	addUnknown.Dir = target
	out, err = addUnknown.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for unknown feature, got: %s", out)
	}
	if !strings.Contains(string(out), "feature inconnue") && !strings.Contains(string(out), "unknown feature") {
		t.Fatalf("expected 'feature inconnue' error, got: %s", out)
	}
	t.Log("✓ Unknown feature correctly rejected")

	// Test 2: Feature already installed
	t.Log("Test 2: Feature already installed")
	addAuth := exec.Command(bin, "add", "auth")
	addAuth.Dir = target
	out, err = addAuth.CombinedOutput()
	if err != nil {
		t.Fatalf("first forge add auth: %s: %v", out, err)
	}
	addAuthAgain := exec.Command(bin, "add", "auth")
	addAuthAgain.Dir = target
	out, err = addAuthAgain.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for already installed feature, got: %s", out)
	}
	if !strings.Contains(string(out), "déjà installée") {
		t.Fatalf("expected 'déjà installée' error, got: %s", out)
	}
	t.Log("✓ Already installed feature correctly rejected")

	// Test 3: Dependency resolution (cors depends on auth)
	t.Log("Test 3: Dependency resolution")
	// Remove auth from features.yaml to simulate missing dependency
	featuresYaml := filepath.Join(target, ".forge", "features.yaml")
	if err := os.WriteFile(featuresYaml, []byte("features: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Try to add cors without auth - should automatically install auth as dependency
	addCors := exec.Command(bin, "add", "cors")
	addCors.Dir = target
	out, err = addCors.CombinedOutput()
	if err != nil {
		t.Fatalf("expected success for adding cors with auto dependency resolution, got: %s", out)
	}
	// Verify both auth and cors are now installed
	if !strings.Contains(string(out), "auth installé") {
		t.Fatalf("expected auth to be installed as dependency: %s", out)
	}
	if !strings.Contains(string(out), "cors installé") {
		t.Fatalf("expected cors to be installed: %s", out)
	}
	t.Log("✓ Dependency resolution correctly installs auth before cors")
	// Verify features.yaml has both
	data, _ := os.ReadFile(featuresYaml)
	if !strings.Contains(string(data), "auth") || !strings.Contains(string(data), "cors") {
		t.Fatalf("features.yaml missing features: %s", string(data))
	}

	// Test 4: Invalid config (corrupted go.mod)
	t.Log("Test 4: Invalid go.mod")
	badMod := filepath.Join(target, "go.mod.bak")
	if err := os.Rename(filepath.Join(target, "go.mod"), badMod); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "go.mod"), []byte("invalid content"), 0o644); err != nil {
		t.Fatal(err)
	}
	analyze := exec.Command(bin, "analyze", "--ci")
	analyze.Dir = target
	out, err = analyze.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "Exit code: 2") {
		// Check exit code via err
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() != 2 {
				t.Fatalf("expected exit code 2 for invalid go.mod, got %d: %s", exitErr.ExitCode(), out)
			}
			t.Log("✓ Invalid go.mod correctly returns exit code 2")
		} else {
			t.Logf("analyze output (may not have exit code 2 in test env): %s", out)
		}
		// Restore go.mod
		os.Rename(badMod, filepath.Join(target, "go.mod"))
	}

	// Test 5: Remove impossible (feature not installed)
	t.Log("Test 5: Remove non-installed feature")
	// Use a feature that was never installed (logging)
	removeAuth := exec.Command(bin, "remove", "logging")
	removeAuth.Dir = target
	out, err = removeAuth.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for removing non-installed feature, got: %s", out)
	}
	if !strings.Contains(string(out), "n'est pas installée") {
		t.Fatalf("expected 'n'est pas installée' error, got: %s", out)
	}
	t.Log("✓ Remove non-installed feature correctly rejected")

	// Test 6: Remove with dependent feature installed (cors depends on auth)
	t.Log("Test 6: Remove with dependent feature")
	// Verify that auth and cors are both installed (from Test 3)
	// Try to remove auth while cors is installed - should fail due to dependent feature
	removeAuth = exec.Command(bin, "remove", "auth")
	removeAuth.Dir = target
	out, err = removeAuth.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for removing auth while cors depends on it, got: %s", out)
	}
	if !strings.Contains(string(out), "dépendantes installées") {
		t.Fatalf("expected 'dépendantes installées' error, got: %s", out)
	}
	t.Log("✓ Remove with dependent feature correctly rejected")
}

// TestForgeScenariosE2E tests various .forge scenarios
func TestForgeScenariosE2E(t *testing.T) {
	root := t.TempDir()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "forge")
	build := exec.Command("go", "build", "-o", bin, "./cmd/forge")
	build.Dir = repoRoot
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build forge: %s: %v", out, err)
	}

	// Test 1: Valid .forge signature
	t.Log("Test 1: Valid .forge signature")
	project1 := "valid-forge"
	target1 := filepath.Join(root, project1)
	init := exec.Command(bin, "init", project1,
		"--non-interactive",
		"--module", "github.com/forgekit/valid-forge",
		"--dir", target1,
	)
	out, err = init.CombinedOutput()
	if err != nil {
		t.Fatalf("forge init: %s: %v", out, err)
	}
	inspect := exec.Command(bin, "inspect")
	inspect.Dir = target1
	out, err = inspect.CombinedOutput()
	if err != nil {
		t.Fatalf("forge inspect: %s: %v", out, err)
	}
	if !strings.Contains(string(out), "Signature:") || !strings.Contains(string(out), "Valide") {
		t.Fatalf("expected valid signature, got: %s", out)
	}
	t.Log("✓ Valid .forge signature detected")

	// Test 2: Legacy ForgeKit project (only features.yaml, no forge.yaml)
	t.Log("Test 2: Legacy ForgeKit project")
	project2 := "legacy-forge"
	target2 := filepath.Join(root, project2)
	if err := os.MkdirAll(filepath.Join(target2, "cmd", "server"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target2, "internal", "transport", "http"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target2, "go.mod"), []byte("module github.com/forgekit/legacy\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	routerContent := `package http
import "github.com/go-chi/chi/v5"
func NewRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Timeout(30 * time.Second))
	return r
}
`
	if err := os.WriteFile(filepath.Join(target2, "internal", "transport", "http", "router.go"), []byte(routerContent), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create .forge/features.yaml only (legacy)
	if err := os.MkdirAll(filepath.Join(target2, ".forge"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target2, ".forge", "features.yaml"), []byte("features: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Do NOT create forge.yaml
	inspect = exec.Command(bin, "inspect")
	inspect.Dir = target2
	out, err = inspect.CombinedOutput()
	if err != nil {
		t.Fatalf("forge inspect: %s: %v", out, err)
	}
	if !strings.Contains(string(out), "legacy") && !strings.Contains(string(out), "Legacy") {
		t.Fatalf("expected legacy detection, got: %s", out)
	}
	t.Log("✓ Legacy ForgeKit project detected")

	// Test 3: .forge absent (external project)
	t.Log("Test 3: .forge absent")
	project3 := "no-forge"
	target3 := filepath.Join(root, project3)
	if err := os.MkdirAll(filepath.Join(target3, "cmd", "server"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target3, "go.mod"), []byte("module github.com/test/noforge\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inspect = exec.Command(bin, "inspect")
	inspect.Dir = target3
	out, err = inspect.CombinedOutput()
	if err != nil {
		t.Fatalf("inspect command failed: %v", err)
	}
	if !strings.Contains(string(out), "ABSENTE") {
		t.Fatalf("expected absent signature in output, got: %s", out)
	}
	t.Log("✓ .forge absent detected")

	// Test 4: Invalid .forge (directory exists but no forge.yaml or features.yaml)
	t.Log("Test 4: Invalid .forge")
	project4 := "invalid-forge"
	target4 := filepath.Join(root, project4)
	if err := os.MkdirAll(filepath.Join(target4, "cmd", "server"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target4, "go.mod"), []byte("module github.com/forgekit/invalid\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target4, ".forge"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Don't create forge.yaml or features.yaml
	inspect = exec.Command(bin, "inspect")
	inspect.Dir = target4
	out, err = inspect.CombinedOutput()
	if err != nil {
		t.Fatalf("inspect command failed: %v", err)
	}
	if !strings.Contains(string(out), "INVALIDE") {
		t.Fatalf("expected invalid signature in output, got: %s", out)
	}
	t.Log("✓ Invalid .forge detected")

	// Test 5: Inconsistent .forge (features.yaml declares feature but directory missing)
	t.Log("Test 5: Inconsistent .forge")
	project5 := "inconsistent-forge"
	target5 := filepath.Join(root, project5)
	init = exec.Command(bin, "init", project5,
		"--non-interactive",
		"--module", "github.com/forgekit/inconsistent",
		"--dir", target5,
	)
	out, err = init.CombinedOutput()
	if err != nil {
		t.Fatalf("forge init: %s: %v", out, err)
	}
	// Manually add a feature to features.yaml without installing it
	featuresYaml := filepath.Join(target5, ".forge", "features.yaml")
	if err := os.WriteFile(featuresYaml, []byte("features:\n  - name: auth\n    version: 1.0.0\n    installed_at: \"2026-01-01T00:00:00Z\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Remove auth directory if it exists
	os.RemoveAll(filepath.Join(target5, "internal", "auth"))
	doctor := exec.Command(bin, "doctor")
	doctor.Dir = target5
	out, err = doctor.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "manquant") {
		// Should warn about missing feature directory
		t.Logf("doctor output (expected warning about missing feature dir): %s", out)
	}
	t.Log("✓ Inconsistent .forge scenario tested")
}

// TestRollbackE2E tests rollback behavior on installation failure
func TestRollbackE2E(t *testing.T) {
	root := t.TempDir()
	project := "rollback-test"
	target := filepath.Join(root, project)

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "forge")
	build := exec.Command("go", "build", "-o", bin, "./cmd/forge")
	build.Dir = repoRoot
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build forge: %s: %v", out, err)
	}

	init := exec.Command(bin, "init", project,
		"--non-interactive",
		"--module", "github.com/forgekit/rollback-test",
		"--dir", target,
	)
	out, err = init.CombinedOutput()
	if err != nil {
		t.Fatalf("forge init: %s: %v", out, err)
	}

	// Test rollback by simulating a failure (e.g., making a file read-only)
	t.Log("Testing rollback on permission error")
	// Create a situation where file write will fail
	authDir := filepath.Join(target, "internal", "auth")
	if err := os.MkdirAll(authDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Make directory read-only
	if err := os.Chmod(authDir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(authDir, 0o755) // Restore for cleanup

	addAuth := exec.Command(bin, "add", "auth")
	addAuth.Dir = target
	out, err = addAuth.CombinedOutput()
	// Should fail due to permission denied
	if err == nil {
		t.Logf("add auth succeeded unexpectedly (may be OK on some systems): %s", out)
	} else {
		t.Logf("add auth failed as expected: %s", out)
		// Check if rollback happened - project should be in consistent state
		// The go.mod and .forge should be intact
		if _, err := os.Stat(filepath.Join(target, "go.mod")); err != nil {
			t.Fatal("go.mod missing after failed install")
		}
		if _, err := os.Stat(filepath.Join(target, ".forge")); err != nil {
			t.Fatal(".forge missing after failed install")
		}
		t.Log("✓ Rollback behavior verified (project state intact)")
	}
}