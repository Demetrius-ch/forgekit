package cli

import (
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
	err := runReport(g, "analyze", ".", reg)
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
		_ = runAnalyze(g, dir, loader)
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
		_ = runAnalyze(g, dir, loader)
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
