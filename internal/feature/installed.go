package feature

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// InstalledFeature represents a feature installed in the project.
type InstalledFeature struct {
	Name        string    `yaml:"name"`
	Version     string    `yaml:"version"`
	InstalledAt time.Time `yaml:"installed_at"`
}

// FeaturesFile represents the .forge/features.yaml file.
type FeaturesFile struct {
	Features []InstalledFeature `yaml:"features"`
}

// FeaturesPath returns the path to the features tracking file.
func FeaturesPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".forge", "features.yaml")
}

// LoadFeatures loads the installed features from .forge/features.yaml.
func LoadFeatures(projectRoot string) (FeaturesFile, error) {
	path := FeaturesPath(projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return FeaturesFile{}, nil
		}
		return FeaturesFile{}, fmt.Errorf("lire features.yaml : %w", err)
	}

	var f FeaturesFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return FeaturesFile{}, fmt.Errorf("parser features.yaml : %w", err)
	}
	return f, nil
}

// SaveFeatures saves the installed features to .forge/features.yaml.
func SaveFeatures(projectRoot string, f FeaturesFile) error {
	path := FeaturesPath(projectRoot)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("créer le répertoire .forge : %w", err)
	}

	data, err := yaml.Marshal(f)
	if err != nil {
		return fmt.Errorf("sérialiser features.yaml : %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

// IsInstalled checks if a feature is already installed.
func IsInstalled(projectRoot, name string) (bool, *InstalledFeature, error) {
	f, err := LoadFeatures(projectRoot)
	if err != nil {
		return false, nil, err
	}

	for _, feat := range f.Features {
		if feat.Name == name {
			return true, &feat, nil
		}
	}
	return false, nil, nil
}

// AddInstalledFeature adds a feature to the installed list.
func AddInstalledFeature(projectRoot, name, version string) error {
	f, err := LoadFeatures(projectRoot)
	if err != nil {
		return err
	}

	// Check if already exists
	for i, feat := range f.Features {
		if feat.Name == name {
			// Update version and timestamp
			f.Features[i].Version = version
			f.Features[i].InstalledAt = time.Now()
			return SaveFeatures(projectRoot, f)
		}
	}

	// Add new feature
	f.Features = append(f.Features, InstalledFeature{
		Name:        name,
		Version:     version,
		InstalledAt: time.Now(),
	})

	return SaveFeatures(projectRoot, f)
}

// RemoveInstalledFeature removes a feature from the installed list.
func RemoveInstalledFeature(projectRoot, name string) error {
	f, err := LoadFeatures(projectRoot)
	if err != nil {
		return err
	}

	for i, feat := range f.Features {
		if feat.Name == name {
			f.Features = append(f.Features[:i], f.Features[i+1:]...)
			return SaveFeatures(projectRoot, f)
		}
	}
	return nil
}
