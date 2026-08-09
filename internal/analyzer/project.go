package analyzer

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// PackageInfo describes a Go package in the project.
type PackageInfo struct {
	ImportPath string
	Dir        string
	Layer      string
	Imports    []string
}

// Project holds parsed project structure.
type Project struct {
	Root     string
	Module   string
	Packages []PackageInfo
}

// LoadProject parses go.mod module and Go files under internal/.
func LoadProject(root string) (*Project, error) {
	module, err := readModule(root)
	if err != nil {
		return nil, err
	}

	p := &Project{Root: root, Module: module}
	internalDir := filepath.Join(root, "internal")
	if _, err := os.Stat(internalDir); os.IsNotExist(err) {
		return p, nil
	}

	err = filepath.WalkDir(internalDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		hasGo, err := dirHasGoFiles(path)
		if err != nil || !hasGo {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		importPath := module + "/" + filepath.ToSlash(rel)
		imports, err := parseImports(path)
		if err != nil {
			return err
		}

		p.Packages = append(p.Packages, PackageInfo{
			ImportPath: importPath,
			Dir:        rel,
			Layer:      detectLayer(rel),
			Imports:    imports,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func readModule(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("go.mod introuvable : %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("directive module introuvable dans go.mod")
}

func dirHasGoFiles(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			return true, nil
		}
	}
	return false, nil
}

func parseImports(dir string) ([]string, error) {
	fset := token.NewFileSet()
	var imports []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.ImportsOnly)
		if err != nil {
			return nil, err
		}
		for _, imp := range f.Imports {
			imports = append(imports, strings.Trim(imp.Path.Value, `"`))
		}
	}
	return unique(imports), nil
}

func detectLayer(rel string) string {
	rel = filepath.ToSlash(rel)
	switch {
	case strings.Contains(rel, "/domain"):
		return "domain"
	case strings.Contains(rel, "/application"):
		return "application"
	case strings.Contains(rel, "/infrastructure"):
		return "infrastructure"
	case strings.Contains(rel, "/transport"):
		return "transport"
	default:
		return "unknown"
	}
}

func unique(in []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// ImportGraph returns adjacency list module -> imported project packages.
func (p *Project) ImportGraph() map[string][]string {
	g := make(map[string][]string)
	for _, pkg := range p.Packages {
		var deps []string
		for _, imp := range pkg.Imports {
			if strings.HasPrefix(imp, p.Module) {
				deps = append(deps, imp)
			}
		}
		g[pkg.ImportPath] = deps
	}
	return g
}
