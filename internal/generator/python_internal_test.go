package generator

import (
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
