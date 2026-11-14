//go:build e2e

package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestForgeInitE2E(t *testing.T) {
	root := t.TempDir()
	project := "test-api"
	target := filepath.Join(root, project)

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "forge")
	build := exec.Command("go", "build", "-o", bin, "./cmd/forge")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build forge: %s: %v", out, err)
	}

	init := exec.Command(bin, "init", project,
		"--non-interactive",
		"--module", "github.com/forgekit/e2e-api",
		"--dir", target,
	)
	if out, err := init.CombinedOutput(); err != nil {
		t.Fatalf("forge init: %s: %v", out, err)
	}

	// v0.4 hexagonal layout (docs/v0.4-custom-generation.md); the v0.3 user
	// example (domain/user.go, application/user, postgres/pool.go) is gone.
	required := []string{
		"cmd/server/main.go",
		"internal/domain/health.go",
		"internal/application/health.go",
		"internal/infrastructure/postgres/database.go",
		"internal/transport/http/router.go",
		"forge.yaml",
		"go.mod",
		"README.md",
	}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(target, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}

	mod, err := os.ReadFile(filepath.Join(target, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mod), "module github.com/forgekit/e2e-api") {
		t.Fatalf("unexpected go.mod: %s", mod)
	}

	test := exec.Command("go", "test", "./...")
	test.Dir = target
	if out, err := test.CombinedOutput(); err != nil {
		t.Fatalf("go test: %s: %v", out, err)
	}

	buildAPI := exec.Command("go", "build", "-o", "/dev/null", "./cmd/server")
	buildAPI.Dir = target
	if out, err := buildAPI.CombinedOutput(); err != nil {
		t.Fatalf("go build: %s: %v", out, err)
	}

	// architecture layers present
	for _, layer := range []string{"domain", "application", "infrastructure", "transport"} {
		path := filepath.Join(target, "internal", layer)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing layer %s", layer)
		}
	}
}

// TestForgeInitInteractivePipedAnswersE2E answers the interactive prompts
// through a pipe: every answer must be used, not only the first one, and
// choosing MySQL must not keep the PostgreSQL configuration.
func TestForgeInitInteractivePipedAnswersE2E(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "piped-api")

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "forge")
	build := exec.Command("go", "build", "-o", bin, "./cmd/forge")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build forge: %s: %v", out, err)
	}

	// language (asked first: the module prompt depends on it), module, port,
	// architecture, database, docker, auth, docs, tests, ci
	// (FORGE_AUTHOR set: no author prompt)
	answers := "\ngithub.com/forgekit/piped-api\n\n\nmysql\n\n\n\n\n\n"
	init := exec.Command(bin, "init", "piped-api", "--dir", target, "--skip-postprocess")
	init.Env = append(os.Environ(), "FORGE_AUTHOR=Tester")
	init.Stdin = strings.NewReader(answers)
	if out, err := init.CombinedOutput(); err != nil {
		t.Fatalf("forge init: %s: %v", out, err)
	}

	env, err := os.ReadFile(filepath.Join(target, ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(env), "@tcp(localhost:") {
		t.Fatalf("expected a MySQL DATABASE_URL, got:\n%s", env)
	}
	if _, err := os.Stat(filepath.Join(target, "internal", "infrastructure", "mysql", "database.go")); err != nil {
		t.Fatalf("expected MySQL infrastructure: %v", err)
	}
}
