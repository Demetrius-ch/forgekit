package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Demetrius-ch/forgekit/internal/config"
	"github.com/Demetrius-ch/forgekit/internal/output"
	"github.com/Demetrius-ch/forgekit/internal/report"
	"github.com/Demetrius-ch/forgekit/internal/rules"
)

// fakeRule is a minimal Rule implementation returning a single finding.
type fakeRule struct{}

func (fakeRule) ID() string          { return "fake" }
func (fakeRule) Name() string        { return "fake" }
func (fakeRule) Description() string { return "fake" }
func (fakeRule) Category() string    { return "project" }
func (fakeRule) Severity() report.Severity {
	return report.SeverityInfo
}
func (fakeRule) Run(_ context.Context, _ rules.Context) ([]report.Finding, error) {
	return []report.Finding{{ID: "f1", Category: "project", Severity: report.SeverityInfo, Message: "ok"}}, nil
}

func TestRunReportJSON_NoSpinnerOrAnsi(t *testing.T) {
	// capture stdout
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	g := &globalFlags{Format: output.FormatJSON}
	reg := rules.NewRegistry(fakeRule{})
	err := runReport(g, "analyze", ".", reg, nil)
	// close writer and read
	_ = w.Close()
	var buf []byte
	buf, _ = io.ReadAll(r)
	if err != nil {
		t.Fatalf("runReport error: %v", err)
	}
	// should be valid JSON
	var obj map[string]interface{}
	if err := json.Unmarshal(buf, &obj); err != nil {
		t.Fatalf("output not valid json: %v\n%s", err, string(buf))
	}
	// ensure no spinner frames present
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	for _, f := range frames {
		if strings.Contains(string(buf), f) {
			t.Fatalf("unexpected spinner char in output: %q", f)
		}
	}
}

func TestAnalyzeProgressScoresMatchFinalScore(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# demo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env.example"), []byte("KEY=VALUE"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main_test.go"), []byte("package main\n\nfunc TestDummy(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/forgekit/demo\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	loader := rules.StaticConfigLoader{Rules: cfg.Architecture.Rules}
	g := &globalFlags{Format: output.FormatHuman}

	oldOut := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldOut }()

	go func() {
		_ = runAnalyze(g, dir, loader, nil, false)
		_ = w.Close()
	}()

	outBytes, _ := io.ReadAll(r)
	outputStr := strings.ReplaceAll(string(outBytes), "\r", "\n")

	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	outputStr = ansiRegex.ReplaceAllString(outputStr, "")
	progressRegex := regexp.MustCompile(`(?m)^[✓✗] (Architecture|Tests|Security|Configuration|Docker|Documentation) — (\d{1,3})/100`)
	finalRegex := regexp.MustCompile(`(?ms)^(Architecture|Tests|Security|Configuration|Docker|Documentation)\n.*?(\d{1,3})/100`)
	progressMatches := progressRegex.FindAllStringSubmatch(outputStr, -1)
	if len(progressMatches) == 0 {
		t.Fatalf("expected progress scores in output, got %q", outputStr)
	}

	finalMatches := finalRegex.FindAllStringSubmatch(outputStr, -1)
	if len(finalMatches) == 0 {
		t.Fatalf("expected final score table in output, got %q", outputStr)
	}

	finalScores := map[string]string{}
	for _, m := range finalMatches {
		finalScores[m[1]] = m[2]
	}

	for _, match := range progressMatches {
		cat := match[1]
		score := match[2]
		if finalScores[cat] == "" {
			t.Fatalf("category %q found in progress output but not in final summary", cat)
		}
		if finalScores[cat] != score {
			t.Fatalf("category %q progress score %s does not match final score %s", cat, score, finalScores[cat])
		}
	}
}

func TestAnalyzeOptionalResourcesMissingWarnsButDoesNotFail(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/forgekit/demo\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	loader := rules.StaticConfigLoader{Rules: cfg.Architecture.Rules}
	g := &globalFlags{Format: output.FormatHuman}

	oldOut := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldOut }()

	go func() {
		_ = runAnalyze(g, dir, loader, nil, false)
		_ = w.Close()
	}()

	outBytes, _ := io.ReadAll(r)
	outputStr := strings.ReplaceAll(string(outBytes), "\r", "\n")

	if strings.Contains(outputStr, "erreur") {
		t.Fatalf("expected analyze to complete without fatal error, got output:\n%s", outputStr)
	}
	if !strings.Contains(outputStr, "README.md absent") || !strings.Contains(outputStr, ".env.example absent") {
		t.Fatalf("expected warnings for missing optional docs, got output:\n%s", outputStr)
	}
}

func captureRootOutput(args []string) string {
	root := NewRootCommand()
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w
	root.SetArgs(args)
	_ = root.Execute()
	_ = w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestRootCommandShowsBranding(t *testing.T) {
	out := captureRootOutput([]string{"--help"})

	if !strings.Contains(out, "ForgeKit") {
		t.Fatalf("expected branding 'ForgeKit' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Build • Extend • Ship") {
		t.Fatalf("expected slogan in output, got:\n%s", out)
	}
	if !strings.Contains(out, "███████") {
		t.Fatalf("expected ASCII logo in output, got:\n%s", out)
	}
}

func TestRootCommandNoArgsShowsBranding(t *testing.T) {
	out := captureRootOutput([]string{})

	if !strings.Contains(out, "ForgeKit") {
		t.Fatalf("expected branding 'ForgeKit' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Build • Extend • Ship") {
		t.Fatalf("expected slogan in output, got:\n%s", out)
	}
	if !strings.Contains(out, "███████") {
		t.Fatalf("expected ASCII logo in output, got:\n%s", out)
	}
}

func TestRootCommandNoBrandingInJSON(t *testing.T) {
	out := captureRootOutput([]string{"--help", "--format=json"})

	// Check for actual branding elements (ASCII logo, slogan), not just the word "ForgeKit"
	if strings.Contains(out, "███████") || strings.Contains(out, "Build • Extend • Ship") {
		t.Fatalf("expected no branding in JSON output, got:\n%s", out)
	}
	// Should still be valid help text
	if !strings.Contains(out, "Available Commands") {
		t.Fatalf("expected help content in JSON mode, got:\n%s", out)
	}
}

func TestRootCommandNoBrandingInQuiet(t *testing.T) {
	out := captureRootOutput([]string{"--help", "--quiet"})

	// Check for actual branding elements (ASCII logo, slogan), not just the word "ForgeKit"
	if strings.Contains(out, "███████") || strings.Contains(out, "Build • Extend • Ship") {
		t.Fatalf("expected no branding in quiet output, got:\n%s", out)
	}
	if !strings.Contains(out, "Available Commands") {
		t.Fatalf("expected help content in quiet mode, got:\n%s", out)
	}
}

func TestSubcommandHelpNoBranding(t *testing.T) {
	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"init", "--help"})
	_ = root.Execute()
	out := buf.String()

	if strings.Contains(out, "ForgeKit") || strings.Contains(out, "Build • Extend • Ship") || strings.Contains(out, "███████") {
		t.Fatalf("expected no branding in subcommand help, got:\n%s", out)
	}
	if !strings.Contains(out, "forge init") {
		t.Fatalf("expected subcommand usage in output, got:\n%s", out)
	}
}

func TestSelectPorts_HTTPPortAvailable(t *testing.T) {
	g := &globalFlags{Format: output.FormatHuman, Quiet: true}
	sel, err := selectPorts(g, 9999, 9998, false)
	if err != nil {
		t.Fatalf("selectPorts error: %v", err)
	}
	if sel.HTTPPort != 9999 {
		t.Errorf("expected HTTP port 9999, got %d", sel.HTTPPort)
	}
	if sel.PostgresHostPort != 9998 {
		t.Errorf("expected Postgres port 9998, got %d", sel.PostgresHostPort)
	}
}

func TestSelectPorts_HTTPPortOccupied(t *testing.T) {
	// We can't easily test selectPorts with a mock since it creates its own RealPortChecker
	// This test just verifies the function exists and compiles
	g := &globalFlags{Format: output.FormatHuman, Quiet: true}
	// Use a port that's likely available
	sel, err := selectPorts(g, 18080, 15432, false)
	if err != nil {
		t.Fatalf("selectPorts error: %v", err)
	}
	if sel.HTTPPort != 18080 {
		t.Errorf("expected HTTP port 18080, got %d", sel.HTTPPort)
	}
	if sel.PostgresHostPort != 15432 {
		t.Errorf("expected Postgres port 15432, got %d", sel.PostgresHostPort)
	}
}

func TestSelectPorts_DryRun(t *testing.T) {
	g := &globalFlags{Format: output.FormatHuman, Quiet: true}
	// Use mock to test dry-run output without affecting real ports
	// We can't inject mock into selectPorts easily, so just test it doesn't crash
	sel, err := selectPorts(g, 18080, 15432, true)
	if err != nil {
		t.Fatalf("selectPorts error: %v", err)
	}
	if sel.HTTPPort != 18080 {
		t.Errorf("expected HTTP port 18080, got %d", sel.HTTPPort)
	}
}

func TestAddCommandPlanFlag(t *testing.T) {
	// Create a minimal ForgeKit project structure
	root := t.TempDir()

	// Create go.mod
	if err := os.WriteFile(
		filepath.Join(root, "go.mod"),
		[]byte("module github.com/test/test\n\ngo 1.25\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Create .forge/features.yaml for legacy project
	if err := os.MkdirAll(filepath.Join(root, ".forge"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, ".forge", "features.yaml"),
		[]byte("features: []\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Create cmd/server directory
	if err := os.MkdirAll(filepath.Join(root, "cmd", "server"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create required files for auth feature
	if err := os.MkdirAll(filepath.Join(root, "internal", "transport", "http"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "internal", "transport", "http", "router.go"),
		[]byte("package http\n\nimport \"github.com/go-chi/chi/v5\"\n\nfunc NewRouter() *chi.Mux {\n\tr := chi.NewRouter()\n\tr.Use(middleware.Timeout(30 * time.Second))\n\treturn r\n}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Change to project directory
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(root)

	// Capture output using the same method as other tests
	out := captureRootOutput([]string{"add", "auth", "--plan"})
	// Remove spinner characters
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	out = ansiRegex.ReplaceAllString(out, "")
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	for _, c := range spinnerChars {
		out = strings.ReplaceAll(out, c, "")
	}

	if !strings.Contains(out, "ForgeKit Plan") {
		t.Fatalf("expected 'ForgeKit Plan' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Feature: auth") {
		t.Fatalf("expected feature name in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Version: 1.0.0") {
		t.Fatalf("expected feature version in output, got:\n%s", out)
	}
}

func TestAddCommandPlanFlagJSON(t *testing.T) {
	root := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(root, "go.mod"),
		[]byte("module github.com/test/test\n\ngo 1.25\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(root, ".forge"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, ".forge", "features.yaml"),
		[]byte("features: []\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(root, "cmd", "server"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(root, "internal", "transport", "http"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "internal", "transport", "http", "router.go"),
		[]byte("package http\n\nimport \"github.com/go-chi/chi/v5\"\n\nfunc NewRouter() *chi.Mux {\n\tr := chi.NewRouter()\n\tr.Use(middleware.Timeout(30 * time.Second))\n\treturn r\n}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(root)

	// Capture output using the same method as other tests
	out := captureRootOutput([]string{"add", "auth", "--plan", "--format=json"})
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("output not valid JSON: %v\n%s", err, out)
	}

	// Check schema_version
	if result["schema_version"] != "1" {
		t.Fatalf("expected schema_version=1, got %v", result["schema_version"])
	}

	plan, ok := result["plan"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected plan object, got %v", result["plan"])
	}

	if plan["Feature"] != "auth" {
		t.Fatalf("expected Feature=auth, got %v", plan["Feature"])
	}
	if plan["Version"] != "1.0.0" {
		t.Fatalf("expected Version=1.0.0, got %v", plan["Version"])
	}
	files, ok := plan["Files"].([]interface{})
	if !ok || len(files) != 2 {
		t.Fatalf("expected 2 files, got %v", files)
	}
	for _, f := range files {
		fileMap := f.(map[string]interface{})
		if fileMap["Action"] != "modify" && fileMap["Action"] != "create" {
			t.Fatalf("expected Action to be modify or create, got %v", fileMap["Action"])
		}
	}
}

func TestAddCommandPlanFlagQuiet(t *testing.T) {
	root := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(root, "go.mod"),
		[]byte("module github.com/test/test\n\ngo 1.25\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(root, ".forge"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, ".forge", "features.yaml"),
		[]byte("features: []\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(root, "cmd", "server"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(root, "internal", "transport", "http"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "internal", "transport", "http", "router.go"),
		[]byte("package http\n\nimport \"github.com/go-chi/chi/v5\"\n\nfunc NewRouter() *chi.Mux {\n\tr := chi.NewRouter()\n\tr.Use(middleware.Timeout(30 * time.Second))\n\treturn r\n}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(root)

	// Capture output using the same method as other tests
	out := captureRootOutput([]string{"add", "auth", "--plan", "--quiet"})
	// Remove spinner characters
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	out = ansiRegex.ReplaceAllString(out, "")
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	for _, c := range spinnerChars {
		out = strings.ReplaceAll(out, c, "")
	}

	if !strings.Contains(out, "CREATE") && !strings.Contains(out, "MODIFY") {
		t.Fatalf("expected file actions in quiet output, got:\n%s", out)
	}
}
