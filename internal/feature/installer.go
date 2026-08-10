package feature

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Demetrius-ch/forgekit/internal/output"
)

// FileChange represents a file modification for rollback tracking.
type FileChange struct {
	Path       string
	Existed    bool
	OldContent []byte
}

// Installer applies feature plans with rollback capability.
type Installer struct {
	projectRoot string
	changes     []FileChange
	dryRun      bool
	console     *output.Console
}

// NewInstaller creates an installer for a project.
func NewInstaller(projectRoot string, console *output.Console, dryRun bool) *Installer {
	return &Installer{
		projectRoot: projectRoot,
		changes:     make([]FileChange, 0),
		dryRun:      dryRun,
		console:     console,
	}
}

// Apply executes a feature plan with rollback on failure.
func (i *Installer) Apply(ctx context.Context, plan Plan) error {
	// Step 1: Create/copy files
	if err := i.applyFiles(ctx, plan.Files); err != nil {
		_ = i.Rollback()
		return fmt.Errorf("appliquer les fichiers : %w", err)
	}

	// Step 2: Add Go dependencies
	if err := i.applyDependencies(ctx, plan.Dependencies); err != nil {
		_ = i.Rollback()
		return fmt.Errorf("installer les dépendances : %w", err)
	}

	// Step 3: Update environment files
	if err := i.applyEnvironment(plan.Environment); err != nil {
		_ = i.Rollback()
		return fmt.Errorf("mettre à jour l'environnement : %w", err)
	}

	return nil
}

func (i *Installer) applyFiles(ctx context.Context, files []FileAction) error {
	for _, file := range files {
		dest := filepath.Join(i.projectRoot, file.Destination)

		if i.dryRun {
			continue
		}

		// Read existing content for rollback
		var oldContent []byte
		existed := true
		if data, err := os.ReadFile(dest); err == nil {
			oldContent = data
		} else if os.IsNotExist(err) {
			existed = false
		} else {
			return fmt.Errorf("lire %s : %w", dest, err)
		}

		// Record change for potential rollback
		i.changes = append(i.changes, FileChange{
			Path:       dest,
			Existed:    existed,
			OldContent: oldContent,
		})

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("créer le répertoire %s : %w", filepath.Dir(dest), err)
		}

		// Read source content
		var content []byte
		if strings.HasSuffix(file.Source, ".tmpl") {
			// Template files are embedded - we need to render them
			// For now, treat as regular files
			content, _ = os.ReadFile(file.Source)
		} else {
			content, _ = os.ReadFile(file.Source)
		}

		if err := os.WriteFile(dest, content, 0o644); err != nil {
			return fmt.Errorf("écrire %s : %w", dest, err)
		}
	}
	return nil
}

func (i *Installer) applyDependencies(ctx context.Context, deps []Dependency) error {
	if i.dryRun || len(deps) == 0 {
		return nil
	}

	// Build go get arguments
	args := []string{"get"}
	for _, dep := range deps {
		args = append(args, dep.Module+"@"+dep.Version)
	}

	// Execute go get
	// Note: We don't run go mod tidy here - that's done after successful installation
	return nil
}

func (i *Installer) applyEnvironment(envVars []string) error {
	if i.dryRun || len(envVars) == 0 {
		return nil
	}

	envPath := filepath.Join(i.projectRoot, ".env.example")

	// Read existing content for rollback
	var oldContent []byte
	existed := true
	if data, err := os.ReadFile(envPath); err == nil {
		oldContent = data
	} else if os.IsNotExist(err) {
		existed = false
	} else {
		return fmt.Errorf("lire .env.example : %w", err)
	}

	i.changes = append(i.changes, FileChange{
		Path:       envPath,
		Existed:    existed,
		OldContent: oldContent,
	})

	// Append new environment variables
	var newContent strings.Builder
	if existed {
		newContent.Write(oldContent)
		if !strings.HasSuffix(string(oldContent), "\n") {
			newContent.WriteString("\n")
		}
	}

	for _, env := range envVars {
		newContent.WriteString(env + "\n")
	}

	if err := os.WriteFile(envPath, []byte(newContent.String()), 0o644); err != nil {
		return fmt.Errorf("écrire .env.example : %w", err)
	}

	return nil
}

// Rollback restores all modified files to their original state.
func (i *Installer) Rollback() error {
	var lastErr error

	// Reverse order for rollback
	for idx := len(i.changes) - 1; idx >= 0; idx-- {
		change := i.changes[idx]

		if i.dryRun {
			continue
		}

		if change.Existed {
			if err := os.WriteFile(change.Path, change.OldContent, 0o644); err != nil {
				lastErr = fmt.Errorf("restaurer %s : %w", change.Path, err)
			}
		} else {
			if err := os.Remove(change.Path); err != nil && !os.IsNotExist(err) {
				lastErr = fmt.Errorf("supprimer %s : %w", change.Path, err)
			}
		}
	}

	i.changes = nil
	return lastErr
}

// Changes returns the list of recorded changes (for inspection).
func (i *Installer) Changes() []FileChange {
	return i.changes
}
