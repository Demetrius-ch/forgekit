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

// ProjectContext describes the ForgeKit project receiving a feature.
type ProjectContext struct {
	Root      string
	Module    string
	GoVersion string
	HTTPPort  int
	Type      ProjectType
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
// Loose mode: accepts ForgeKit projects, legacy projects, and external compatible Go projects.
func (Detector) DetectLoose(root string) (ProjectContext, error) {
	ctx, ptype := detectProject(root, true)
	ctx.Type = ptype
	if ptype == ProjectTypeUnknown {
		return ctx, fmt.Errorf("projet Go non détecté")
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

	goModPath := filepath.Join(root, "go.mod")

	data, err := os.ReadFile(goModPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ProjectContext{}, ProjectTypeUnknown
		}
		return ProjectContext{}, ProjectTypeUnknown
	}

	module := parseModule(data)
	if module == "" {
		return ProjectContext{}, ProjectTypeUnknown
	}

	goVersion := parseGoVersion(data)

	// Check for ForgeKit signature
	forgeDir := filepath.Join(root, ".forge")
	_, forgeDirErr := os.Stat(forgeDir)
	_, forgeYamlErr := os.Stat(filepath.Join(forgeDir, "forge.yaml"))
	_, featuresYamlErr := os.Stat(filepath.Join(forgeDir, "features.yaml"))

	hasForgeDir := forgeDirErr == nil
	hasForgeYaml := forgeYamlErr == nil
	hasFeaturesYaml := featuresYamlErr == nil

	// Check for ForgeKit structure
	hasCmdServer := true
	if _, err := os.Stat(filepath.Join(root, "cmd", "server")); err != nil {
		hasCmdServer = false
	}

	var ptype ProjectType

	if hasForgeDir && hasForgeYaml {
		ptype = ProjectTypeForgeKit
	} else if hasForgeDir && hasFeaturesYaml && !hasForgeYaml {
		ptype = ProjectTypeLegacyForgeKit
	} else if loose && hasCmdServer {
		// External compatible: has Go module + cmd/server structure
		ptype = ProjectTypeExternalCompatible
	} else if hasForgeDir && !hasForgeYaml && !hasFeaturesYaml {
		ptype = ProjectTypeInvalidForgeKit
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
	}, ptype
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
