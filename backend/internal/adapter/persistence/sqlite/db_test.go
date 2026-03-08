package sqlite

import (
	"database/sql"
	"testing"
	"time"

	"context"

	"github.com/pressly/goose/v3"

	_ "modernc.org/sqlite"
)

func TestOpen_CreatesDBAndMigrates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// Verify goose version table exists and has entries.
	var version int64
	if err := db.QueryRow("SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied = 1").Scan(&version); err != nil {
		t.Fatalf("query goose_db_version: %v", err)
	}
	if version < 2 {
		t.Errorf("goose version: want >= 2, got %d", version)
	}

	// Verify key tables exist.
	tables := []string{"config", "projects", "agents", "board_cards", "pr_tracking",
		"quota_usage", "locks", "worktrees", "traces", "spans"}
	for _, table := range tables {
		if !tableExists(t, db, table) {
			t.Errorf("table %q does not exist", table)
		}
	}

	// Verify agents.name column exists (migration 002).
	if !columnExists(t, db, "agents", "name") {
		t.Error("agents.name column does not exist")
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

	var version int64
	if err := db2.QueryRow("SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied = 1").Scan(&version); err != nil {
		t.Fatalf("query goose_db_version: %v", err)
	}
	if version < 2 {
		t.Errorf("goose version: want >= 2, got %d", version)
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

func TestBridge_LegacySchemaVersion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Simulate an old database: create schema_version with version 2 and all tables.
	dbPath := dir + "/kiloforge.db"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}

	// Create legacy schema_version table.
	if _, err := db.Exec(`CREATE TABLE schema_version (
		version    INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create schema_version: %v", err)
	}

	// Record versions 1 and 2 as applied.
	now := time.Now().UTC().Format(time.RFC3339)
	db.Exec("INSERT INTO schema_version (version, applied_at) VALUES (1, ?)", now)
	db.Exec("INSERT INTO schema_version (version, applied_at) VALUES (2, ?)", now)

	// Create all the tables that V1+V2 would have created.
	db.Exec(`CREATE TABLE config (key TEXT PRIMARY KEY, value TEXT NOT NULL)`)
	db.Exec(`CREATE TABLE projects (slug TEXT PRIMARY KEY, repo_name TEXT, project_dir TEXT, origin_remote TEXT, ssh_key_path TEXT, registered_at TEXT, active INTEGER DEFAULT 1)`)
	db.Exec(`CREATE TABLE agents (id TEXT PRIMARY KEY, role TEXT, ref TEXT, status TEXT, session_id TEXT, pid INTEGER, worktree_dir TEXT, log_file TEXT, started_at TEXT, updated_at TEXT, suspended_at TEXT, shutdown_reason TEXT, resume_error TEXT, model TEXT, name TEXT NOT NULL DEFAULT '')`)
	db.Exec(`CREATE TABLE board_cards (track_id TEXT, project_slug TEXT, title TEXT, type TEXT, column_name TEXT, position INTEGER, agent_id TEXT, agent_status TEXT, assigned_worker TEXT, pr_number INTEGER, trace_id TEXT, moved_at TEXT, created_at TEXT, PRIMARY KEY (track_id, project_slug))`)
	db.Exec(`CREATE TABLE pr_tracking (pr_number INTEGER, project_slug TEXT, track_id TEXT, developer_agent_id TEXT, developer_session TEXT, developer_work_dir TEXT, reviewer_agent_id TEXT, reviewer_session TEXT, review_cycle_count INTEGER DEFAULT 0, max_review_cycles INTEGER DEFAULT 3, status TEXT, PRIMARY KEY (pr_number, project_slug))`)
	db.Exec(`CREATE TABLE quota_usage (agent_id TEXT PRIMARY KEY, total_cost_usd REAL DEFAULT 0, input_tokens INTEGER DEFAULT 0, output_tokens INTEGER DEFAULT 0, cache_read_tokens INTEGER DEFAULT 0, cache_creation_tokens INTEGER DEFAULT 0, result_count INTEGER DEFAULT 0)`)
	db.Exec(`CREATE TABLE locks (scope TEXT PRIMARY KEY, holder TEXT, acquired_at TEXT, expires_at TEXT)`)
	db.Exec(`CREATE TABLE worktrees (name TEXT PRIMARY KEY, path TEXT, branch TEXT, status TEXT DEFAULT 'idle', track_id TEXT, agent_id TEXT, acquired_at TEXT)`)
	db.Exec(`CREATE TABLE traces (trace_id TEXT PRIMARY KEY, root_span_name TEXT, span_count INTEGER DEFAULT 0, started_at TEXT, ended_at TEXT, duration_ms INTEGER, status TEXT, track_id TEXT, session_id TEXT)`)
	db.Exec(`CREATE TABLE spans (span_id TEXT PRIMARY KEY, trace_id TEXT REFERENCES traces(trace_id), parent_id TEXT, name TEXT, start_time TEXT, end_time TEXT, duration_ms INTEGER, status TEXT, attributes TEXT, events TEXT)`)
	db.Close()

	// Now open via our Open() which should bridge and succeed.
	db2, err := Open(dir)
	if err != nil {
		t.Fatalf("Open after bridge: %v", err)
	}
	defer db2.Close()

	// Verify schema_version is gone.
	var count int
	db2.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_version'").Scan(&count)
	if count != 0 {
		t.Error("schema_version table should be dropped after bridge")
	}

	// Verify goose_db_version exists with correct version.
	var gooseVersion int64
	if err := db2.QueryRow("SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied = 1").Scan(&gooseVersion); err != nil {
		t.Fatalf("query goose version: %v", err)
	}
	if gooseVersion < 2 {
		t.Errorf("goose version after bridge: want >= 2, got %d", gooseVersion)
	}
}

func TestGooseDown_RollsBack(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Verify agents.name exists before rollback.
	if !columnExists(t, db, "agents", "name") {
		t.Fatal("agents.name should exist before down")
	}

	// Create a provider for rollback.
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrationsFS())
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	// Roll back migration 003.
	if _, err := provider.Down(context.Background()); err != nil {
		t.Fatalf("goose.Down (003): %v", err)
	}

	// Roll back migration 002.
	if _, err := provider.Down(context.Background()); err != nil {
		t.Fatalf("goose.Down (002): %v", err)
	}

	// agents.name should be gone.
	if columnExists(t, db, "agents", "name") {
		t.Error("agents.name should not exist after rolling back 002")
	}

	// Roll back migration 001.
	if _, err := provider.Down(context.Background()); err != nil {
		t.Fatalf("goose.Down (001): %v", err)
	}

	// All tables should be gone.
	if tableExists(t, db, "agents") {
		t.Error("agents table should not exist after rolling back 001")
	}

	db.Close()
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

func columnExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("pragma table_info(%s): %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt *string
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			continue
		}
		if name == column {
			return true
		}
	}
	return false
}
