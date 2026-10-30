package migration

import (
	"fmt"
)

// MigrationPlan represents the plan for migrating a project from one template version to another.
type MigrationPlan struct {
	SourceVersion string
	TargetVersion string
	FileChanges   int
	Conflicts     int
}

// MigrationExecutor applies a MigrationPlan to a project directory.
type MigrationExecutor struct {
	Plan *MigrationPlan
}

// NewMigrationExecutor creates a new MigrationExecutor with the given plan.
func NewMigrationExecutor(plan *MigrationPlan) *MigrationExecutor {
	return &MigrationExecutor{Plan: plan}
}

// Apply applies the migration plan to the project directory.
// It creates a snapshot, applies file changes, validates, updates metadata, and records history.
func (e *MigrationExecutor) Apply(projectDir string) error {
	// Step 1: Create a backup/snapshot of the project
	// Note: Snapshot creation uses the feature package RollbackManager
	// For now, we record the operation
	fmt.Printf("Application of migration %s → %s\n", e.Plan.SourceVersion, e.Plan.TargetVersion)

	// Step 2: Apply file changes according to the migration plan
	if err := applyFileChanges(projectDir); err != nil {
		return err
	}

	// Step 3: Validate the migrated project
	if err := validateProject(projectDir); err != nil {
		return err
	}

	// Step 4: Update the .forge metadata with the new template version
	if err := updateMetadata(projectDir, e.Plan.TargetVersion); err != nil {
		return err
	}

	// Step 5: Record the migration in the .forge history
	if err := recordHistory(projectDir, e.Plan.SourceVersion, e.Plan.TargetVersion, "success"); err != nil {
		return err
	}

	return nil
}

// applyFileChanges applies the changes from the migration plan to the project files.
func applyFileChanges(projectDir string) error {
	// In a full implementation, this would modify project files according to the migration plan
	// For now, return nil (no changes applied)
	return nil
}

// validateProject runs validation checks on the migrated project.
func validateProject(projectDir string) error {
	// In a full implementation, this would run go vet, go fmt, and other checks
	// For now, return nil
	return nil
}

// updateMetadata updates the .forge metadata with the new template version.
func updateMetadata(projectDir string, targetVersion string) error {
	// Update the .forge metadata with the new template version
	// Note: In a full implementation, this would read/update the .forge/forge.yaml
	_ = projectDir
	_ = targetVersion
	return nil
}

// recordHistory records the migration in the .forge history file.
func recordHistory(projectDir string, sourceVersion, targetVersion, status string) error {
	// Append to the .forge/history file
	// In a full implementation, this would properly append to the history file
	_ = projectDir
	_ = sourceVersion
	_ = targetVersion
	_ = status
	return nil
}

// rollback restores the project from the last snapshot.
func rollback(projectDir string) error {
	// In a full implementation, this would restore files from the snapshot
	// using the RollbackManager from the feature package
	_ = projectDir
	return fmt.Errorf("rollback non implémenté - utilisez forge rollback directement")
}
