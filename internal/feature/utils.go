package feature

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AddDependencies adds Go module dependencies to go.mod.
func AddDependencies(projectRoot string, deps []Dependency) error {
	if len(deps) == 0 {
		return nil
	}

	for _, dep := range deps {
		cmd := exec.Command("go", "get", dep.Module+"@"+dep.Version)
		cmd.Dir = projectRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go get %s@%s : %w", dep.Module, dep.Version, err)
		}
	}

	return nil
}

// RunGoModTidy runs go mod tidy in the project directory.
func RunGoModTidy(projectRoot string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunGoFmt runs gofmt on all Go files in the project.
func RunGoFmt(projectRoot string) error {
	var files []string
	err := filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if filepath.Base(path) == ".git" || filepath.Base(path) == "vendor" {
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

	if len(files) == 0 {
		return nil
	}

	cmd := exec.Command("gofmt", append([]string{"-w"}, files...)...)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// FileExists checks if a file exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ReadFile reads a file content.
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes content to a file.
func WriteFile(path string, content []byte) error {
	return os.WriteFile(path, content, 0o644)
}

// MkdirAll creates directories.
func MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// TemplateRenderer renders template files.
type TemplateRenderer struct {
	data map[string]string
}

// NewTemplateRenderer creates a template renderer.
func NewTemplateRenderer(data map[string]string) *TemplateRenderer {
	return &TemplateRenderer{data: data}
}

// RenderFile reads a template file and replaces placeholders.
func (t *TemplateRenderer) RenderFile(sourcePath, destPath string) error {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	result := string(content)
	for key, value := range t.data {
		placeholder := "{{." + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return os.WriteFile(destPath, []byte(result), 0o644)
}

// UpdateEnvironment appends environment variables to .env.example.
func UpdateEnvironment(projectRoot string, envVars []string) error {
	if len(envVars) == 0 {
		return nil
	}

	envPath := filepath.Join(projectRoot, ".env.example")

	// Read existing content
	var oldContent []byte
	existed := true
	if data, err := os.ReadFile(envPath); err == nil {
		oldContent = data
	} else if os.IsNotExist(err) {
		existed = false
	} else {
		return fmt.Errorf("lire .env.example : %w", err)
	}

	// Append new environment variables
	var newContent strings.Builder
	if existed {
		newContent.Write(oldContent)
		if !strings.HasSuffix(string(oldContent), "\n") {
			newContent.WriteString("\n")
		}
	}

	for _, env := range envVars {
		newContent.WriteString(env + "\n")
	}

	if err := os.WriteFile(envPath, []byte(newContent.String()), 0o644); err != nil {
		return fmt.Errorf("écrire .env.example : %w", err)
	}

	return nil
}
