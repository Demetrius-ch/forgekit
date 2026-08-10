package feature

// Plan describes the changes a feature intends to make.
type Plan struct {
	Feature      string
	Version      string
	Files        []FileAction
	Dependencies []Dependency
	Environment  []string
}
