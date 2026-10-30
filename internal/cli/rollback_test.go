package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Demetrius-ch/forgekit/internal/feature"
)

// TestRestoreInstallSnapshotRevertsFeatureFiles checks that a failed install
// leaves no partial state: created files are deleted (with their now-empty
// directories) and modified files, including .env.example, are restored.
func TestRestoreInstallSnapshotRevertsFeatureFiles(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".env.example", "HTTP_PORT=8080\n")
	write("internal/transport/http/router.go", "package transport\n")

	plan := feature.Plan{Files: []feature.FileAction{{Destination: "internal/cors/cors.go"}}}
	snapshot, err := createInstallSnapshot(root, snapshotPaths(root, plan)...)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a partially applied feature.
	write(".env.example", "HTTP_PORT=8080\nCORS_ENABLED=false\n")
	write("internal/transport/http/router.go", "package transport\n// modified\n")
	write("internal/cors/cors.go", "package cors\n")

	if err := restoreInstallSnapshot(root, snapshot); err != nil {
		t.Fatalf("restoreInstallSnapshot: %v", err)
	}

	if data, _ := os.ReadFile(filepath.Join(root, ".env.example")); string(data) != "HTTP_PORT=8080\n" {
		t.Errorf(".env.example not restored: %q", data)
	}
	if data, _ := os.ReadFile(filepath.Join(root, "internal/transport/http/router.go")); string(data) != "package transport\n" {
		t.Errorf("router.go not restored: %q", data)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "cors")); !os.IsNotExist(err) {
		t.Errorf("internal/cors should be removed after rollback, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "transport")); err != nil {
		t.Errorf("pre-existing directories must be kept: %v", err)
	}
}
