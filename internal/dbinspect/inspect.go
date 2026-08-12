package dbinspect

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DatabaseInfo represents information about a PostgreSQL database.
type DatabaseInfo struct {
	Name             string
	Owner            string
	Encoding         string
	Collation        string
	CType            string
	IsTemplate       bool
	Size             string
	HasTables        bool
	TableCount       int
	HasMigrations    bool
	MigrationVersion *int
	IsDirty          bool
}

// Inspector provides methods to inspect PostgreSQL databases.
type Inspector struct {
	connStr string
}

// NewInspector creates a new database inspector.
func NewInspector(connStr string) *Inspector {
	return &Inspector{connStr: connStr}
}

// ListDatabases returns all databases accessible with the connection string.
func (i *Inspector) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	db, err := sql.Open("pgx", i.connStr)
	if err != nil {
		return nil, fmt.Errorf("open connection: %w", err)
	}
	defer db.Close()

	// Query to list databases with their info
	query := `
		SELECT
			datname,
			pg_get_userbyid(datdba) as owner,
			pg_encoding_to_char(encoding) as encoding,
			datcollate,
			datctype,
			datistemplate,
			pg_size_pretty(pg_database_size(datname)) as size
		FROM pg_database
		WHERE datistemplate = false
		ORDER BY datname
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query databases: %w", err)
	}
	defer rows.Close()

	var databases []DatabaseInfo
	for rows.Next() {
		var db DatabaseInfo
		if err := rows.Scan(&db.Name, &db.Owner, &db.Encoding, &db.Collation, &db.CType, &db.IsTemplate, &db.Size); err != nil {
			return nil, fmt.Errorf("scan database: %w", err)
		}
		databases = append(databases, db)
	}

	return databases, nil
}

// DatabaseExists checks if a database exists.
func (i *Inspector) DatabaseExists(ctx context.Context, name string) (bool, error) {
	db, err := sql.Open("pgx", i.connStr)
	if err != nil {
		return false, fmt.Errorf("open connection: %w", err)
	}
	defer db.Close()

	var exists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check database existence: %w", err)
	}
	return exists, nil
}

// InspectDatabase performs a detailed inspection of a specific database.
func (i *Inspector) InspectDatabase(ctx context.Context, name string) (*DatabaseInfo, error) {
	db, err := sql.Open("pgx", i.getDatabaseConnStr(name))
	if err != nil {
		return nil, fmt.Errorf("open connection to database: %w", err)
	}
	defer db.Close()

	info := &DatabaseInfo{Name: name}

	// Check basic connection
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	info.HasTables = true // Connection works

	// Check if database has tables
	hasTables, tableCount, err := i.checkTables(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("check tables: %w", err)
	}
	info.HasTables = hasTables
	info.TableCount = tableCount

	// Check for migration table and version
	hasMigrations, version, isDirty, err := i.checkMigrations(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("check migrations: %w", err)
	}
	info.HasMigrations = hasMigrations
	info.MigrationVersion = version
	info.IsDirty = isDirty

	return info, nil
}

// checkTables checks if the database has any tables.
func (i *Inspector) checkTables(ctx context.Context, db *sql.DB) (bool, int, error) {
	query := `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
	`
	var count int
	err := db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return false, 0, err
	}
	return count > 0, count, nil
}

// checkMigrations checks for the golang-migrate migration table.
func (i *Inspector) checkMigrations(ctx context.Context, db *sql.DB) (bool, *int, bool, error) {
	// Check if migration table exists
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'schema_migrations'
		)
	`).Scan(&exists)
	if err != nil {
		return false, nil, false, err
	}

	if !exists {
		return false, nil, false, nil
	}

	// Get current version and dirty state
	var version sql.NullInt64
	var dirty bool
	err = db.QueryRowContext(ctx, `
		SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 1
	`).Scan(&version, &dirty)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil, false, nil
		}
		return false, nil, false, err
	}

	var versionPtr *int
	if version.Valid {
		v := int(version.Int64)
		versionPtr = &v
	}

	return true, versionPtr, dirty, nil
}

// CheckPermissions checks if the user has necessary permissions.
func (i *Inspector) CheckPermissions(ctx context.Context, name string) error {
	db, err := sql.Open("pgx", i.getDatabaseConnStr(name))
	if err != nil {
		return fmt.Errorf("open connection: %w", err)
	}
	defer db.Close()

	// Try to create and drop a test table to verify permissions
	_, err = db.ExecContext(ctx, "CREATE TEMP TABLE forgekit_perm_test (id INT)")
	if err != nil {
		return fmt.Errorf("create temp table: %w", err)
	}
	_, err = db.ExecContext(ctx, "DROP TABLE forgekit_perm_test")
	if err != nil {
		return fmt.Errorf("drop temp table: %w", err)
	}

	return nil
}

// GetDatabaseURL returns a connection string for a specific database.
func (i *Inspector) GetDatabaseURL(name string) string {
	return i.getDatabaseConnStr(name)
}

// getDatabaseConnStr builds a connection string for a specific database.
func (i *Inspector) getDatabaseConnStr(dbName string) string {
	// Replace the database name in the connection string
	// The connection string format is typically: postgres://user:pass@host:port/dbname?params
	// We need to replace the dbname part
	parts := strings.Split(i.connStr, "/")
	if len(parts) < 2 {
		return i.connStr
	}

	// Find the last part before query parameters
	lastPart := parts[len(parts)-1]
	// Split by ? to separate dbname from params
	dbAndParams := strings.SplitN(lastPart, "?", 2)

	newLastPart := dbName
	if len(dbAndParams) > 1 {
		newLastPart = dbName + "?" + dbAndParams[1]
	}

	parts[len(parts)-1] = newLastPart
	return strings.Join(parts, "/")
}

// CheckConnection tests if a connection can be established.
func (i *Inspector) CheckConnection(ctx context.Context) error {
	db, err := sql.Open("pgx", i.connStr)
	if err != nil {
		return fmt.Errorf("open connection: %w", err)
	}
	defer db.Close()
	return db.PingContext(ctx)
}

// MaskPassword masks the password in a connection string for display.
func MaskPassword(connStr string) string {
	// postgres://user:password@host:port/dbname -> postgres://user:***@host:port/dbname
	// postgres://user@host:port/dbname -> postgres://user@host:port/dbname (no password to mask)

	// Find the @ that separates user:pass from host
	atIdx := strings.LastIndex(connStr, "@")
	if atIdx == -1 {
		return connStr
	}

	beforeAt := connStr[:atIdx]
	afterAt := connStr[atIdx:]

	// Find the last : in the beforeAt part (which separates user from password)
	colonIdx := strings.LastIndex(beforeAt, ":")
	if colonIdx == -1 {
		// No colon, no password to mask
		return connStr
	}

	// Check if the colon is followed by "//" (protocol like postgres://)
	if colonIdx+2 < len(beforeAt) && beforeAt[colonIdx+1:colonIdx+3] == "//" {
		// The colon is part of the protocol (e.g., postgres://), not user:pass
		return connStr
	}

	user := beforeAt[:colonIdx]
	return user + ":***" + afterAt
}
