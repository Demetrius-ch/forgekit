package rules

import (
	"context"
	"fmt"
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
