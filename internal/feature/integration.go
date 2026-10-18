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
	MiddlewareCall  string // Statement inserted in the router, e.g. r.Use(cors.Middleware())
	MiddlewareCheck string // String to check if already integrated
	// IsRoute marks route registrations (e.g. swagger.RegisterRoutes(r)): they are
	// inserted before the final "return r" so that every middleware precedes them.
	IsRoute    bool
	RouterPath string // Optional: explicit path to router.go (auto-detected if empty)
}

// MainIntegration holds the configuration for integrating a feature into main.go
type MainIntegration struct {
	ModulePath   string
	ImportPath   string
	ImportCheck  string
	BlankImport  bool // If true, use blank import (_ "path") instead of regular import
	Replacements []MainReplacement
	MainPath     string // Optional: explicit path to main.go (auto-detected if empty)
}

// MainReplacement represents a string replacement in main.go
type MainReplacement struct {
	OldStr string
	NewStr string
	Check  string // Optional: only replace if Check is found
}

// findRouterPath searches for router.go across known architecture layouts.
// Returns empty string if not found.
func findRouterPath(projectRoot string) string {
	candidates := []string{
		filepath.Join(projectRoot, "internal", "transport", "http", "router.go"),  // hexagonal
		filepath.Join(projectRoot, "internal", "interfaces", "http", "router.go"), // clean
		filepath.Join(projectRoot, "internal", "handlers", "http.go"),             // layered
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// findMainPath searches for main.go in known locations.
// Returns empty string if not found.
func findMainPath(projectRoot string) string {
	candidates := []string{
		filepath.Join(projectRoot, "cmd", "server", "main.go"),
		filepath.Join(projectRoot, "cmd", "main.go"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// RouterPath returns the router file of a supported Go layout, or "" if none is found.
func RouterPath(projectRoot string) string {
	return findRouterPath(projectRoot)
}

// CheckGoHTTPProject verifies that projectRoot is a Go project with a router
// (any supported architecture) and, when requireMain is set, a server entry point.
func CheckGoHTTPProject(projectRoot string, requireMain bool) error {
	goMod := filepath.Join(projectRoot, "go.mod")
	if _, err := os.Stat(goMod); err != nil {
		return fmt.Errorf("fichier requis manquant : %s", goMod)
	}
	if findRouterPath(projectRoot) == "" {
		return fmt.Errorf("router introuvable dans le projet (architectures supportées: hexagonal, clean, layered)")
	}
	if requireMain && findMainPath(projectRoot) == "" {
		return fmt.Errorf("main.go introuvable dans le projet (attendu : cmd/server/main.go)")
	}
	return nil
}

// resolveRouterPath returns the explicit path or auto-detects it.
func resolveRouterPath(projectRoot string, integration RouterIntegration) (string, error) {
	if integration.RouterPath != "" {
		return integration.RouterPath, nil
	}
	if p := findRouterPath(projectRoot); p != "" {
		return p, nil
	}
	return "", fmt.Errorf("router.go introuvable dans le projet (architectures supportées: hexagonal, clean, layered)")
}

// resolveMainPath returns the explicit path or auto-detects it.
func resolveMainPath(projectRoot string, integration MainIntegration) (string, error) {
	if integration.MainPath != "" {
		return integration.MainPath, nil
	}
	if p := findMainPath(projectRoot); p != "" {
		return p, nil
	}
	return "", fmt.Errorf("main.go introuvable dans le projet")
}

// addImport appends spec at the end of the first import block, i.e. in the
// group holding the module's own imports. It reports false without a block.
func addImport(src, spec string) (string, bool) {
	start := strings.Index(src, "import (")
	if start < 0 {
		return src, false
	}
	end := strings.Index(src[start:], "\n)")
	if end < 0 {
		return src, false
	}
	end += start
	return src[:end] + "\n\t" + spec + src[end:], true
}

// routerAnchors are the lines after which feature statements are inserted, in
// order of preference: the v0.3 middleware stack, then the router creation used
// by every v0.4 architecture.
var routerAnchors = []string{
	`r.Use(middleware.Timeout(30 * time.Second))`,
	`r := chi.NewRouter()`,
}

// IntegrateRouterGo safely integrates a feature into router.go
// It handles multiple features by properly merging imports and middleware calls
func IntegrateRouterGo(projectRoot string, integration RouterIntegration) error {
	routerPath, err := resolveRouterPath(projectRoot, integration)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(routerPath)
	if err != nil {
		return fmt.Errorf("lire router.go : %w", err)
	}

	src := string(content)

	// Check if already integrated
	if strings.Contains(src, integration.MiddlewareCheck) || strings.Contains(src, `"`+integration.ImportPath+`"`) {
		return nil // Already integrated
	}

	// 1. Add import
	src, ok := addImport(src, `"`+integration.ImportPath+`"`)
	if !ok {
		return fmt.Errorf("bloc import introuvable dans router.go")
	}

	// 2. Insert the statement. Middlewares go right after a known anchor and
	// routes right before "return r": chi panics when r.Use follows a route.
	inserted := false
	if integration.IsRoute {
		const ret = "\n\treturn r\n"
		if i := strings.LastIndex(src, ret); i >= 0 {
			src = src[:i] + "\n\t" + integration.MiddlewareCall + src[i:]
			inserted = true
		}
	} else {
		for _, anchor := range routerAnchors {
			if strings.Contains(src, anchor) {
				src = strings.Replace(src, anchor, anchor+"\n\t"+integration.MiddlewareCall, 1)
				inserted = true
				break
			}
		}
	}
	if !inserted {
		return fmt.Errorf("point d'intégration middleware introuvable dans router.go")
	}

	if err := os.WriteFile(routerPath, []byte(src), 0o644); err != nil {
		return fmt.Errorf("écrire router.go : %w", err)
	}

	return nil
}

// RemoveRouterGo safely removes a feature's integration from router.go
// It only removes the specific feature's import and middleware call, preserving other features
func RemoveRouterGo(projectRoot string, integration RouterIntegration) error {
	routerPath, err := resolveRouterPath(projectRoot, integration)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(routerPath)
	if err != nil {
		return fmt.Errorf("lire router.go : %w", err)
	}

	src := string(content)

	// 1. Remove the statement. Matching on MiddlewareCheck also removes the
	// forms written by earlier ForgeKit versions.
	lines := strings.Split(src, "\n")
	kept := lines[:0]
	for _, line := range lines {
		if strings.Contains(line, integration.MiddlewareCheck) {
			continue
		}
		kept = append(kept, line)
	}
	src = strings.Join(kept, "\n")

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
	mainPath, err := resolveMainPath(projectRoot, integration)
	if err != nil {
		return err
	}
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

// IntegrateMainGo safely integrates a feature into main.go.
// When the integration declares replacements and none of them matches, main.go
// does not use the layout the feature targets: it is left untouched rather than
// receiving an import nothing would use (which would break compilation).
func IntegrateMainGo(projectRoot string, integration MainIntegration) error {
	mainPath, err := resolveMainPath(projectRoot, integration)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("lire main.go : %w", err)
	}

	src := string(content)

	// Check if already integrated (either import format)
	importSpec := `"` + integration.ImportPath + `"`
	if strings.Contains(src, importSpec) {
		return nil
	}

	// 1. Apply replacements
	applied := 0
	for _, repl := range integration.Replacements {
		if repl.Check != "" && !strings.Contains(src, repl.Check) {
			continue // Skip if check string not found
		}
		if strings.Contains(src, repl.OldStr) {
			src = strings.Replace(src, repl.OldStr, repl.NewStr, 1)
			applied++
		}
	}
	if len(integration.Replacements) > 0 && applied == 0 && !integration.BlankImport {
		return nil
	}

	// 2. Add import
	if integration.BlankImport {
		importSpec = `_ ` + importSpec
	}
	src, ok := addImport(src, importSpec)
	if !ok {
		return fmt.Errorf("bloc import introuvable dans main.go")
	}

	if err := os.WriteFile(mainPath, []byte(src), 0o644); err != nil {
		return fmt.Errorf("écrire main.go : %w", err)
	}

	return nil
}
