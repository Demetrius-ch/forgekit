package feature

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RouterIntegration holds the configuration for integrating a feature into router.go
type RouterIntegration struct {
	ModulePath      string
	ImportPath      string
	MiddlewareCall  string
	MiddlewareCheck string // String to check if already integrated
}

// IntegrateRouterGo safely integrates a feature into router.go
// It handles multiple features by properly merging imports and middleware calls
func IntegrateRouterGo(projectRoot string, integration RouterIntegration) error {
	routerPath := filepath.Join(projectRoot, "internal", "transport", "http", "router.go")
	content, err := os.ReadFile(routerPath)
	if err != nil {
		return fmt.Errorf("lire router.go : %w", err)
	}

	src := string(content)

	// Check if already integrated
	if strings.Contains(src, integration.MiddlewareCheck) || strings.Contains(src, `"`+integration.ImportPath+`"`) {
		return nil // Already integrated
	}

	// 1. Add import if not present
	if !strings.Contains(src, `"`+integration.ImportPath+`"`) {
		importBlock := `import (`
		newImport := importBlock + "\n\t" + `"` + integration.ImportPath + `"`
		if strings.Contains(src, importBlock) {
			src = strings.Replace(src, importBlock, newImport, 1)
		} else {
			return fmt.Errorf("bloc import introuvable dans router.go")
		}
	}

	// 2. Add middleware after timeout middleware
	// Find the timeout middleware line and add our middleware after it
	timeoutMiddleware := `r.Use(middleware.Timeout(30 * time.Second))`
	if strings.Contains(src, timeoutMiddleware) {
		newMiddleware := timeoutMiddleware + "\n\t" + integration.MiddlewareCall
		src = strings.Replace(src, timeoutMiddleware, newMiddleware, 1)
	} else {
		// Fallback: try to find any r.Use and add after the first one
		lines := strings.Split(src, "\n")
		found := false
		for i, line := range lines {
			if strings.Contains(line, "r.Use(") && !strings.Contains(line, integration.MiddlewareCall) {
				lines[i] = line + "\n\t" + integration.MiddlewareCall
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("point d'intégration middleware introuvable dans router.go")
		}
		src = strings.Join(lines, "\n")
	}

	if err := os.WriteFile(routerPath, []byte(src), 0o644); err != nil {
		return fmt.Errorf("écrire router.go : %w", err)
	}

	return nil
}

// MainIntegration holds the configuration for integrating a feature into main.go
type MainIntegration struct {
	ModulePath   string
	ImportPath   string
	ImportCheck  string
	BlankImport  bool // If true, use blank import (_ "path") instead of regular import
	Replacements []MainReplacement
}

// MainReplacement represents a string replacement in main.go
type MainReplacement struct {
	OldStr string
	NewStr string
	Check  string // Optional: only replace if Check is found
}

// RemoveRouterGo safely removes a feature's integration from router.go
// It only removes the specific feature's import and middleware call, preserving other features
func RemoveRouterGo(projectRoot string, integration RouterIntegration) error {
	routerPath := filepath.Join(projectRoot, "internal", "transport", "http", "router.go")
	content, err := os.ReadFile(routerPath)
	if err != nil {
		return fmt.Errorf("lire router.go : %w", err)
	}

	src := string(content)

	// 1. Remove the middleware call
	// The middleware call is on its own line after the timeout middleware
	middlewareLine := "\n\t" + integration.MiddlewareCall
	if strings.Contains(src, middlewareLine) {
		src = strings.Replace(src, middlewareLine, "", 1)
	} else {
		// Try alternative format (without leading newline)
		altMiddlewareLine := integration.MiddlewareCall
		if strings.Contains(src, altMiddlewareLine) {
			// Find the line and remove it entirely
			lines := strings.Split(src, "\n")
			var newLines []string
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed == altMiddlewareLine {
					continue // Skip this line
				}
				newLines = append(newLines, line)
			}
			src = strings.Join(newLines, "\n")
		}
	}

	// 2. Remove the import if it exists
	importPath := `"` + integration.ImportPath + `"`
	if strings.Contains(src, importPath) {
		src = removeImportFromBlock(src, integration.ImportPath)
	}

	if err := os.WriteFile(routerPath, []byte(src), 0o644); err != nil {
		return fmt.Errorf("écrire router.go : %w", err)
	}

	return nil
}

// removeImportFromBlock removes a specific import from the import block
func removeImportFromBlock(src, importPath string) string {
	lines := strings.Split(src, "\n")
	var newLines []string
	inImportBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "import (" {
			inImportBlock = true
		} else if inImportBlock && trimmed == ")" {
			inImportBlock = false
		}

		if inImportBlock && strings.Contains(line, importPath) {
			// Skip this import line
			continue
		}

		newLines = append(newLines, line)
	}

	return strings.Join(newLines, "\n")
}

// RemoveMainGo safely removes a feature's integration from main.go
func RemoveMainGo(projectRoot string, integration MainIntegration) error {
	mainPath := filepath.Join(projectRoot, "cmd", "server", "main.go")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("lire main.go : %w", err)
	}

	src := string(content)

	importPath := `"` + integration.ImportPath + `"`
	blankImportPath := `_ "` + integration.ImportPath + `"`
	var importToRemove string

	// Determine which import format to remove
	if integration.BlankImport {
		if strings.Contains(src, blankImportPath) {
			importToRemove = blankImportPath
		} else if strings.Contains(src, importPath) {
			importToRemove = importPath
		}
	} else if strings.Contains(src, importPath) {
		importToRemove = importPath
	}

	if importToRemove != "" {
		src = removeImportFromBlock(src, importToRemove)
	}

	// Revert replacements in reverse order
	for i := len(integration.Replacements) - 1; i >= 0; i-- {
		repl := integration.Replacements[i]
		if repl.Check != "" && !strings.Contains(src, repl.Check) {
			continue
		}
		if strings.Contains(src, repl.NewStr) {
			src = strings.Replace(src, repl.NewStr, repl.OldStr, 1)
		}
	}

	if err := os.WriteFile(mainPath, []byte(src), 0o644); err != nil {
		return fmt.Errorf("écrire main.go : %w", err)
	}

	return nil
}

// IntegrateMainGo safely integrates a feature into main.go
func IntegrateMainGo(projectRoot string, integration MainIntegration) error {
	mainPath := filepath.Join(projectRoot, "cmd", "server", "main.go")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("lire main.go : %w", err)
	}

	src := string(content)

	// Check if already integrated
	// For blank imports, check with both _ "path" and "path" prefixes
	importCheck := `"` + integration.ImportPath + `"`
	blankImportCheck := `_ "` + integration.ImportPath + `"`
	if integration.BlankImport {
		// Check for both blank and regular import formats
		if strings.Contains(src, blankImportCheck) || strings.Contains(src, importCheck) {
			return nil // Already integrated (either format)
		}
	} else {
		if strings.Contains(src, importCheck) {
			return nil // Already integrated
		}
	}

	// 1. Add import if not present
	if integration.BlankImport {
		// For blank import, add if neither format exists
		if !strings.Contains(src, blankImportCheck) && !strings.Contains(src, importCheck) {
			importBlock := `import (`
			newImport := importBlock + "\n\t" + `_ "` + integration.ImportPath + `"`
			if strings.Contains(src, importBlock) {
				src = strings.Replace(src, importBlock, newImport, 1)
			} else {
				if strings.Contains(src, "import(") {
					src = strings.Replace(src, "import(", "import (\n\t"+`_ "`+integration.ImportPath+`"`, 1)
				} else {
					return fmt.Errorf("bloc import introuvable dans main.go")
				}
			}
		}
	} else {
		// Regular import
		if !strings.Contains(src, importCheck) {
			importBlock := `import (`
			newImport := importBlock + "\n\t" + `"` + integration.ImportPath + `"`
			if strings.Contains(src, importBlock) {
				src = strings.Replace(src, importBlock, newImport, 1)
			} else {
				if strings.Contains(src, "import(") {
					src = strings.Replace(src, "import(", "import (\n\t"+`"`+integration.ImportPath+`"`, 1)
				} else {
					return fmt.Errorf("bloc import introuvable dans main.go")
				}
			}
		}
	}

	// 2. Apply replacements
	for _, repl := range integration.Replacements {
		if repl.Check != "" && !strings.Contains(src, repl.Check) {
			continue // Skip if check string not found
		}
		if strings.Contains(src, repl.OldStr) {
			src = strings.Replace(src, repl.OldStr, repl.NewStr, 1)
		}
	}

	if err := os.WriteFile(mainPath, []byte(src), 0o644); err != nil {
		return fmt.Errorf("écrire main.go : %w", err)
	}

	return nil
}
