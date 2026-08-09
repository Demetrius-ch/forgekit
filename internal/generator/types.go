package generator

import "github.com/forgekit/forgekit/internal/template"

// InitOptions holds parameters for project scaffolding.
type InitOptions struct {
	ProjectName  string
	ModulePath   string
	HTTPPort     int
	DatabaseName string
	TargetDir    string
	Author       string
	DryRun       bool
}

// TemplateData is passed to text/template when rendering files.
type TemplateData = template.Data
