package feature

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallerApply(t *testing.T) {
	root := t.TempDir()

	// Create a test file
	testFile := filepath.Join(root, "test.txt")
	if err := os.WriteFile(testFile, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	installer := NewInstaller(root, nil, false)

	plan := Plan{
		Files: []FileAction{
			{
				Source:      testFile,
				Destination: "new.txt",
			},
		},
	}

	err := installer.Apply(context.Background(), plan)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify file was copied
	newFile := filepath.Join(root, "new.txt")
	content, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("Failed to read new file: %v", err)
	}
	if string(content) != "original" {
		t.Fatalf("Expected 'original', got %s", string(content))
	}
}

func TestInstallerRollback(t *testing.T) {
	root := t.TempDir()

	// Create original file
	originalFile := filepath.Join(root, "test.txt")
	if err := os.WriteFile(originalFile, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	installer := NewInstaller(root, nil, false)

	plan := Plan{
		Files: []FileAction{
			{
				Source:      originalFile,
				Destination: "test.txt", // Same file - will be overwritten
			},
		},
	}

	err := installer.Apply(context.Background(), plan)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Modify the file
	if err := os.WriteFile(originalFile, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Rollback
	err = installer.Rollback()
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify original content restored
	content, err := os.ReadFile(originalFile)
	if err != nil {
		t.Fatalf("Failed to read file after rollback: %v", err)
	}
	if string(content) != "original" {
		t.Fatalf("Expected 'original' after rollback, got %s", string(content))
	}
}

func TestInstallerRollbackNewFile(t *testing.T) {
	root := t.TempDir()

	installer := NewInstaller(root, nil, false)

	// Create a source file
	sourceFile := filepath.Join(root, "source.txt")
	if err := os.WriteFile(sourceFile, []byte("source content"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := Plan{
		Files: []FileAction{
			{
				Source:      sourceFile,
				Destination: "newfile.txt",
			},
		},
	}

	err := installer.Apply(context.Background(), plan)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify new file exists
	newFile := filepath.Join(root, "newfile.txt")
	if _, err := os.Stat(newFile); err != nil {
		t.Fatalf("New file not created: %v", err)
	}

	// Rollback
	err = installer.Rollback()
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify new file is removed
	if _, err := os.Stat(newFile); !os.IsNotExist(err) {
		t.Fatal("New file should be removed after rollback")
	}
}

func TestInstalledFeatures(t *testing.T) {
	root := t.TempDir()

	// Test empty features
	f, err := LoadFeatures(root)
	if err != nil {
		t.Fatalf("LoadFeatures failed: %v", err)
	}
	if len(f.Features) != 0 {
		t.Fatal("Expected empty features list")
	}

	// Add a feature
	err = AddInstalledFeature(root, "auth", "1.0.0")
	if err != nil {
		t.Fatalf("AddInstalledFeature failed: %v", err)
	}

	// Check if installed
	installed, feat, err := IsInstalled(root, "auth")
	if err != nil {
		t.Fatalf("IsInstalled failed: %v", err)
	}
	if !installed {
		t.Fatal("Expected feature to be installed")
	}
	if feat.Version != "1.0.0" {
		t.Fatalf("Expected version 1.0.0, got %s", feat.Version)
	}

	// Add same feature again (should update)
	err = AddInstalledFeature(root, "auth", "1.0.1")
	if err != nil {
		t.Fatalf("AddInstalledFeature failed: %v", err)
	}

	installed, feat, err = IsInstalled(root, "auth")
	if err != nil {
		t.Fatalf("IsInstalled failed: %v", err)
	}
	if !installed {
		t.Fatal("Expected feature to be installed")
	}
	if feat.Version != "1.0.1" {
		t.Fatalf("Expected version 1.0.1, got %s", feat.Version)
	}

	// Add another feature
	err = AddInstalledFeature(root, "redis", "1.0.0")
	if err != nil {
		t.Fatalf("AddInstalledFeature failed: %v", err)
	}

	f, err = LoadFeatures(root)
	if err != nil {
		t.Fatalf("LoadFeatures failed: %v", err)
	}
	if len(f.Features) != 2 {
		t.Fatalf("Expected 2 features, got %d", len(f.Features))
	}

	// Remove feature
	err = RemoveInstalledFeature(root, "auth")
	if err != nil {
		t.Fatalf("RemoveInstalledFeature failed: %v", err)
	}

	f, err = LoadFeatures(root)
	if err != nil {
		t.Fatalf("LoadFeatures failed: %v", err)
	}
	if len(f.Features) != 1 {
		t.Fatalf("Expected 1 feature after removal, got %d", len(f.Features))
	}
	if f.Features[0].Name != "redis" {
		t.Fatalf("Expected redis, got %s", f.Features[0].Name)
	}
}

func TestInstalledFeaturesInvalidYAML(t *testing.T) {
	root := t.TempDir()

	// Create invalid YAML
	featuresDir := filepath.Join(root, ".forge")
	if err := os.MkdirAll(featuresDir, 0o755); err != nil {
		t.Fatal(err)
	}
	invalidYAML := "invalid: yaml: ["
	if err := os.WriteFile(filepath.Join(featuresDir, "features.yaml"), []byte(invalidYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFeatures(root)
	if err == nil {
		t.Fatal("Expected error for invalid YAML")
	}
}
