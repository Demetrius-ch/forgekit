package generator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgekit/forgekit/internal/generator"
)

func TestValidateProjectName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid", input: "my-api", wantErr: false},
		{name: "single letter", input: "a", wantErr: false},
		{name: "empty", input: "", wantErr: true},
		{name: "uppercase", input: "MyAPI", wantErr: true},
		{name: "double dash", input: "my--api", wantErr: true},
		{name: "reserved", input: "con", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := generator.ValidateProjectName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateProjectName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateModulePath(t *testing.T) {
	t.Parallel()

	if err := generator.ValidateModulePath("github.com/user/my-api"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := generator.ValidateModulePath("invalid"); err == nil {
		t.Fatal("expected error for invalid module path")
	}
}

func TestValidateTargetDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := generator.ValidateTargetDir(dir); err != nil {
		t.Fatalf("empty dir should be valid: %v", err)
	}

	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := generator.ValidateTargetDir(dir); err == nil {
		t.Fatal("expected error for non-empty dir")
	}
}

func TestValidateDatabaseName(t *testing.T) {
	t.Parallel()

	if err := generator.ValidateDatabaseName("demo_api"); err != nil {
		t.Fatalf("expected valid db name: %v", err)
	}
	if err := generator.ValidateDatabaseName("demo-api"); err == nil {
		t.Fatal("expected invalid db name")
	}
}

func TestGeneratorInitGeneratesGoModuleDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test in short mode")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "dep-api")

	gen, err := generator.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = gen.Init(generator.InitOptions{
		ProjectName:  "dep-api",
		ModulePath:   "github.com/forgekit/dep-api",
		HTTPPort:     8081,
		DatabaseName: "dep_api",
		TargetDir:    target,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	goModPath := filepath.Join(target, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("ReadFile(go.mod) error = %v", err)
	}
	content := string(data)
	for _, want := range []string{"github.com/go-chi/chi/v5", "github.com/jackc/pgx/v5"} {
		if !strings.Contains(content, want) {
			t.Fatalf("go.mod missing dependency %q\n%s", want, content)
		}
	}
}

func TestPackageNameFromProject(t *testing.T) {
	t.Parallel()

	if got := generator.PackageNameFromProject("my-api"); got != "myapi" {
		t.Fatalf("got %q, want myapi", got)
	}
}

func TestPostProcessProjectRunsFormattingAndTests(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test in short mode")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "cmd", "server"), 0o755); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "cmd", "server", "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\n\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	gen := &generator.Generator{}
	if err := gen.PostProcessProject(dir); err != nil {
		t.Fatalf("PostProcessProject() error = %v", err)
	}
}

func TestGeneratorInit(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test in short mode")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "demo-api")

	gen, err := generator.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	opts := generator.InitOptions{
		ProjectName:  "demo-api",
		ModulePath:   "github.com/forgekit/demo-api",
		HTTPPort:     8080,
		DatabaseName: "demo_api",
		TargetDir:    target,
	}

	_, err = gen.Init(opts)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	required := []string{
		"cmd/server/main.go",
		"internal/domain/user.go",
		"internal/application/user/service.go",
		"internal/infrastructure/postgres/pool.go",
		"internal/transport/http/router.go",
		"migrations/000001_init.up.sql",
		"docker/docker-compose.yml",
		"forge.yaml",
		".env.example",
		"README.md",
		"go.mod",
	}

	for _, rel := range required {
		path := filepath.Join(target, rel)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing generated file %s: %v", rel, err)
		}
	}
}
