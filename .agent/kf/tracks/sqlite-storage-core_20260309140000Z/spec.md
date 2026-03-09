# Specification: SQLite Storage Layer — Core Schema and Migration

**Track ID:** sqlite-storage-core_20260309140000Z
**Type:** Refactor
**Created:** 2026-03-09T14:00:00Z
**Status:** Draft

## Summary

Replace all flat-file JSON stores with a single SQLite database (`~/.kiloforge/kiloforge.db`). Introduce the `modernc.org/sqlite` pure-Go driver (no CGo), define the schema, implement a migration framework, and port all stores: config, projects, agents, board, PR tracking, quota, locks, pool, and traces (now persistent).

## Context

Kiloforge currently stores all metadata in 8+ JSON files scattered across `~/.kiloforge/`. This has multiple problems:
- **No atomicity** — concurrent writes can corrupt state
- **No querying** — loading entire files to find one record
- **No referential integrity** — orphaned records accumulate
- **No persistence for traces** — lost on restart
- **Thread safety is ad-hoc** — each store has different mutex patterns

SQLite provides ACID transactions, schema enforcement, efficient querying, and a single file that's easy to backup. Using `modernc.org/sqlite` (pure Go, no CGo) keeps the build simple and cross-platform.

## Codebase Analysis

### Existing port interfaces (clean swap)
- `core/port/agent_store.go` — `AgentStore` interface with Get, List, Add, Update, etc.
- `core/port/project_store.go` — `ProjectStore` interface with Get, List, Add, Remove, etc.

### Hardcoded adapters (need port extraction first)
- `persistence/jsonfile/board_store.go` — no port interface
- `persistence/jsonfile/pr_tracking_store.go` — no port interface
- `agent/tracker.go` — QuotaTracker, no port interface
- `lock/manager.go` — LockManager, no port interface
- `pool/pool.go` — Pool, no port interface

### Current adapter location
- `backend/internal/adapter/persistence/jsonfile/` — existing JSON stores

### New adapter location
- `backend/internal/adapter/persistence/sqlite/` — new SQLite stores

### Config special case
Config uses a layered resolution chain (defaults → JSON → env → flags). SQLite replaces only the JSON persistence layer, not the resolution chain. The `JSONAdapter` becomes `SQLiteAdapter` that reads/writes config to a `config` table.

## Acceptance Criteria

- [ ] `modernc.org/sqlite` added to `go.mod`
- [ ] SQLite database created at `{DataDir}/kiloforge.db`
- [ ] Schema migration framework with version tracking
- [ ] All tables created: `config`, `projects`, `agents`, `board_cards`, `pr_tracking`, `quota_usage`, `locks`, `worktrees`, `traces`, `spans`
- [ ] Port interfaces extracted for all stores that lack them (board, PR tracking, quota, locks, pool)
- [ ] SQLite adapter implements all port interfaces
- [ ] All existing JSON stores replaced with SQLite adapter calls
- [ ] Traces persisted to SQLite (survive restart)
- [ ] Data migration from existing JSON files on first run
- [ ] `go test ./...` passes
- [ ] `make build` succeeds
- [ ] Existing `kf init` → `kf up` → `kf implement` flow works end-to-end

## Dependencies

None — this is foundational infrastructure.

## Blockers

- **All other pending tracks** should ideally complete before this, OR this track should be implemented with backward compatibility so in-flight work isn't disrupted. Recommend a migration-on-startup approach: if JSON files exist and DB doesn't, auto-migrate.

## Conflict Risk

- **tracing-default-on-be_20260309133000Z** — MEDIUM. Both touch config persistence and trace storage. This track supersedes the config API's JSON persistence.
- **dashboard-root-routing-be_20260309130000Z** — LOW. Touches config defaults but not persistence layer.
- **All other pending tracks** — LOW individually, but cumulative risk if many are in-flight.

## Out of Scope

- Frontend changes (stores are backend-only)
- Changing the config resolution chain (defaults → DB → env → flags still works the same)
- Removing JSON files after migration (keep as backup, can add cleanup later)

## Technical Notes

### Driver choice
`modernc.org/sqlite` — pure Go, no CGo, cross-compiles cleanly. Slightly slower than `mattn/go-sqlite3` but eliminates build complexity.

### Database location
```
~/.kiloforge/kiloforge.db
```

### Schema (v1)
```sql
CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
);

CREATE TABLE config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE projects (
    slug TEXT PRIMARY KEY,
    repo_name TEXT NOT NULL,
    project_dir TEXT NOT NULL,
    origin_remote TEXT,
    ssh_key_path TEXT,
    registered_at TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE agents (
    id TEXT PRIMARY KEY,
    role TEXT NOT NULL,
    ref TEXT NOT NULL,
    status TEXT NOT NULL,
    session_id TEXT,
    pid INTEGER,
    worktree_dir TEXT,
    log_file TEXT,
    started_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    suspended_at TEXT,
    shutdown_reason TEXT,
    resume_error TEXT,
    model TEXT
);

CREATE TABLE board_cards (
    track_id TEXT NOT NULL,
    project_slug TEXT NOT NULL,
    title TEXT NOT NULL,
    type TEXT NOT NULL,
    column_name TEXT NOT NULL,
    position INTEGER NOT NULL,
    agent_id TEXT,
    agent_status TEXT,
    assigned_worker TEXT,
    pr_number INTEGER,
    trace_id TEXT,
    moved_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (track_id, project_slug)
);

CREATE TABLE pr_tracking (
    pr_number INTEGER NOT NULL,
    project_slug TEXT NOT NULL,
    track_id TEXT NOT NULL,
    developer_agent_id TEXT,
    developer_session TEXT,
    developer_work_dir TEXT,
    reviewer_agent_id TEXT,
    reviewer_session TEXT,
    review_cycle_count INTEGER NOT NULL DEFAULT 0,
    max_review_cycles INTEGER NOT NULL DEFAULT 3,
    status TEXT NOT NULL,
    PRIMARY KEY (pr_number, project_slug)
);

CREATE TABLE quota_usage (
    agent_id TEXT PRIMARY KEY,
    total_cost_usd REAL NOT NULL DEFAULT 0,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    result_count INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE locks (
    scope TEXT PRIMARY KEY,
    holder TEXT NOT NULL,
    acquired_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

CREATE TABLE worktrees (
    name TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    branch TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'idle',
    track_id TEXT,
    agent_id TEXT,
    acquired_at TEXT
);

CREATE TABLE traces (
    trace_id TEXT PRIMARY KEY,
    root_span_name TEXT,
    span_count INTEGER NOT NULL DEFAULT 0,
    started_at TEXT NOT NULL,
    duration_ms INTEGER,
    status TEXT,
    track_id TEXT,
    session_id TEXT
);

CREATE TABLE spans (
    span_id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL REFERENCES traces(trace_id),
    parent_id TEXT,
    name TEXT NOT NULL,
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    duration_ms INTEGER NOT NULL,
    status TEXT NOT NULL,
    attributes TEXT,  -- JSON
    events TEXT       -- JSON
);

CREATE INDEX idx_spans_trace ON spans(trace_id);
CREATE INDEX idx_traces_track ON traces(track_id);
CREATE INDEX idx_traces_session ON traces(session_id);
CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agents_ref ON agents(ref);
CREATE INDEX idx_board_project ON board_cards(project_slug);
```

### Migration strategy
1. On startup, check if `kiloforge.db` exists
2. If not, create DB and run schema migrations
3. If JSON files exist, auto-import data into SQLite tables
4. JSON files are NOT deleted (kept as backup)
5. Once DB exists, JSON files are ignored

### Port interface extraction
Before implementing SQLite adapters, extract port interfaces for stores that lack them:
- `BoardStore` interface in `core/port/board_store.go`
- `PRTrackingStore` interface in `core/port/pr_tracking_store.go`
- `QuotaStore` interface in `core/port/quota_store.go`
- `LockStore` interface in `core/port/lock_store.go`
- `WorktreeStore` interface in `core/port/worktree_store.go`

### Connection management
Single `*sql.DB` instance shared across all stores. WAL mode enabled for concurrent reads. Connection created in server startup, passed to all store constructors.

```go
db, err := sql.Open("sqlite", filepath.Join(dataDir, "kiloforge.db"))
db.Exec("PRAGMA journal_mode=WAL")
db.Exec("PRAGMA foreign_keys=ON")
```

---

_Generated by conductor-track-generator from prompt: "migrate all flat file storage to SQLite"_
