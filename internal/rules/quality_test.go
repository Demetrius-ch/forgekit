package rules

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGracefulShutdownRule(t *testing.T) {
	t.Run("warns when shutdown is missing", func(t *testing.T) {
		dir := t.TempDir()
		mainPath := filepath.Join(dir, "cmd", "server", "main.go")
		if err := os.MkdirAll(filepath.Dir(mainPath), 0o755); err != nil {
			t.Fatal(err)
		}
		content := []byte(`package main

import "net/http"

func main() {
	server := &http.Server{}
	_ = server.ListenAndServe()
}
`)
		if err := os.WriteFile(mainPath, content, 0o644); err != nil {
			t.Fatal(err)
		}

		rule := GracefulShutdownRule{}
		findings, err := rule.Run(context.Background(), Context{ProjectRoot: dir})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if len(findings) != 1 {
			t.Fatalf("expected one finding, got %d", len(findings))
		}
		if findings[0].Severity != "warning" {
			t.Fatalf("expected warning severity, got %s", findings[0].Severity)
		}
	})

	t.Run("passes when shutdown is configured", func(t *testing.T) {
		dir := t.TempDir()
		mainPath := filepath.Join(dir, "cmd", "server", "main.go")
		if err := os.MkdirAll(filepath.Dir(mainPath), 0o755); err != nil {
			t.Fatal(err)
		}
		content := []byte(`package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	server := &http.Server{}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	_ = server.ListenAndServe()
	_ = server.Shutdown(nil)
}
`)
		if err := os.WriteFile(mainPath, content, 0o644); err != nil {
			t.Fatal(err)
		}

		rule := GracefulShutdownRule{}
		findings, err := rule.Run(context.Background(), Context{ProjectRoot: dir})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if len(findings) != 1 {
			t.Fatalf("expected one finding, got %d", len(findings))
		}
		if findings[0].Severity != "info" {
			t.Fatalf("expected info severity, got %s", findings[0].Severity)
		}
	})
}
