package feature

import (
	"context"
	"testing"
)

type testFeature struct {
	name string
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
