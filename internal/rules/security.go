package rules

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Demetrius-ch/forgekit/internal/config"
	"github.com/Demetrius-ch/forgekit/internal/report"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(api[_-]?key|secret|password|token|private[_-]?key)\s*=\s*["']?[A-Za-z0-9_\-]{8,}`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`-----BEGIN (RSA |EC )?PRIVATE KEY-----`),
}

// SecuritySecretsRule detects potential hardcoded secrets.
// Limit: heuristic only — many false positives possible on placeholders.
type SecuritySecretsRule struct{}

func (SecuritySecretsRule) ID() string          { return "security.secrets" }
func (SecuritySecretsRule) Name() string        { return "Secrets potentiels" }
func (SecuritySecretsRule) Description() string { return "Heuristique de secrets en dur dans le code" }
func (SecuritySecretsRule) Category() string    { return "security" }
func (SecuritySecretsRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (r SecuritySecretsRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding
	_ = filepath.WalkDir(rctx.ProjectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.Contains(path, ".git") || strings.Contains(path, "vendor") {
			return nil
		}
		base := filepath.Base(path)
		if !strings.HasSuffix(base, ".go") && base != ".env" && base != ".env.example" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		if strings.Contains(content, "changeme") || strings.Contains(content, "your_") {
			return nil
		}
		for _, re := range secretPatterns {
			if loc := re.FindStringIndex(content); loc != nil {
				rel, _ := filepath.Rel(rctx.ProjectRoot, path)
				findings = append(findings, report.Finding{
					ID: "security.secrets", Category: "security", Severity: report.SeverityWarning,
					File: rel, Message: "Secret potentiel détecté (heuristique)",
					Explanation: "ForgeKit n'est pas un scanner SAST ; vérifiez manuellement",
					Suggestion:  "Utilisez des variables d'environnement ou un gestionnaire de secrets",
				})
				break
			}
		}
		return nil
	})
	return findings, nil
}

// SecuritySensitiveFilesRule warns when sensitive files may be committed.
type SecuritySensitiveFilesRule struct{}

func (SecuritySensitiveFilesRule) ID() string          { return "security.files" }
func (SecuritySensitiveFilesRule) Name() string        { return "Fichiers sensibles" }
func (SecuritySensitiveFilesRule) Description() string { return "Détecte .env versionné" }
func (SecuritySensitiveFilesRule) Category() string    { return "security" }
func (SecuritySensitiveFilesRule) Severity() report.Severity {
	return report.SeverityError
}

func (SecuritySensitiveFilesRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	envPath := filepath.Join(rctx.ProjectRoot, ".env")
	if _, err := os.Stat(envPath); err == nil {
		gitignore, _ := os.ReadFile(filepath.Join(rctx.ProjectRoot, ".gitignore"))
		if !strings.Contains(string(gitignore), ".env") {
			return []report.Finding{{
				ID: "security.files", Category: "security", Severity: report.SeverityError,
				File: ".env", Message: ".env présent mais non ignoré par .gitignore",
				Suggestion: "Ajoutez .env à .gitignore et ne commitez jamais de secrets",
			}}, nil
		}
		return []report.Finding{{
			ID: "security.files", Category: "security", Severity: report.SeverityWarning,
			File: ".env", Message: ".env présent localement",
			Explanation: "Normal en dev ; assurez-vous qu'il n'est pas commité",
		}}, nil
	}
	return nil, nil
}

// SecurityCORSRule detects permissive CORS patterns in source.
type SecurityCORSRule struct{}

func (SecurityCORSRule) ID() string          { return "security.cors" }
func (SecurityCORSRule) Name() string        { return "CORS permissif" }
func (SecurityCORSRule) Description() string { return "Détecte Access-Control-Allow-Origin: *" }
func (SecurityCORSRule) Category() string    { return "security" }
func (SecurityCORSRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (r SecurityCORSRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding
	_ = filepath.WalkDir(rctx.ProjectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, _ := os.ReadFile(path)
		if strings.Contains(string(data), "AllowOrigin") && strings.Contains(string(data), `"*"`) {
			rel, _ := filepath.Rel(rctx.ProjectRoot, path)
			findings = append(findings, report.Finding{
				ID: "security.cors", Category: "security", Severity: report.SeverityWarning,
				File: rel, Message: "CORS permissif détecté (origine wildcard)",
				Suggestion: "Restreignez les origines autorisées en production",
			})
		}
		return nil
	})
	return findings, nil
}

type GracefulShutdownRule struct{}

func (GracefulShutdownRule) ID() string   { return "project.shutdown" }
func (GracefulShutdownRule) Name() string { return "Arrêt gracieux" }
func (GracefulShutdownRule) Description() string {
	return "Vérifie la présence d’un shutdown gracieux dans le point d’entrée"
}
func (GracefulShutdownRule) Category() string { return "project" }
func (GracefulShutdownRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (GracefulShutdownRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	mainPath := filepath.Join(rctx.ProjectRoot, "cmd", "server", "main.go")
	data, err := os.ReadFile(mainPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	content := string(data)
	if strings.Contains(content, "signal.Notify") && strings.Contains(content, "Shutdown") {
		return []report.Finding{{
			ID: "project.shutdown", Category: "pass", Severity: report.SeverityInfo,
			Message: "Arrêt gracieux configuré",
		}}, nil
	}
	return []report.Finding{{
		ID: "project.shutdown", Category: "project", Severity: report.SeverityWarning,
		File: filepath.Join("cmd", "server", "main.go"), Message: "Aucun shutdown gracieux détecté",
		Suggestion: "Ajoutez signal.Notify et un appel à server.Shutdown()",
	}}, nil
}

func DoctorRules() *Registry {
	return NewRegistry(
		GoVersionRule{},
		GitRule{},
		DockerRule{},
		GoModRule{},
		EnvFileRule{},
		SecuritySensitiveFilesRule{},
		GracefulShutdownRule{},
	)
}

func AnalyzeRules(cfg configLoader) *Registry {
	return NewRegistry(
		GoModRule{},
		ArchitectureRule{Rules: cfg.ArchitectureRules()},
		SecuritySecretsRule{},
		SecurityCORSRule{},
	)
}

func CheckRules(cfg configLoader) *Registry {
	return NewRegistry(
		ArchitectureRule{Rules: cfg.ArchitectureRules()},
	)
}

type configLoader interface {
	ArchitectureRules() []config.ArchitectureRule
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
