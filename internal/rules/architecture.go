package rules

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/Demetrius-ch/forgekit/internal/analyzer"
	"github.com/Demetrius-ch/forgekit/internal/config"
	"github.com/Demetrius-ch/forgekit/internal/report"
)

// ArchitectureRule checks layer import violations.
type ArchitectureRule struct {
	Rules []config.ArchitectureRule
}

func (r ArchitectureRule) ID() string          { return "arch.layers" }
func (r ArchitectureRule) Name() string        { return "Règles d'architecture" }
func (r ArchitectureRule) Description() string { return "Vérifie les dépendances entre couches" }
func (r ArchitectureRule) Category() string    { return "architecture" }
func (r ArchitectureRule) Severity() report.Severity {
	return report.SeverityError
}

func (r ArchitectureRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	project, err := analyzer.LoadProject(rctx.ProjectRoot)
	if err != nil {
		return nil, err
	}
	rules := r.Rules
	if len(rules) == 0 {
		rules = config.DefaultArchitectureRules()
	}

	var findings []report.Finding
	violationCount := 0

	// 1. Check layer import violations (existing logic)
	for _, pkg := range project.Packages {
		if pkg.Layer == "unknown" {
			continue
		}
		for _, imp := range pkg.Imports {
			if !strings.HasPrefix(imp, project.Module) {
				continue
			}
			targetLayer := layerForImport(project, imp)
			if targetLayer == "" {
				continue
			}
			for _, rule := range rules {
				if rule.From == pkg.Layer && rule.To == targetLayer && !rule.Allow {
					violationCount++
					findings = append(findings, report.Finding{
						ID:          "arch.layers",
						Category:    "architecture",
						Severity:    report.SeverityError,
						File:        pkg.Dir,
						Message:     fmt.Sprintf("Violation d'architecture: %s importe %s (interdit)", pkg.Layer, targetLayer),
						Explanation: fmt.Sprintf("La couche '%s' ne doit pas dépendre de la couche '%s'. Règle: %s -> %s = forbidden", pkg.Layer, targetLayer, rule.From, rule.To),
						Suggestion:  "Introduisez une interface dans domain et injectez l'implémentation via l'injection de dépendances",
					})
				}
			}
		}
	}

	// 2. Check for missing layers in hexagonal architecture
	expectedLayers := []string{"domain", "application", "infrastructure", "transport"}
	hasLayer := make(map[string]bool)
	for _, pkg := range project.Packages {
		if pkg.Layer != "unknown" {
			hasLayer[pkg.Layer] = true
		}
	}
	for _, layer := range expectedLayers {
		if !hasLayer[layer] {
			findings = append(findings, report.Finding{
				ID:          "arch.missing_layer",
				Category:    "architecture",
				Severity:    report.SeverityWarning,
				Message:     fmt.Sprintf("Couche '%s' manquante dans l'architecture hexagonale", layer),
				Explanation: fmt.Sprintf("L'architecture hexagonale recommande la présence de la couche %s", layer),
				Suggestion:  fmt.Sprintf("Ajoutez un package dans internal/%s/", layer),
			})
		}
	}

	// 3. Check for domain layer purity (no external dependencies)
	for _, pkg := range project.Packages {
		if pkg.Layer == "domain" {
			for _, imp := range pkg.Imports {
				if isExternalImport(imp) {
					findings = append(findings, report.Finding{
						ID:          "arch.domain.purity",
						Category:    "architecture",
						Severity:    report.SeverityError,
						File:        pkg.Dir,
						Message:     fmt.Sprintf("Couche domain importe un package externe: %s", imp),
						Explanation: "La couche domain ne doit pas avoir de dépendances externes (frameworks, DB, HTTP, etc.)",
						Suggestion:  "Supprimez l'import externe ou déplacez le code vers infrastructure/application",
					})
				}
			}
		}
	}

	// 4. Check for infrastructure leaking into application
	for _, pkg := range project.Packages {
		if pkg.Layer == "application" {
			for _, imp := range pkg.Imports {
				if strings.Contains(imp, "/infrastructure/") {
					findings = append(findings, report.Finding{
						ID:          "arch.app.infra_leak",
						Category:    "architecture",
						Severity:    report.SeverityWarning,
						File:        pkg.Dir,
						Message:     fmt.Sprintf("Couche application importe infrastructure directement: %s", imp),
						Explanation: "La couche application ne doit dépendre que de domain; utilisez des interfaces",
						Suggestion:  "Introduisez une interface dans domain et injectez l'implémentation infrastructure",
					})
				}
			}
		}
	}

	// 5. Check for transport depending on infrastructure (should only depend on application + domain)
	for _, pkg := range project.Packages {
		if pkg.Layer == "transport" {
			for _, imp := range pkg.Imports {
				if strings.Contains(imp, "/infrastructure/") {
					findings = append(findings, report.Finding{
						ID:          "arch.transport.infra_leak",
						Category:    "architecture",
						Severity:    report.SeverityWarning,
						File:        pkg.Dir,
						Message:     fmt.Sprintf("Couche transport importe infrastructure: %s", imp),
						Explanation: "La couche transport ne doit dépendre que d'application et domain",
						Suggestion:  "Utilisez les services d'application au lieu d'accéder directement à l'infrastructure",
					})
				}
			}
		}
	}

	if violationCount == 0 && len(project.Packages) > 0 {
		findings = append(findings, report.Finding{
			ID: "arch.layers", Category: "pass", Severity: report.SeverityInfo,
			Message: "Architecture respectée: aucune violation de couche détectée",
		})
	}
	return findings, nil
}

func layerForImport(p *analyzer.Project, imp string) string {
	for _, pkg := range p.Packages {
		if pkg.ImportPath == imp {
			return pkg.Layer
		}
	}
	return ""
}

func isExternalImport(imp string) bool {
	// Domain should not import standard library packages that are "external" like database/sql, net/http, etc.
	// But we allow standard library. We consider external anything that's not the project module.
	// For domain layer, we flag any import that is not standard library or project module
	return !isStandardLib(imp)
}

func isStandardLib(imp string) bool {
	// Common Go standard library packages
	stdLib := []string{
		"context", "fmt", "strings", "errors", "time", "encoding/json",
		"net/http", "database/sql", "io", "os", "path/filepath", "sync",
		"regexp", "strconv", "math", "sort", "bytes", "bufio", "crypto",
		"hash", "encoding", "reflect", "runtime", "unsafe", "plugin",
	}
	for _, std := range stdLib {
		if imp == std || strings.HasPrefix(imp, std+"/") {
			return true
		}
	}
	return false
}

// ArchitectureComplexityRule checks for overly complex packages
type ArchitectureComplexityRule struct{}

func (ArchitectureComplexityRule) ID() string   { return "arch.complexity" }
func (ArchitectureComplexityRule) Name() string { return "Complexité des packages" }
func (ArchitectureComplexityRule) Description() string {
	return "Détecte les packages trop complexes (trop de fichiers/go)"
}
func (ArchitectureComplexityRule) Category() string { return "architecture" }
func (ArchitectureComplexityRule) Severity() report.Severity {
	return report.SeverityWarning
}

func (ArchitectureComplexityRule) Run(_ context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding

	_ = filepath.WalkDir(rctx.ProjectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		f, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return nil
		}
		if err != nil {
			return nil
		}

		// Count functions and structs
		funcCount := 0
		structCount := 0
		ast.Inspect(f, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.FuncDecl:
				funcCount++
			case *ast.TypeSpec:
				if node.Type != nil {
					if _, ok := node.Type.(*ast.StructType); ok {
						structCount++
					}
				}
			}
			return true
		})

		relPath, _ := filepath.Rel(rctx.ProjectRoot, path)

		// Flag if too many functions in a single file (> 15)
		if funcCount > 15 {
			findings = append(findings, report.Finding{
				ID:         "arch.complexity.funcs",
				Category:   "architecture",
				Severity:   report.SeverityWarning,
				File:       relPath,
				Message:    fmt.Sprintf("Fichier avec %d fonctions (recommandé < 15)", funcCount),
				Suggestion: "Séparez en plusieurs fichiers ou packages plus petits",
			})
		}

		// Flag if too many structs in a single file (> 8)
		if structCount > 8 {
			findings = append(findings, report.Finding{
				ID:         "arch.complexity.structs",
				Category:   "architecture",
				Severity:   report.SeverityWarning,
				File:       relPath,
				Message:    fmt.Sprintf("Fichier avec %d structures (recommandé < 8)", structCount),
				Suggestion: "Considérez séparer les types dans des fichiers distincts",
			})
		}

		return nil
	})

	return findings, nil
}
