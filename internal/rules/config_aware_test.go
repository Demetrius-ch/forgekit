package rules

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Demetrius-ch/forgekit/internal/projectconfig"
	"github.com/Demetrius-ch/forgekit/internal/report"
)

func writeRuleFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDockerRule_DisabledInProjectConfigIsInfo(t *testing.T) {
	cfg := projectconfig.Default("app", "example.com/app")
	cfg.Docker = false
	findings, err := DockerRule{}.Run(context.Background(), Context{ProjectRoot: t.TempDir(), Project: &cfg})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].ID != "docker.disabled" || findings[0].Severity != report.SeverityInfo {
		t.Fatalf("expected a single docker.disabled info finding, got %+v", findings)
	}
}

func TestPostgreSQLRule_SkippedWhenProjectUsesAnotherDatabase(t *testing.T) {
	root := t.TempDir()
	writeRuleFile(t, root, ".env.example", "HTTP_PORT=8080\n")
	for _, db := range []projectconfig.Database{projectconfig.DatabaseNone, projectconfig.DatabaseSQLite, projectconfig.DatabaseMySQL} {
		cfg := projectconfig.Default("app", "example.com/app")
		cfg.Database = db
		findings, err := PostgreSQLRule{}.Run(context.Background(), Context{ProjectRoot: root, Project: &cfg})
		if err != nil {
			t.Fatal(err)
		}
		if len(findings) != 0 {
			t.Fatalf("database %s: expected no PostgreSQL finding, got %+v", db, findings)
		}
	}

	// Without project configuration the rule keeps checking .env.example.
	findings, err := PostgreSQLRule{}.Run(context.Background(), Context{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) == 0 {
		t.Fatal("expected PostgreSQL findings when the project configuration is unknown")
	}
}

func TestSecuritySensitiveFilesRule_IgnoredEnvIsInfo(t *testing.T) {
	root := t.TempDir()
	writeRuleFile(t, root, ".env", "KEY=value\n")
	writeRuleFile(t, root, ".gitignore", ".env\n")
	findings, err := SecuritySensitiveFilesRule{}.Run(context.Background(), Context{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Severity != report.SeverityInfo {
		t.Fatalf("expected one info finding for an ignored .env, got %+v", findings)
	}
}

func TestSecuritySecretsRule(t *testing.T) {
	const awsSecret = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYzzzzzzzzzz"
	tests := []struct {
		name    string
		file    string
		content string
		want    bool
	}{
		{
			name:    "long import path",
			file:    "cmd/server/main.go",
			content: "package main\n\nimport \"github.com/acme/d3/internal/infrastructure/config\"\n",
		},
		{
			name:    "aws secret assignment",
			file:    "internal/aws/aws.go",
			content: "package aws\n\nconst awsSecretAccessKey = \"" + awsSecret + "\"\n",
			want:    true,
		},
		{
			name:    "secret in a file that mentions test elsewhere",
			file:    "internal/aws/client.go",
			content: "package aws\n\n// latest client\nconst awsSecretAccessKey = \"" + awsSecret + "\"\n",
			want:    true,
		},
		{
			name:    "default local credentials",
			file:    ".env.example",
			content: "DATABASE_URL=postgres://postgres:postgres@localhost:5432/app?sslmode=disable\n",
		},
		{
			name:    "real connection string password",
			file:    "config.yaml",
			content: "url: postgres://admin:S3cr3tPassw0rd@db.internal:5432/app\n",
			want:    true,
		},
		{
			name:    "test files are skipped",
			file:    "internal/aws/aws_test.go",
			content: "package aws\n\nconst awsSecretAccessKey = \"" + awsSecret + "\"\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeRuleFile(t, root, tt.file, tt.content)
			findings, err := SecuritySecretsRule{}.Run(context.Background(), Context{ProjectRoot: root})
			if err != nil {
				t.Fatal(err)
			}
			if got := len(findings) > 0; got != tt.want {
				t.Fatalf("detected = %v, want %v (%+v)", got, tt.want, findings)
			}
			for _, f := range findings {
				if strings.Contains(f.Explanation, "S3cr3t") || strings.Contains(f.Explanation, awsSecret) {
					t.Fatalf("finding must not echo the secret value: %q", f.Explanation)
				}
			}
		})
	}
}
