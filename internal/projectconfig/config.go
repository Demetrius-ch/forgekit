// Package projectconfig defines the single, typed source of truth for a
// ForgeKit project's generation choices.
package projectconfig

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Language identifies the programming language for the generated project.
type Language string

const (
	LanguageGo         Language = "go"
	LanguageTypeScript Language = "typescript"
	LanguageJavaScript Language = "javascript"
	LanguagePython     Language = "python"
)

// Runtime identifies the execution environment for JS/TS projects.
// For Go projects this is always RuntimeNone.
type Runtime string

const (
	RuntimeNone Runtime = "none"
	RuntimeNode Runtime = "node"
)

// Architecture identifies the project layout to generate.
type Architecture string

const (
	ArchitectureHexagonal Architecture = "hexagonal"
	ArchitectureClean     Architecture = "clean"
	ArchitectureLayered   Architecture = "layered"
)

// Database identifies the persistence technology selected for a project.
type Database string

const (
	DatabasePostgres Database = "postgres"
	DatabaseMySQL    Database = "mysql"
	DatabaseSQLite   Database = "sqlite"
	DatabaseNone     Database = "none"
)

// Authentication identifies the authentication component selected for a
// project.
type Authentication string

const (
	AuthenticationNone Authentication = "none"
	AuthenticationJWT  Authentication = "jwt"
)

// Documentation identifies the API documentation component selected for a
// project.
type Documentation string

const (
	DocumentationNone    Documentation = "none"
	DocumentationSwagger Documentation = "swagger"
)

// TestStrategy identifies the test suite generated with a project.
type TestStrategy string

const (
	TestStrategyUnit            TestStrategy = "unit"
	TestStrategyUnitIntegration TestStrategy = "unit-integration"
)

// CIStrategy identifies the CI configuration generated with a project.
type CIStrategy string

const (
	CIStrategyNone   CIStrategy = "none"
	CIStrategyGitHub CIStrategy = "github-actions"
)

// ProjectConfig is the central v0.6 configuration for a generated project.
// It is persisted in .forge/forge.yaml and drives generation planning.
type ProjectConfig struct {
	Name           string         `yaml:"name" json:"name"`
	ModulePath     string         `yaml:"module" json:"module"`
	Language       Language       `yaml:"language" json:"language"`
	Runtime        Runtime        `yaml:"runtime" json:"runtime"`
	Architecture   Architecture   `yaml:"architecture" json:"architecture"`
	Database       Database       `yaml:"database" json:"database"`
	Docker         bool           `yaml:"docker" json:"docker"`
	Authentication Authentication `yaml:"authentication" json:"authentication"`
	Documentation  Documentation  `yaml:"documentation" json:"documentation"`
	Tests          TestStrategy   `yaml:"tests" json:"tests"`
	CI             CIStrategy     `yaml:"ci" json:"ci"`
}

// Default returns the v0.6-compatible generation profile. Optional components
// are disabled until a caller selects them explicitly.
func Default(name, modulePath string) ProjectConfig {
	return ProjectConfig{
		Name:           name,
		ModulePath:     modulePath,
		Language:       LanguageGo,
		Runtime:        RuntimeNone,
		Architecture:   ArchitectureHexagonal,
		Database:       DatabasePostgres,
		Docker:         true,
		Authentication: AuthenticationNone,
		Documentation:  DocumentationNone,
		Tests:          TestStrategyUnit,
		CI:             CIStrategyNone,
	}
}

// IsZero reports whether no v0.6 configuration was supplied. It permits the
// existing v0.3 API to receive the compatibility defaults during migration.
func (c ProjectConfig) IsZero() bool {
	return c == (ProjectConfig{})
}

// IsGo reports whether this is a Go project.
func (c ProjectConfig) IsGo() bool {
	return c.Language == LanguageGo
}

// IsNode reports whether this project uses Node.js runtime.
func (c ProjectConfig) IsNode() bool {
	return c.Runtime == RuntimeNode
}

// IsPython reports whether this is a Python project.
func (c ProjectConfig) IsPython() bool {
	return c.Language == LanguagePython
}

// Validate verifies the complete configuration before a generation plan is
// built. Keeping validation here prevents component-specific checks from
// drifting apart across the CLI and generator.
func (c ProjectConfig) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("project name is required")
	}
	if strings.TrimSpace(c.ModulePath) == "" {
		return fmt.Errorf("module path is required")
	}
	if !isLanguage(c.Language) {
		return fmt.Errorf("invalid language %q; supported languages: go, typescript, javascript, python", c.Language)
	}
	if !isRuntime(c.Runtime) {
		return fmt.Errorf("invalid runtime %q; supported runtimes: none, node", c.Runtime)
	}
	if !isArchitecture(c.Architecture) {
		return fmt.Errorf("invalid architecture %q; supported architectures: hexagonal, clean, layered", c.Architecture)
	}
	if !isDatabase(c.Database) {
		return fmt.Errorf("invalid database %q; supported databases: postgres, mysql, sqlite, none", c.Database)
	}
	if !isAuthentication(c.Authentication) {
		return fmt.Errorf("invalid authentication %q; supported authentication modes: none, jwt", c.Authentication)
	}
	if !isDocumentation(c.Documentation) {
		return fmt.Errorf("invalid documentation %q; supported documentation modes: none, swagger", c.Documentation)
	}
	if !isTestStrategy(c.Tests) {
		return fmt.Errorf("invalid test strategy %q; supported test strategies: unit, unit-integration", c.Tests)
	}
	if !isCIStrategy(c.CI) {
		return fmt.Errorf("invalid CI strategy %q; supported CI strategies: none, github-actions", c.CI)
	}
	if err := validateLanguageRuntime(c.Language, c.Runtime); err != nil {
		return err
	}
	return nil
}

// LoadFile reads a YAML project configuration. Project identity may be omitted
// from the file and supplied by `forge init <name> --module ...` instead.
func LoadFile(path string) (ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProjectConfig{}, fmt.Errorf("read project configuration: %w", err)
	}
	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return ProjectConfig{}, fmt.Errorf("parse project configuration: %w", err)
	}
	return config, nil
}

// WithIdentity applies the CLI project identity to a file configuration. A
// conflicting explicit identity is rejected instead of silently generating a
// project different from the user's command.
func (c ProjectConfig) WithIdentity(name, modulePath string) (ProjectConfig, error) {
	if c.Name == "" {
		c.Name = name
	} else if c.Name != name {
		return ProjectConfig{}, fmt.Errorf("configuration name %q does not match init project name %q", c.Name, name)
	}
	if c.ModulePath == "" {
		c.ModulePath = modulePath
	} else if c.ModulePath != modulePath {
		return ProjectConfig{}, fmt.Errorf("configuration module %q does not match init module %q", c.ModulePath, modulePath)
	}
	return c, nil
}

func isLanguage(value Language) bool {
	return value == LanguageGo || value == LanguageTypeScript || value == LanguageJavaScript || value == LanguagePython
}

func isRuntime(value Runtime) bool {
	return value == RuntimeNone || value == RuntimeNode
}

func validateLanguageRuntime(language Language, runtime Runtime) error {
	switch language {
	case LanguageGo:
		if runtime != RuntimeNone {
			return fmt.Errorf("Go projects do not support a runtime; use runtime=none")
		}
	case LanguageTypeScript, LanguageJavaScript:
		if runtime != RuntimeNode {
			return fmt.Errorf("%s projects require runtime=node for v0.6", language)
		}
	case LanguagePython:
		if runtime != RuntimeNone {
			return fmt.Errorf("Python projects do not support a runtime; use runtime=none")
		}
	}
	return nil
}

func isArchitecture(value Architecture) bool {
	return value == ArchitectureHexagonal || value == ArchitectureClean || value == ArchitectureLayered
}

func isDatabase(value Database) bool {
	return value == DatabasePostgres || value == DatabaseMySQL || value == DatabaseSQLite || value == DatabaseNone
}

func isAuthentication(value Authentication) bool {
	return value == AuthenticationNone || value == AuthenticationJWT
}

func isDocumentation(value Documentation) bool {
	return value == DocumentationNone || value == DocumentationSwagger
}

func isTestStrategy(value TestStrategy) bool {
	return value == TestStrategyUnit || value == TestStrategyUnitIntegration
}

func isCIStrategy(value CIStrategy) bool {
	return value == CIStrategyNone || value == CIStrategyGitHub
}
