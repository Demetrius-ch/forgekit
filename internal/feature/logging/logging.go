package logging

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
			},
			{
				Source:      "internal/logging/middleware.go.tmpl",
				Destination: "internal/logging/middleware.go",
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

	// Integrate logging into main.go
	if err := integrateMainGo(project.Root, project.Module); err != nil {
		return fmt.Errorf("intégrer dans main.go : %w", err)
	}

	// Integrate logging middleware into router.go
	if err := integrateRouterGo(project.Root, project.Module); err != nil {
		return fmt.Errorf("intégrer dans router.go : %w", err)
	}

	// Run go mod tidy to resolve new imports before gofmt
	if err := feature.RunGoModTidy(project.Root); err != nil {
		return fmt.Errorf("go mod tidy après intégration : %w", err)
	}

	if err := feature.RunGoFmt(project.Root); err != nil {
		return fmt.Errorf("gofmt après intégration : %w", err)
	}

	if err := feature.AddInstalledFeature(project.Root, "logging", "1.0.0"); err != nil {
		return fmt.Errorf("enregistrer l'installation : %w", err)
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
	if strings.Contains(src, "logging.DefaultLogger") || strings.Contains(src, `"`+modulePath+`/internal/logging"`) {
		return nil // Already integrated
	}

	// Add logging import
	importBlock := `import (`
	newImport := importBlock + "\n\t" + `"` + modulePath + "/internal/logging" + `"`
	src = strings.Replace(src, importBlock, newImport, 1)

	// Replace slog.New with logging.DefaultLogger() - use more flexible matching
	oldLogger := `slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))`
	newLogger := `logging.DefaultLogger()`
	if !strings.Contains(src, oldLogger) {
		// Try without the extra space
		oldLogger = `slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))`
		if !strings.Contains(src, oldLogger) {
			return fmt.Errorf("point d'intégration main.go introuvable : logger initialization")
		}
	}
	src = strings.Replace(src, oldLogger, newLogger, 1)

	// Replace logger with logger.Logger for transport.NewRouter and transport.NewServer
	oldRouter := `transport.NewRouter(logger, pool, userRepo)`
	newRouter := `transport.NewRouter(logger.Logger, pool, userRepo)`
	if strings.Contains(src, oldRouter) {
		src = strings.Replace(src, oldRouter, newRouter, 1)
	}

	oldServer := `transport.NewServer(cfg.HTTPAddr, router, logger)`
	newServer := `transport.NewServer(cfg.HTTPAddr, router, logger.Logger)`
	if strings.Contains(src, oldServer) {
		src = strings.Replace(src, oldServer, newServer, 1)
	}

	// Remove unused "log/slog" import if present
	oldSlogImport := `"log/slog"`
	if strings.Contains(src, oldSlogImport) {
		src = strings.Replace(src, oldSlogImport+"\n", "", 1)
		src = strings.Replace(src, oldSlogImport+"\t", "", 1)
	}

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
	if strings.Contains(src, "logging.LoggingMiddleware") || strings.Contains(src, `"`+modulePath+`/internal/logging"`) {
		return nil // Already integrated
	}

	// Add logging import
	importBlock := `import (`
	newImport := importBlock + "\n\t" + `"` + modulePath + "/internal/logging" + `"`
	src = strings.Replace(src, importBlock, newImport, 1)

	// Add LoggingMiddleware to the router chain
	// Find the middleware chain and add logging middleware
	oldMiddleware := `r.Use(middleware.Timeout(30 * time.Second))`
	newMiddleware := `r.Use(middleware.Timeout(30 * time.Second))
	r.Use(logging.LoggingMiddleware(logger))`
	if !strings.Contains(src, oldMiddleware) {
		return fmt.Errorf("point d'intégration router.go introuvable : middleware chain")
	}
	src = strings.Replace(src, oldMiddleware, newMiddleware, 1)

	if err := os.WriteFile(routerPath, []byte(src), 0o644); err != nil {
		return fmt.Errorf("écrire router.go : %w", err)
	}

	return nil
}
