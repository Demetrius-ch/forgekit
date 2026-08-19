package auth

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Demetrius-ch/forgekit/internal/feature"
	"github.com/Demetrius-ch/forgekit/internal/template"
)

// AuthFeature implements the Feature interface for JWT authentication.
type AuthFeature struct{}

// Name returns the feature name.
func (AuthFeature) Name() string {
	return "auth"
}

// Description returns the feature description.
func (AuthFeature) Description() string {
	return "Infrastructure d'authentification JWT (middleware, validation de token)"
}

// Version returns the feature version.
func (AuthFeature) Version() string {
	return "1.0.0"
}

// Check verifies prerequisites for installing auth.
func (AuthFeature) Check(ctx context.Context, project feature.ProjectContext) error {
	// Check if already installed
	installed, existing, err := feature.IsInstalled(project.Root, "auth")
	if err != nil {
		return fmt.Errorf("vérifier l'installation existante : %w", err)
	}
	if installed {
		return fmt.Errorf("auth version %s déjà installée", existing.Version)
	}

	// Check if project has required structure
	requiredPaths := []string{
		filepath.Join(project.Root, "internal", "transport", "http", "router.go"),
		filepath.Join(project.Root, "go.mod"),
	}

	for _, p := range requiredPaths {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("fichier requis manquant : %s", p)
		}
	}

	return nil
}

// Plan returns the installation plan for auth.
func (AuthFeature) Plan(ctx context.Context, project feature.ProjectContext) (feature.Plan, error) {
	return feature.Plan{
		Feature: "auth",
		Version: "1.0.0",
		Files: []feature.FileAction{
			{
				Source:      "internal/auth/jwt.go.tmpl",
				Destination: "internal/auth/jwt.go",
				Action:      feature.FileActionCreate,
			},
			{
				Source:      "internal/auth/middleware.go.tmpl",
				Destination: "internal/auth/middleware.go",
				Action:      feature.FileActionCreate,
			},
		},
		Dependencies: []feature.Dependency{
			{Module: "github.com/golang-jwt/jwt/v5", Version: "v5.2.0"},
		},
		Environment: []string{
			"JWT_SECRET=your-secret-key-change-in-production",
		},
	}, nil
}

// Apply installs the auth feature.
func (AuthFeature) Apply(ctx context.Context, project feature.ProjectContext, plan feature.Plan) error {
	// Load template filesystem
	tmplFS, err := template.LoadAPI()
	if err != nil {
		return fmt.Errorf("charger les templates : %w", err)
	}

	// Prepare template data
	data := template.Data{
		ProjectName:  filepath.Base(project.Root),
		ModulePath:   project.Module,
		PackageName:  filepath.Base(project.Root),
		HTTPPort:     8080,
		DatabaseName: "",
		GoVersion:    project.GoVersion,
		Author:       "",
	}

	// Render and write template files
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

	// Update go.mod with new dependencies
	if err := feature.AddDependencies(project.Root, plan.Dependencies); err != nil {
		return fmt.Errorf("ajouter les dépendances : %w", err)
	}

	// Run go mod tidy
	if err := feature.RunGoModTidy(project.Root); err != nil {
		return fmt.Errorf("go mod tidy : %w", err)
	}

	// Run gofmt
	if err := feature.RunGoFmt(project.Root); err != nil {
		return fmt.Errorf("gofmt : %w", err)
	}

	// Update environment variables
	if err := feature.UpdateEnvironment(project.Root, plan.Environment); err != nil {
		return fmt.Errorf("mettre à jour l'environnement : %w", err)
	}

	// Record installation
	if err := feature.AddInstalledFeature(project.Root, "auth", AuthFeature{}.Version()); err != nil {
		return fmt.Errorf("enregistrer l'installation : %w", err)
	}

	return nil
}

// Remove uninstalls the auth feature.
func (AuthFeature) Remove(ctx context.Context, project feature.ProjectContext, plan feature.Plan) error {
	// Remove files
	for _, file := range plan.Files {
		dest := filepath.Join(project.Root, file.Destination)
		if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("supprimer %s : %w", dest, err)
		}
	}

	// Remove empty directories if they exist
	authDir := filepath.Join(project.Root, "internal", "auth")
	if err := os.Remove(authDir); err != nil && !os.IsNotExist(err) {
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
	if err := feature.RemoveInstalledFeature(project.Root, "auth"); err != nil {
		return fmt.Errorf("supprimer l'enregistrement d'installation : %w", err)
	}

	return nil
}
