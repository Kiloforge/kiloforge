# Implementation Plan: Fix Unhandled SQLite Write Errors Across All Stores

**Track ID:** fix-sqlite-error-handling_20260310041001Z

## Phase 1: Fix Agent Store Errors

### Task 1.1: Fix AddAgent() error handling
- In `agent_store.go`, check error return from `db.Exec()` in `AddAgent()`
- Return `fmt.Errorf("agent store: add agent %s: %w", id, err)` on failure
- Update callers to handle the error

### Task 1.2: Fix UpdateStatus() error handling
- Check error return from `db.Exec()` in `UpdateStatus()`
- Return wrapped error on failure
- Update callers (spawner, lifecycle handlers) to handle the error

### Task 1.3: Add tests for error propagation
- Test AddAgent with a closed/corrupt DB — verify error returned
- Test UpdateStatus with a closed DB — verify error returned

### Task 1.4: Verify Phase 1
- `go test ./internal/adapter/persistence/sqlite/... -race` passes

## Phase 2: Fix Quota and Trace Store Errors

### Task 2.1: Fix RecordUsage() error handling
- In `quota_store.go`, check error return from `db.Exec()` in `RecordUsage()`
- Return wrapped error
- Update quota tracker caller to handle/log the error

### Task 2.2: Fix trace Record() error handling
- In `trace_store.go`, check all `db.Exec()` error returns in `Record()`
- Use transaction if multiple writes need atomicity
- Return wrapped error on any failure

### Task 2.3: Fix migrateConfig() error handling
- In `migrate_json.go`, check `db.Exec()` error return
- Return wrapped error — migration failure should be surfaced

### Task 2.4: Verify Phase 2
- All store tests pass with race detector

## Phase 3: Verify All Callers Handle Errors

### Task 3.1: Audit and fix all callers
- Search for all call sites of the 5 fixed methods
- Ensure each caller checks the error return
- Add error logging or propagation where missing

### Task 3.2: Verify Phase 3
- Full test suite passes: `make test`
