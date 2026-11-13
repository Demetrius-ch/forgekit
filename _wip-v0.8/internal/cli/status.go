package cli

// TemplateMetadata contient les métadonnées de template stockées dans .forge.
type TemplateMetadata struct {
	ID              string `yaml:"id,omitempty"`
	Version         string `yaml:"version,omitempty"`
	Source          string `yaml:"source,omitempty"`
	Latest          string `yaml:"latest,omitempty"`
	Available       bool   `yaml:"available,omitempty"`
	UpdateAvailable bool   `yaml:"update_available,omitempty"`
}

// TemplateInfo est un alias pour TemplateMetadata (utilisé pour la sérialisation JSON/YAML).
type TemplateInfo = TemplateMetadata

// ProjectStatus représente l'état complet d'un projet ForgeKit.
type ProjectStatus struct {
	ProjectName  string       `json:"project_name"`
	Language     string       `json:"language"`
	ForgeVersion string       `json:"forge_version"`
	HasTemplate  bool         `json:"has_template"`
	Template     TemplateInfo `json:"template,omitempty"`
}
