# Implementation Plan: SQLite Storage Layer — Core Schema and Migration

**Track ID:** sqlite-storage-core_20260309140000Z

## Phase 1: Foundation — Driver, Schema, Migrations

- [ ] Task 1.1: Add `modernc.org/sqlite` to `go.mod` / `go.sum`
- [ ] Task 1.2: Create `backend/internal/adapter/persistence/sqlite/db.go` — `Open(dataDir)` function that creates/opens DB, enables WAL mode, runs migrations
- [ ] Task 1.3: Create `backend/internal/adapter/persistence/sqlite/migrate.go` — versioned migration framework with `schema_version` table
- [ ] Task 1.4: Create migration v1 — all CREATE TABLE and CREATE INDEX statements
- [ ] Task 1.5: Add tests for DB open, migration, and schema verification

## Phase 2: Extract Missing Port Interfaces

- [ ] Task 2.1: Create `core/port/board_store.go` — `BoardStore` interface (GetBoard, SaveBoard)
- [ ] Task 2.2: Create `core/port/pr_tracking_store.go` — `PRTrackingStore` interface
- [ ] Task 2.3: Create `core/port/quota_store.go` — `QuotaStore` interface (RecordUsage, GetAgentUsage, GetTotalUsage)
- [ ] Task 2.4: Create `core/port/lock_store.go` — `LockStore` interface (Acquire, Release, Heartbeat, List)
- [ ] Task 2.5: Create `core/port/worktree_store.go` — `WorktreeStore` interface (Acquire, Return, List)
- [ ] Task 2.6: Update existing JSON adapters to implement the new interfaces (backward compat)

## Phase 3: SQLite Adapters — Core Stores

- [ ] Task 3.1: Implement `sqlite.ProjectStore` — implements `port.ProjectStore`
- [ ] Task 3.2: Implement `sqlite.AgentStore` — implements `port.AgentStore`
- [ ] Task 3.3: Implement `sqlite.ConfigStore` — replaces `JSONAdapter` for config persistence
- [ ] Task 3.4: Add tests for project, agent, and config stores

## Phase 4: SQLite Adapters — Supporting Stores

- [ ] Task 4.1: Implement `sqlite.BoardStore` — implements `port.BoardStore`
- [ ] Task 4.2: Implement `sqlite.PRTrackingStore` — implements `port.PRTrackingStore`
- [ ] Task 4.3: Implement `sqlite.QuotaStore` — implements `port.QuotaStore`
- [ ] Task 4.4: Implement `sqlite.LockStore` — implements `port.LockStore`
- [ ] Task 4.5: Implement `sqlite.WorktreeStore` — implements `port.WorktreeStore`
- [ ] Task 4.6: Add tests for all supporting stores

## Phase 5: SQLite Adapters — Trace Persistence

- [ ] Task 5.1: Implement `sqlite.TraceStore` — persistent trace and span storage
- [ ] Task 5.2: Update OTel `StoreProcessor` to write spans to SQLite instead of in-memory store
- [ ] Task 5.3: Update trace API handlers to query from SQLite
- [ ] Task 5.4: Add tests for trace persistence and querying

## Phase 6: JSON Migration and Wiring

- [ ] Task 6.1: Create `sqlite/migrate_json.go` — auto-import from JSON files if DB is fresh
- [ ] Task 6.2: Update `cli/serve.go` — open SQLite DB, create all stores, pass to server
- [ ] Task 6.3: Update `adapter/rest/server.go` — accept SQLite-backed stores
- [ ] Task 6.4: Update `cli/implement.go` — use SQLite stores for agent and trace operations
- [ ] Task 6.5: Update config resolution chain — `SQLiteAdapter` replaces `JSONAdapter`

## Phase 7: Verification

- [ ] Task 7.1: Verify `go test ./...` passes
- [ ] Task 7.2: Verify `make build` succeeds
- [ ] Task 7.3: Verify fresh `kf init` → `kf up` works with new SQLite storage
- [ ] Task 7.4: Verify migration from existing JSON files works correctly
- [ ] Task 7.5: Verify traces persist across orchestrator restart
