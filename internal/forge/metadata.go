package forge

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Demetrius-ch/forgekit/internal/app"

	"gopkg.in/yaml.v3"
)

const (
	SchemaVersion    = 1
	MetadataFileName = "forge.yaml"
	FeaturesFileName = "features.yaml"
	ForgeDirName     = ".forge"
)

type ForgeMetadata struct {
	Version   string    `yaml:"version"`
	Schema    int       `yaml:"schema"`
	Project   string    `yaml:"project,omitempty"`
	Language  string    `yaml:"language,omitempty"`
	Type      string    `yaml:"type,omitempty"`
	CreatedAt time.Time `yaml:"created_at,omitempty"`
}

func MetadataPath(projectRoot string) string {
	return filepath.Join(projectRoot, ForgeDirName, MetadataFileName)
}

func FeaturesPath(projectRoot string) string {
	return filepath.Join(projectRoot, ForgeDirName, FeaturesFileName)
}

func ForgeDirPath(projectRoot string) string {
	return filepath.Join(projectRoot, ForgeDirName)
}

func LoadMetadata(projectRoot string) (ForgeMetadata, error) {
	path := MetadataPath(projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ForgeMetadata{}, ErrMetadataNotFound
		}
		return ForgeMetadata{}, fmt.Errorf("read %s: %w", path, err)
	}

	var meta ForgeMetadata
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return ForgeMetadata{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return meta, nil
}

func SaveMetadata(projectRoot string, meta ForgeMetadata) error {
	path := MetadataPath(projectRoot)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create .forge dir: %w", err)
	}

	data, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

func CreateInitialMetadata(projectRoot, projectName, modulePath, goVersion string) ForgeMetadata {
	return ForgeMetadata{
		Version:   app.Version,
		Schema:    SchemaVersion,
		Project:   projectName,
		Language:  "go",
		Type:      "backend-api",
		CreatedAt: time.Now().UTC(),
	}
}

var ErrMetadataNotFound = fmt.Errorf("forge metadata not found")
