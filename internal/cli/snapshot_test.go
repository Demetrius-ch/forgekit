package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRestoreInstallSnapshotRevertsAllChanges(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/app\n")
	writeTestFile(t, root, "cmd/server/main.go", "package main\n")
	writeTestFile(t, root, "internal/transport/http/router.go", "package transport\n")
	writeTestFile(t, root, ".env.example", "HTTP_PORT=8080\n")
	writeTestFile(t, root, "internal/old/old.go", "package old\n")

	snapshot, err := createInstallSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a feature installation that fails halfway.
	writeTestFile(t, root, "go.mod", "module example.com/app\n\nrequire github.com/rs/cors v1.10.1\n")
	writeTestFile(t, root, "cmd/server/main.go", "package main\n\nimport _ \"example.com/app/internal/logging\"\n")
	writeTestFile(t, root, ".env.example", "HTTP_PORT=8080\nLOG_LEVEL=info\n")
	writeTestFile(t, root, "go.sum", "github.com/rs/cors v1.10.1 h1:x\n")
	writeTestFile(t, root, "internal/logging/logger.go", "package logging\n")
	writeTestFile(t, root, "internal/auth/sub/jwt.go", "package sub\n")
	if err := os.RemoveAll(filepath.Join(root, "internal", "old")); err != nil {
		t.Fatal(err)
	}

	if err := restoreInstallSnapshot(root, snapshot); err != nil {
		t.Fatalf("restore: %v", err)
	}

	for rel, want := range map[string]string{
		"go.mod":                            "module example.com/app\n",
		"cmd/server/main.go":                "package main\n",
		"internal/transport/http/router.go": "package transport\n",
		".env.example":                      "HTTP_PORT=8080\n",
		"internal/old/old.go":               "package old\n",
	} {
		if got := readTestFile(t, root, rel); got != want {
			t.Errorf("%s = %q, want %q", rel, got, want)
		}
	}
	for _, rel := range []string{"go.sum", "internal/logging", "internal/auth"} {
		if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Errorf("%s should have been removed (err=%v)", rel, err)
		}
	}
}

func TestRestoreInstallSnapshotKeepsSkippedAndLargeEntries(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/app\n")
	large := make([]byte, snapshotMaxFileSize+1)
	if err := os.WriteFile(filepath.Join(root, "big.bin"), large, 0o644); err != nil {
		t.Fatal(err)
	}

	snapshot, err := createInstallSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, root, ".git/index", "new git data\n")
	writeTestFile(t, root, "vendor/modules.txt", "# vendored\n")

	if err := restoreInstallSnapshot(root, snapshot); err != nil {
		t.Fatalf("restore: %v", err)
	}
	for _, rel := range []string{"big.bin", ".git/index", "vendor/modules.txt"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("%s should have been kept: %v", rel, err)
		}
	}
}
