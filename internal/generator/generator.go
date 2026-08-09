package generator

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/forgekit/forgekit/internal/engine"
	"github.com/forgekit/forgekit/internal/template"
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

	data := template.Data{
		ProjectName:  opts.ProjectName,
		ModulePath:   opts.ModulePath,
		PackageName:  PackageNameFromProject(opts.ProjectName),
		HTTPPort:     opts.HTTPPort,
		DatabaseName: opts.DatabaseName,
		GoVersion:    "1.25",
		Author:       opts.Author,
	}

	if !opts.DryRun {
		if err := os.MkdirAll(opts.TargetDir, 0o755); err != nil {
			return nil, fmt.Errorf("créer le répertoire %q : %w", opts.TargetDir, err)
		}
	}

	plan, err := g.eng.Execute(engine.Options{
		SourceFS:  g.templates,
		TargetDir: opts.TargetDir,
		Data:      data,
		DryRun:    opts.DryRun,
	})
	if err != nil {
		if !opts.DryRun {
			_ = os.RemoveAll(opts.TargetDir)
		}
		return plan, err
	}

	if opts.DryRun {
		return plan, nil
	}

	if err := g.initGoMod(opts.TargetDir, opts.ModulePath, data.GoVersion); err != nil {
		_ = os.RemoveAll(opts.TargetDir)
		return plan, err
	}

	if err := g.PostProcessProject(opts.TargetDir); err != nil {
		return plan, err
	}

	return plan, nil
}

func (g *Generator) initGoMod(dir, modulePath, goVersion string) error {
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

	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = dir

	out, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[warn] go mod tidy a échoué : %s\n", trimOutput(string(out)))
	}

	return nil
}

// PostProcessProject formats Go sources and runs the test suite for the generated project.
func (g *Generator) PostProcessProject(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("le répertoire cible ne peut pas être vide")
	}

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
