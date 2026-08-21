package rules

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Demetrius-ch/forgekit/internal/feature"
	"github.com/Demetrius-ch/forgekit/internal/report"
)

// Secret patterns with improved precision - avoiding common false positives
var secretPatterns = []*regexp.Regexp{
	// AWS Access Key ID
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	// AWS Secret Access Key
	regexp.MustCompile(`[A-Za-z0-9/+=]{40}`),
	// Private keys
	regexp.MustCompile(`-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
	// Generic high-entropy strings that look like secrets (min 20 chars, high entropy)
	regexp.MustCompile(`(?i)(api[_-]?key|secret|password|token|private[_-]?key|access[_-]?token)\s*[:=]\s*["'][A-Za-z0-9_\-\.]{20,}["']`),
	// Generic assignment with high entropy value (base64-like, 32+ chars)
	regexp.MustCompile(`(?i)(api[_-]?key|secret|password|token|private[_-]?key|access[_-]?token)\s*[:=]\s*[A-Za-z0-9+/=]{32,}`),
	// JWT tokens
	regexp.MustCompile(`eyJ[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}`),
	// Database connection strings with passwords
	regexp.MustCompile(`(?i)(postgres|mysql|mongodb|redis)://[^:]+:[^@]+@`),
	// Generic key=value with high entropy (32+ chars)
	regexp.MustCompile(`(?i)(key|secret|password|token|api[_-]?key)\s*[:=]\s*[A-Za-z0-9+/=_\-]{32,}`),
}

// Common false positive patterns to exclude
var falsePositivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(changeme|your_|example|test|dummy|placeholder|xxx|todo|fixme)`),
	regexp.MustCompile(`(?i)(password|secret|token)\s*[:=]\s*["']?\$`), // Template variables like ${PASSWORD}
	regexp.MustCompile(`["']?\$\{`),                                    // Template syntax
	regexp.MustCompile(`(?i)test`),                                     // Test files
	regexp.MustCompile(`(?i)mock`),                                     // Mock data
	regexp.MustCompile(`(?i)example`),                                  // Example code
	regexp.MustCompile(`(?i)placeholder`),                              // Placeholders
	regexp.MustCompile(`(?i)dummy`),                                    // Dummy values
}

// SecuritySecretsRule detects potential hardcoded secrets with reduced false positives.
type SecuritySecretsRule struct{}

func (SecuritySecretsRule) ID() string   { return "security.secrets" }
func (SecuritySecretsRule) Name() string { return "Secrets potentiels" }
func (SecuritySecretsRule) Description() string {
	return "Détection de secrets en dur dans le code (heuristique améliorée)"
}
func (SecuritySecretsRule) Category() string { return "security" }
func (SecuritySecretsRule) Severity() report.Severity {
	return report.SeverityWarning
}

func isFalsePositive(content string) bool {
	for _, fp := range falsePositivePatterns {
		if fp.MatchString(content) {
			return true
		}
	}
	return false
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
		// Check Go files, .env, .env.example, and config files
		if !strings.HasSuffix(base, ".go") && base != ".env" && base != ".env.example" &&
			!strings.HasSuffix(base, ".yaml") && !strings.HasSuffix(base, ".yml") &&
			!strings.HasSuffix(base, ".json") && !strings.HasSuffix(base, ".toml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		// Skip if content contains obvious placeholder patterns
		if isFalsePositive(content) {
			return nil
		}
		for _, re := range secretPatterns {
			if loc := re.FindStringIndex(content); loc != nil {
				rel, _ := filepath.Rel(rctx.ProjectRoot, path)
				// Extract context around the match for better reporting
				start := max(0, loc[0]-50)
				end := min(len(content), loc[1]+50)
				context := strings.TrimSpace(content[start:end])
				context = strings.ReplaceAll(context, "\n", " ")
				findings = append(findings, report.Finding{
					ID:          "security.secrets",
					Category:    "security",
					Severity:    report.SeverityWarning,
					File:        rel,
					Message:     "Secret potentiel détecté dans le code",
					Explanation: fmt.Sprintf("Pattern suspect trouvé: ...%s... (ForgeKit n'est pas un scanner SAST complet)", context),
					Suggestion:  "Utilisez des variables d'environnement ou un gestionnaire de secrets (Vault, AWS Secrets Manager, etc.)",
				})
				break // Only report first match per file
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

type SecuritySQLInjectionRule struct{}

func (SecuritySQLInjectionRule) ID() string   { return "security.sql_injection" }
func (SecuritySQLInjectionRule) Name() string { return "Injection SQL potentielle" }
func (SecuritySQLInjectionRule) Description() string {
	return "Détecte les concaténations SQL dangereuses"
}
func (SecuritySQLInjectionRule) Category() string { return "security" }
func (SecuritySQLInjectionRule) Severity() report.Severity {
	return report.SeverityError
}

func (r SecuritySQLInjectionRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding
	_ = filepath.WalkDir(rctx.ProjectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, _ := os.ReadFile(path)
		content := string(data)

		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Patterns for potential SQL injection
		patterns := []*regexp.Regexp{
			regexp.MustCompile(`fmt\.Sprintf\([^)]*%[sq][^)]*\)`),        // fmt.Sprintf with %s or %q in SQL
			regexp.MustCompile(`fmt\.Printf\([^)]*%[sq][^)]*\)`),         // fmt.Printf with %s or %q in SQL
			regexp.MustCompile(`strings\.Join\([^)]*\)`),                 // strings.Join for SQL building
			regexp.MustCompile(`\+\s*["'].*SELECT|INSERT|UPDATE|DELETE`), // String concatenation with SQL
			regexp.MustCompile(`query\s*:=\s*["'].*%s.*["']`),            // Query with fmt placeholder
			regexp.MustCompile(`db\.Query\([^)]*\+`),                     // db.Query with concatenation
			regexp.MustCompile(`db\.Exec\([^)]*\+`),                      // db.Exec with concatenation
		}

		for _, re := range patterns {
			if re.MatchString(content) {
				rel, _ := filepath.Rel(rctx.ProjectRoot, path)
				findings = append(findings, report.Finding{
					ID:          "security.sql_injection",
					Category:    "security",
					Severity:    report.SeverityWarning,
					File:        rel,
					Message:     "Pattern potentiel d'injection SQL détecté",
					Explanation: "Utilisez des requêtes paramétrées (db.Query/Exec avec $1, $2) au lieu de concaténation",
					Suggestion:  "Remplacez la concaténation de chaînes par des requêtes préparées avec $1, $2, etc.",
				})
				break // Only report first match per file
			}
		}
		return nil
	})
	return findings, nil
}

// SecurityXSSRule detects potential XSS vulnerabilities.
type SecurityXSSRule struct{}

func (SecurityXSSRule) ID() string   { return "security.xss" }
func (SecurityXSSRule) Name() string { return "XSS potentiel" }
func (SecurityXSSRule) Description() string {
	return "Détecte les sorties non échappées dans les handlers HTTP"
}
func (SecurityXSSRule) Category() string { return "security" }
func (SecurityXSSRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (r SecurityXSSRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding
	_ = filepath.WalkDir(rctx.ProjectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, _ := os.ReadFile(path)
		content := string(data)

		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Look for direct user input in HTTP responses
		patterns := []*regexp.Regexp{
			regexp.MustCompile(`w\.Write\([^)]*r\.FormValue`),       // Direct form value in Write
			regexp.MustCompile(`w\.Write\([^)]*r\.URL\.Query`),      // Direct query param in Write
			regexp.MustCompile(`fmt\.Fprintf\(w,[^)]*r\.FormValue`), // FormValue in Fprintf to ResponseWriter
			regexp.MustCompile(`w\.WriteHeader\(\d+\)`),             // WriteHeader without content-type
		}

		for _, re := range patterns {
			if re.MatchString(content) {
				rel, _ := filepath.Rel(rctx.ProjectRoot, path)
				findings = append(findings, report.Finding{
					ID:          "security.xss",
					Category:    "security",
					Severity:    report.SeverityWarning,
					File:        rel,
					Message:     "Pattern potentiel XSS détecté",
					Explanation: "Les entrées utilisateur doivent être échappées avant d'être incluses dans les réponses HTTP",
					Suggestion:  "Utilisez html/template avec auto-escaping ou échappez manuellement les entrées",
				})
				break
			}
		}
		return nil
	})
	return findings, nil
}

// SecurityHardcodedIPRule detects hardcoded IP addresses.
type SecurityHardcodedIPRule struct{}

func (SecurityHardcodedIPRule) ID() string          { return "security.hardcoded_ip" }
func (SecurityHardcodedIPRule) Name() string        { return "Adresses IP en dur" }
func (SecurityHardcodedIPRule) Description() string { return "Détecte les adresses IP codées en dur" }
func (SecurityHardcodedIPRule) Category() string    { return "security" }
func (SecurityHardcodedIPRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (r SecurityHardcodedIPRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding
	ipPattern := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)
	_ = filepath.WalkDir(rctx.ProjectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, _ := os.ReadFile(path)
		content := string(data)

		// Look for hardcoded IPs (excluding localhost and common patterns)
		matches := ipPattern.FindAllString(content, -1)
		for _, match := range matches {
			if match != "127.0.0.1" && match != "0.0.0.0" && match != "::1" {
				rel, _ := filepath.Rel(rctx.ProjectRoot, path)
				findings = append(findings, report.Finding{
					ID:          "security.hardcoded_ip",
					Category:    "security",
					Severity:    report.SeverityWarning,
					File:        rel,
					Message:     fmt.Sprintf("Adresse IP en dur détectée: %s", match),
					Explanation: "Les adresses IP en dur empêchent la portabilité et la configuration par environnement",
					Suggestion:  "Utilisez des variables d'environnement pour les adresses de service",
				})
			}
		}
		return nil
	})
	return findings, nil
}

// SecurityWeakCryptoRule detects weak cryptographic practices.
type SecurityWeakCryptoRule struct{}

func (SecurityWeakCryptoRule) ID() string   { return "security.weak_crypto" }
func (SecurityWeakCryptoRule) Name() string { return "Cryptographie faible" }
func (SecurityWeakCryptoRule) Description() string {
	return "Détecte l'utilisation de MD5, SHA1, etc."
}
func (SecurityWeakCryptoRule) Category() string { return "security" }
func (SecurityWeakCryptoRule) Severity() report.Severity {
	return report.SeverityError
}

func (r SecurityWeakCryptoRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding
	_ = filepath.WalkDir(rctx.ProjectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, _ := os.ReadFile(path)
		content := string(data)

		weakAlgorithms := []struct {
			pattern *regexp.Regexp
			name    string
		}{
			{regexp.MustCompile(`crypto/md5`), "MD5"},
			{regexp.MustCompile(`crypto/sha1`), "SHA1"},
			{regexp.MustCompile(`md5\.Sum`), "MD5"},
			{regexp.MustCompile(`sha1\.Sum`), "SHA1"},
			{regexp.MustCompile(`DES|3DES`), "DES/3DES"},
			{regexp.MustCompile(`RC4`), "RC4"},
		}

		for _, algo := range weakAlgorithms {
			if algo.pattern.MatchString(content) {
				rel, _ := filepath.Rel(rctx.ProjectRoot, path)
				findings = append(findings, report.Finding{
					ID:          "security.weak_crypto",
					Category:    "security",
					Severity:    report.SeverityError,
					File:        rel,
					Message:     fmt.Sprintf("Algorithme cryptographique faible détecté: %s", algo.name),
					Explanation: fmt.Sprintf("%s est considéré comme cryptographiquement cassé", algo.name),
					Suggestion:  "Utilisez SHA-256, SHA-3, AES-GCM, ou ChaCha20-Poly1305",
				})
			}
		}
		return nil
	})
	return findings, nil
}

// PostgreSQLRule checks PostgreSQL connectivity and configuration.
type PostgreSQLRule struct{}

func (PostgreSQLRule) ID() string   { return "postgres.connection" }
func (PostgreSQLRule) Name() string { return "PostgreSQL" }
func (PostgreSQLRule) Description() string {
	return "Vérifie la connectivité et la configuration PostgreSQL"
}
func (PostgreSQLRule) Category() string { return "postgresql" }
func (PostgreSQLRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (r PostgreSQLRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding

	// Check if docker-compose.yml has PostgreSQL service
	dockerComposePath := filepath.Join(rctx.ProjectRoot, "docker", "docker-compose.yml")
	if _, err := os.Stat(dockerComposePath); err == nil {
		data, err := os.ReadFile(dockerComposePath)
		if err == nil {
			content := string(data)
			if strings.Contains(content, "postgres") || strings.Contains(content, "postgresql") {
				findings = append(findings, report.Finding{
					ID: "postgres.docker", Category: "pass", Severity: report.SeverityInfo,
					Message: "Service PostgreSQL configuré dans docker-compose.yml",
				})
			}
		}
	}

	// Check .env.example for PostgreSQL config
	envExamplePath := filepath.Join(rctx.ProjectRoot, ".env.example")
	if _, err := os.Stat(envExamplePath); err == nil {
		data, _ := os.ReadFile(envExamplePath)
		content := string(data)
		hasPostgresConfig := strings.Contains(content, "POSTGRES") || strings.Contains(content, "DATABASE_URL")
		if hasPostgresConfig {
			findings = append(findings, report.Finding{
				ID: "postgres.env", Category: "pass", Severity: report.SeverityInfo,
				Message: "Configuration PostgreSQL présente dans .env.example",
			})
		} else {
			findings = append(findings, report.Finding{
				ID: "postgres.env.missing", Category: "postgresql", Severity: report.SeverityWarning,
				Message:    "Variables PostgreSQL manquantes dans .env.example",
				Suggestion: "Ajoutez POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB",
			})
		}
	}

	// Try to connect to PostgreSQL if port is available (optional - only if not in CI)
	if _, err := exec.LookPath("psql"); err == nil {
		// Could try a connection test here, but keep it simple for now
		findings = append(findings, report.Finding{
			ID: "postgres.client", Category: "pass", Severity: report.SeverityInfo,
			Message: "Client PostgreSQL (psql) disponible",
		})
	}

	if len(findings) == 0 {
		findings = append(findings, report.Finding{
			ID: "postgres.not_configured", Category: "postgresql", Severity: report.SeverityWarning,
			Message:    "PostgreSQL non configuré détecté",
			Suggestion: "Vérifiez docker-compose.yml et .env.example pour la configuration PostgreSQL",
		})
	}

	return findings, nil
}

// DependenciesRule checks Go module dependencies.
type DependenciesRule struct{}

func (DependenciesRule) ID() string          { return "deps.gomod" }
func (DependenciesRule) Name() string        { return "Dépendances Go" }
func (DependenciesRule) Description() string { return "Vérifie go.mod et go.sum" }
func (DependenciesRule) Category() string    { return "dependencies" }
func (DependenciesRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (r DependenciesRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding

	// Check go.mod
	goModPath := filepath.Join(rctx.ProjectRoot, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		if os.IsNotExist(err) {
			findings = append(findings, report.Finding{
				ID: "deps.gomod.missing", Category: "dependencies", Severity: report.SeverityError,
				Message:    "go.mod manquant",
				Suggestion: "Exécutez 'go mod init' ou 'forge init'",
			})
			return findings, nil
		}
		return nil, err
	}

	// Check go.sum
	goSumPath := filepath.Join(rctx.ProjectRoot, "go.sum")
	if _, err := os.Stat(goSumPath); err != nil {
		if os.IsNotExist(err) {
			findings = append(findings, report.Finding{
				ID: "deps.gosum.missing", Category: "dependencies", Severity: report.SeverityWarning,
				Message:    "go.sum manquant",
				Suggestion: "Exécutez 'go mod tidy' pour générer go.sum",
			})
		} else {
			return nil, err
		}
	} else {
		findings = append(findings, report.Finding{
			ID: "deps.gosum.present", Category: "pass", Severity: report.SeverityInfo,
			Message: "go.sum présent",
		})
	}

	// Check if go.mod is valid by trying to parse it
	data, err := os.ReadFile(goModPath)
	if err != nil {
		findings = append(findings, report.Finding{
			ID: "deps.gomod.read_error", Category: "dependencies", Severity: report.SeverityError,
			Message: fmt.Sprintf("Impossible de lire go.mod: %v", err),
		})
	} else {
		content := string(data)
		if strings.HasPrefix(strings.TrimSpace(content), "module ") {
			findings = append(findings, report.Finding{
				ID: "deps.gomod.valid", Category: "pass", Severity: report.SeverityInfo,
				Message: "go.mod valide avec déclaration de module",
			})
		} else {
			findings = append(findings, report.Finding{
				ID: "deps.gomod.invalid", Category: "dependencies", Severity: report.SeverityError,
				Message: "go.mod invalide: déclaration de module manquante",
			})
		}
	}

	// Check for common outdated dependencies (optional)
	// Re-read go.mod for this check
	goModData, err := os.ReadFile(filepath.Join(rctx.ProjectRoot, "go.mod"))
	if err == nil {
		goModContent := string(goModData)
		if strings.Contains(goModContent, "github.com/go-chi/chi/v5") {
			findings = append(findings, report.Finding{
				ID: "deps.chi.present", Category: "pass", Severity: report.SeverityInfo,
				Message: "Router Chi détecté",
			})
		}
	}

	return findings, nil
}

// FeaturesConsistencyRule checks consistency of installed features.
type FeaturesConsistencyRule struct{}

func (FeaturesConsistencyRule) ID() string   { return "features.consistency" }
func (FeaturesConsistencyRule) Name() string { return "Cohérence des features" }
func (FeaturesConsistencyRule) Description() string {
	return "Vérifie que les features déclarées sont bien installées"
}
func (FeaturesConsistencyRule) Category() string { return "features" }
func (FeaturesConsistencyRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (r FeaturesConsistencyRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding

	features, err := feature.LoadFeatures(rctx.ProjectRoot)
	if err != nil {
		return nil, err
	}

	if len(features.Features) == 0 {
		findings = append(findings, report.Finding{
			ID: "features.none", Category: "features", Severity: report.SeverityInfo,
			Message: "Aucune feature installée",
		})
		return findings, nil
	}

	for _, feat := range features.Features {
		// Check if feature directory exists
		featurePath := filepath.Join(rctx.ProjectRoot, "internal", feat.Name)
		if _, err := os.Stat(featurePath); err != nil {
			if os.IsNotExist(err) {
				findings = append(findings, report.Finding{
					ID: "features.missing_dir", Category: "features", Severity: report.SeverityWarning,
					File: featurePath, Message: fmt.Sprintf("Feature %q déclarée mais répertoire manquant", feat.Name),
					Suggestion: "Réinstallez la feature avec 'forge add' ou supprimez-la de .forge/features.yaml",
				})
			}
		} else {
			findings = append(findings, report.Finding{
				ID: "features.dir.present", Category: "pass", Severity: report.SeverityInfo,
				Message: fmt.Sprintf("Feature %q: répertoire présent", feat.Name),
			})
		}

		// Check if feature is registered in ForgeKit registry (optional)
		// This would require access to the registry, skip for now
	}

	return findings, nil
}

// ConfigValidationRule validates project configuration.
type ConfigValidationRule struct{}

func (ConfigValidationRule) ID() string          { return "config.validation" }
func (ConfigValidationRule) Name() string        { return "Validation de la configuration" }
func (ConfigValidationRule) Description() string { return "Valide forge.yaml et .env.example" }
func (ConfigValidationRule) Category() string    { return "configuration" }
func (ConfigValidationRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (r ConfigValidationRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding

	// Check forge.yaml
	forgeYamlPath := filepath.Join(rctx.ProjectRoot, "forge.yaml")
	if _, err := os.Stat(forgeYamlPath); err != nil {
		if os.IsNotExist(err) {
			findings = append(findings, report.Finding{
				ID: "config.forgeyaml.missing", Category: "configuration", Severity: report.SeverityWarning,
				Message:    "forge.yaml manquant",
				Suggestion: "Exécutez 'forge config init' pour créer la configuration par défaut",
			})
		} else {
			return nil, err
		}
	} else {
		// Try to parse it
		data, err := os.ReadFile(forgeYamlPath)
		if err != nil {
			findings = append(findings, report.Finding{
				ID: "config.forgeyaml.read_error", Category: "configuration", Severity: report.SeverityError,
				Message: fmt.Sprintf("Impossible de lire forge.yaml: %v", err),
			})
		} else {
			// Basic YAML validation - check for required fields
			content := string(data)
			if strings.Contains(content, "architecture:") || strings.Contains(content, "project:") {
				findings = append(findings, report.Finding{
					ID: "config.forgeyaml.valid", Category: "pass", Severity: report.SeverityInfo,
					Message: "forge.yaml présent et semble valide",
				})
			} else {
				findings = append(findings, report.Finding{
					ID: "config.forgeyaml.incomplete", Category: "configuration", Severity: report.SeverityWarning,
					Message:    "forge.yaml incomplet: sections architecture/project manquantes",
					Suggestion: "Vérifiez la structure de forge.yaml",
				})
			}
		}
	}

	// Check .env.example
	envExamplePath := filepath.Join(rctx.ProjectRoot, ".env.example")
	if _, err := os.Stat(envExamplePath); err != nil {
		if os.IsNotExist(err) {
			findings = append(findings, report.Finding{
				ID: "config.envexample.missing", Category: "configuration", Severity: report.SeverityWarning,
				Message:    ".env.example manquant",
				Suggestion: "Créez un fichier .env.example pour documenter les variables d'environnement",
			})
		} else {
			return nil, err
		}
	} else {
		data, _ := os.ReadFile(envExamplePath)
		content := string(data)
		if strings.TrimSpace(content) != "" {
			findings = append(findings, report.Finding{
				ID: "config.envexample.present", Category: "pass", Severity: report.SeverityInfo,
				Message: ".env.example présent et non vide",
			})
		} else {
			findings = append(findings, report.Finding{
				ID: "config.envexample.empty", Category: "configuration", Severity: report.SeverityWarning,
				Message:    ".env.example vide",
				Suggestion: "Ajoutez les variables d'environnement nécessaires",
			})
		}
	}

	return findings, nil
}
