package sqlite

import (
	"database/sql"
	"fmt"
	"time"
)

// migration is a versioned schema migration.
type migration struct {
	version int
	sql     string
}

// migrations is the ordered list of schema migrations.
var migrations = []migration{
	{version: 1, sql: migrationV1},
	{version: 2, sql: migrationV2},
}

// Migrate runs all pending migrations, creating the schema_version table if needed.
func Migrate(db *sql.DB) error {
	// Create the version-tracking table if it doesn't exist.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version    INTEGER PRIMARY KEY,
		applied_at TEXT    NOT NULL
	)`); err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}

	current, err := currentVersion(db)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		if _, err := db.Exec(m.sql); err != nil {
			return fmt.Errorf("migration v%d: %w", m.version, err)
		}
		if _, err := db.Exec(
			"INSERT INTO schema_version (version, applied_at) VALUES (?, ?)",
			m.version, time.Now().UTC().Format(time.RFC3339),
		); err != nil {
			return fmt.Errorf("record migration v%d: %w", m.version, err)
		}
	}
	return nil
}

// currentVersion returns the highest applied migration version, or 0.
func currentVersion(db *sql.DB) (int, error) {
	var v int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return v, nil
}

const migrationV1 = `
CREATE TABLE config (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE projects (
	slug          TEXT PRIMARY KEY,
	repo_name     TEXT NOT NULL,
	project_dir   TEXT NOT NULL,
	origin_remote TEXT,
	ssh_key_path  TEXT,
	registered_at TEXT NOT NULL,
	active        INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE agents (
	id              TEXT PRIMARY KEY,
	role            TEXT NOT NULL,
	ref             TEXT NOT NULL,
	status          TEXT NOT NULL,
	session_id      TEXT,
	pid             INTEGER,
	worktree_dir    TEXT,
	log_file        TEXT,
	started_at      TEXT NOT NULL,
	updated_at      TEXT NOT NULL,
	suspended_at    TEXT,
	shutdown_reason TEXT,
	resume_error    TEXT,
	model           TEXT
);

CREATE TABLE board_cards (
	track_id       TEXT NOT NULL,
	project_slug   TEXT NOT NULL,
	title          TEXT NOT NULL,
	type           TEXT NOT NULL,
	column_name    TEXT NOT NULL,
	position       INTEGER NOT NULL,
	agent_id       TEXT,
	agent_status   TEXT,
	assigned_worker TEXT,
	pr_number      INTEGER,
	trace_id       TEXT,
	moved_at       TEXT NOT NULL,
	created_at     TEXT NOT NULL,
	PRIMARY KEY (track_id, project_slug)
);

CREATE TABLE pr_tracking (
	pr_number           INTEGER NOT NULL,
	project_slug        TEXT NOT NULL,
	track_id            TEXT NOT NULL,
	developer_agent_id  TEXT,
	developer_session   TEXT,
	developer_work_dir  TEXT,
	reviewer_agent_id   TEXT,
	reviewer_session    TEXT,
	review_cycle_count  INTEGER NOT NULL DEFAULT 0,
	max_review_cycles   INTEGER NOT NULL DEFAULT 3,
	status              TEXT NOT NULL,
	PRIMARY KEY (pr_number, project_slug)
);

CREATE TABLE quota_usage (
	agent_id             TEXT PRIMARY KEY,
	total_cost_usd       REAL NOT NULL DEFAULT 0,
	input_tokens         INTEGER NOT NULL DEFAULT 0,
	output_tokens        INTEGER NOT NULL DEFAULT 0,
	cache_read_tokens    INTEGER NOT NULL DEFAULT 0,
	cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
	result_count         INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE locks (
	scope       TEXT PRIMARY KEY,
	holder      TEXT NOT NULL,
	acquired_at TEXT NOT NULL,
	expires_at  TEXT NOT NULL
);

CREATE TABLE worktrees (
	name        TEXT PRIMARY KEY,
	path        TEXT NOT NULL,
	branch      TEXT NOT NULL,
	status      TEXT NOT NULL DEFAULT 'idle',
	track_id    TEXT,
	agent_id    TEXT,
	acquired_at TEXT
);

CREATE TABLE traces (
	trace_id       TEXT PRIMARY KEY,
	root_span_name TEXT,
	span_count     INTEGER NOT NULL DEFAULT 0,
	started_at     TEXT NOT NULL,
	ended_at       TEXT,
	duration_ms    INTEGER,
	status         TEXT,
	track_id       TEXT,
	session_id     TEXT
);

CREATE TABLE spans (
	span_id    TEXT PRIMARY KEY,
	trace_id   TEXT NOT NULL REFERENCES traces(trace_id),
	parent_id  TEXT,
	name       TEXT NOT NULL,
	start_time TEXT NOT NULL,
	end_time   TEXT NOT NULL,
	duration_ms INTEGER NOT NULL,
	status     TEXT NOT NULL,
	attributes TEXT,
	events     TEXT
);

CREATE INDEX idx_spans_trace ON spans(trace_id);
CREATE INDEX idx_traces_track ON traces(track_id);
CREATE INDEX idx_traces_session ON traces(session_id);
CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agents_ref ON agents(ref);
CREATE INDEX idx_board_project ON board_cards(project_slug);
`

const migrationV2 = `
ALTER TABLE agents ADD COLUMN name TEXT NOT NULL DEFAULT '';
`
