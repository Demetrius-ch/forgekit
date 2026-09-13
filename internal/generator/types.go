package generator

import (
	"github.com/Demetrius-ch/forgekit/internal/projectconfig"
	"github.com/Demetrius-ch/forgekit/internal/template"
)

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
	// ProjectConfig is the central v0.4 configuration. When absent, Init uses
	// the v0.3-compatible defaults while callers migrate to explicit choices.
	ProjectConfig projectconfig.ProjectConfig
	// SkipPostprocess when true avoids running gofmt/go test on generated project.
	SkipPostprocess bool
	// SkipNpmInstall when true avoids running npm install for Node.js projects.
	SkipNpmInstall bool
}

// TemplateData is passed to text/template when rendering files.
type TemplateData = template.Data
