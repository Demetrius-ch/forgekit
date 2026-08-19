package feature

import (
	"context"
	"testing"
)

type testFeature struct {
	name string
	deps []string
}

func (f testFeature) Name() string {
	return f.name
}

func (f testFeature) Description() string {
	return "feature de test"
}

func (f testFeature) Version() string {
	return "0.1.0"
}

func (f testFeature) Check(context.Context, ProjectContext) error {
	return nil
}

func (f testFeature) Plan(context.Context, ProjectContext) (Plan, error) {
	return Plan{
		Feature: f.name,
		Version: "0.1.0",
	}, nil
}

func (f testFeature) Apply(
	context.Context,
	ProjectContext,
	Plan,
) error {
	return nil
}

func (f testFeature) DependsOn() []string {
	return f.deps
}

func TestRegistryRegisterAndGet(t *testing.T) {
	registry := NewRegistry()

	feature := testFeature{name: "auth"}

	registry.Register(feature)

	got, ok := registry.Get("auth")
	if !ok {
		t.Fatal("expected feature to be registered")
	}

	if got.Name() != "auth" {
		t.Fatalf("expected auth, got %s", got.Name())
	}
}

func TestRegistryUnknownFeature(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Require("unknown")
	if err == nil {
		t.Fatal("expected error for unknown feature")
	}
}

func TestRegistryListSorted(t *testing.T) {
	registry := NewRegistry(
		testFeature{name: "swagger"},
		testFeature{name: "auth"},
		testFeature{name: "redis"},
	)

	list := registry.List()

	if len(list) != 3 {
		t.Fatalf("expected 3 features, got %d", len(list))
	}

	expected := []string{
		"auth",
		"redis",
		"swagger",
	}

	for i, name := range expected {
		if list[i].Name() != name {
			t.Fatalf(
				"expected feature %d to be %s, got %s",
				i,
				name,
				list[i].Name(),
			)
		}
	}
}

func TestRegistryResolveDependencies(t *testing.T) {
	registry := NewRegistry(
		testFeature{name: "auth"},
		testFeature{name: "cors", deps: []string{"auth"}},
		testFeature{name: "logging", deps: []string{"auth"}},
		testFeature{name: "swagger", deps: []string{"cors", "logging"}},
	)

	// Test resolving a single feature with no deps
	resolved, err := registry.ResolveDependencies([]string{"auth"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 || resolved[0].Name() != "auth" {
		t.Fatalf("expected [auth], got %v", resolved)
	}

	// Test resolving feature with dependencies
	resolved, err = registry.ResolveDependencies([]string{"cors"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 2 || resolved[0].Name() != "auth" || resolved[1].Name() != "cors" {
		t.Fatalf("expected [auth, cors], got %v", resolved)
	}

	// Test resolving feature with transitive dependencies
	resolved, err = registry.ResolveDependencies([]string{"swagger"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 4 {
		t.Fatalf("expected 4 features, got %d", len(resolved))
	}
	// auth must come before cors and logging
	authIdx := -1
	corsIdx := -1
	loggingIdx := -1
	swaggerIdx := -1
	for i, f := range resolved {
		switch f.Name() {
		case "auth":
			authIdx = i
		case "cors":
			corsIdx = i
		case "logging":
			loggingIdx = i
		case "swagger":
			swaggerIdx = i
		}
	}
	if authIdx >= corsIdx || authIdx >= loggingIdx {
		t.Fatalf("auth must come before cors and logging: %v", resolved)
	}
	if corsIdx >= swaggerIdx || loggingIdx >= swaggerIdx {
		t.Fatalf("cors and logging must come before swagger: %v", resolved)
	}
}

func TestRegistryResolveDependenciesMissing(t *testing.T) {
	registry := NewRegistry(
		testFeature{name: "auth"},
	)

	_, err := registry.ResolveDependencies([]string{"cors"})
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}

func TestRegistryResolveDependenciesCycle(t *testing.T) {
	registry := NewRegistry(
		testFeature{name: "a", deps: []string{"b"}},
		testFeature{name: "b", deps: []string{"a"}},
	)

	_, err := registry.ResolveDependencies([]string{"a"})
	if err == nil {
		t.Fatal("expected error for circular dependency")
	}
}

func TestRegistryHasCycles(t *testing.T) {
	registry := NewRegistry(
		testFeature{name: "a", deps: []string{"b"}},
		testFeature{name: "b", deps: []string{"a"}},
	)

	if !registry.HasCycles([]string{"a"}) {
		t.Fatal("expected cycle detection")
	}

	registry2 := NewRegistry(
		testFeature{name: "auth"},
		testFeature{name: "cors", deps: []string{"auth"}},
	)

	if registry2.HasCycles([]string{"cors"}) {
		t.Fatal("expected no cycle")
	}
}

func TestRegistryFeaturesWithoutDependencies(t *testing.T) {
	// Features that don't implement FeatureDependencies should work fine
	registry := NewRegistry(
		testFeature{name: "auth"}, // no DependsOn method
	)

	resolved, err := registry.ResolveDependencies([]string{"auth"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 || resolved[0].Name() != "auth" {
		t.Fatalf("expected [auth], got %v", resolved)
	}
}
