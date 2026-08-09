package rules

import (
	"context"

	"github.com/forgekit/forgekit/internal/report"
)

// Context carries shared state for rule evaluation.
type Context struct {
	ProjectRoot string
	Language    string
	GoVersion   string
	Extra       map[string]any
}

// Rule is a pluggable diagnostic rule.
type Rule interface {
	ID() string
	Name() string
	Description() string
	Category() string
	Severity() report.Severity
	Run(ctx context.Context, rctx Context) ([]report.Finding, error)
}

// Registry holds ordered rules.
type Registry struct {
	rules []Rule
}

func NewRegistry(rules ...Rule) *Registry {
	return &Registry{rules: rules}
}

func (r *Registry) Add(rule Rule) {
	r.rules = append(r.rules, rule)
}

func (r *Registry) Run(ctx context.Context, rctx Context) ([]report.Finding, error) {
	var findings []report.Finding
	for _, rule := range r.rules {
		fs, err := rule.Run(ctx, rctx)
		if err != nil {
			return findings, err
		}
		findings = append(findings, fs...)
	}
	return findings, nil
}

func (r *Registry) Rules() []Rule {
	out := make([]Rule, len(r.rules))
	copy(out, r.rules)
	return out
}
