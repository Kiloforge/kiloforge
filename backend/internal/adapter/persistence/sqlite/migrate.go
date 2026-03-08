package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

// migrationsFS returns the sub-filesystem containing migration SQL files.
func migrationsFS() fs.FS {
	sub, err := fs.Sub(embeddedMigrations, "migrations")
	if err != nil {
		panic("embedded migrations sub: " + err.Error())
	}
	return sub
}

// Migrate runs all pending goose migrations against the database.
// On first run after upgrading from the old custom migrator, it bridges
// the schema_version table into goose's tracking table so existing
// migrations are not re-applied.
func Migrate(db *sql.DB) error {
	if err := bridgeFromLegacy(db); err != nil {
		return fmt.Errorf("bridge legacy migrations: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrationsFS())
	if err != nil {
		return fmt.Errorf("create goose provider: %w", err)
	}

	if _, err := provider.Up(context.Background()); err != nil {
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

	// Ensure goose's version table exists by running a temporary provider.
	// This creates goose_db_version if it doesn't exist.
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrationsFS())
	if err != nil {
		return fmt.Errorf("create goose provider for bridge: %w", err)
	}
	// GetDBVersion creates the version table as a side effect.
	currentGooseVersion, err := provider.GetDBVersion(context.Background())
	if err != nil {
		return fmt.Errorf("get goose version: %w", err)
	}

	if currentGooseVersion > 0 {
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
