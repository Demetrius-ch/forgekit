package feature

import (
	"fmt"
	"sort"
)

// Registry stores the available ForgeKit features.
type Registry struct {
	features map[string]Feature
}

// NewRegistry creates a feature registry.
func NewRegistry(features ...Feature) *Registry {
	r := &Registry{
		features: make(map[string]Feature),
	}

	for _, f := range features {
		r.Register(f)
	}

	return r
}

// Register adds a feature to the registry.
func (r *Registry) Register(f Feature) {
	if f == nil {
		return
	}

	r.features[f.Name()] = f
}

// Get returns a feature by name.
func (r *Registry) Get(name string) (Feature, bool) {
	f, ok := r.features[name]
	return f, ok
}

// Require returns a feature or an explicit error.
func (r *Registry) Require(name string) (Feature, error) {
	f, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("feature inconnue : %s", name)
	}

	return f, nil
}

// List returns all registered features sorted by name.
func (r *Registry) List() []Feature {
	result := make([]Feature, 0, len(r.features))

	for _, f := range r.features {
		result = append(result, f)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})

	return result
}
