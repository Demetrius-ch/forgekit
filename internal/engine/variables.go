package engine

import "github.com/Demetrius-ch/forgekit/internal/template"

// Variables exposes template placeholders documented in ForgeKit specs.
type Variables struct {
	PROJECT_NAME string
	MODULE_NAME  string
	DATABASE     string
	AUTHOR       string
	HTTP_PORT    int
	GO_VERSION   string
	PackageName  string
}

// FromData converts internal template data to public variable names.
func FromData(d template.Data) Variables {
	return Variables{
		PROJECT_NAME: d.ProjectName,
		MODULE_NAME:  d.ModulePath,
		DATABASE:     d.DatabaseName,
		AUTHOR:       d.Author,
		HTTP_PORT:    d.HTTPPort,
		GO_VERSION:   d.GoVersion,
		PackageName:  d.PackageName,
	}
}

// ToTemplateData converts engine variables back to render data.
func (v Variables) ToTemplateData() template.Data {
	return template.Data{
		ProjectName:  v.PROJECT_NAME,
		ModulePath:   v.MODULE_NAME,
		DatabaseName: v.DATABASE,
		Author:       v.AUTHOR,
		HTTPPort:     v.HTTP_PORT,
		GoVersion:    v.GO_VERSION,
		PackageName:  v.PackageName,
	}
}
