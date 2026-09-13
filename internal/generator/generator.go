package generator

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Demetrius-ch/forgekit/internal/engine"
	"github.com/Demetrius-ch/forgekit/internal/forge"
	"github.com/Demetrius-ch/forgekit/internal/generationplan"
	"github.com/Demetrius-ch/forgekit/internal/projectconfig"
	"github.com/Demetrius-ch/forgekit/internal/template"
)

// Generator scaffolds projects from embedded templates.
type Generator struct {
	templates fs.FS
	eng       *engine.Engine
}

// New creates a Generator with embedded API templates.
func New() (*Generator, error) {
	tmplFS, err := template.LoadAPI()
	if err != nil {
		return nil, err
	}
	return &Generator{templates: tmplFS, eng: engine.NewEngine()}, nil
}

// Init creates a new project at opts.TargetDir.
func (g *Generator) Init(opts InitOptions) ([]engine.PlanEntry, error) {
	if err := ValidateProjectName(opts.ProjectName); err != nil {
		return nil, err
	}
	if err := ValidateModulePath(opts.ModulePath); err != nil {
		return nil, err
	}
	if err := ValidateHTTPPort(opts.HTTPPort); err != nil {
		return nil, err
	}
	if err := ValidateDatabaseName(opts.DatabaseName); err != nil {
		return nil, err
	}
	if !opts.DryRun {
		if err := ValidateTargetDir(opts.TargetDir); err != nil {
			return nil, err
		}
	}

	projectConfig := opts.ProjectConfig
	if projectConfig.IsZero() {
		projectConfig = projectconfig.Default(opts.ProjectName, opts.ModulePath)
	}
	if projectConfig.Name != opts.ProjectName {
		return nil, fmt.Errorf("project configuration name %q does not match init project name %q", projectConfig.Name, opts.ProjectName)
	}
	if projectConfig.ModulePath != opts.ModulePath {
		return nil, fmt.Errorf("project configuration module %q does not match init module %q", projectConfig.ModulePath, opts.ModulePath)
	}
	if err := projectConfig.Validate(); err != nil {
		return nil, fmt.Errorf("validate project configuration: %w", err)
	}

	data := template.Data{
		ProjectName:        opts.ProjectName,
		ModulePath:         opts.ModulePath,
		PackageName:        PackageNameFromProject(opts.ProjectName),
		HTTPPort:           opts.HTTPPort,
		PostgresHostPort:   opts.PostgresHostPort,
		DBHostPort:         opts.PostgresHostPort,
		DatabaseName:       opts.DatabaseName,
		GoVersion:          "1.25",
		Author:             opts.Author,
		UseExistingDB:      opts.UseExistingDB,
		ExternalDBHost:     opts.ExternalDBHost,
		ExternalDBPort:     opts.ExternalDBPort,
		ExternalDBUser:     opts.ExternalDBUser,
		ExternalDBPassword: opts.ExternalDBPassword,
		ExternalDBName:     opts.ExternalDBName,
		ProjectConfig:      projectConfig,
	}

	if !opts.DryRun {
		if err := os.MkdirAll(opts.TargetDir, 0o755); err != nil {
			return nil, fmt.Errorf("créer le répertoire %q : %w", opts.TargetDir, err)
		}
	}

	generationPlan, err := generationplan.RegistryForLanguage(projectConfig.Language).Build(projectConfig)
	if err != nil {
		return nil, fmt.Errorf("build generation plan: %w", err)
	}
	executionFiles := make([]engine.TemplateFile, 0, len(generationPlan.Files))
	for _, file := range generationPlan.Files {
		executionFiles = append(executionFiles, engine.TemplateFile{
			Template:    file.Template,
			Destination: file.Destination,
		})
	}

	plan, err := g.eng.ExecuteFiles(engine.Options{
		SourceFS:  g.templates,
		TargetDir: opts.TargetDir,
		Data:      data,
		DryRun:    opts.DryRun,
	}, executionFiles)
	if err != nil {
		if !opts.DryRun {
			_ = os.RemoveAll(opts.TargetDir)
		}
		return plan, err
	}

	if opts.DryRun {
		return plan, nil
	}

	meta := forge.CreateInitialMetadataWithConfig(opts.TargetDir, data.GoVersion, projectConfig)
	if err := forge.SaveMetadata(opts.TargetDir, meta); err != nil {
		_ = os.RemoveAll(opts.TargetDir)
		return plan, fmt.Errorf("create forge metadata: %w", err)
	}

	// Initialize project based on language
	if projectConfig.IsGo() {
		if err := g.initGoMod(opts.TargetDir, opts.ModulePath, data.GoVersion, generationPlan.Dependencies); err != nil {
			_ = os.RemoveAll(opts.TargetDir)
			return plan, err
		}
	} else if projectConfig.IsNode() && !opts.SkipNpmInstall {
		if err := g.initNpmProject(opts.TargetDir); err != nil {
			_ = os.RemoveAll(opts.TargetDir)
			return plan, err
		}
	} else if projectConfig.IsPython() && !opts.SkipNpmInstall {
		if err := g.initPythonProject(opts.TargetDir); err != nil {
			_ = os.RemoveAll(opts.TargetDir)
			return plan, err
		}
	}

	if !opts.SkipPostprocess {
		if err := g.PostProcessProject(opts.TargetDir); err != nil {
			return plan, err
		}
	}

	return plan, nil
}

func (g *Generator) initGoMod(dir, modulePath, goVersion string, dependencies []generationplan.Dependency) error {
	cmd := exec.Command("go", "mod", "init", modulePath)
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod init : %s: %w", trimOutput(string(out)), err)
	}

	cmd = exec.Command("go", "mod", "edit", "-go="+goVersion)
	cmd.Dir = dir

	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod edit : %s: %w", trimOutput(string(out)), err)
	}

	for _, dependency := range dependencies {
		cmd = exec.Command("go", "mod", "edit", "-require="+dependency.Module+"@"+dependency.Version)
		cmd.Dir = dir
		out, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("add dependency %s: %s: %w", dependency.Module, trimOutput(string(out)), err)
		}
	}

	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = dir

	out, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[warn] go mod tidy a échoué : %s\n", trimOutput(string(out)))
	}

	return nil
}

func (g *Generator) initNpmProject(dir string) error {
	cmd := exec.Command("npm", "install")
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install: %s: %w", trimOutput(string(out)), err)
	}

	return nil
}

func (g *Generator) initPythonProject(dir string) error {
	// Create virtual environment
	cmd := exec.Command("python3", "-m", "venv", ".venv")
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create venv: %s: %w", trimOutput(string(out)), err)
	}

	// Install dependencies
	cmd = exec.Command(".venv/bin/pip", "install", "-r", "requirements.txt")
	cmd.Dir = dir

	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pip install: %s: %w", trimOutput(string(out)), err)
	}

	return nil
}

// PostProcessProject formats Go sources and runs the test suite for the generated project.
func (g *Generator) PostProcessProject(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("le répertoire cible ne peut pas être vide")
	}

	// Check if this is a Go project
	goMod := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(goMod); err == nil {
		// Go project - run gofmt
		var files []string
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path != dir && (filepath.Base(path) == ".git" || filepath.Base(path) == "vendor") {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) == ".go" {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("parcourir le projet : %w", err)
		}

		if len(files) > 0 {
			cmd := exec.Command("gofmt", append([]string{"-w"}, files...)...)
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("gofmt : %s: %w", trimOutput(string(out)), err)
			}
		}

		cmd := exec.Command("go", "test", "./...")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("go test : %s: %w", trimOutput(string(out)), err)
		}
		return nil
	}

	// Check if this is a Python project
	requirementsTxt := filepath.Join(dir, "requirements.txt")
	if _, err := os.Stat(requirementsTxt); err == nil {
		// Python project - run syntax check on .py files
		var files []string
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path != dir && (filepath.Base(path) == ".git" || filepath.Base(path) == ".venv") {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) == ".py" {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("parcourir le projet : %w", err)
		}

		// Syntax check using python -m py_compile
		for _, file := range files {
			cmd := exec.Command("python3", "-m", "py_compile", file)
			cmd.Dir = dir
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("python syntax check %s : %s: %w", file, trimOutput(string(out)), err)
			}
		}
		return nil
	}

	// Not a Go or Python project, skip post-processing
	return nil
}

func ValidateProjectName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("le nom du projet ne peut pas être vide")
	}
	if strings.Contains(name, "_") || strings.Contains(name, " ") {
		return fmt.Errorf("le nom du projet doit utiliser des tirets et non des underscores ou espaces")
	}
	if strings.ToLower(name) != name {
		return fmt.Errorf("le nom du projet doit être en minuscules")
	}
	if regexp.MustCompile(`[^a-z0-9-]+`).MatchString(name) {
		return fmt.Errorf("le nom du projet ne peut contenir que des lettres minuscules, chiffres et tirets")
	}
	if strings.Contains(name, "--") {
		return fmt.Errorf("le nom du projet ne peut pas contenir deux tirets consécutifs")
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("le nom du projet ne peut pas commencer ou finir par un tiret")
	}
	if name == "con" || name == "test" || name == "import" {
		return fmt.Errorf("le nom du projet est réservé")
	}
	return nil
}

func ValidateModulePath(module string) error {
	module = strings.TrimSpace(module)
	if module == "" {
		return fmt.Errorf("le chemin du module ne peut pas être vide")
	}
	if !strings.Contains(module, "/") {
		return fmt.Errorf("le chemin du module doit être au format github.com/owner/name")
	}
	return nil
}

func ValidateHTTPPort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("le port HTTP doit être compris entre 1 et 65535")
	}
	return nil
}

func ValidateDatabaseName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("le nom de la base de données ne peut pas être vide")
	}
	if regexp.MustCompile(`[^a-z0-9_]+`).MatchString(name) {
		return fmt.Errorf("le nom de la base de données ne peut contenir que des lettres minuscules, chiffres et underscores")
	}
	return nil
}

func ValidateTargetDir(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("le répertoire cible ne peut pas être vide")
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("la cible %q n'est pas un répertoire", path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("le répertoire %q n'est pas vide", path)
	}
	return nil
}

func PackageNameFromProject(projectName string) string {
	return strings.ReplaceAll(projectName, "-", "")
}

func trimOutput(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
