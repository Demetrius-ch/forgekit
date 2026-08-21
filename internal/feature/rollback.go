package feature

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RollbackManager manages project state snapshots for rollback operations.
type RollbackManager struct {
	projectRoot string
	snapshots   []*Snapshot
}

// Snapshot represents a complete project state at a point in time.
type Snapshot struct {
	ID          string
	Timestamp   time.Time
	GoMod       []byte
	GoSum       []byte
	ForgeFiles  map[string][]byte
	EnvExample  []byte
	Description string
}

// NewRollbackManager creates a new rollback manager for a project.
func NewRollbackManager(projectRoot string) *RollbackManager {
	return &RollbackManager{
		projectRoot: projectRoot,
		snapshots:   make([]*Snapshot, 0),
	}
}

// CreateSnapshot captures the current project state.
func (rm *RollbackManager) CreateSnapshot(description string) (*Snapshot, error) {
	snap := &Snapshot{
		ID:          fmt.Sprintf("snap-%d", time.Now().UnixNano()),
		Timestamp:   time.Now(),
		ForgeFiles:  make(map[string][]byte),
		Description: description,
	}

	// Capture go.mod
	if data, err := os.ReadFile(filepath.Join(rm.projectRoot, "go.mod")); err == nil {
		snap.GoMod = data
	}

	// Capture go.sum
	if data, err := os.ReadFile(filepath.Join(rm.projectRoot, "go.sum")); err == nil {
		snap.GoSum = data
	}

	// Capture .env.example
	if data, err := os.ReadFile(filepath.Join(rm.projectRoot, ".env.example")); err == nil {
		snap.EnvExample = data
	}

	// Capture .forge directory
	forgeDir := filepath.Join(rm.projectRoot, ".forge")
	if info, err := os.Stat(forgeDir); err == nil && info.IsDir() {
		filepath.WalkDir(forgeDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if data, err := os.ReadFile(path); err == nil {
				relPath, _ := filepath.Rel(rm.projectRoot, path)
				snap.ForgeFiles[relPath] = data
			}
			return nil
		})
	}

	rm.snapshots = append(rm.snapshots, snap)
	return snap, nil
}

// RollbackTo restores the project state to a specific snapshot.
func (rm *RollbackManager) RollbackTo(snap *Snapshot) error {
	var lastErr error

	// Restore go.mod
	if len(snap.GoMod) > 0 {
		if err := os.WriteFile(filepath.Join(rm.projectRoot, "go.mod"), snap.GoMod, 0o644); err != nil {
			lastErr = fmt.Errorf("restaurer go.mod: %w", err)
		}
	} else {
		_ = os.Remove(filepath.Join(rm.projectRoot, "go.mod"))
	}

	// Restore go.sum
	if len(snap.GoSum) > 0 {
		if err := os.WriteFile(filepath.Join(rm.projectRoot, "go.sum"), snap.GoSum, 0o644); err != nil {
			lastErr = fmt.Errorf("restaurer go.sum: %w", err)
		}
	} else {
		_ = os.Remove(filepath.Join(rm.projectRoot, "go.sum"))
	}

	// Restore .env.example
	if len(snap.EnvExample) > 0 {
		if err := os.WriteFile(filepath.Join(rm.projectRoot, ".env.example"), snap.EnvExample, 0o644); err != nil {
			lastErr = fmt.Errorf("restaurer .env.example: %w", err)
		}
	} else {
		_ = os.Remove(filepath.Join(rm.projectRoot, ".env.example"))
	}

	// Restore .forge directory
	forgeDir := filepath.Join(rm.projectRoot, ".forge")
	_ = os.RemoveAll(forgeDir)
	for relPath, data := range snap.ForgeFiles {
		fullPath := filepath.Join(rm.projectRoot, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			lastErr = fmt.Errorf("créer répertoire %s: %w", filepath.Dir(fullPath), err)
			continue
		}
		if err := os.WriteFile(fullPath, data, 0o644); err != nil {
			lastErr = fmt.Errorf("restaurer %s: %w", relPath, err)
		}
	}

	return lastErr
}

// RollbackLast restores to the most recent snapshot.
func (rm *RollbackManager) RollbackLast() error {
	if len(rm.snapshots) == 0 {
		return fmt.Errorf("aucun snapshot disponible pour rollback")
	}
	lastSnap := rm.snapshots[len(rm.snapshots)-1]
	return rm.RollbackTo(lastSnap)
}

// VerifyRollback verifies that the project state matches the snapshot.
func (rm *RollbackManager) VerifyRollback(snap *Snapshot) error {
	// Verify go.mod
	if len(snap.GoMod) > 0 {
		current, err := os.ReadFile(filepath.Join(rm.projectRoot, "go.mod"))
		if err != nil {
			return fmt.Errorf("go.mod manquant après rollback: %w", err)
		}
		if string(current) != string(snap.GoMod) {
			return fmt.Errorf("go.mod ne correspond pas au snapshot")
		}
	}

	// Verify go.sum
	if len(snap.GoSum) > 0 {
		current, err := os.ReadFile(filepath.Join(rm.projectRoot, "go.sum"))
		if err != nil {
			return fmt.Errorf("go.sum manquant après rollback: %w", err)
		}
		if string(current) != string(snap.GoSum) {
			return fmt.Errorf("go.sum ne correspond pas au snapshot")
		}
	}

	// Verify .env.example
	if len(snap.EnvExample) > 0 {
		current, err := os.ReadFile(filepath.Join(rm.projectRoot, ".env.example"))
		if err != nil {
			return fmt.Errorf(".env.example manquant après rollback: %w", err)
		}
		if string(current) != string(snap.EnvExample) {
			return fmt.Errorf(".env.example ne correspond pas au snapshot")
		}
	}

	// Verify .forge files
	for relPath, expectedData := range snap.ForgeFiles {
		fullPath := filepath.Join(rm.projectRoot, relPath)
		current, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("%s manquant après rollback: %w", relPath, err)
		}
		if string(current) != string(expectedData) {
			return fmt.Errorf("%s ne correspond pas au snapshot", relPath)
		}
	}

	return nil
}

// ListSnapshots returns all available snapshots.
func (rm *RollbackManager) ListSnapshots() []*Snapshot {
	return rm.snapshots
}

// ClearSnapshots removes all snapshots.
func (rm *RollbackManager) ClearSnapshots() {
	rm.snapshots = nil
}

// RollbackLimitation describes what rollback can and cannot do.
type RollbackLimitation struct {
	CanRollback    []string
	CannotRollback []string
}

// GetRollbackLimitations returns the rollback capabilities and limitations.
func GetRollbackLimitations() RollbackLimitation {
	return RollbackLimitation{
		CanRollback: []string{
			"go.mod / go.sum file content",
			"go.sum file content",
			".env.example file content",
			".forge/forge.yaml metadata",
			".forge/features.yaml installed features",
			"Feature-generated files (internal/*)",
		},
		CannotRollback: []string{
			"go get / go mod tidy side effects (downloaded modules in GOMODCACHE)",
			"Docker container state (databases, volumes, networks)",
			"Git history (commits, tags, branches)",
			"System packages installed via apt/brew/yum",
			"External service state (PostgreSQL databases, Redis, etc.)",
			"gofmt formatting changes (non-semantic whitespace)",
			"Files not tracked in snapshot (manual user files)",
		},
	}
}
