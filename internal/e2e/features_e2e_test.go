//go:build e2e

package e2e_test

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// buildForge compiles the CLI into dir and returns the binary path.
func buildForge(t *testing.T, dir string) string {
	t.Helper()
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "forge")
	build := exec.Command("go", "build", "-o", bin, "./cmd/forge")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build forge: %s: %v", out, err)
	}
	return bin
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %s: %s: %v", name, strings.Join(args, " "), out, err)
	}
}

// treeDigest returns one line per file (path and content hash), sorted.
func treeDigest(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			lines = append(lines, rel+"/")
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines = append(lines, fmt.Sprintf("%s %x", rel, sha256.Sum256(data)))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// TestAddFeaturesAllArchitecturesE2E installs then removes every feature on a
// freshly generated project of each architecture; the project must build and
// pass its tests at each end state.
func TestAddFeaturesAllArchitecturesE2E(t *testing.T) {
	bin := buildForge(t, t.TempDir())
	features := []string{"logging", "swagger", "auth", "cors"}

	for _, arch := range []string{"hexagonal", "clean", "layered"} {
		t.Run(arch, func(t *testing.T) {
			root := t.TempDir()
			project := "arch-" + arch
			target := filepath.Join(root, project)
			run(t, root, bin, "init", project,
				"--non-interactive",
				"--module", "github.com/forgekit/"+project,
				"--architecture", arch,
				"--database", "sqlite",
				"--docker=false",
				"--quiet",
			)

			for _, f := range features {
				run(t, target, bin, "add", f, "--quiet")
			}
			run(t, target, "go", "build", "./...")
			run(t, target, "go", "vet", "./...")
			run(t, target, "go", "test", "./...")

			// Dependents first: swagger needs cors, cors and logging need auth.
			for _, f := range []string{"swagger", "cors", "logging", "auth"} {
				run(t, target, bin, "remove", f, "--quiet")
			}
			run(t, target, "go", "build", "./...")
			run(t, target, "go", "test", "./...")
		})
	}
}

// TestRollbackRestoresProjectE2E makes an installation fail after files were
// written and dependencies added, and checks that the project is restored
// byte for byte and still builds.
func TestRollbackRestoresProjectE2E(t *testing.T) {
	root := t.TempDir()
	bin := buildForge(t, root)
	project := "rollback-restore"
	target := filepath.Join(root, project)
	run(t, root, bin, "init", project,
		"--non-interactive",
		"--module", "github.com/forgekit/"+project,
		"--database", "none",
		"--docker=false",
		"--quiet",
	)

	// A router ForgeKit cannot integrate with: logging fails after auth (its
	// dependency) has been fully installed and logging's own files written.
	routerPath := filepath.Join(target, "internal", "transport", "http", "router.go")
	router, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatal(err)
	}
	custom := strings.Replace(string(router), "r := chi.NewRouter()", "r := chi.NewMux()", 1)
	if err := os.WriteFile(routerPath, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	before := treeDigest(t, target)

	cmd := exec.Command(bin, "add", "logging", "--quiet")
	cmd.Dir = target
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("forge add logging should fail on an unsupported router: %s", out)
	}

	if after := treeDigest(t, target); after != before {
		t.Fatalf("project not restored after failed install\nbefore:\n%s\nafter:\n%s", before, after)
	}
	run(t, target, "go", "build", "./...")
}
