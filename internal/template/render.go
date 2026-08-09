package template

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"text/template"
)

//go:embed all:api
var apiFS embed.FS

const apiRoot = "api"

// LoadAPI returns the embedded API template filesystem rooted at "".
func LoadAPI() (fs.FS, error) {
	sub, err := fs.Sub(apiFS, apiRoot)
	if err != nil {
		return nil, fmt.Errorf("charger les templates api : %w", err)
	}
	return sub, nil
}

// RenderFile executes a template file and returns rendered content.
func RenderFile(tmplFS fs.FS, relPath string, data Data) ([]byte, error) {
	content, err := fs.ReadFile(tmplFS, relPath)
	if err != nil {
		return nil, fmt.Errorf("lire le template %q : %w", relPath, err)
	}

	body := stripMetaLines(string(content))

	funcMap := template.FuncMap{
		"replace": strings.ReplaceAll,
	}

	tmpl, err := template.New(path.Base(relPath)).Funcs(funcMap).Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parser le template %q : %w", relPath, err)
	}

	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		return nil, fmt.Errorf("exécuter le template %q : %w", relPath, err)
	}
	return []byte(b.String()), nil
}

func stripMetaLines(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "forge:") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// OutputPath maps a template path to the generated project path.
func OutputPath(relPath string) string {
	relPath = strings.TrimSuffix(relPath, ".skip.tmpl")
	relPath = strings.TrimSuffix(relPath, ".tmpl")
	return relPath
}
