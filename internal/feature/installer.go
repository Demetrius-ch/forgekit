package feature

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Demetrius-ch/forgekit/internal/output"
	"github.com/Demetrius-ch/forgekit/internal/prompt"
)

// FileChange represents a file modification for rollback tracking.
type FileChange struct {
	Path       string
	Existed    bool
	OldContent []byte
	OldHash    string
}

// ConflictInfo represents a detected conflict
type ConflictInfo struct {
	File        string
	Description string
	UserContent []byte
	PlanContent []byte
}

// Installer applies feature plans with rollback capability.
type Installer struct {
	projectRoot string
	changes     []FileChange
	conflicts   []ConflictInfo
	dryRun      bool
	console     *output.Console
	prompt      *prompt.Prompter
	rollbackMgr *RollbackManager
}

// NewInstaller creates an installer for a project.
func NewInstaller(projectRoot string, console *output.Console, dryRun bool) *Installer {
	return &Installer{
		projectRoot: projectRoot,
		changes:     make([]FileChange, 0),
		conflicts:   make([]ConflictInfo, 0),
		dryRun:      dryRun,
		console:     console,
		prompt:      prompt.New(os.Stdin, os.Stdout),
		rollbackMgr: NewRollbackManager(projectRoot),
	}
}

// computeHash computes SHA256 hash of content
func computeHash(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h)
}

// Apply executes a feature plan with rollback on failure.
func (i *Installer) Apply(ctx context.Context, plan Plan) error {
	// Create snapshot before applying
	if !i.dryRun {
		_, err := i.rollbackMgr.CreateSnapshot(fmt.Sprintf("pre-install-%s", plan.Feature))
		if err != nil {
			return fmt.Errorf("créer snapshot pour rollback: %w", err)
		}
	}

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

		// Read existing content for rollback and conflict detection
		var oldContent []byte
		existed := true
		if data, err := os.ReadFile(dest); err == nil {
			oldContent = data
		} else if os.IsNotExist(err) {
			existed = false
		} else {
			return fmt.Errorf("lire %s : %w", dest, err)
		}

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("créer le répertoire %s : %w", filepath.Dir(dest), err)
		}

		// Read source content (plan content)
		var planContent []byte
		if strings.HasSuffix(file.Source, ".tmpl") {
			// Template files are embedded - we need to render them
			// For now, treat as regular files
			planContent, _ = os.ReadFile(file.Source)
		} else {
			planContent, _ = os.ReadFile(file.Source)
		}

		// Check for conflicts if file exists and will be modified
		if existed && (file.Action == FileActionModify || file.Action == FileActionCreate) {
			oldHash := computeHash(oldContent)
			planHash := computeHash(planContent)

			// If hashes differ, user has modified the file
			if oldHash != planHash {
				conflict := ConflictInfo{
					File:        file.Destination,
					Description: "fichier modifié par l'utilisateur",
					UserContent: oldContent,
					PlanContent: planContent,
				}
				i.conflicts = append(i.conflicts, conflict)

				// If not in dry-run, ask user what to do
				if !i.dryRun {
					action, err := i.handleConflict(conflict)
					if err != nil {
						return err
					}
					if action == "skip" {
						// Record as skipped (no change)
						continue
					}
					// If action == "overwrite", continue with installation
				}
			}
		}

		// Record change for potential rollback
		i.changes = append(i.changes, FileChange{
			Path:       dest,
			Existed:    existed,
			OldContent: oldContent,
			OldHash:    computeHash(oldContent),
		})

		if err := os.WriteFile(dest, planContent, 0o644); err != nil {
			return fmt.Errorf("écrire %s : %w", dest, err)
		}
	}
	return nil
}

// handleConflict prompts user for conflict resolution
func (i *Installer) handleConflict(conflict ConflictInfo) (string, error) {
	if i.console.Format == output.FormatJSON || i.console.Quiet {
		// In JSON/quiet mode, skip by default
		return "skip", nil
	}

	fmt.Fprintf(i.console.Out, "\n⚠ Conflit détecté : %s\n", conflict.File)
	fmt.Fprintf(i.console.Out, "  Description: %s\n", conflict.Description)
	fmt.Fprintln(i.console.Out, "  Que voulez-vous faire ?")
	fmt.Fprintln(i.console.Out, "  1. Écraser (installer la version ForgeKit)")
	fmt.Fprintln(i.console.Out, "  2. Conserver (ignorer ce fichier)")

	choice, err := i.prompt.AskInt("Choix", 1)
	if err != nil {
		return "", err
	}

	switch choice {
	case 1:
		return "overwrite", nil
	case 2:
		return "skip", nil
	default:
		return "skip", nil
	}
}

// Conflicts returns the list of detected conflicts
func (i *Installer) Conflicts() []ConflictInfo {
	return i.conflicts
}

func (i *Installer) applyDependencies(ctx context.Context, deps []Dependency) error {
	if i.dryRun || len(deps) == 0 {
		return nil
	}

	for _, dep := range deps {
		cmd := exec.Command("go", "get", dep.Module+"@"+dep.Version)
		cmd.Dir = i.projectRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go get %s@%s : %w", dep.Module, dep.Version, err)
		}
	}

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
	// Use the rollback manager for comprehensive rollback
	err := i.rollbackMgr.RollbackLast()
	if err != nil {
		return fmt.Errorf("rollback manager: %w", err)
	}

	// Also restore individual file changes (for files not covered by snapshot)
	var lastErr error
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
	i.conflicts = nil
	return lastErr
}

// Changes returns the list of recorded changes (for inspection).
func (i *Installer) Changes() []FileChange {
	return i.changes
}
