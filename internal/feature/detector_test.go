package feature

import (
	"os"
	"path/filepath"
	"testing"
)

func createGoProject(t *testing.T, goMod string) string {
	t.Helper()

	root := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(root, "go.mod"),
		[]byte(goMod),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(
		filepath.Join(root, "cmd", "server"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestDetectorDetect(t *testing.T) {
	root := createGoProject(
		t,
		"module github.com/example/demo\n\ngo 1.25\n",
	)

	ctx, err := (Detector{}).Detect(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.Module != "github.com/example/demo" {
		t.Fatalf("unexpected module: %s", ctx.Module)
	}

	if ctx.GoVersion != "1.25" {
		t.Fatalf("unexpected Go version: %s", ctx.GoVersion)
	}

	if ctx.Root != root {
		t.Fatalf("unexpected root: %s", ctx.Root)
	}
}

func TestDetectorMissingGoMod(t *testing.T) {
	root := t.TempDir()

	_, err := (Detector{}).Detect(root)
	if err == nil {
		t.Fatal("expected error when go.mod is missing")
	}
}

func TestDetectorMissingModule(t *testing.T) {
	root := createGoProject(
		t,
		"go 1.25\n",
	)

	_, err := (Detector{}).Detect(root)
	if err == nil {
		t.Fatal("expected error when module is missing")
	}
}
