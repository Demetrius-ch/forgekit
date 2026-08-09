package rules

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/forgekit/forgekit/internal/analyzer"
	"github.com/forgekit/forgekit/internal/report"
)

type GoVersionRule struct{}

func (GoVersionRule) ID() string          { return "env.go.version" }
func (GoVersionRule) Name() string        { return "Version Go" }
func (GoVersionRule) Description() string { return "Vérifie que Go est installé" }
func (GoVersionRule) Category() string    { return "environment" }
func (GoVersionRule) Severity() report.Severity {
	return report.SeverityError
}

func (GoVersionRule) Run(_ context.Context, _ Context) ([]report.Finding, error) {
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

func (DockerRule) ID() string          { return "env.docker" }
func (DockerRule) Name() string        { return "Docker" }
func (DockerRule) Description() string { return "Vérifie le daemon Docker" }
func (DockerRule) Category() string    { return "environment" }
func (DockerRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (DockerRule) Run(_ context.Context, _ Context) ([]report.Finding, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return []report.Finding{{
			ID: "env.docker", Category: "environment", Severity: report.SeverityWarning,
			Message: "Docker CLI introuvable", Suggestion: "Installez Docker pour le développement local",
		}}, nil
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		return []report.Finding{{
			ID: "env.docker", Category: "environment", Severity: report.SeverityWarning,
			Message: "Docker est installé mais le daemon ne répond pas", Suggestion: "Démarrez Docker Desktop ou le service docker",
		}}, nil
	}
	return []report.Finding{{
		ID: "env.docker", Category: "pass", Severity: report.SeverityInfo,
		Message: "Docker daemon accessible",
	}}, nil
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
	exKeys, err := parseEnvKeys(example)
	if err != nil {
		return nil, err
	}
	if len(exKeys) == 0 {
		return nil, nil
	}
	envKeys, err := parseEnvKeys(env)
	if err != nil {
		return []report.Finding{{
			ID: "project.env", Category: "project", Severity: report.SeverityWarning,
			Message: ".env manquant", Suggestion: "cp .env.example .env",
		}}, nil
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
