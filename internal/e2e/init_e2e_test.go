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

	required := []string{
		"cmd/server/main.go",
		"internal/domain/user.go",
		"internal/application/user/service.go",
		"internal/infrastructure/postgres/pool.go",
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
