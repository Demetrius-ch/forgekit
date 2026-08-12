package generator

import "github.com/Demetrius-ch/forgekit/internal/template"

// InitOptions holds parameters for project scaffolding.
type InitOptions struct {
	ProjectName        string
	ModulePath         string
	HTTPPort           int
	PostgresHostPort   int
	DatabaseName       string
	TargetDir          string
	Author             string
	DryRun             bool
	UseExistingDB      bool
	ExternalDBHost     string
	ExternalDBPort     int
	ExternalDBUser     string
	ExternalDBPassword string
	ExternalDBName     string
	// SkipPostprocess when true avoids running gofmt/go test on generated project.
	SkipPostprocess bool
}

// TemplateData is passed to text/template when rendering files.
type TemplateData = template.Data
