package feature

import "context"

// ProjectContext describes the ForgeKit project receiving a feature.
type ProjectContext struct {
	Root      string
	Module    string
	GoVersion string
	HTTPPort  int
}

// Feature represents an installable ForgeKit feature.
type Feature interface {
	Name() string
	Description() string
	Version() string

	Check(ctx context.Context, project ProjectContext) error
	Plan(ctx context.Context, project ProjectContext) (Plan, error)
	Apply(ctx context.Context, project ProjectContext, plan Plan) error
}
