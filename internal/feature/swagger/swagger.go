package swagger

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Demetrius-ch/forgekit/internal/feature"
	"github.com/Demetrius-ch/forgekit/internal/template"
)

type SwaggerFeature struct{}

func (SwaggerFeature) Name() string {
	return "swagger"
}

func (SwaggerFeature) Description() string {
	return "Documentation OpenAPI/Swagger"
}

func (SwaggerFeature) Version() string {
	return "1.0.1"
}

// DependsOn returns the list of features this feature depends on.
func (SwaggerFeature) DependsOn() []string {
	return []string{"cors"}
}

func (SwaggerFeature) Check(ctx context.Context, project feature.ProjectContext) error {
	installed, existing, err := feature.IsInstalled(project.Root, "swagger")
	if err != nil {
		return fmt.Errorf("vérifier l'installation existante : %w", err)
	}
	if installed {
		return fmt.Errorf("swagger version %s déjà installée", existing.Version)
	}

	requiredPaths := []string{
		filepath.Join(project.Root, "internal", "transport", "http", "router.go"),
		filepath.Join(project.Root, "cmd", "server", "main.go"),
		filepath.Join(project.Root, "go.mod"),
	}

	for _, p := range requiredPaths {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("fichier requis manquant : %s", p)
		}
	}

	return nil
}

func (SwaggerFeature) Plan(ctx context.Context, project feature.ProjectContext) (feature.Plan, error) {
	return feature.Plan{
		Feature: "swagger",
		Version: "1.0.1",
		Files: []feature.FileAction{
			{
				Source:      "internal/swagger/swagger.go.tmpl",
				Destination: "internal/swagger/swagger.go",
				Action:      feature.FileActionCreate,
			},
			{
				Source:      "internal/swagger/openapi.yaml.tmpl",
				Destination: "internal/swagger/openapi.yaml",
				Action:      feature.FileActionCreate,
			},
		},
		Dependencies: []feature.Dependency{
			{Module: "github.com/swaggo/http-swagger", Version: "v1.3.4"},
			{Module: "gopkg.in/yaml.v3", Version: "v3.0.1"},
		},
		Environment: []string{
			"SWAGGER_ENABLED=true",
		},
	}, nil
}

func (SwaggerFeature) Apply(ctx context.Context, project feature.ProjectContext, plan feature.Plan) error {
	tmplFS, err := template.LoadAPI()
	if err != nil {
		return fmt.Errorf("charger les templates : %w", err)
	}

	data := template.Data{
		ProjectName:  filepath.Base(project.Root),
		ModulePath:   project.Module,
		PackageName:  filepath.Base(project.Root),
		HTTPPort:     project.HTTPPort,
		DatabaseName: "",
		GoVersion:    project.GoVersion,
		Author:       "",
	}

	for _, file := range plan.Files {
		content, err := template.RenderFile(tmplFS, file.Source, data)
		if err != nil {
			return fmt.Errorf("rendre le template %s : %w", file.Source, err)
		}

		dest := filepath.Join(project.Root, file.Destination)

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("créer le répertoire %s : %w", filepath.Dir(dest), err)
		}

		if err := os.WriteFile(dest, content, 0o644); err != nil {
			return fmt.Errorf("écrire %s : %w", dest, err)
		}
	}

	if err := feature.AddDependencies(project.Root, plan.Dependencies); err != nil {
		return fmt.Errorf("ajouter les dépendances : %w", err)
	}

	if err := feature.RunGoModTidy(project.Root); err != nil {
		return fmt.Errorf("go mod tidy : %w", err)
	}

	if err := feature.UpdateEnvironment(project.Root, plan.Environment); err != nil {
		return fmt.Errorf("mettre à jour l'environnement : %w", err)
	}

	// Integrate into main.go using shared utility - blank import for swagger
	if err := feature.IntegrateMainGo(project.Root, feature.MainIntegration{
		ModulePath:   project.Module,
		ImportPath:   project.Module + "/internal/swagger",
		ImportCheck:  `swagger`,
		BlankImport:  true,
		Replacements: []feature.MainReplacement{},
	}); err != nil {
		return fmt.Errorf("intégrer dans main.go : %w", err)
	}

	// Integrate into router.go using shared utility
	if err := feature.IntegrateRouterGo(project.Root, feature.RouterIntegration{
		ModulePath:      project.Module,
		ImportPath:      project.Module + "/internal/swagger",
		MiddlewareCall:  "swagger.RegisterRoutes(r)",
		MiddlewareCheck: "swagger.RegisterRoutes",
	}); err != nil {
		return fmt.Errorf("intégrer dans router.go : %w", err)
	}

	if err := feature.RunGoModTidy(project.Root); err != nil {
		return fmt.Errorf("go mod tidy après intégration : %w", err)
	}

	if err := feature.RunGoFmt(project.Root); err != nil {
		return fmt.Errorf("gofmt après intégration : %w", err)
	}

	if err := feature.AddInstalledFeature(project.Root, "swagger", SwaggerFeature{}.Version()); err != nil {
		return fmt.Errorf("enregistrer l'installation : %w", err)
	}

	return nil
}

// Remove uninstalls the swagger feature.
func (SwaggerFeature) Remove(ctx context.Context, project feature.ProjectContext, plan feature.Plan) error {
	// Remove files
	for _, file := range plan.Files {
		dest := filepath.Join(project.Root, file.Destination)
		if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("supprimer %s : %w", dest, err)
		}
	}

	// Remove empty directories if they exist
	swaggerDir := filepath.Join(project.Root, "internal", "swagger")
	if err := os.Remove(swaggerDir); err != nil && !os.IsNotExist(err) {
		// Directory not empty or other error, ignore
	}

	// Remove router.go integration
	if err := feature.RemoveRouterGo(project.Root, feature.RouterIntegration{
		ModulePath:      project.Module,
		ImportPath:      project.Module + "/internal/swagger",
		MiddlewareCall:  "swagger.RegisterRoutes(r)",
		MiddlewareCheck: "swagger.RegisterRoutes",
	}); err != nil {
		return fmt.Errorf("supprimer intégration router.go : %w", err)
	}

	// Remove main.go integration (blank import)
	if err := feature.RemoveMainGo(project.Root, feature.MainIntegration{
		ModulePath:   project.Module,
		ImportPath:   project.Module + "/internal/swagger",
		ImportCheck:  `swagger`,
		BlankImport:  true,
		Replacements: []feature.MainReplacement{},
	}); err != nil {
		return fmt.Errorf("supprimer intégration main.go : %w", err)
	}

	// Remove dependencies from go.mod
	for _, dep := range plan.Dependencies {
		cmd := exec.Command("go", "mod", "edit", "-droprequire", dep.Module)
		cmd.Dir = project.Root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			// Log but continue - dependency might not be in go.mod
		}
	}

	// Run go mod tidy
	if err := feature.RunGoModTidy(project.Root); err != nil {
		return fmt.Errorf("go mod tidy : %w", err)
	}

	// Run gofmt
	if err := feature.RunGoFmt(project.Root); err != nil {
		return fmt.Errorf("gofmt : %w", err)
	}

	// Remove environment variables from .env.example
	if err := feature.RemoveEnvironment(project.Root, plan.Environment); err != nil {
		return fmt.Errorf("supprimer les variables d'environnement : %w", err)
	}

	// Remove installation record
	if err := feature.RemoveInstalledFeature(project.Root, "swagger"); err != nil {
		return fmt.Errorf("supprimer l'enregistrement d'installation : %w", err)
	}

	return nil
}
