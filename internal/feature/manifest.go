package feature

// Manifest describes the resources required by a feature.
type Manifest struct {
	Name         string
	Version      string
	Description  string
	Dependencies []Dependency
	Files        []FileAction
	Environment  []string
}

// Dependency represents a Go module dependency.
type Dependency struct {
	Module  string
	Version string
}

// FileAction describes a file operation performed by a feature.
type FileAction struct {
	Source      string
	Destination string
}
