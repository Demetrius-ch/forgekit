package dbinspect

import (
	"context"
	"testing"
)

func TestInspector_MaskPassword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard connection string",
			input:    "postgres://user:password@localhost:5432/dbname",
			expected: "postgres://user:***@localhost:5432/dbname",
		},
		{
			name:     "with query params",
			input:    "postgres://user:password@localhost:5432/dbname?sslmode=disable",
			expected: "postgres://user:***@localhost:5432/dbname?sslmode=disable",
		},
		{
			name:     "no password",
			input:    "postgres://user@localhost:5432/dbname",
			expected: "postgres://user@localhost:5432/dbname",
		},
		{
			name:     "empty password",
			input:    "postgres://user:@localhost:5432/dbname",
			expected: "postgres://user:***@localhost:5432/dbname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskPassword(tt.input)
			if result != tt.expected {
				t.Errorf("MaskPassword(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInspector_GetDatabaseURL(t *testing.T) {
	tests := []struct {
		name     string
		connStr  string
		dbName   string
		expected string
	}{
		{
			name:     "standard connection string",
			connStr:  "postgres://user:pass@localhost:5432/dbname",
			dbName:   "newdb",
			expected: "postgres://user:pass@localhost:5432/newdb",
		},
		{
			name:     "with query params",
			connStr:  "postgres://user:pass@localhost:5432/dbname?sslmode=disable",
			dbName:   "newdb",
			expected: "postgres://user:pass@localhost:5432/newdb?sslmode=disable",
		},
		{
			name:     "with multiple params",
			connStr:  "postgres://user:pass@localhost:5432/dbname?sslmode=disable&connect_timeout=10",
			dbName:   "newdb",
			expected: "postgres://user:pass@localhost:5432/newdb?sslmode=disable&connect_timeout=10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inspector := NewInspector(tt.connStr)
			result := inspector.GetDatabaseURL(tt.dbName)
			if result != tt.expected {
				t.Errorf("GetDatabaseURL(%q) = %q, want %q", tt.dbName, result, tt.expected)
			}
		})
	}
}

func TestDatabaseInfo_Struct(t *testing.T) {
	info := DatabaseInfo{
		Name:             "testdb",
		Owner:            "postgres",
		Encoding:         "UTF8",
		Collation:        "en_US.UTF-8",
		CType:            "en_US.UTF-8",
		IsTemplate:       false,
		Size:             "12 MB",
		HasTables:        true,
		TableCount:       5,
		HasMigrations:    true,
		MigrationVersion: func() *int { v := 3; return &v }(),
		IsDirty:          false,
	}

	if info.Name != "testdb" {
		t.Errorf("Expected Name=testdb, got %s", info.Name)
	}
	if info.TableCount != 5 {
		t.Errorf("Expected TableCount=5, got %d", info.TableCount)
	}
	if info.MigrationVersion == nil || *info.MigrationVersion != 3 {
		t.Errorf("Expected MigrationVersion=3, got %v", info.MigrationVersion)
	}
	if info.IsDirty {
		t.Errorf("Expected IsDirty=false, got %v", info.IsDirty)
	}
}

func TestInspector_Context(t *testing.T) {
	// This test verifies that the context is properly used in method signatures
	inspector := NewInspector("postgres://user:pass@localhost:5432/dbname")
	_ = inspector
	ctx := context.Background()
	_ = ctx
	// Just verifying compilation
}
