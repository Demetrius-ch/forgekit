package rules

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Demetrius-ch/forgekit/internal/report"
)

func TestEnvFileRule_MissingEnvExampleIsWarning(t *testing.T) {
	dir := t.TempDir()
	rule := EnvFileRule{}
	findings, err := rule.Run(context.Background(), Context{ProjectRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(findings))
	}
	f := findings[0]
	if f.ID != "project.env.example.missing" {
		t.Fatalf("expected project.env.example.missing, got %q", f.ID)
	}
	if f.Severity != report.SeverityWarning {
		t.Fatalf("expected warning severity, got %q", f.Severity)
	}
}

func TestEnvFileRule_MissingEnvIsWarning(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env.example"), []byte("KEY=VALUE\n"), 0o644); err != nil {
		t.Fatalf("write .env.example: %v", err)
	}
	rule := EnvFileRule{}
	findings, err := rule.Run(context.Background(), Context{ProjectRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(findings))
	}
	f := findings[0]
	if f.ID != "project.env" {
		t.Fatalf("expected project.env, got %q", f.ID)
	}
	if f.Severity != report.SeverityWarning {
		t.Fatalf("expected warning severity, got %q", f.Severity)
	}
}

func TestDockerRule_NoDockerFilesWarnsOnce(t *testing.T) {
	dir := t.TempDir()
	rule := DockerRule{}
	findings, err := rule.Run(context.Background(), Context{ProjectRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(findings))
	}
	if findings[0].ID != "docker.project.missing" {
		t.Fatalf("expected docker.project.missing, got %q", findings[0].ID)
	}
}

func TestDockerRule_ContainerFilesPresentButCLIOrDaemonIssue(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch"), 0o644); err != nil {
		t.Fatalf("write Dockerfile: %v", err)
	}
	rule := DockerRule{}
	findings, err := rule.Run(context.Background(), Context{ProjectRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(findings))
	}
	if findings[0].ID != "docker.cli.missing" && findings[0].ID != "docker.daemon.stopped" && findings[0].ID != "docker.ready" {
		t.Fatalf("unexpected docker finding id %q", findings[0].ID)
	}
}
