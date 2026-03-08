package sqlite

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate runs all pending goose migrations against the database.
// On first run after upgrading from the old custom migrator, it bridges
// the schema_version table into goose's tracking table so existing
// migrations are not re-applied.
func Migrate(db *sql.DB) error {
	if err := bridgeFromLegacy(db); err != nil {
		return fmt.Errorf("bridge legacy migrations: %w", err)
	}

	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	// Suppress goose log output.
	goose.SetLogger(goose.NopLogger())

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

// bridgeFromLegacy detects the old custom schema_version table and seeds
// goose's version table so that already-applied migrations are not re-run.
// After seeding, the legacy table is dropped.
func bridgeFromLegacy(db *sql.DB) error {
	// Check if the old schema_version table exists.
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_version'",
	).Scan(&count)
	if err != nil || count == 0 {
		return nil // no legacy table — nothing to bridge
	}

	// Read the highest applied version from the legacy table.
	var legacyVersion int
	if err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&legacyVersion); err != nil {
		return fmt.Errorf("read legacy version: %w", err)
	}
	if legacyVersion == 0 {
		// Table exists but empty — just drop it.
		_, _ = db.Exec("DROP TABLE schema_version")
		return nil
	}

	// Ensure goose's version table exists.
	if _, err := goose.EnsureDBVersion(db); err != nil {
		return fmt.Errorf("ensure goose version table: %w", err)
	}

	// Check if goose already has versions recorded (bridge already done).
	var gooseMax int64
	if err := db.QueryRow(
		"SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE version_id > 0",
	).Scan(&gooseMax); err == nil && gooseMax > 0 {
		// Already bridged — just drop the legacy table.
		_, _ = db.Exec("DROP TABLE schema_version")
		return nil
	}

	// Seed goose with each legacy version as already-applied.
	for v := 1; v <= legacyVersion; v++ {
		if _, err := db.Exec(
			"INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, ?)",
			v, true,
		); err != nil {
			return fmt.Errorf("seed goose version %d: %w", v, err)
		}
	}

	// Drop the legacy table.
	if _, err := db.Exec("DROP TABLE schema_version"); err != nil {
		return fmt.Errorf("drop legacy table: %w", err)
	}

	return nil
}
