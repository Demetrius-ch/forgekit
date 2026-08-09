package template

// Data is passed to text/template when rendering scaffold files.
type Data struct {
	ProjectName  string
	ModulePath   string
	PackageName  string
	HTTPPort     int
	DatabaseName string
	GoVersion    string
	Author       string
}

// Aliases exposes {{PROJECT_NAME}} style access inside templates.
func (d Data) PROJECT_NAME() string  { return d.ProjectName }
func (d Data) MODULE_NAME() string   { return d.ModulePath }
func (d Data) DATABASE() string      { return d.DatabaseName }
func (d Data) AUTHOR() string        { return d.Author }
func (d Data) HTTP_PORT() int        { return d.HTTPPort }
func (d Data) GO_VERSION() string    { return d.GoVersion }
