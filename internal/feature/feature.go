package feature

import (
	"context"
	"errors"
)

// ErrAlreadyInstalled is wrapped by Feature.Check when the feature is already
// present in the project; installation of that feature is then skipped.
var ErrAlreadyInstalled = errors.New("déjà installée")

// Feature represents an installable ForgeKit feature.
type Feature interface {
	Name() string
	Description() string
	Version() string

	Check(ctx context.Context, project ProjectContext) error
	Plan(ctx context.Context, project ProjectContext) (Plan, error)
	Apply(ctx context.Context, project ProjectContext, plan Plan) error
}

// FeatureDependencies is an optional interface for features that declare dependencies on other features.
type FeatureDependencies interface {
	DependsOn() []string
}

// FeatureRemover is an optional interface for features that support removal.
type FeatureRemover interface {
	Remove(ctx context.Context, project ProjectContext, plan Plan) error
}
