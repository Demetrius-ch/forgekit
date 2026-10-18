package feature

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runGo runs a go toolchain command in dir. Its output is only reported when
// the command fails, so that --quiet and --format json stay clean.
func runGo(dir string, args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go %s : %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// AddDependencies adds Go module dependencies to go.mod.
func AddDependencies(projectRoot string, deps []Dependency) error {
	for _, dep := range deps {
		if err := runGo(projectRoot, "get", dep.Module+"@"+dep.Version); err != nil {
			return err
		}
	}
	return nil
}

// DropDependencies removes the requirements of deps from go.mod. A module that
// is not required is not an error; go mod tidy settles the rest afterwards.
func DropDependencies(projectRoot string, deps []Dependency) {
	for _, dep := range deps {
		_ = runGo(projectRoot, "mod", "edit", "-droprequire", dep.Module)
	}
}

// RunGoModTidy runs go mod tidy in the project directory.
func RunGoModTidy(projectRoot string) error {
	return runGo(projectRoot, "mod", "tidy")
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
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gofmt : %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
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

// RemoveEnvironment removes environment variables from .env.example.
func RemoveEnvironment(projectRoot string, envVars []string) error {
	if len(envVars) == 0 {
		return nil
	}

	envPath := filepath.Join(projectRoot, ".env.example")

	// Read existing content
	data, err := os.ReadFile(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, nothing to do
		}
		return fmt.Errorf("lire .env.example : %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	// Build a set of keys to remove
	keysToRemove := make(map[string]bool)
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) > 0 {
			keysToRemove[strings.TrimSpace(parts[0])] = true
		}
	}

	// Filter out lines with keys to remove
	var newLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			newLines = append(newLines, line)
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) > 0 {
			key := strings.TrimSpace(parts[0])
			if keysToRemove[key] {
				continue // Skip this line
			}
		}
		newLines = append(newLines, line)
	}

	newContent := strings.Join(newLines, "\n")
	// Ensure no trailing empty lines beyond one
	newContent = strings.TrimRight(newContent, "\n") + "\n"

	if err := os.WriteFile(envPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("écrire .env.example : %w", err)
	}

	return nil
}
