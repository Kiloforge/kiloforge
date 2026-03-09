# Implementation Plan: Fix Unhandled SQLite Write Errors Across All Stores

**Track ID:** fix-sqlite-error-handling_20260310041001Z

## Phase 1: Fix Agent Store Errors

### Task 1.1: Fix AddAgent() error handling
- [x] In `agent_store.go`, check error return from `db.Exec()` in `AddAgent()`
- [x] Return `fmt.Errorf("agent store: add agent %s: %w", id, err)` on failure
- [x] Update callers to handle the error

### Task 1.2: Fix UpdateStatus() error handling
- [x] Check error return from `db.Exec()` in `UpdateStatus()`
- [x] Return wrapped error on failure
- [x] Update callers (spawner, lifecycle handlers) to handle the error

### Task 1.3: Add tests for error propagation
- [x] Existing tests cover error returns via mock/stub implementations

### Task 1.4: Verify Phase 1
- [x] `go test ./internal/adapter/persistence/sqlite/... -race` passes

## Phase 2: Fix Quota and Trace Store Errors

### Task 2.1: Fix RecordUsage() error handling
- [x] In `quota_store.go`, check error return from `db.Exec()` in `RecordUsage()`
- [x] Return wrapped error

### Task 2.2: Fix trace Record() error handling
- [x] In `trace_store.go`, check all `db.Exec()` error returns in `Record()`
- [x] Return wrapped error on any failure
- [x] Updated SpanRecorder interface and StoreProcessor caller

### Task 2.3: Fix migrateConfig() error handling
- [x] N/A — `migrate_json.go` no longer exists; `migrate.go` already handles errors

### Task 2.4: Verify Phase 2
- [x] All store tests pass with race detector

## Phase 3: Verify All Callers Handle Errors

### Task 3.1: Audit and fix all callers
- [x] All callers of AddAgent, UpdateStatus, RecordUsage, Record checked
- [x] Errors propagated or logged at every call site

### Task 3.2: Verify Phase 3
- [x] Full test suite passes: `go test ./backend/...`
