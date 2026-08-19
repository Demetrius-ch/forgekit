package forge

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Demetrius-ch/forgekit/internal/feature"
)

type SignatureStatus int

const (
	SignatureValid SignatureStatus = iota
	SignatureAbsent
	SignatureInvalid
)

type ValidationResult struct {
	Status        SignatureStatus
	Metadata      ForgeMetadata
	Features      feature.FeaturesFile
	Errors        []string
	Warnings      []string
	LegacyProject bool
}

func ValidateSignature(projectRoot string) ValidationResult {
	result := ValidationResult{}

	forgeDir := ForgeDirPath(projectRoot)
	if _, err := os.Stat(forgeDir); err != nil {
		if os.IsNotExist(err) {
			result.Status = SignatureAbsent
			result.Errors = append(result.Errors, "ForgeKit signature not found (.forge directory missing)")
			return result
		}
		result.Status = SignatureInvalid
		result.Errors = append(result.Errors, fmt.Sprintf("cannot access .forge: %v", err))
		return result
	}

	meta, err := LoadMetadata(projectRoot)
	if err != nil {
		if err == ErrMetadataNotFound {
			// Check if features.yaml exists
			featuresPath := filepath.Join(projectRoot, ".forge", "features.yaml")
			if _, err := os.Stat(featuresPath); err != nil {
				if os.IsNotExist(err) {
					result.Status = SignatureInvalid
					result.Errors = append(result.Errors, ".forge/forge.yaml missing and .forge/features.yaml missing")
					return result
				}
				result.Status = SignatureInvalid
				result.Errors = append(result.Errors, fmt.Sprintf("cannot access .forge/features.yaml: %v", err))
				return result
			}
			// features.yaml exists, treat as legacy
			legacyFeatures, featErr := feature.LoadFeatures(projectRoot)
			if featErr != nil {
				result.Status = SignatureInvalid
				result.Errors = append(result.Errors, fmt.Sprintf("invalid features.yaml: %v", featErr))
				return result
			}
			result.Status = SignatureValid
			result.LegacyProject = true
			result.Metadata = ForgeMetadata{
				Schema:  0,
				Project: "(legacy)",
			}
			result.Features = legacyFeatures
			result.Warnings = append(result.Warnings, "Legacy project detected: .forge/forge.yaml missing, only features.yaml present")
			return result
		}
		result.Status = SignatureInvalid
		result.Errors = append(result.Errors, fmt.Sprintf("invalid forge.yaml: %v", err))
		return result
	}

	if meta.Schema == 0 {
		result.Status = SignatureInvalid
		result.Errors = append(result.Errors, "schema version missing or invalid (must be >= 1)")
		return result
	}

	if meta.Schema > SchemaVersion {
		result.Status = SignatureInvalid
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported schema version %d (current: %d)", meta.Schema, SchemaVersion))
		return result
	}

	if meta.Version == "" {
		result.Status = SignatureInvalid
		result.Errors = append(result.Errors, "ForgeKit version missing in metadata")
		return result
	}

	if meta.Project == "" {
		result.Status = SignatureInvalid
		result.Errors = append(result.Errors, "project name missing in metadata")
		return result
	}

	features, featErr := feature.LoadFeatures(projectRoot)
	if featErr != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("cannot load features.yaml: %v", featErr))
	} else {
		result.Features = features
	}

	if !result.LegacyProject {
		if len(features.Features) == 0 {
			result.Warnings = append(result.Warnings, "no features registered in .forge/features.yaml")
		}
	}

	for _, feat := range features.Features {
		featurePath := filepath.Join(projectRoot, "internal", feat.Name)
		if _, err := os.Stat(featurePath); os.IsNotExist(err) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("feature %q declared but configuration missing (expected at %s)", feat.Name, featurePath))
		}
	}

	result.Status = SignatureValid
	result.Metadata = meta
	return result
}

func (r ValidationResult) IsValid() bool {
	return r.Status == SignatureValid
}

func (r ValidationResult) IsAbsent() bool {
	return r.Status == SignatureAbsent
}

func (r ValidationResult) IsInvalid() bool {
	return r.Status == SignatureInvalid
}
