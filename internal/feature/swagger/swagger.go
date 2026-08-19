package swagger

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	if err := integrateMainGo(project.Root, project.Module); err != nil {
		return fmt.Errorf("intégrer dans main.go : %w", err)
	}

	if err := integrateRouterGo(project.Root, project.Module); err != nil {
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

func integrateMainGo(projectRoot, modulePath string) error {
	mainPath := filepath.Join(projectRoot, "cmd", "server", "main.go")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("lire main.go : %w", err)
	}

	src := string(content)

	// Check if already integrated
	if strings.Contains(src, "swagger") && strings.Contains(src, `"`+modulePath+`/internal/swagger"`) {
		return nil // Already integrated
	}

	// Add swagger import
	importBlock := `import (`
	newImport := importBlock + "\n\t" + `_ "` + modulePath + "/internal/swagger" + `"`
	src = strings.Replace(src, importBlock, newImport, 1)

	// Remove unused import if present (log/slog or logging might be there)
	// This is handled by go mod tidy/gofmt later

	if err := os.WriteFile(mainPath, []byte(src), 0o644); err != nil {
		return fmt.Errorf("écrire main.go : %w", err)
	}

	return nil
}

func integrateRouterGo(projectRoot, modulePath string) error {
	routerPath := filepath.Join(projectRoot, "internal", "transport", "http", "router.go")
	content, err := os.ReadFile(routerPath)
	if err != nil {
		return fmt.Errorf("lire router.go : %w", err)
	}

	src := string(content)

	// Check if already integrated
	if strings.Contains(src, "swagger.RegisterRoutes") && strings.Contains(src, `"`+modulePath+`/internal/swagger"`) {
		return nil // Already integrated
	}

	// Add swagger import
	importBlock := `import (`
	newImport := importBlock + "\n\t" + `"` + modulePath + "/internal/swagger" + `"`
	src = strings.Replace(src, importBlock, newImport, 1)

	// Add Swagger routes registration - insert after the route definitions
	// Try multiple patterns to find the integration point
	patterns := []struct {
		oldStr string
		newStr string
	}{
		// Pattern 1: Standard ForgeKit router with users route
		{
			oldStr: `	r.Post("/users", handler.NewUser(logger, userSvc).Create)`,
			newStr: `	r.Post("/users", handler.NewUser(logger, userSvc).Create)

	// Swagger documentation routes
	swagger.RegisterRoutes(r)`,
		},
		// Pattern 2: Minimal router with only health route (test scenario)
		{
			oldStr: `	r.Get("/health", handler.NewHealth(logger, healthSvc).ServeHTTP)`,
			newStr: `	r.Get("/health", handler.NewHealth(logger, healthSvc).ServeHTTP)

	// Swagger documentation routes
	swagger.RegisterRoutes(r)`,
		},
	}

	integrated := false
	for _, p := range patterns {
		if strings.Contains(src, p.oldStr) {
			src = strings.Replace(src, p.oldStr, p.newStr, 1)
			integrated = true
			break
		}
	}

	if !integrated {
		return fmt.Errorf("point d'intégration router.go introuvable : routes")
	}

	if err := os.WriteFile(routerPath, []byte(src), 0o644); err != nil {
		return fmt.Errorf("écrire router.go : %w", err)
	}

	return nil
}
