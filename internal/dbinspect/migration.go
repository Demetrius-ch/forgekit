package dbinspect

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// MigrationFile represents a migration file.
type MigrationFile struct {
	Version int
	Name    string
	Path    string
	UpSQL   string
	DownSQL string
}

// MigrationState represents the state of migrations.
type MigrationState struct {
	CurrentVersion *int
	IsDirty        bool
	Available      []MigrationFile
	Applied        []int
	Pending        []MigrationFile
	HasDestructive bool
}

// MigrationChecker checks migration state against database.
type MigrationChecker struct {
	inspector     *Inspector
	migrationsDir string
}

// NewMigrationChecker creates a new migration checker.
func NewMigrationChecker(inspector *Inspector, migrationsDir string) *MigrationChecker {
	return &MigrationChecker{
		inspector:     inspector,
		migrationsDir: migrationsDir,
	}
}

// LoadMigrations loads all migration files from the migrations directory.
func (m *MigrationChecker) LoadMigrations() ([]MigrationFile, error) {
	files, err := os.ReadDir(m.migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	var migrations []MigrationFile
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}

		// Parse version from filename (e.g., "000001_init.up.sql")
		versionStr := strings.Split(name, "_")[0]
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			continue // Skip files that don't match pattern
		}

		upPath := filepath.Join(m.migrationsDir, name)
		downName := strings.Replace(name, ".up.sql", ".down.sql", 1)
		downPath := filepath.Join(m.migrationsDir, downName)

		upSQL, _ := os.ReadFile(upPath)
		downSQL, _ := os.ReadFile(downPath)

		migrations = append(migrations, MigrationFile{
			Version: version,
			Name:    strings.TrimSuffix(name, ".up.sql"),
			Path:    upPath,
			UpSQL:   string(upSQL),
			DownSQL: string(downSQL),
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// CheckDestructive checks if a migration contains potentially destructive operations.
func (m *MigrationChecker) CheckDestructive(sql string) bool {
	sql = strings.ToUpper(sql)
	destructivePatterns := []string{
		"DROP TABLE",
		"DROP COLUMN",
		"TRUNCATE",
		"DELETE FROM",
		"DROP INDEX",
		"DROP SEQUENCE",
		"DROP VIEW",
		"DROP FUNCTION",
		"DROP TYPE",
		"DROP SCHEMA",
	}

	// Check for destructive ALTER TABLE operations (but not ADD COLUMN)
	if strings.Contains(sql, "ALTER TABLE") {
		// Check if it's a non-destructive ALTER TABLE (ADD COLUMN, ADD CONSTRAINT, etc.)
		nonDestructiveAlter := []string{
			"ADD COLUMN",
			"ADD CONSTRAINT",
			"ADD INDEX",
			"ALTER COLUMN",
			"ALTER COLUMN",
			"SET DEFAULT",
			"DROP DEFAULT",
			"SET NOT NULL",
			"DROP NOT NULL",
		}
		isNonDestructive := false
		for _, nd := range nonDestructiveAlter {
			if strings.Contains(sql, nd) {
				isNonDestructive = true
				break
			}
		}
		if !isNonDestructive {
			return true
		}
	}

	for _, pattern := range destructivePatterns {
		if strings.Contains(sql, pattern) {
			return true
		}
	}
	return false
}

// CheckState checks the migration state against the database.
func (m *MigrationChecker) CheckState(ctx context.Context, dbName string) (*MigrationState, error) {
	// Load available migrations
	available, err := m.LoadMigrations()
	if err != nil {
		return nil, fmt.Errorf("load migrations: %w", err)
	}

	// Inspect database
	info, err := m.inspector.InspectDatabase(ctx, dbName)
	if err != nil {
		return nil, fmt.Errorf("inspect database: %w", err)
	}

	state := &MigrationState{
		Available:      available,
		IsDirty:        info.IsDirty,
		CurrentVersion: info.MigrationVersion,
	}

	// Determine applied and pending migrations
	if info.HasMigrations && info.MigrationVersion != nil {
		for _, mig := range available {
			if mig.Version <= *info.MigrationVersion {
				state.Applied = append(state.Applied, mig.Version)
			} else {
				state.Pending = append(state.Pending, mig)
			}
		}
	} else {
		// No migrations applied yet
		state.Pending = available
	}

	// Check for destructive operations in pending migrations
	for _, mig := range state.Pending {
		if m.CheckDestructive(mig.UpSQL) {
			state.HasDestructive = true
			break
		}
	}

	return state, nil
}

// CanApplyMigrations returns whether migrations can be safely applied.
func (m *MigrationChecker) CanApplyMigrations(state *MigrationState, dbHasTables bool) (bool, string) {
	if state.IsDirty {
		return false, "Migration state is dirty - manual intervention required"
	}

	if state.HasDestructive && dbHasTables {
		return false, "Pending migrations contain destructive operations on a non-empty database"
	}

	return true, ""
}
