package template

// Data is passed to text/template when rendering scaffold files.
type Data struct {
	ProjectName        string
	ModulePath         string
	PackageName        string
	HTTPPort           int
	PostgresHostPort   int
	DatabaseName       string
	GoVersion          string
	Author             string
	UseExistingDB      bool
	ExternalDBHost     string
	ExternalDBPort     int
	ExternalDBUser     string
	ExternalDBPassword string
	ExternalDBName     string
}

// Aliases exposes {{PROJECT_NAME}} style access inside templates.
func (d Data) PROJECT_NAME() string         { return d.ProjectName }
func (d Data) MODULE_NAME() string          { return d.ModulePath }
func (d Data) DATABASE() string             { return d.DatabaseName }
func (d Data) AUTHOR() string               { return d.Author }
func (d Data) HTTP_PORT() int               { return d.HTTPPort }
func (d Data) POSTGRES_HOST_PORT() int      { return d.PostgresHostPort }
func (d Data) GO_VERSION() string           { return d.GoVersion }
func (d Data) USE_EXISTING_DB() bool        { return d.UseExistingDB }
func (d Data) EXTERNAL_DB_HOST() string     { return d.ExternalDBHost }
func (d Data) EXTERNAL_DB_PORT() int        { return d.ExternalDBPort }
func (d Data) EXTERNAL_DB_USER() string     { return d.ExternalDBUser }
func (d Data) EXTERNAL_DB_PASSWORD() string { return d.ExternalDBPassword }
func (d Data) EXTERNAL_DB_NAME() string     { return d.ExternalDBName }
