package rules

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Demetrius-ch/forgekit/internal/analyzer"
	"github.com/Demetrius-ch/forgekit/internal/config"
	"github.com/Demetrius-ch/forgekit/internal/report"
)

type GoVersionRule struct{}

func (GoVersionRule) ID() string          { return "env.go.version" }
func (GoVersionRule) Name() string        { return "Version Go" }
func (GoVersionRule) Description() string { return "Vérifie que Go est installé" }
func (GoVersionRule) Category() string    { return "environment" }
func (GoVersionRule) Severity() report.Severity {
	return report.SeverityError
}

func (GoVersionRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	// Skip for non-Go projects
	if rctx.Language != "" && rctx.Language != "go" {
		return nil, nil
	}
	out, err := exec.Command("go", "version").CombinedOutput()
	if err != nil {
		return []report.Finding{{
			ID: "env.go.version", Category: "environment", Severity: report.SeverityError,
			Message: "Go n'est pas installé ou inaccessible", Suggestion: "Installez Go 1.22+ et vérifiez PATH",
		}}, nil
	}
	return []report.Finding{{
		ID: "env.go.version", Category: "pass", Severity: report.SeverityInfo,
		Message: strings.TrimSpace(string(out)),
	}}, nil
}

type NodeVersionRule struct{}

func (NodeVersionRule) ID() string          { return "env.node.version" }
func (NodeVersionRule) Name() string        { return "Version Node.js" }
func (NodeVersionRule) Description() string { return "Vérifie que Node.js est installé" }
func (NodeVersionRule) Category() string    { return "environment" }
func (NodeVersionRule) Severity() report.Severity {
	return report.SeverityError
}

func (NodeVersionRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	// Only for Node.js projects
	if rctx.Language != "typescript" && rctx.Language != "javascript" {
		return nil, nil
	}
	out, err := exec.Command("node", "--version").CombinedOutput()
	if err != nil {
		return []report.Finding{{
			ID: "env.node.version", Category: "environment", Severity: report.SeverityError,
			Message: "Node.js n'est pas installé ou inaccessible", Suggestion: "Installez Node.js 18+ et vérifiez PATH",
		}}, nil
	}
	return []report.Finding{{
		ID: "env.node.version", Category: "pass", Severity: report.SeverityInfo,
		Message: strings.TrimSpace(string(out)),
	}}, nil
}

type NpmRule struct{}

func (NpmRule) ID() string          { return "env.npm" }
func (NpmRule) Name() string        { return "npm" }
func (NpmRule) Description() string { return "Vérifie que npm est installé" }
func (NpmRule) Category() string    { return "environment" }
func (NpmRule) Severity() report.Severity {
	return report.SeverityError
}

func (NpmRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	// Only for Node.js projects
	if rctx.Language != "typescript" && rctx.Language != "javascript" {
		return nil, nil
	}
	out, err := exec.Command("npm", "--version").CombinedOutput()
	if err != nil {
		return []report.Finding{{
			ID: "env.npm", Category: "environment", Severity: report.SeverityError,
			Message: "npm n'est pas installé ou inaccessible", Suggestion: "Installez npm avec Node.js",
		}}, nil
	}
	return []report.Finding{{
		ID: "env.npm", Category: "pass", Severity: report.SeverityInfo,
		Message: "npm " + strings.TrimSpace(string(out)),
	}}, nil
}

type PythonVersionRule struct{}

func (PythonVersionRule) ID() string          { return "env.python.version" }
func (PythonVersionRule) Name() string        { return "Version Python" }
func (PythonVersionRule) Description() string { return "Vérifie que Python est installé" }
func (PythonVersionRule) Category() string    { return "environment" }
func (PythonVersionRule) Severity() report.Severity {
	return report.SeverityError
}

func (PythonVersionRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	// Only for Python projects
	if rctx.Language != "python" {
		return nil, nil
	}
	out, err := exec.Command("python3", "--version").CombinedOutput()
	if err != nil {
		return []report.Finding{{
			ID: "env.python.version", Category: "environment", Severity: report.SeverityError,
			Message: "Python n'est pas installé ou inaccessible", Suggestion: "Installez Python 3.11+ et vérifiez PATH",
		}}, nil
	}
	return []report.Finding{{
		ID: "env.python.version", Category: "pass", Severity: report.SeverityInfo,
		Message: strings.TrimSpace(string(out)),
	}}, nil
}

type PipRule struct{}

func (PipRule) ID() string          { return "env.pip" }
func (PipRule) Name() string        { return "pip" }
func (PipRule) Description() string { return "Vérifie que pip est installé" }
func (PipRule) Category() string    { return "environment" }
func (PipRule) Severity() report.Severity {
	return report.SeverityError
}

func (PipRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	// Only for Python projects
	if rctx.Language != "python" {
		return nil, nil
	}
	out, err := exec.Command("pip3", "--version").CombinedOutput()
	if err != nil {
		return []report.Finding{{
			ID: "env.pip", Category: "environment", Severity: report.SeverityError,
			Message: "pip n'est pas installé ou inaccessible", Suggestion: "Installez pip avec Python",
		}}, nil
	}
	return []report.Finding{{
		ID: "env.pip", Category: "pass", Severity: report.SeverityInfo,
		Message: "pip " + strings.TrimSpace(string(out)),
	}}, nil
}

type GitRule struct{}

func (GitRule) ID() string          { return "env.git" }
func (GitRule) Name() string        { return "Git" }
func (GitRule) Description() string { return "Vérifie que Git est disponible" }
func (GitRule) Category() string    { return "environment" }
func (GitRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (GitRule) Run(_ context.Context, _ Context) ([]report.Finding, error) {
	out, err := exec.Command("git", "--version").CombinedOutput()
	if err != nil {
		return []report.Finding{{
			ID: "env.git", Category: "environment", Severity: report.SeverityWarning,
			Message: "Git n'est pas installé", Suggestion: "Installez git pour le versioning",
		}}, nil
	}
	return []report.Finding{{
		ID: "env.git", Category: "pass", Severity: report.SeverityInfo,
		Message: strings.TrimSpace(string(out)),
	}}, nil
}

type DockerRule struct{}

func (DockerRule) ID() string          { return "docker.status" }
func (DockerRule) Name() string        { return "Docker" }
func (DockerRule) Description() string { return "Vérifie la configuration Docker du projet" }
func (DockerRule) Category() string    { return "docker" }
func (DockerRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (DockerRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	if !hasDockerConfig(rctx.ProjectRoot) {
		return []report.Finding{{
			ID: "docker.project.missing", Category: "docker", Severity: report.SeverityWarning,
			Message:    "Aucun fichier Docker détecté dans le projet",
			Suggestion: "Ajoutez un Dockerfile ou docker/docker-compose.yml si vous souhaitez containeriser le projet",
		}}, nil
	}

	if _, err := exec.LookPath("docker"); err != nil {
		return []report.Finding{{
			ID: "docker.cli.missing", Category: "docker", Severity: report.SeverityWarning,
			Message:    "Docker CLI introuvable",
			Suggestion: "Installez Docker pour exécuter les images locales",
		}}, nil
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		return []report.Finding{{
			ID: "docker.daemon.stopped", Category: "docker", Severity: report.SeverityWarning,
			Message:    "Docker est configuré mais le daemon ne répond pas",
			Suggestion: "Démarrez le service Docker ou Docker Desktop",
		}}, nil
	}
	return []report.Finding{{
		ID: "docker.ready", Category: "docker", Severity: report.SeverityInfo,
		Message: "Docker configuré et daemon accessible",
	}}, nil
}

func hasDockerConfig(root string) bool {
	if _, err := os.Stat(filepath.Join(root, "Dockerfile")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(root, "docker", "docker-compose.yml")); err == nil {
		return true
	}
	return false
}

type GoModRule struct{}

func (GoModRule) ID() string          { return "project.gomod" }
func (GoModRule) Name() string        { return "go.mod" }
func (GoModRule) Description() string { return "Vérifie la présence de go.mod" }
func (GoModRule) Category() string    { return "project" }
func (GoModRule) Severity() report.Severity {
	return report.SeverityError
}

func (GoModRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	// Skip for non-Go projects
	if rctx.Language != "" && rctx.Language != "go" {
		return nil, nil
	}
	if _, err := os.Stat(filepath.Join(rctx.ProjectRoot, "go.mod")); err != nil {
		return []report.Finding{{
			ID: "project.gomod", Category: "project", Severity: report.SeverityError,
			Message: "go.mod introuvable", Suggestion: "Exécutez forge init ou go mod init",
		}}, nil
	}
	p, err := analyzer.LoadProject(rctx.ProjectRoot)
	if err != nil {
		return []report.Finding{{
			ID: "project.gomod", Category: "project", Severity: report.SeverityError,
			Message: err.Error(),
		}}, nil
	}
	return []report.Finding{{
		ID: "project.gomod", Category: "pass", Severity: report.SeverityInfo,
		Message: fmt.Sprintf("Module Go : %s", p.Module),
	}}, nil
}

type PackageJSONRule struct{}

func (PackageJSONRule) ID() string          { return "project.packagejson" }
func (PackageJSONRule) Name() string        { return "package.json" }
func (PackageJSONRule) Description() string { return "Vérifie la présence de package.json" }
func (PackageJSONRule) Category() string    { return "project" }
func (PackageJSONRule) Severity() report.Severity {
	return report.SeverityError
}

func (PackageJSONRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	// Only for Node.js projects
	if rctx.Language != "typescript" && rctx.Language != "javascript" {
		return nil, nil
	}
	if _, err := os.Stat(filepath.Join(rctx.ProjectRoot, "package.json")); err != nil {
		return []report.Finding{{
			ID: "project.packagejson", Category: "project", Severity: report.SeverityError,
			Message: "package.json introuvable", Suggestion: "Exécutez forge init ou npm init",
		}}, nil
	}
	return []report.Finding{{
		ID: "project.packagejson", Category: "pass", Severity: report.SeverityInfo,
		Message: "package.json présent",
	}}, nil
}

type RequirementsTxtRule struct{}

func (RequirementsTxtRule) ID() string          { return "project.requirementstxt" }
func (RequirementsTxtRule) Name() string        { return "requirements.txt" }
func (RequirementsTxtRule) Description() string { return "Vérifie la présence de requirements.txt" }
func (RequirementsTxtRule) Category() string    { return "project" }
func (RequirementsTxtRule) Severity() report.Severity {
	return report.SeverityError
}

func (RequirementsTxtRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	// Only for Python projects
	if rctx.Language != "python" {
		return nil, nil
	}
	if _, err := os.Stat(filepath.Join(rctx.ProjectRoot, "requirements.txt")); err != nil {
		return []report.Finding{{
			ID: "project.requirementstxt", Category: "project", Severity: report.SeverityError,
			Message: "requirements.txt introuvable", Suggestion: "Exécutez forge init ou pip freeze > requirements.txt",
		}}, nil
	}
	return []report.Finding{{
		ID: "project.requirementstxt", Category: "pass", Severity: report.SeverityInfo,
		Message: "requirements.txt présent",
	}}, nil
}

type EnvFileRule struct{}

func (EnvFileRule) ID() string          { return "project.env" }
func (EnvFileRule) Name() string        { return "Variables d'environnement" }
func (EnvFileRule) Description() string { return "Compare .env et .env.example" }
func (EnvFileRule) Category() string    { return "project" }
func (EnvFileRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (EnvFileRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	example := filepath.Join(rctx.ProjectRoot, ".env.example")
	env := filepath.Join(rctx.ProjectRoot, ".env")
	if _, err := os.Stat(example); err != nil {
		if os.IsNotExist(err) {
			return []report.Finding{{
				ID: "project.env.example.missing", Category: "project", Severity: report.SeverityWarning,
				Message:    ".env.example absent",
				Suggestion: "Ajoutez un fichier .env.example pour documenter les variables d'environnement",
			}}, nil
		}
		return nil, err
	}

	exKeys, err := parseEnvKeys(example)
	if err != nil {
		return nil, err
	}
	if len(exKeys) == 0 {
		return []report.Finding{{
			ID: "project.env.example.empty", Category: "project", Severity: report.SeverityWarning,
			Message:    ".env.example vide",
			Suggestion: "Remplissez .env.example avec les variables nécessaires",
		}}, nil
	}
	envKeys, err := parseEnvKeys(env)
	if err != nil {
		if os.IsNotExist(err) {
			return []report.Finding{{
				ID: "project.env", Category: "project", Severity: report.SeverityWarning,
				Message:    ".env manquant",
				Suggestion: "cp .env.example .env",
			}}, nil
		}
		return nil, err
	}
	var missing []string
	for k := range exKeys {
		if _, ok := envKeys[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return []report.Finding{{
			ID: "project.env", Category: "project", Severity: report.SeverityWarning,
			Message:    fmt.Sprintf("Clés manquantes dans .env : %s", strings.Join(missing, ", ")),
			Suggestion: "Copiez les clés depuis .env.example",
		}}, nil
	}
	return []report.Finding{{
		ID: "project.env", Category: "pass", Severity: report.SeverityInfo,
		Message: "Fichier .env complet par rapport à .env.example",
	}}, nil
}

func parseEnvKeys(path string) (map[string]struct{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	keys := make(map[string]struct{})
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) > 0 {
			keys[strings.TrimSpace(parts[0])] = struct{}{}
		}
	}
	return keys, nil
}

// ConfigurationLintRule checks for common configuration issues
type ConfigurationLintRule struct{}

func (ConfigurationLintRule) ID() string   { return "config.lint" }
func (ConfigurationLintRule) Name() string { return "Qualité de la configuration" }
func (ConfigurationLintRule) Description() string {
	return "Vérifie les problèmes courants dans les fichiers de configuration"
}
func (ConfigurationLintRule) Category() string { return "configuration" }
func (ConfigurationLintRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (ConfigurationLintRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding

	// Check go.mod for common issues
	goModPath := filepath.Join(rctx.ProjectRoot, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err == nil {
		content := string(data)
		lines := strings.Split(content, "\n")

		// Check for replace directives
		hasReplace := false
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "replace ") {
				hasReplace = true
				break
			}
		}
		if hasReplace {
			findings = append(findings, report.Finding{
				ID:          "config.replace_directive",
				Category:    "configuration",
				Severity:    report.SeverityWarning,
				File:        "go.mod",
				Message:     "Directive 'replace' détectée dans go.mod",
				Explanation: "Les directives replace peuvent causer des problèmes de reproductibilité",
				Suggestion:  "Utilisez des versions spécifiques ou des forks officiels",
			})
		}

		// Check for missing module declaration
		if !strings.HasPrefix(strings.TrimSpace(content), "module ") {
			findings = append(findings, report.Finding{
				ID:         "config.gomod.missing_module",
				Category:   "configuration",
				Severity:   report.SeverityError,
				File:       "go.mod",
				Message:    "Déclaration de module manquante dans go.mod",
				Suggestion: "Ajoutez 'module <votre-module>' au début du fichier",
			})
		}
	}

	// Check forge.yaml for completeness
	forgeYamlPath := filepath.Join(rctx.ProjectRoot, "forge.yaml")
	data, _ = os.ReadFile(forgeYamlPath)
	if err == nil {
		content := string(data)
		hasArchitecture := strings.Contains(content, "architecture:")
		hasProject := strings.Contains(content, "project:")
		hasFeatures := strings.Contains(content, "features:")

		if !hasArchitecture {
			findings = append(findings, report.Finding{
				ID:         "config.forgeyaml.missing_architecture",
				Category:   "configuration",
				Severity:   report.SeverityWarning,
				File:       "forge.yaml",
				Message:    "Section 'architecture' manquante dans forge.yaml",
				Suggestion: "Ajoutez la section architecture avec vos règles personnalisées",
			})
		}
		if !hasProject {
			findings = append(findings, report.Finding{
				ID:         "config.forgeyaml.missing_project",
				Category:   "configuration",
				Severity:   report.SeverityWarning,
				File:       "forge.yaml",
				Message:    "Section 'project' manquante dans forge.yaml",
				Suggestion: "Ajoutez la section project avec les métadonnées du projet",
			})
		}
		if !hasFeatures {
			findings = append(findings, report.Finding{
				ID:         "config.forgeyaml.missing_features",
				Category:   "configuration",
				Severity:   report.SeverityInfo,
				File:       "forge.yaml",
				Message:    "Section 'features' manquante dans forge.yaml",
				Suggestion: "Ajoutez la section features pour tracker les features installées",
			})
		}
	}

	return findings, nil
}

// DockerBestPracticesRule checks Docker configuration best practices
type DockerBestPracticesRule struct{}

func (DockerBestPracticesRule) ID() string   { return "docker.best_practices" }
func (DockerBestPracticesRule) Name() string { return "Bonnes pratiques Docker" }
func (DockerBestPracticesRule) Description() string {
	return "Vérifie les bonnes pratiques dans les fichiers Docker"
}
func (DockerBestPracticesRule) Category() string { return "docker" }
func (DockerBestPracticesRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (DockerBestPracticesRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding

	dockerComposePath := filepath.Join(rctx.ProjectRoot, "docker", "docker-compose.yml")
	data, err := os.ReadFile(dockerComposePath)
	if err != nil {
		if os.IsNotExist(err) {
			return findings, nil
		}
		return nil, err
	}

	content := string(data)

	// Check for health checks
	if !strings.Contains(content, "healthcheck:") {
		findings = append(findings, report.Finding{
			ID:          "docker.missing_healthcheck",
			Category:    "docker",
			Severity:    report.SeverityWarning,
			File:        "docker/docker-compose.yml",
			Message:     "Aucun healthcheck configuré dans docker-compose.yml",
			Explanation: "Les healthchecks permettent à Docker de vérifier l'état des services",
			Suggestion:  "Ajoutez une section healthcheck aux services (ex: pg_isready pour PostgreSQL)",
		})
	}

	// Check for resource limits
	if !strings.Contains(content, "deploy:") && !strings.Contains(content, "limits:") {
		findings = append(findings, report.Finding{
			ID:          "docker.missing_limits",
			Category:    "docker",
			Severity:    report.SeverityWarning,
			File:        "docker/docker-compose.yml",
			Message:     "Aucune limite de ressources (CPU/mémoire) configurée",
			Explanation: "Les limites empêchent un service de monopoliser les ressources",
			Suggestion:  "Ajoutez 'deploy: resources: limits:' sous chaque service",
		})
	}

	// Check for PostgreSQL healthcheck
	if strings.Contains(content, "postgres") || strings.Contains(content, "postgresql") {
		if !strings.Contains(content, "pg_isready") {
			findings = append(findings, report.Finding{
				ID:         "docker.postgres_healthcheck",
				Category:   "docker",
				Severity:   report.SeverityWarning,
				File:       "docker/docker-compose.yml",
				Message:    "Service PostgreSQL sans healthcheck pg_isready",
				Suggestion: "Ajoutez 'test: [\"CMD-SHELL\", \"pg_isready -U postgres\"]' dans healthcheck",
			})
		}
	}

	// Check for version pinning in images
	if strings.Contains(content, "image:") && !strings.Contains(content, ":") {
		findings = append(findings, report.Finding{
			ID:         "docker.unpinned_image",
			Category:   "docker",
			Severity:   report.SeverityWarning,
			File:       "docker/docker-compose.yml",
			Message:    "Images Docker sans tag de version (latest implicite)",
			Suggestion: "Utilisez des tags de version explicites (ex: postgres:16-alpine)",
		})
	}

	return findings, nil
}

// DocumentationCompletenessRule checks documentation completeness
type DocumentationCompletenessRule struct{}

func (DocumentationCompletenessRule) ID() string   { return "docs.completeness" }
func (DocumentationCompletenessRule) Name() string { return "Complétude de la documentation" }
func (DocumentationCompletenessRule) Description() string {
	return "Vérifie la présence et la qualité de la documentation"
}
func (DocumentationCompletenessRule) Category() string { return "documentation" }
func (DocumentationCompletenessRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (DocumentationCompletenessRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding

	// Check for README.md
	readmePath := filepath.Join(rctx.ProjectRoot, "README.md")
	if _, err := os.Stat(readmePath); err != nil {
		if os.IsNotExist(err) {
			findings = append(findings, report.Finding{
				ID:         "docs.readme.missing",
				Category:   "documentation",
				Severity:   report.SeverityWarning,
				Message:    "README.md manquant",
				Suggestion: "Créez un README.md avec description, installation, usage",
			})
		}
	} else {
		// Check README content quality
		data, _ := os.ReadFile(readmePath)
		content := string(data)
		if len(content) < 200 {
			findings = append(findings, report.Finding{
				ID:         "docs.readme.minimal",
				Category:   "documentation",
				Severity:   report.SeverityWarning,
				File:       "README.md",
				Message:    "README.md très court (< 200 caractères)",
				Suggestion: "Ajoutez description, installation, configuration, usage, tests",
			})
		}
		if !strings.Contains(strings.ToLower(content), "install") {
			findings = append(findings, report.Finding{
				ID:         "docs.readme.no_install",
				Category:   "documentation",
				Severity:   report.SeverityInfo,
				File:       "README.md",
				Message:    "README.md ne mentionne pas l'installation",
				Suggestion: "Ajoutez une section 'Installation'",
			})
		}
		if !strings.Contains(strings.ToLower(content), "usage") && !strings.Contains(strings.ToLower(content), "utilisation") {
			findings = append(findings, report.Finding{
				ID:         "docs.readme.no_usage",
				Category:   "documentation",
				Severity:   report.SeverityInfo,
				File:       "README.md",
				Message:    "README.md ne mentionne pas l'utilisation",
				Suggestion: "Ajoutez une section 'Usage' ou 'Utilisation'",
			})
		}
	}

	// Check for .env.example
	envExamplePath := filepath.Join(rctx.ProjectRoot, ".env.example")
	if _, err := os.Stat(envExamplePath); err != nil {
		if os.IsNotExist(err) {
			findings = append(findings, report.Finding{
				ID:         "docs.env.example.missing",
				Category:   "documentation",
				Severity:   report.SeverityWarning,
				Message:    ".env.example manquant",
				Suggestion: "Créez un .env.example documentant toutes les variables d'environnement",
			})
		}
	} else {
		data, _ := os.ReadFile(envExamplePath)
		content := string(data)
		if strings.TrimSpace(content) == "" {
			findings = append(findings, report.Finding{
				ID:         "docs.env.example.empty",
				Category:   "documentation",
				Severity:   report.SeverityWarning,
				File:       ".env.example",
				Message:    ".env.example vide",
				Suggestion: "Documentez toutes les variables d'environnement requises",
			})
		}
		// Check for comments/documentation in .env.example
		if !strings.Contains(content, "#") {
			findings = append(findings, report.Finding{
				ID:         "docs.env.example.no_comments",
				Category:   "documentation",
				Severity:   report.SeverityInfo,
				File:       ".env.example",
				Message:    ".env.example sans commentaires explicatifs",
				Suggestion: "Ajoutez des commentaires pour expliquer chaque variable",
			})
		}
	}

	// Check for API documentation (Swagger/OpenAPI)
	swaggerPath := filepath.Join(rctx.ProjectRoot, "internal", "swagger")
	if _, err := os.Stat(swaggerPath); err == nil {
		findings = append(findings, report.Finding{
			ID:       "docs.swagger.present",
			Category: "documentation",
			Severity: report.SeverityInfo,
			Message:  "Documentation Swagger/OpenAPI détectée",
		})
	}

	// Check for CHANGELOG
	changelogPath := filepath.Join(rctx.ProjectRoot, "CHANGELOG.md")
	if _, err := os.Stat(changelogPath); err == nil {
		findings = append(findings, report.Finding{
			ID:       "docs.changelog.present",
			Category: "documentation",
			Severity: report.SeverityInfo,
			File:     "CHANGELOG.md",
			Message:  "CHANGELOG.md présent",
		})
	}

	// Check for LICENSE
	licensePath := filepath.Join(rctx.ProjectRoot, "LICENSE")
	if _, err := os.Stat(licensePath); err == nil {
		findings = append(findings, report.Finding{
			ID:       "docs.license.present",
			Category: "documentation",
			Severity: report.SeverityInfo,
			File:     "LICENSE",
			Message:  "LICENSE présent",
		})
	}

	return findings, nil
}

type configLoader interface {
	ArchitectureRules() []config.ArchitectureRule
}

func AnalyzeRules(cfg configLoader) *Registry {
	return NewRegistry(
		GoModRule{},
		ArchitectureRule{Rules: cfg.ArchitectureRules()},
		SecuritySecretsRule{},
		SecurityCORSRule{},
		SecuritySQLInjectionRule{},
		SecurityXSSRule{},
		SecurityHardcodedIPRule{},
		SecurityWeakCryptoRule{},
	)
}

func CheckRules(cfg configLoader) *Registry {
	return NewRegistry(
		ArchitectureRule{Rules: cfg.ArchitectureRules()},
	)
}

func DoctorRules() *Registry {
	return NewRegistry(
		GoVersionRule{},
		NodeVersionRule{},
		NpmRule{},
		PythonVersionRule{},
		PipRule{},
		GitRule{},
		DockerRule{},
		GoModRule{},
		PackageJSONRule{},
		RequirementsTxtRule{},
		EnvFileRule{},
		SecuritySensitiveFilesRule{},
		GracefulShutdownRule{},
		PostgreSQLRule{},
		DependenciesRule{},
		FeaturesConsistencyRule{},
		ConfigValidationRule{},
		ConfigurationLintRule{},
		DockerBestPracticesRule{},
		DocumentationCompletenessRule{},
	)
}

// StaticConfigLoader wraps fixed rules for tests.
type StaticConfigLoader struct {
	Rules []config.ArchitectureRule
}

func (s StaticConfigLoader) ArchitectureRules() []config.ArchitectureRule {
	if len(s.Rules) == 0 {
		return config.DefaultArchitectureRules()
	}
	return s.Rules
}
