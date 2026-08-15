package feature

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Detector detects and loads information about a ForgeKit project.
type Detector struct{}

// Detect inspects the project root and returns its context.
func (Detector) Detect(root string) (ProjectContext, error) {
	root = strings.TrimSpace(root)

	if root == "" {
		return ProjectContext{}, fmt.Errorf("le répertoire du projet ne peut pas être vide")
	}

	root, err := filepath.Abs(root)
	if err != nil {
		return ProjectContext{}, fmt.Errorf("résoudre le répertoire du projet : %w", err)
	}

	info, err := os.Stat(root)
	if err != nil {
		return ProjectContext{}, fmt.Errorf("projet introuvable : %w", err)
	}

	if !info.IsDir() {
		return ProjectContext{}, fmt.Errorf("le projet n'est pas un répertoire")
	}

	goModPath := filepath.Join(root, "go.mod")

	data, err := os.ReadFile(goModPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ProjectContext{}, fmt.Errorf("go.mod absent : projet Go non détecté")
		}

		return ProjectContext{}, fmt.Errorf("lire go.mod : %w", err)
	}

	module := parseModule(data)
	if module == "" {
		return ProjectContext{}, fmt.Errorf("module Go introuvable dans go.mod")
	}

	goVersion := parseGoVersion(data)

	if _, err := os.Stat(filepath.Join(root, "forge.yaml")); err != nil {
		if !os.IsNotExist(err) {
			return ProjectContext{}, fmt.Errorf("vérifier forge.yaml : %w", err)
		}
	}

	if _, err := os.Stat(filepath.Join(root, "cmd", "server")); err != nil {
		if !os.IsNotExist(err) {
			return ProjectContext{}, fmt.Errorf("vérifier cmd/server : %w", err)
		}
	}

	httpPort := readHTTPPort(root)

	return ProjectContext{
		Root:      root,
		Module:    module,
		GoVersion: goVersion,
		HTTPPort:  httpPort,
	}, nil
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
