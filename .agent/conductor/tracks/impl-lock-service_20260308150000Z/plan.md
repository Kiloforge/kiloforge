# Implementation Plan: HTTP-Based Scoped Lock Service in Relay Server

**Track ID:** impl-lock-service_20260308150000Z

## Phase 1: Lock Manager Core (4 tasks)

### Task 1.1: Define lock types and manager struct
- [x] Create `internal/lock/manager.go`
- [x] `Lock` struct: scope, holder, acquired_at, expires_at
- [x] `Manager` struct: mutex, locks map, waiters map, dataDir
- [x] `New(dataDir)` constructor, `ErrTimeout`, `ErrNotHolder` sentinel errors

### Task 1.2: Implement acquire and release
- [x] `Acquire(ctx, scope, holder, ttl)` — immediate acquire if free, block if held
- [x] Long-poll via waiter channels — notified when lock freed
- [x] `Release(scope, holder)` — only holder can release; notify waiters
- [x] Holder validation: release by non-holder returns error

### Task 1.3: Implement heartbeat and TTL reaper
- [x] `Heartbeat(scope, holder, ttl)` — extend expiry time
- [x] `startReaper(ctx)` — background goroutine, check expired locks every second
- [x] Expired locks auto-released and waiters notified
- [x] `List()` — return all active (non-expired) locks

### Task 1.4: Write lock manager tests
- [x] Test: acquire free lock → immediate success
- [x] Test: acquire held lock → blocks until released
- [x] Test: acquire with timeout → returns ErrTimeout
- [x] Test: release by non-holder → returns ErrNotHolder
- [x] Test: heartbeat extends TTL
- [x] Test: TTL expiry auto-releases lock and unblocks waiter
- [x] Test: concurrent acquire from multiple goroutines (race detector)
- [x] `t.Parallel()` for independent tests

## Phase 2: Persistence (2 tasks)

### Task 2.1: Implement lock state persistence
- [x] `Save()` — write active locks to `{DataDir}/locks.json`
- [x] `Load()` — restore on startup, discard expired locks
- [x] Auto-save on acquire/release

### Task 2.2: Write persistence tests
- [x] Round-trip: save → load → verify locks restored
- [x] Expired locks discarded on load
- [x] Missing file → empty state (no error)
- [x] Corrupt file → graceful error

## Phase 3: HTTP API Endpoints (4 tasks)

### Task 3.1: Implement acquire endpoint
- [x] `POST /api/locks/:scope/acquire`
- [x] Request body: `{"holder": "...", "ttl_seconds": N, "timeout_seconds": N}`
- [x] 200 on success with lock details, 409 on timeout
- [x] Long-poll: request blocks until lock acquired or timeout

### Task 3.2: Implement heartbeat and release endpoints
- [x] `POST /api/locks/:scope/heartbeat` — body: `{"holder": "...", "ttl_seconds": N}`
- [x] `DELETE /api/locks/:scope` — body: `{"holder": "..."}`
- [x] 200 on success, 404 if lock not held by this holder

### Task 3.3: Implement list endpoint
- [x] `GET /api/locks` — return all active locks as JSON array
- [x] Include: scope, holder, acquired_at, expires_at, ttl_remaining_seconds

### Task 3.4: Write HTTP handler tests
- [x] Table-driven tests for each endpoint
- [x] Test acquire with immediate success and with timeout
- [x] Test release by holder and by non-holder
- [x] Test heartbeat extending TTL
- [x] Test list with zero, one, and multiple locks

## Phase 4: Integration and Verification (3 tasks)

### Task 4.1: Register endpoints in relay server
- [x] Create lock manager in relay server startup
- [x] Register all `/api/locks/` routes on existing mux
- [x] Start reaper goroutine with server context

### Task 4.2: Integration test
- [x] Acquire → hold → release → re-acquire flow via HTTP
- [x] TTL expiry via HTTP verified

### Task 4.3: Full build and test
- [x] `go build -buildvcs=false ./...`
- [x] `go test -buildvcs=false -race ./...`
- [x] No regressions — all 15 packages pass

---

**Total: 13 tasks across 4 phases — ALL COMPLETE**
