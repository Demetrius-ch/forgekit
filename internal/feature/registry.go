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

// ResolveDependencies returns features in topological order based on their DependsOn() declarations.
// Features without FeatureDependencies interface are treated as having no dependencies.
// Returns an error if a cycle is detected or a dependency is not registered.
func (r *Registry) ResolveDependencies(names []string) ([]Feature, error) {
	// Build complete dependency graph (including transitive dependencies)
	graph := make(map[string][]string)
	allFeatures := make(map[string]Feature)

	// Collect all features needed (requested + their transitive dependencies)
	var collect func([]string) error
	collect = func(featureNames []string) error {
		for _, name := range featureNames {
			if _, ok := allFeatures[name]; ok {
				continue // already processed
			}
			f, ok := r.Get(name)
			if !ok {
				return fmt.Errorf("feature %q non enregistrée", name)
			}
			allFeatures[name] = f

			var deps []string
			if fd, ok := f.(FeatureDependencies); ok {
				deps = fd.DependsOn()
			}
			graph[name] = deps

			// Ensure all dependencies are registered
			for _, dep := range deps {
				if _, ok := r.Get(dep); !ok {
					return fmt.Errorf("feature %q dépend de %q qui n'est pas enregistrée", name, dep)
				}
			}

			// Recursively collect dependencies
			if err := collect(deps); err != nil {
				return err
			}
		}
		return nil
	}

	if err := collect(names); err != nil {
		return nil, err
	}

	// Topological sort (Kahn's algorithm)
	var result []Feature
	inDegree := make(map[string]int)
	// reverseGraph maps dependency -> list of features that depend on it
	reverseGraph := make(map[string][]string)

	// Initialize inDegree for all nodes in graph
	for name := range graph {
		inDegree[name] = 0
		reverseGraph[name] = nil
	}
	// Build reverse graph and calculate in-degrees
	for name, deps := range graph {
		for _, dep := range deps {
			inDegree[name]++ // name depends on dep, so name's in-degree increases
			reverseGraph[dep] = append(reverseGraph[dep], name)
		}
	}

	queue := make([]string, 0)
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}
	// Sort queue for deterministic order
	sort.Strings(queue)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if f, ok := allFeatures[current]; ok {
			result = append(result, f)
		}

		for _, dependent := range reverseGraph[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
		// Sort queue for deterministic order
		sort.Strings(queue)
	}

	if len(result) != len(allFeatures) {
		return nil, fmt.Errorf("dépendance circulaire détectée entre features")
	}

	return result, nil
}

// HasCycles checks if there are circular dependencies among the given feature names.
func (r *Registry) HasCycles(names []string) bool {
	_, err := r.ResolveDependencies(names)
	return err != nil
}
