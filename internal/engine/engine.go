package engine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/forgekit/forgekit/internal/template"
)

const (
	metaPrefix     = "forge:"
	skipSuffix     = ".skip.tmpl"
	conditionalDir = "_conditional/"
)

// Options controls template execution.
type Options struct {
	SourceFS  fs.FS
	TargetDir string
	Data      template.Data
	DryRun    bool
}

// Engine renders embedded templates onto disk or produces a dry-run plan.
type Engine struct{}

// NewEngine returns a template engine instance.
func NewEngine() *Engine { return &Engine{} }

// Execute walks source templates and writes files or returns a plan.
func (e *Engine) Execute(opts Options) ([]PlanEntry, error) {
	var plan []PlanEntry
	err := e.walk(opts.SourceFS, ".", opts.TargetDir, opts.Data, opts.DryRun, &plan)
	if err != nil {
		return plan, err
	}
	return plan, nil
}

func (e *Engine) walk(src fs.FS, srcDir, dstDir string, data template.Data, dryRun bool, plan *[]PlanEntry) error {
	entries, err := fs.ReadDir(src, srcDir)
	if err != nil {
		return fmt.Errorf("lire %q : %w", srcDir, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "_") && entry.IsDir() {
			continue
		}
		if name == conditionalDir {
			continue
		}

		srcPath := filepath.ToSlash(filepath.Join(srcDir, name))
		if entry.IsDir() {
			outName := strings.TrimPrefix(name, conditionalDir)
			outDir := filepath.Join(dstDir, outName)
			if !dryRun {
				if err := os.MkdirAll(outDir, 0o755); err != nil {
					return err
				}
			}
			if err := e.walk(src, srcPath, outDir, data, dryRun, plan); err != nil {
				return err
			}
			continue
		}

		if strings.HasSuffix(name, skipSuffix) {
			*plan = append(*plan, PlanEntry{
				Path:   filepath.Join(dstDir, template.OutputPath(strings.TrimSuffix(name, skipSuffix))),
				Action: ActionSkip,
				Reason: "condition non satisfaite",
			})
			continue
		}

		if meta, skip, err := readMeta(src, srcPath); err != nil {
			return err
		} else if skip {
			*plan = append(*plan, PlanEntry{
				Path:   filepath.Join(dstDir, template.OutputPath(name)),
				Action: ActionSkip,
				Reason: meta,
			})
			continue
		}

		outName := template.OutputPath(name)
		dstPath := filepath.Join(dstDir, outName)
		mode := fileMode(name)

		rendered, err := template.RenderFile(src, srcPath, data)
		if err != nil {
			return err
		}

		*plan = append(*plan, PlanEntry{
			Path:   dstPath,
			Action: ActionCreate,
			Mode:   mode,
		})

		if dryRun {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return fmt.Errorf("créer %q : %w", filepath.Dir(dstPath), err)
		}
		if err := os.WriteFile(dstPath, rendered, mode); err != nil {
			return fmt.Errorf("écrire %q : %w", dstPath, err)
		}
	}

	return nil
}

func readMeta(src fs.FS, path string) (reason string, skip bool, err error) {
	content, err := fs.ReadFile(src, path)
	if err != nil {
		return "", false, err
	}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, metaPrefix) {
			if line != "" && !strings.HasPrefix(line, "{{") && !strings.HasPrefix(line, "package") {
				break
			}
			continue
		}
		if strings.HasPrefix(line, metaPrefix+"skip:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, metaPrefix+"skip:"))
			if val == "true" {
				return "forge:skip", true, nil
			}
		}
	}
	return "", false, nil
}

func fileMode(name string) os.FileMode {
	if strings.Contains(name, "Makefile") {
		return 0o644
	}
	return 0o644
}
