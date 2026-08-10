// Package generator exposes a stable public API for external Go applications.
package generator

import (
	"github.com/Demetrius-ch/forgekit/internal/engine"
	intgen "github.com/Demetrius-ch/forgekit/internal/generator"
)

// Options configures project generation.
type Options struct {
	ProjectName     string
	ModulePath      string
	HTTPPort        int
	DatabaseName    string
	TargetDir       string
	Author          string
	DryRun          bool
	SkipPostprocess bool
}

// PlanEntry describes a planned filesystem operation.
type PlanEntry = engine.PlanEntry

// Generate scaffolds a ForgeKit backend project.
func Generate(opts Options) ([]PlanEntry, error) {
	g, err := intgen.New()
	if err != nil {
		return nil, err
	}
	return g.Init(intgen.InitOptions{
		ProjectName:     opts.ProjectName,
		ModulePath:      opts.ModulePath,
		HTTPPort:        opts.HTTPPort,
		DatabaseName:    opts.DatabaseName,
		TargetDir:       opts.TargetDir,
		Author:          opts.Author,
		DryRun:          opts.DryRun,
		SkipPostprocess: opts.SkipPostprocess,
	})
}

// ValidateProjectName checks project naming rules.
func ValidateProjectName(name string) error {
	return intgen.ValidateProjectName(name)
}
