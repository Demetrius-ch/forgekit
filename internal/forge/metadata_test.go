package forge

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Demetrius-ch/forgekit/internal/feature"
)

func createTestProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd", "server"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/test/test\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCreateInitialMetadata(t *testing.T) {
	meta := CreateInitialMetadata("/tmp/test", "my-project", "github.com/test/my-project", "1.25")

	if meta.Version == "" {
		t.Fatal("version should not be empty")
	}
	if meta.Schema != SchemaVersion {
		t.Fatalf("expected schema %d, got %d", SchemaVersion, meta.Schema)
	}
	if meta.Project != "my-project" {
		t.Fatalf("expected project 'my-project', got %s", meta.Project)
	}
	if meta.Language != "go" {
		t.Fatalf("expected language 'go', got %s", meta.Language)
	}
	if meta.Type != "backend-api" {
		t.Fatalf("expected type 'backend-api', got %s", meta.Type)
	}
	if meta.CreatedAt.IsZero() {
		t.Fatal("created_at should not be zero")
	}
}

func TestSaveAndLoadMetadata(t *testing.T) {
	root := createTestProject(t)

	meta := ForgeMetadata{
		Version:   "0.12.0",
		Schema:    SchemaVersion,
		Project:   "test-project",
		Language:  "go",
		Type:      "backend-api",
		CreatedAt: time.Now().UTC(),
	}

	if err := SaveMetadata(root, meta); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	loaded, err := LoadMetadata(root)
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}

	if loaded.Version != meta.Version {
		t.Fatalf("version mismatch: %s != %s", loaded.Version, meta.Version)
	}
	if loaded.Schema != meta.Schema {
		t.Fatalf("schema mismatch: %d != %d", loaded.Schema, meta.Schema)
	}
	if loaded.Project != meta.Project {
		t.Fatalf("project mismatch: %s != %s", loaded.Project, meta.Project)
	}
	if loaded.Language != meta.Language {
		t.Fatalf("language mismatch: %s != %s", loaded.Language, meta.Language)
	}
	if loaded.Type != meta.Type {
		t.Fatalf("type mismatch: %s != %s", loaded.Type, meta.Type)
	}
}

func TestLoadMetadataNotFound(t *testing.T) {
	root := createTestProject(t)

	_, err := LoadMetadata(root)
	if err != ErrMetadataNotFound {
		t.Fatalf("expected ErrMetadataNotFound, got %v", err)
	}
}

func TestValidateSignature_Valid(t *testing.T) {
	root := createTestProject(t)

	meta := ForgeMetadata{
		Version:   "0.12.0",
		Schema:    SchemaVersion,
		Project:   "test-project",
		Language:  "go",
		Type:      "backend-api",
		CreatedAt: time.Now().UTC(),
	}
	if err := SaveMetadata(root, meta); err != nil {
		t.Fatal(err)
	}

	// Create empty features.yaml
	feature.SaveFeatures(root, feature.FeaturesFile{})

	result := ValidateSignature(root)

	if !result.IsValid() {
		t.Fatalf("expected valid signature, got status %d: errors=%v warnings=%v", result.Status, result.Errors, result.Warnings)
	}
	if result.Metadata.Project != "test-project" {
		t.Fatalf("expected project 'test-project', got %s", result.Metadata.Project)
	}
	if result.LegacyProject {
		t.Fatal("should not be legacy project")
	}
}

func TestValidateSignature_Absent(t *testing.T) {
	root := createTestProject(t)
	// No .forge directory created

	result := ValidateSignature(root)

	if !result.IsAbsent() {
		t.Fatalf("expected absent signature, got status %d", result.Status)
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected errors for absent signature")
	}
}

func TestValidateSignature_InvalidMissingForgeYamlNoFeatures(t *testing.T) {
	root := createTestProject(t)

	// Create .forge directory but no forge.yaml and no features.yaml
	forgeDir := ForgeDirPath(root)
	if err := os.MkdirAll(forgeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result := ValidateSignature(root)

	if !result.IsInvalid() {
		t.Fatalf("expected invalid signature, got status %d", result.Status)
	}
	found := false
	for _, e := range result.Errors {
		if e == ".forge/forge.yaml missing but .forge directory exists" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing forge.yaml error, got: %v", result.Errors)
	}
}

func TestValidateSignature_LegacyProject(t *testing.T) {
	root := createTestProject(t)

	// Create .forge directory but no forge.yaml
	forgeDir := ForgeDirPath(root)
	if err := os.MkdirAll(forgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create features.yaml with features (legacy project)
	feature.SaveFeatures(root, feature.FeaturesFile{
		Features: []feature.InstalledFeature{{Name: "auth", Version: "1.0.0", InstalledAt: time.Now()}},
	})

	result := ValidateSignature(root)

	if !result.IsValid() {
		t.Fatalf("expected valid signature for legacy project, got status %d: errors=%v", result.Status, result.Errors)
	}
	if !result.LegacyProject {
		t.Fatal("expected legacy project")
	}
	if result.Metadata.Schema != 0 {
		t.Fatalf("expected schema 0 for legacy, got %d", result.Metadata.Schema)
	}
}

func TestValidateSignature_InvalidSchemaTooHigh(t *testing.T) {
	root := createTestProject(t)

	meta := ForgeMetadata{
		Version:   "0.12.0",
		Schema:    SchemaVersion + 1,
		Project:   "test-project",
		Language:  "go",
		Type:      "backend-api",
		CreatedAt: time.Now().UTC(),
	}
	if err := SaveMetadata(root, meta); err != nil {
		t.Fatal(err)
	}
	feature.SaveFeatures(root, feature.FeaturesFile{})

	result := ValidateSignature(root)

	if !result.IsInvalid() {
		t.Fatalf("expected invalid signature for future schema, got status %d", result.Status)
	}
	found := false
	for _, e := range result.Errors {
		if e == "unsupported schema version 2 (current: 1)" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected unsupported schema error, got: %v", result.Errors)
	}
}

func TestValidateSignature_InvalidMissingVersion(t *testing.T) {
	root := createTestProject(t)

	meta := ForgeMetadata{
		Schema:    SchemaVersion,
		Project:   "test-project",
		Language:  "go",
		Type:      "backend-api",
		CreatedAt: time.Now().UTC(),
	}
	if err := SaveMetadata(root, meta); err != nil {
		t.Fatal(err)
	}
	feature.SaveFeatures(root, feature.FeaturesFile{})

	result := ValidateSignature(root)

	if !result.IsInvalid() {
		t.Fatalf("expected invalid signature for missing version, got status %d", result.Status)
	}
	found := false
	for _, e := range result.Errors {
		if e == "ForgeKit version missing in metadata" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing version error, got: %v", result.Errors)
	}
}

func TestValidateSignature_InvalidMissingProject(t *testing.T) {
	root := createTestProject(t)

	meta := ForgeMetadata{
		Version:   "0.12.0",
		Schema:    SchemaVersion,
		Language:  "go",
		Type:      "backend-api",
		CreatedAt: time.Now().UTC(),
	}
	if err := SaveMetadata(root, meta); err != nil {
		t.Fatal(err)
	}
	feature.SaveFeatures(root, feature.FeaturesFile{})

	result := ValidateSignature(root)

	if !result.IsInvalid() {
		t.Fatalf("expected invalid signature for missing project, got status %d", result.Status)
	}
	found := false
	for _, e := range result.Errors {
		if e == "project name missing in metadata" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing project error, got: %v", result.Errors)
	}
}

func TestValidateSignature_WarningForMissingFeatureConfig(t *testing.T) {
	root := createTestProject(t)

	meta := ForgeMetadata{
		Version:   "0.12.0",
		Schema:    SchemaVersion,
		Project:   "test-project",
		Language:  "go",
		Type:      "backend-api",
		CreatedAt: time.Now().UTC(),
	}
	if err := SaveMetadata(root, meta); err != nil {
		t.Fatal(err)
	}
	// features.yaml declares a feature but the config is missing
	feature.SaveFeatures(root, feature.FeaturesFile{
		Features: []feature.InstalledFeature{{Name: "auth", Version: "1.0.0", InstalledAt: time.Now()}},
	})

	result := ValidateSignature(root)

	if !result.IsValid() {
		t.Fatalf("expected valid signature, got status %d: errors=%v", result.Status, result.Errors)
	}
	found := false
	for _, w := range result.Warnings {
		if w == "feature \"auth\" declared but configuration missing (expected at "+filepath.Join(root, "internal", "auth")+")" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected warning for missing feature config, got warnings: %v", result.Warnings)
	}
}
