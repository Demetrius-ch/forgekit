package logging

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Demetrius-ch/forgekit/internal/feature"
	"github.com/Demetrius-ch/forgekit/internal/template"
)

type LoggingFeature struct{}

func (LoggingFeature) Name() string {
	return "logging"
}

func (LoggingFeature) Description() string {
	return "Logging structuré et middleware HTTP"
}

func (LoggingFeature) Version() string {
	return "1.0.0"
}

// DependsOn returns the list of features this feature depends on.
func (LoggingFeature) DependsOn() []string {
	return []string{"auth"}
}

func (LoggingFeature) Check(ctx context.Context, project feature.ProjectContext) error {
	installed, existing, err := feature.IsInstalled(project.Root, "logging")
	if err != nil {
		return fmt.Errorf("vérifier l'installation existante : %w", err)
	}
	if installed {
		return fmt.Errorf("logging version %s déjà installée", existing.Version)
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

func (LoggingFeature) Plan(ctx context.Context, project feature.ProjectContext) (feature.Plan, error) {
	return feature.Plan{
		Feature: "logging",
		Version: "1.0.0",
		Files: []feature.FileAction{
			{
				Source:      "internal/logging/logger.go.tmpl",
				Destination: "internal/logging/logger.go",
				Action:      feature.FileActionCreate,
			},
			{
				Source:      "internal/logging/middleware.go.tmpl",
				Destination: "internal/logging/middleware.go",
				Action:      feature.FileActionCreate,
			},
		},
		Dependencies: []feature.Dependency{},
		Environment: []string{
			"LOG_LEVEL=info",
			"LOG_FORMAT=text",
		},
	}, nil
}

func (LoggingFeature) Apply(ctx context.Context, project feature.ProjectContext, plan feature.Plan) error {
	tmplFS, err := template.LoadAPI()
	if err != nil {
		return fmt.Errorf("charger les templates : %w", err)
	}

	data := template.Data{
		ProjectName:  filepath.Base(project.Root),
		ModulePath:   project.Module,
		PackageName:  filepath.Base(project.Root),
		HTTPPort:     8080,
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

	// Integrate into main.go using shared utility
	if err := feature.IntegrateMainGo(project.Root, feature.MainIntegration{
		ModulePath:  project.Module,
		ImportPath:  project.Module + "/internal/logging",
		ImportCheck: "logging.DefaultLogger()",
		Replacements: []feature.MainReplacement{
			{
				OldStr: `slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))`,
				NewStr: `logging.DefaultLogger()`,
				Check:  "slog.New",
			},
			{
				OldStr: `transport.NewRouter(logger, pool, userRepo)`,
				NewStr: `transport.NewRouter(logger.Logger, pool, userRepo)`,
				Check:  "transport.NewRouter(logger, pool, userRepo)",
			},
			{
				OldStr: `transport.NewServer(cfg.HTTPAddr, router, logger)`,
				NewStr: `transport.NewServer(cfg.HTTPAddr, router, logger.Logger)`,
				Check:  "transport.NewServer(cfg.HTTPAddr, router, logger)",
			},
		},
	}); err != nil {
		return fmt.Errorf("intégrer dans main.go : %w", err)
	}

	// Integrate into router.go using shared utility
	if err := feature.IntegrateRouterGo(project.Root, feature.RouterIntegration{
		ModulePath:      project.Module,
		ImportPath:      project.Module + "/internal/logging",
		MiddlewareCall:  "logging.LoggingMiddleware(logger)",
		MiddlewareCheck: "logging.LoggingMiddleware",
	}); err != nil {
		return fmt.Errorf("intégrer dans router.go : %w", err)
	}

	// Run go mod tidy to resolve new imports before gofmt
	if err := feature.RunGoModTidy(project.Root); err != nil {
		return fmt.Errorf("go mod tidy après intégration : %w", err)
	}

	if err := feature.RunGoFmt(project.Root); err != nil {
		return fmt.Errorf("gofmt après intégration : %w", err)
	}

	if err := feature.AddInstalledFeature(project.Root, "logging", LoggingFeature{}.Version()); err != nil {
		return fmt.Errorf("enregistrer l'installation : %w", err)
	}

	return nil
}

// Remove uninstalls the logging feature.
func (LoggingFeature) Remove(ctx context.Context, project feature.ProjectContext, plan feature.Plan) error {
	// Remove files
	for _, file := range plan.Files {
		dest := filepath.Join(project.Root, file.Destination)
		if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("supprimer %s : %w", dest, err)
		}
	}

	// Remove empty directories if they exist
	loggingDir := filepath.Join(project.Root, "internal", "logging")
	if err := os.Remove(loggingDir); err != nil && !os.IsNotExist(err) {
		// Directory not empty or other error, ignore
	}

	// Remove router.go integration
	if err := feature.RemoveRouterGo(project.Root, feature.RouterIntegration{
		ModulePath:      project.Module,
		ImportPath:      project.Module + "/internal/logging",
		MiddlewareCall:  "logging.LoggingMiddleware(logger)",
		MiddlewareCheck: "logging.LoggingMiddleware",
	}); err != nil {
		return fmt.Errorf("supprimer intégration router.go : %w", err)
	}

	// Remove main.go integration
	if err := feature.RemoveMainGo(project.Root, feature.MainIntegration{
		ModulePath:  project.Module,
		ImportPath:  project.Module + "/internal/logging",
		ImportCheck: "logging.DefaultLogger()",
		Replacements: []feature.MainReplacement{
			{
				OldStr: `slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))`,
				NewStr: `logging.DefaultLogger()`,
				Check:  "slog.New",
			},
			{
				OldStr: `transport.NewRouter(logger, pool, userRepo)`,
				NewStr: `transport.NewRouter(logger.Logger, pool, userRepo)`,
				Check:  "transport.NewRouter(logger, pool, userRepo)",
			},
			{
				OldStr: `transport.NewServer(cfg.HTTPAddr, router, logger)`,
				NewStr: `transport.NewServer(cfg.HTTPAddr, router, logger.Logger)`,
				Check:  "transport.NewServer(cfg.HTTPAddr, router, logger)",
			},
		},
	}); err != nil {
		return fmt.Errorf("supprimer intégration main.go : %w", err)
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
	if err := feature.RemoveInstalledFeature(project.Root, "logging"); err != nil {
		return fmt.Errorf("supprimer l'enregistrement d'installation : %w", err)
	}

	return nil
}
