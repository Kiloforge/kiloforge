package sqlite

import (
	"database/sql"
	"testing"
)

func TestOpen_CreatesDBAndMigrates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// Verify schema version.
	var version int
	if err := db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version); err != nil {
		t.Fatalf("query schema_version: %v", err)
	}
	if version != 1 {
		t.Errorf("schema version: want 1, got %d", version)
	}

	// Verify key tables exist.
	tables := []string{"config", "projects", "agents", "board_cards", "pr_tracking",
		"quota_usage", "locks", "worktrees", "traces", "spans"}
	for _, table := range tables {
		if !tableExists(t, db, table) {
			t.Errorf("table %q does not exist", table)
		}
	}
}

func TestOpen_IdempotentMigration(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	db1, err := Open(dir)
	if err != nil {
		t.Fatalf("Open (1st): %v", err)
	}
	db1.Close()

	db2, err := Open(dir)
	if err != nil {
		t.Fatalf("Open (2nd): %v", err)
	}
	defer db2.Close()

	var version int
	if err := db2.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version); err != nil {
		t.Fatalf("query schema_version: %v", err)
	}
	if version != 1 {
		t.Errorf("schema version: want 1, got %d", version)
	}
}

func TestOpen_WALEnabled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode: want wal, got %q", mode)
	}
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", name,
	).Scan(&count)
	if err != nil {
		t.Fatalf("check table %q: %v", name, err)
	}
	return count > 0
}
