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

	// Add .forge/features.yaml to make it a valid legacy ForgeKit project
	if err := os.MkdirAll(
		filepath.Join(root, ".forge"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(root, ".forge", "features.yaml"),
		[]byte("features: []\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	return root
}

func createForgeKitProject(t *testing.T, goMod string) string {
	root := createGoProject(t, goMod)

	// Add forge.yaml to make it a full ForgeKit project
	if err := os.WriteFile(
		filepath.Join(root, ".forge", "forge.yaml"),
		[]byte("version: 0.2.0\nschema: 1\nproject: test\nlanguage: go\ntype: backend-api\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	return root
}

func createExternalProject(t *testing.T, goMod string) string {
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

	// No .forge directory - external compatible project
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

	if ctx.Type != ProjectTypeLegacyForgeKit {
		t.Fatalf("expected LegacyForgeKit, got %d", ctx.Type)
	}
}

func TestDetectorDetectForgeKit(t *testing.T) {
	root := createForgeKitProject(
		t,
		"module github.com/example/demo\n\ngo 1.25\n",
	)

	ctx, err := (Detector{}).Detect(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.Type != ProjectTypeForgeKit {
		t.Fatalf("expected ForgeKit, got %d", ctx.Type)
	}
}

func TestDetectorDetectLooseLegacy(t *testing.T) {
	root := createGoProject(
		t,
		"module github.com/example/demo\n\ngo 1.25\n",
	)

	ctx, err := (Detector{}).DetectLoose(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.Type != ProjectTypeLegacyForgeKit {
		t.Fatalf("expected LegacyForgeKit, got %d", ctx.Type)
	}
}

func TestDetectorDetectLooseForgeKit(t *testing.T) {
	root := createForgeKitProject(
		t,
		"module github.com/example/demo\n\ngo 1.25\n",
	)

	ctx, err := (Detector{}).DetectLoose(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.Type != ProjectTypeForgeKit {
		t.Fatalf("expected ForgeKit, got %d", ctx.Type)
	}
}

func TestDetectorDetectLooseExternal(t *testing.T) {
	root := createExternalProject(
		t,
		"module github.com/example/external\n\ngo 1.25\n",
	)

	ctx, err := (Detector{}).DetectLoose(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.Type != ProjectTypeExternalCompatible {
		t.Fatalf("expected ExternalCompatible, got %d", ctx.Type)
	}
}

func TestDetectorMissingGoMod(t *testing.T) {
	root := t.TempDir()

	_, err := (Detector{}).Detect(root)
	if err == nil {
		t.Fatal("expected error when go.mod is missing")
	}

	_, err = (Detector{}).DetectLoose(root)
	if err == nil {
		t.Fatal("expected error when go.mod is missing in loose mode")
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

	_, err = (Detector{}).DetectLoose(root)
	if err == nil {
		t.Fatal("expected error when module is missing in loose mode")
	}
}

func TestDetectorInvalidForgeKit(t *testing.T) {
	root := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(root, "go.mod"),
		[]byte("module github.com/example/demo\n\ngo 1.25\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Create .forge directory but no forge.yaml or features.yaml
	if err := os.MkdirAll(
		filepath.Join(root, ".forge"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	ctx, err := (Detector{}).Detect(root)
	if err == nil {
		t.Fatal("expected error for invalid ForgeKit project")
	}

	// Loose mode returns the context with InvalidForgeKit type, not an error
	ctx, err = (Detector{}).DetectLoose(root)
	if err != nil {
		t.Fatalf("unexpected error in loose mode: %v", err)
	}
	if ctx.Type != ProjectTypeInvalidForgeKit {
		t.Fatalf("expected InvalidForgeKit, got %d", ctx.Type)
	}
}
