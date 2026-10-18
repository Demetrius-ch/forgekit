package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVenvPython(t *testing.T) {
	dir := filepath.Join("projects", "api")
	if got, want := venvPython(dir, "windows"), filepath.Join(dir, ".venv", "Scripts", "python.exe"); got != want {
		t.Errorf("windows: got %q, want %q", got, want)
	}
	for _, goos := range []string{"linux", "darwin"} {
		if got, want := venvPython(dir, goos), filepath.Join(dir, ".venv", "bin", "python"); got != want {
			t.Errorf("%s: got %q, want %q", goos, got, want)
		}
	}
}

func TestRemoveGeneratedKeepsPreexistingDirectory(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "project")
	if err := os.MkdirAll(filepath.Join(dir, "internal"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	removeGenerated(dir, true)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("pre-existing directory must be kept: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("generated content must be removed, found %d entries", len(entries))
	}

	removeGenerated(dir, false)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("directory created by init must be removed, stat err = %v", err)
	}
}
