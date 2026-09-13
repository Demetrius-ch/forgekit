package feature

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ProjectType represents the type of project detected.
type ProjectType int

const (
	ProjectTypeUnknown ProjectType = iota
	ProjectTypeForgeKit
	ProjectTypeLegacyForgeKit
	ProjectTypeExternalCompatible
	ProjectTypeInvalidForgeKit
)

// ProjectLanguage identifies the programming language of the project.
type ProjectLanguage string

const (
	LanguageGo         ProjectLanguage = "go"
	LanguageTypeScript ProjectLanguage = "typescript"
	LanguageJavaScript ProjectLanguage = "javascript"
	LanguagePython     ProjectLanguage = "python"
)

// ProjectContext describes the ForgeKit project receiving a feature.
type ProjectContext struct {
	Root      string
	Module    string
	GoVersion string
	HTTPPort  int
	Type      ProjectType
	Language  ProjectLanguage
}

// Detector detects and loads information about a ForgeKit project.
type Detector struct{}

// Detect inspects the project root and returns its context.
// Strict mode: requires ForgeKit project structure (.forge or legacy features.yaml).
func (Detector) Detect(root string) (ProjectContext, error) {
	ctx, ptype := detectProject(root, false)
	ctx.Type = ptype
	if ptype == ProjectTypeInvalidForgeKit || ptype == ProjectTypeUnknown {
		return ctx, fmt.Errorf("projet ForgeKit invalide ou non détecté")
	}
	return ctx, nil
}

// DetectLoose inspects the project root and returns its context.
// Loose mode: accepts ForgeKit projects, legacy projects, and external compatible projects.
func (Detector) DetectLoose(root string) (ProjectContext, error) {
	ctx, ptype := detectProject(root, true)
	ctx.Type = ptype
	if ptype == ProjectTypeUnknown {
		return ctx, fmt.Errorf("projet non détecté")
	}
	return ctx, nil
}

// detectProject performs the actual detection logic.
func detectProject(root string, loose bool) (ProjectContext, ProjectType) {
	root = strings.TrimSpace(root)

	if root == "" {
		return ProjectContext{}, ProjectTypeUnknown
	}

	root, err := filepath.Abs(root)
	if err != nil {
		return ProjectContext{}, ProjectTypeUnknown
	}

	info, err := os.Stat(root)
	if err != nil {
		return ProjectContext{}, ProjectTypeUnknown
	}

	if !info.IsDir() {
		return ProjectContext{}, ProjectTypeUnknown
	}

	// Check for ForgeKit signature
	forgeDir := filepath.Join(root, ".forge")
	_, forgeDirErr := os.Stat(forgeDir)
	_, forgeYamlErr := os.Stat(filepath.Join(forgeDir, "forge.yaml"))
	_, featuresYamlErr := os.Stat(filepath.Join(forgeDir, "features.yaml"))

	hasForgeDir := forgeDirErr == nil
	hasForgeYaml := forgeYamlErr == nil
	hasFeaturesYaml := featuresYamlErr == nil

	// Detect language
	language := detectLanguage(root)

	var module string
	var goVersion string

	// For Go projects, parse go.mod
	if language == LanguageGo || language == "" {
		goModPath := filepath.Join(root, "go.mod")
		data, err := os.ReadFile(goModPath)
		if err == nil {
			module = parseModule(data)
			goVersion = parseGoVersion(data)
			// For Go projects, module is required
			if module == "" {
				return ProjectContext{}, ProjectTypeUnknown
			}
		} else if language == LanguageGo {
			// Go project but no go.mod
			return ProjectContext{}, ProjectTypeUnknown
		}
	} else if language == LanguagePython {
		// For Python projects, use pyproject.toml name or directory name
		pyprojectPath := filepath.Join(root, "pyproject.toml")
		data, err := os.ReadFile(pyprojectPath)
		if err == nil {
			module = parsePyprojectName(data)
		}
		if module == "" {
			// Use directory name as fallback
			module = filepath.Base(root)
		}
	} else {
		// For JS/TS projects, use package.json name
		packageJSONPath := filepath.Join(root, "package.json")
		data, err := os.ReadFile(packageJSONPath)
		if err == nil {
			module = parsePackageName(data)
		}
	}

	var ptype ProjectType

	if hasForgeDir && hasForgeYaml {
		ptype = ProjectTypeForgeKit
	} else if hasForgeDir && hasFeaturesYaml && !hasForgeYaml {
		ptype = ProjectTypeLegacyForgeKit
	} else if hasForgeDir && !hasForgeYaml && !hasFeaturesYaml {
		ptype = ProjectTypeInvalidForgeKit
	} else if loose && language != "" {
		// External compatible: has language-specific files
		ptype = ProjectTypeExternalCompatible
	} else {
		ptype = ProjectTypeUnknown
	}

	httpPort := readHTTPPort(root)

	return ProjectContext{
		Root:      root,
		Module:    module,
		GoVersion: goVersion,
		HTTPPort:  httpPort,
		Type:      ptype,
		Language:  language,
	}, ptype
}

// detectLanguage identifies the project language based on files present.
func detectLanguage(root string) ProjectLanguage {
	// Check for Go
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		return LanguageGo
	}
	// Check for TypeScript
	if _, err := os.Stat(filepath.Join(root, "tsconfig.json")); err == nil {
		return LanguageTypeScript
	}
	// Check for JavaScript
	if _, err := os.Stat(filepath.Join(root, "package.json")); err == nil {
		// Could be JS or TS, default to JS if no tsconfig
		return LanguageJavaScript
	}
	// Check for Python
	if _, err := os.Stat(filepath.Join(root, "requirements.txt")); err == nil {
		return LanguagePython
	}
	if _, err := os.Stat(filepath.Join(root, "pyproject.toml")); err == nil {
		return LanguagePython
	}
	return ""
}

func readHTTPPort(root string) int {
	envPath := filepath.Join(root, ".env.example")
	data, err := os.ReadFile(envPath)
	if err != nil {
		return 8080 // default
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "HTTP_PORT=") {
			portStr := strings.TrimPrefix(line, "HTTP_PORT=")
			port, err := strconv.Atoi(portStr)
			if err == nil && port > 0 {
				return port
			}
		}
	}
	return 8080
}

func parseModule(data []byte) string {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		if len(fields) == 2 && fields[0] == "module" {
			return fields[1]
		}
	}

	return ""
}

func parseGoVersion(data []byte) string {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		if len(fields) == 2 && fields[0] == "go" {
			return fields[1]
		}
	}

	return ""
}

func parsePackageName(data []byte) string {
	// Simple JSON parse for name field
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, `"name"`) && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				name := strings.Trim(strings.TrimSpace(parts[1]), `",`)
				return name
			}
		}
	}
	return ""
}

func parsePyprojectName(data []byte) string {
	// Simple TOML parse for name field in [project] section
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	inProjectSection := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[project]" {
			inProjectSection = true
			continue
		}
		if strings.HasPrefix(line, "[") && inProjectSection {
			break
		}
		if inProjectSection && strings.HasPrefix(line, "name") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.Trim(strings.TrimSpace(parts[1]), `"`)
				return name
			}
		}
	}
	return ""
}
