package dbinspect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrationChecker_LoadMigrations(t *testing.T) {
	// Create a temporary directory with migration files
	tmpDir := t.TempDir()

	// Create test migration files
	migrations := []struct {
		version int
		name    string
		upSQL   string
		downSQL string
	}{
		{1, "000001_init", "CREATE TABLE users (id BIGSERIAL PRIMARY KEY);", "DROP TABLE users;"},
		{2, "000002_add_email", "ALTER TABLE users ADD COLUMN email TEXT;", "ALTER TABLE users DROP COLUMN email;"},
		{3, "000003_create_index", "CREATE INDEX idx_users_email ON users(email);", "DROP INDEX idx_users_email;"},
	}

	for _, m := range migrations {
		upPath := filepath.Join(tmpDir, m.name+".up.sql")
		downPath := filepath.Join(tmpDir, m.name+".down.sql")
		os.WriteFile(upPath, []byte(m.upSQL), 0644)
		os.WriteFile(downPath, []byte(m.downSQL), 0644)
	}

	checker := NewMigrationChecker(nil, tmpDir)
	loaded, err := checker.LoadMigrations()
	if err != nil {
		t.Fatalf("LoadMigrations error: %v", err)
	}

	if len(loaded) != 3 {
		t.Errorf("Expected 3 migrations, got %d", len(loaded))
	}

	for i, m := range loaded {
		if m.Version != migrations[i].version {
			t.Errorf("Migration %d: expected version %d, got %d", i, migrations[i].version, m.Version)
		}
		if m.Name != migrations[i].name {
			t.Errorf("Migration %d: expected name %s, got %s", i, migrations[i].name, m.Name)
		}
	}
}

func TestMigrationChecker_CheckDestructive(t *testing.T) {
	checker := NewMigrationChecker(nil, "")

	tests := []struct {
		name     string
		sql      string
		expected bool
	}{
		{
			name:     "CREATE TABLE",
			sql:      "CREATE TABLE users (id BIGSERIAL PRIMARY KEY);",
			expected: false,
		},
		{
			name:     "DROP TABLE",
			sql:      "DROP TABLE users;",
			expected: true,
		},
		{
			name:     "DROP COLUMN",
			sql:      "ALTER TABLE users DROP COLUMN email;",
			expected: true,
		},
		{
			name:     "TRUNCATE",
			sql:      "TRUNCATE TABLE users;",
			expected: true,
		},
		{
			name:     "DELETE",
			sql:      "DELETE FROM users WHERE id = 1;",
			expected: true,
		},
		{
			name:     "ALTER TABLE (non-destructive)",
			sql:      "ALTER TABLE users ADD COLUMN name TEXT;",
			expected: false, // ALTER TABLE is in the list but this is a false positive
		},
		{
			name:     "DROP INDEX",
			sql:      "DROP INDEX idx_users_email;",
			expected: true,
		},
		{
			name:     "lowercase",
			sql:      "drop table users;",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.CheckDestructive(tt.sql)
			if result != tt.expected {
				t.Errorf("CheckDestructive(%q) = %v, want %v", tt.sql, result, tt.expected)
			}
		})
	}
}

func TestMigrationFile_Struct(t *testing.T) {
	mig := MigrationFile{
		Version: 1,
		Name:    "000001_init",
		Path:    "/path/to/000001_init.up.sql",
		UpSQL:   "CREATE TABLE users (id BIGSERIAL PRIMARY KEY);",
		DownSQL: "DROP TABLE users;",
	}

	if mig.Version != 1 {
		t.Errorf("Expected Version=1, got %d", mig.Version)
	}
	if mig.Name != "000001_init" {
		t.Errorf("Expected Name=000001_init, got %s", mig.Name)
	}
}

func TestMigrationState_Struct(t *testing.T) {
	state := MigrationState{
		CurrentVersion: func() *int { v := 2; return &v }(),
		IsDirty:        false,
		Available: []MigrationFile{
			{Version: 1, Name: "000001_init"},
			{Version: 2, Name: "000002_add_email"},
			{Version: 3, Name: "000003_create_index"},
		},
		Applied: []int{1, 2},
		Pending: []MigrationFile{
			{Version: 3, Name: "000003_create_index"},
		},
		HasDestructive: false,
	}

	if state.CurrentVersion == nil || *state.CurrentVersion != 2 {
		t.Errorf("Expected CurrentVersion=2, got %v", state.CurrentVersion)
	}
	if state.IsDirty {
		t.Errorf("Expected IsDirty=false, got %v", state.IsDirty)
	}
	if len(state.Applied) != 2 {
		t.Errorf("Expected 2 applied migrations, got %d", len(state.Applied))
	}
	if len(state.Pending) != 1 {
		t.Errorf("Expected 1 pending migration, got %d", len(state.Pending))
	}
	if state.Pending[0].Version != 3 {
		t.Errorf("Expected pending version 3, got %d", state.Pending[0].Version)
	}
	if state.HasDestructive {
		t.Errorf("Expected HasDestructive=false, got %v", state.HasDestructive)
	}
}
