package feature

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const v04Router = `package transport

import (
	"encoding/json"
	"net/http"

	"example.com/app/internal/application"
	"github.com/go-chi/chi/v5"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(application.Health())
	})
	return r
}
`

const v04Main = `package main

import (
	"log/slog"
	"os"

	transport "example.com/app/internal/transport/http"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	_ = transport.NewRouter()
	logger.Info("ok")
}
`

func writeV04Project(t *testing.T, routerRel string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range map[string]string{
		"go.mod":             "module example.com/app\n",
		routerRel:            v04Router,
		"cmd/server/main.go": v04Main,
	} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestIntegrateRouterGoV04Architectures(t *testing.T) {
	for _, routerRel := range []string{
		"internal/transport/http/router.go",  // hexagonal
		"internal/interfaces/http/router.go", // clean
		"internal/handlers/http.go",          // layered
	} {
		t.Run(routerRel, func(t *testing.T) {
			root := writeV04Project(t, routerRel)
			if err := CheckGoHTTPProject(root, true); err != nil {
				t.Fatalf("CheckGoHTTPProject: %v", err)
			}

			for _, integration := range []RouterIntegration{
				{ImportPath: "example.com/app/internal/swagger", MiddlewareCall: "swagger.RegisterRoutes(r)", MiddlewareCheck: "swagger.RegisterRoutes", IsRoute: true},
				{ImportPath: "example.com/app/internal/cors", MiddlewareCall: "r.Use(cors.Middleware())", MiddlewareCheck: "cors.Middleware()"},
			} {
				if err := IntegrateRouterGo(root, integration); err != nil {
					t.Fatalf("IntegrateRouterGo(%s): %v", integration.ImportPath, err)
				}
			}

			data, err := os.ReadFile(filepath.Join(root, routerRel))
			if err != nil {
				t.Fatal(err)
			}
			src := string(data)
			use := strings.Index(src, "r.Use(cors.Middleware())")
			get := strings.Index(src, "r.Get(")
			routes := strings.Index(src, "swagger.RegisterRoutes(r)")
			if use < 0 || routes < 0 || !(use < get && get < routes) {
				t.Fatalf("expected middleware before routes and swagger after them:\n%s", src)
			}
			if !strings.Contains(src, `"example.com/app/internal/cors"`) {
				t.Fatalf("cors import missing:\n%s", src)
			}

			if err := RemoveRouterGo(root, RouterIntegration{ImportPath: "example.com/app/internal/cors", MiddlewareCheck: "cors.Middleware()"}); err != nil {
				t.Fatalf("RemoveRouterGo: %v", err)
			}
			data, _ = os.ReadFile(filepath.Join(root, routerRel))
			if strings.Contains(string(data), "cors") {
				t.Fatalf("cors still present after removal:\n%s", data)
			}
		})
	}
}

func TestIntegrateMainGoLeavesUnmatchedLayoutUntouched(t *testing.T) {
	root := writeV04Project(t, "internal/transport/http/router.go")

	err := IntegrateMainGo(root, MainIntegration{
		ImportPath: "example.com/app/internal/logging",
		Replacements: []MainReplacement{{
			OldStr: `slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))`,
			NewStr: `logging.DefaultLogger()`,
		}},
	})
	if err != nil {
		t.Fatalf("IntegrateMainGo: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "cmd", "server", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != v04Main {
		t.Fatalf("main.go must stay untouched when no replacement applies, got:\n%s", data)
	}
}
