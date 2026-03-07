# Implementation Plan: HTTP-Based Scoped Lock Service in Relay Server

**Track ID:** impl-lock-service_20260308150000Z

## Phase 1: Lock Manager Core (4 tasks)

### Task 1.1: Define lock types and manager struct
- [ ] Create `internal/lock/manager.go`
- [ ] `Lock` struct: scope, holder, acquired_at, expires_at
- [ ] `Manager` struct: mutex, locks map, waiters map, dataDir
- [ ] `New(dataDir)` constructor, `ErrTimeout`, `ErrNotHolder` sentinel errors

### Task 1.2: Implement acquire and release
- [ ] `Acquire(ctx, scope, holder, ttl)` — immediate acquire if free, block if held
- [ ] Long-poll via waiter channels — notified when lock freed
- [ ] `Release(scope, holder)` — only holder can release; notify waiters
- [ ] Holder validation: release by non-holder returns error

### Task 1.3: Implement heartbeat and TTL reaper
- [ ] `Heartbeat(scope, holder, ttl)` — extend expiry time
- [ ] `startReaper(ctx)` — background goroutine, check expired locks every second
- [ ] Expired locks auto-released and waiters notified
- [ ] `List()` — return all active (non-expired) locks

### Task 1.4: Write lock manager tests
- [ ] Test: acquire free lock → immediate success
- [ ] Test: acquire held lock → blocks until released
- [ ] Test: acquire with timeout → returns ErrTimeout
- [ ] Test: release by non-holder → returns ErrNotHolder
- [ ] Test: heartbeat extends TTL
- [ ] Test: TTL expiry auto-releases lock and unblocks waiter
- [ ] Test: concurrent acquire from multiple goroutines (race detector)
- [ ] `t.Parallel()` for independent tests

## Phase 2: Persistence (2 tasks)

### Task 2.1: Implement lock state persistence
- [ ] `Save()` — write active locks to `{DataDir}/locks.json`
- [ ] `Load()` — restore on startup, discard expired locks
- [ ] Auto-save on acquire/release

### Task 2.2: Write persistence tests
- [ ] Round-trip: save → load → verify locks restored
- [ ] Expired locks discarded on load
- [ ] Missing file → empty state (no error)
- [ ] Corrupt file → graceful error

## Phase 3: HTTP API Endpoints (4 tasks)

### Task 3.1: Implement acquire endpoint
- [ ] `POST /api/locks/:scope/acquire`
- [ ] Request body: `{"holder": "...", "ttl_seconds": N, "timeout_seconds": N}`
- [ ] 200 on success with lock details, 409 on timeout
- [ ] Long-poll: request blocks until lock acquired or timeout

### Task 3.2: Implement heartbeat and release endpoints
- [ ] `POST /api/locks/:scope/heartbeat` — body: `{"holder": "...", "ttl_seconds": N}`
- [ ] `DELETE /api/locks/:scope` — body: `{"holder": "..."}`
- [ ] 200 on success, 404 if lock not held by this holder

### Task 3.3: Implement list endpoint
- [ ] `GET /api/locks` — return all active locks as JSON array
- [ ] Include: scope, holder, acquired_at, expires_at, ttl_remaining_seconds

### Task 3.4: Write HTTP handler tests
- [ ] Table-driven tests for each endpoint
- [ ] Test acquire with immediate success and with timeout
- [ ] Test release by holder and by non-holder
- [ ] Test heartbeat extending TTL
- [ ] Test list with zero, one, and multiple locks

## Phase 4: Integration and Verification (3 tasks)

### Task 4.1: Register endpoints in relay server
- [ ] Create lock manager in relay server startup
- [ ] Register all `/api/locks/` routes on existing mux
- [ ] Start reaper goroutine with server context

### Task 4.2: Integration test
- [ ] Start relay server with lock manager
- [ ] Acquire lock via HTTP, verify held
- [ ] Attempt second acquire from different holder, verify blocks
- [ ] Release first lock, verify second acquires
- [ ] Test TTL expiry via HTTP

### Task 4.3: Full build and test
- [ ] `go build ./...`
- [ ] `go test -race ./...`
- [ ] Verify no regressions

---

**Total: 13 tasks across 4 phases**
