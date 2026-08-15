package cors

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Demetrius-ch/forgekit/internal/feature"
	"github.com/Demetrius-ch/forgekit/internal/template"
)

type CorsFeature struct{}

func (CorsFeature) Name() string {
	return "cors"
}

func (CorsFeature) Description() string {
	return "Middleware CORS pour les requêtes cross-origin"
}

func (CorsFeature) Version() string {
	return "1.0.2"
}

func (CorsFeature) Check(ctx context.Context, project feature.ProjectContext) error {
	installed, existing, err := feature.IsInstalled(project.Root, "cors")
	if err != nil {
		return fmt.Errorf("vérifier l'installation existante : %w", err)
	}
	if installed {
		return fmt.Errorf("cors version %s déjà installée", existing.Version)
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

func (CorsFeature) Plan(ctx context.Context, project feature.ProjectContext) (feature.Plan, error) {
	return feature.Plan{
		Feature: "cors",
		Version: "1.0.2",
		Files: []feature.FileAction{
			{
				Source:      "internal/cors/cors.go.tmpl",
				Destination: "internal/cors/cors.go",
			},
		},
		Dependencies: []feature.Dependency{
			{Module: "github.com/rs/cors", Version: "v1.10.1"},
		},
		Environment: []string{
			"CORS_ENABLED=false",
			"CORS_ALLOWED_ORIGINS=http://localhost:3000",
			"CORS_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS",
			"CORS_ALLOWED_HEADERS=Content-Type,Authorization",
			"CORS_ALLOW_CREDENTIALS=false",
		},
	}, nil
}

func (CorsFeature) Apply(ctx context.Context, project feature.ProjectContext, plan feature.Plan) error {
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

	if err := integrateRouterGo(project.Root, project.Module); err != nil {
		return fmt.Errorf("intégrer dans router.go : %w", err)
	}

	if err := feature.RunGoModTidy(project.Root); err != nil {
		return fmt.Errorf("go mod tidy après intégration : %w", err)
	}

	if err := feature.RunGoFmt(project.Root); err != nil {
		return fmt.Errorf("gofmt après intégration : %w", err)
	}

	if err := feature.AddInstalledFeature(project.Root, "cors", CorsFeature{}.Version()); err != nil {
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

	if strings.Contains(src, "cors") && strings.Contains(src, `"`+modulePath+`/internal/cors"`) {
		return nil
	}

	importBlock := `import (`
	newImport := importBlock + "\n\t" + `"` + modulePath + "/internal/cors" + `"`
	src = strings.Replace(src, importBlock, newImport, 1)

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

	if strings.Contains(src, "cors.Middleware") && strings.Contains(src, `"`+modulePath+`/internal/cors"`) {
		return nil
	}

	importBlock := `import (`
	newImport := importBlock + "\n\t" + `"` + modulePath + "/internal/cors" + `"`
	src = strings.Replace(src, importBlock, newImport, 1)

	oldMiddleware := `r.Use(middleware.Timeout(30 * time.Second))`
	newMiddleware := `r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Middleware())`
	if !strings.Contains(src, oldMiddleware) {
		return fmt.Errorf("point d'intégration router.go introuvable : middleware chain")
	}
	src = strings.Replace(src, oldMiddleware, newMiddleware, 1)

	if err := os.WriteFile(routerPath, []byte(src), 0o644); err != nil {
		return fmt.Errorf("écrire router.go : %w", err)
	}

	return nil
}
