package projectconfig

import (
	"strings"
	"testing"
)

func TestDefaultIsValid(t *testing.T) {
	t.Parallel()

	config := Default("my-api", "github.com/example/my-api")
	if err := config.Validate(); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
	if config.Language != LanguageGo || config.Runtime != RuntimeNone {
		t.Fatalf("unexpected language/runtime defaults: Language=%s, Runtime=%s", config.Language, config.Runtime)
	}
	if config.Architecture != ArchitectureHexagonal || config.Database != DatabasePostgres || !config.Docker {
		t.Fatalf("unexpected compatibility defaults: %#v", config)
	}
}

func TestValidateAcceptsSupportedCombinations(t *testing.T) {
	t.Parallel()

	tests := []ProjectConfig{
		Default("hex-api", "github.com/example/hex-api"),
		{
			Name:           "clean-api",
			ModulePath:     "github.com/example/clean-api",
			Language:       LanguageGo,
			Runtime:        RuntimeNone,
			Architecture:   ArchitectureClean,
			Database:       DatabaseMySQL,
			Docker:         true,
			Authentication: AuthenticationJWT,
			Documentation:  DocumentationSwagger,
			Tests:          TestStrategyUnitIntegration,
			CI:             CIStrategyGitHub,
		},
		{
			Name:           "layered-api",
			ModulePath:     "github.com/example/layered-api",
			Language:       LanguageGo,
			Runtime:        RuntimeNone,
			Architecture:   ArchitectureLayered,
			Database:       DatabaseNone,
			Docker:         false,
			Authentication: AuthenticationNone,
			Documentation:  DocumentationNone,
			Tests:          TestStrategyUnit,
			CI:             CIStrategyNone,
		},
		{
			Name:           "ts-node-api",
			ModulePath:     "github.com/example/ts-node-api",
			Language:       LanguageTypeScript,
			Runtime:        RuntimeNode,
			Architecture:   ArchitectureClean,
			Database:       DatabasePostgres,
			Docker:         true,
			Authentication: AuthenticationJWT,
			Documentation:  DocumentationNone,
			Tests:          TestStrategyUnit,
			CI:             CIStrategyGitHub,
		},
		{
			Name:           "js-node-api",
			ModulePath:     "github.com/example/js-node-api",
			Language:       LanguageJavaScript,
			Runtime:        RuntimeNode,
			Architecture:   ArchitectureLayered,
			Database:       DatabaseSQLite,
			Docker:         false,
			Authentication: AuthenticationNone,
			Documentation:  DocumentationNone,
			Tests:          TestStrategyUnit,
			CI:             CIStrategyNone,
		},
	}

	for _, config := range tests {
		if err := config.Validate(); err != nil {
			t.Errorf("config %#v should be valid: %v", config, err)
		}
	}
}

func TestValidateRejectsIncompleteOrUnsupportedConfiguration(t *testing.T) {
	t.Parallel()

	base := Default("my-api", "github.com/example/my-api")
	tests := []struct {
		name    string
		mutate  func(*ProjectConfig)
		message string
	}{
		{"missing name", func(c *ProjectConfig) { c.Name = "" }, "project name is required"},
		{"missing module", func(c *ProjectConfig) { c.ModulePath = "" }, "module path is required"},
		{"language", func(c *ProjectConfig) { c.Language = "rust" }, "invalid language"},
		{"runtime", func(c *ProjectConfig) { c.Runtime = "deno" }, "invalid runtime"},
		{"go with node runtime", func(c *ProjectConfig) { c.Language = LanguageGo; c.Runtime = RuntimeNode }, "Go projects do not support a runtime"},
		{"typescript without node", func(c *ProjectConfig) { c.Language = LanguageTypeScript; c.Runtime = RuntimeNone }, "typescript projects require runtime=node"},
		{"javascript without node", func(c *ProjectConfig) { c.Language = LanguageJavaScript; c.Runtime = RuntimeNone }, "javascript projects require runtime=node"},
		{"architecture", func(c *ProjectConfig) { c.Architecture = "ports-and-adapters" }, "invalid architecture"},
		{"database", func(c *ProjectConfig) { c.Database = "mongo" }, "invalid database"},
		{"authentication", func(c *ProjectConfig) { c.Authentication = "session" }, "invalid authentication"},
		{"documentation", func(c *ProjectConfig) { c.Documentation = "openapi" }, "invalid documentation"},
		{"tests", func(c *ProjectConfig) { c.Tests = "e2e" }, "invalid test strategy"},
		{"CI", func(c *ProjectConfig) { c.CI = "gitlab" }, "invalid CI strategy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := base
			tt.mutate(&config)
			err := config.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("Validate() error = %v, want message containing %q", err, tt.message)
			}
		})
	}
}
