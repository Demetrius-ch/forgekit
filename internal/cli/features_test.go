package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Demetrius-ch/forgekit/internal/feature"
	"github.com/Demetrius-ch/forgekit/internal/feature/auth"
	"github.com/Demetrius-ch/forgekit/internal/feature/cors"
	"github.com/Demetrius-ch/forgekit/internal/feature/logging"
	"github.com/Demetrius-ch/forgekit/internal/feature/swagger"
	"github.com/Demetrius-ch/forgekit/internal/projectconfig"
)

func TestBuiltinFeaturesAreIndependent(t *testing.T) {
	for _, f := range []feature.Feature{auth.AuthFeature{}, cors.CorsFeature{}, logging.LoggingFeature{}, swagger.SwaggerFeature{}} {
		if deps, ok := f.(feature.FeatureDependencies); ok && len(deps.DependsOn()) > 0 {
			t.Errorf("feature %q depends on %v; built-in features must be installable on their own", f.Name(), deps.DependsOn())
		}
	}
}

func TestExistingPlanFilesReportsFilesToOverwrite(t *testing.T) {
	root := t.TempDir()
	project := feature.ProjectContext{Root: root, Module: "example.com/app"}
	features := []feature.Feature{auth.AuthFeature{}, cors.CorsFeature{}}

	existing, err := existingPlanFiles(context.Background(), project, features, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(existing) != 0 {
		t.Fatalf("expected no existing files, got %v", existing)
	}

	userFile := filepath.Join(root, "internal", "cors", "cors.go")
	if err := os.MkdirAll(filepath.Dir(userFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userFile, []byte("package cors\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	existing, err = existingPlanFiles(context.Background(), project, features, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(existing) != 1 || existing[0] != "internal/cors/cors.go" {
		t.Fatalf("expected internal/cors/cors.go, got %v", existing)
	}

	// Features skipped as already present are not reported.
	existing, err = existingPlanFiles(context.Background(), project, features, map[string]bool{"cors": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(existing) != 0 {
		t.Fatalf("expected skipped feature to be ignored, got %v", existing)
	}
}

func TestDefaultModulePathFor(t *testing.T) {
	t.Setenv("USER", "dev")
	tests := map[projectconfig.Language]string{
		projectconfig.LanguageGo:         "github.com/dev/my-api",
		"":                               "github.com/dev/my-api",
		projectconfig.LanguageTypeScript: "my-api",
		projectconfig.LanguageJavaScript: "my-api",
		projectconfig.LanguagePython:     "my-api",
	}
	for language, want := range tests {
		if got := defaultModulePathFor(language, "my-api"); got != want {
			t.Errorf("defaultModulePathFor(%q) = %q, want %q", language, got, want)
		}
	}
}
