package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Demetrius-ch/forgekit/internal/generationplan"
	"github.com/Demetrius-ch/forgekit/internal/projectconfig"
)

func BenchmarkInitHexagonal(b *testing.B) {
	benchInit(b, projectconfig.ProjectConfig{
		Name:           "bench-api",
		ModulePath:     "github.com/bench/bench-api",
		Architecture:   projectconfig.ArchitectureHexagonal,
		Database:       projectconfig.DatabasePostgres,
		Docker:         true,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	})
}

func BenchmarkInitClean(b *testing.B) {
	benchInit(b, projectconfig.ProjectConfig{
		Name:           "bench-api",
		ModulePath:     "github.com/bench/bench-api",
		Architecture:   projectconfig.ArchitectureClean,
		Database:       projectconfig.DatabaseMySQL,
		Docker:         true,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	})
}

func BenchmarkInitLayered(b *testing.B) {
	benchInit(b, projectconfig.ProjectConfig{
		Name:           "bench-api",
		ModulePath:     "github.com/bench/bench-api",
		Architecture:   projectconfig.ArchitectureLayered,
		Database:       projectconfig.DatabaseSQLite,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	})
}

func BenchmarkInitMinimal(b *testing.B) {
	benchInit(b, projectconfig.ProjectConfig{
		Name:           "bench-api",
		ModulePath:     "github.com/bench/bench-api",
		Architecture:   projectconfig.ArchitectureLayered,
		Database:       projectconfig.DatabaseNone,
		Docker:         false,
		Authentication: projectconfig.AuthenticationNone,
		Documentation:  projectconfig.DocumentationNone,
		Tests:          projectconfig.TestStrategyUnit,
		CI:             projectconfig.CIStrategyNone,
	})
}

func BenchmarkInitFull(b *testing.B) {
	benchInit(b, projectconfig.ProjectConfig{
		Name:          "bench-api",
		ModulePath:    "github.com/bench/bench-api",
		Architecture:  projectconfig.ArchitectureHexagonal,
		Database:      projectconfig.DatabasePostgres,
		Docker:        true,
		Authentication: projectconfig.AuthenticationJWT,
		Documentation: projectconfig.DocumentationSwagger,
		Tests:         projectconfig.TestStrategyUnitIntegration,
		CI:            projectconfig.CIStrategyGitHub,
	})
}

func benchInit(b *testing.B, cfg projectconfig.ProjectConfig) {
	b.Helper()
	gen, err := New()
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		dir := b.TempDir()
		target := filepath.Join(dir, "bench-api")
		opts := InitOptions{
			ProjectName:   "bench-api",
			ModulePath:    "github.com/bench/bench-api",
			HTTPPort:      8080,
			DatabaseName:  "bench_api",
			TargetDir:     target,
			ProjectConfig: cfg,
		}
		if _, err := gen.Init(opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerationPlan(b *testing.B) {
	b.Helper()
	registry := generationplan.DefaultRegistry()
	cfg := projectconfig.ProjectConfig{
		Name:          "bench-api",
		ModulePath:    "github.com/bench/bench-api",
		Architecture:  projectconfig.ArchitectureHexagonal,
		Database:      projectconfig.DatabasePostgres,
		Docker:        true,
		Authentication: projectconfig.AuthenticationJWT,
		Documentation: projectconfig.DocumentationSwagger,
		Tests:         projectconfig.TestStrategyUnitIntegration,
		CI:            projectconfig.CIStrategyGitHub,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := registry.Build(cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPostProcessProject(b *testing.B) {
	b.Helper()
	gen, err := New()
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		dir := b.TempDir()
		target := filepath.Join(dir, "bench-api")
		opts := InitOptions{
			ProjectName:  "bench-api",
			ModulePath:   "github.com/bench/bench-api",
			HTTPPort:     8080,
			DatabaseName: "bench_api",
			TargetDir:    target,
		}
		if _, err := gen.Init(opts); err != nil {
			b.Fatal(err)
		}
		if err := gen.PostProcessProject(target); err != nil {
			b.Fatal(err)
		}
		// cleanup
		os.RemoveAll(target)
	}
}
