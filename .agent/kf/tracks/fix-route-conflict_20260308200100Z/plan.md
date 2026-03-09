# Implementation Plan: Fix Route Pattern Conflict Panic on Startup

**Track ID:** fix-route-conflict_20260308200100Z

## Phase 1: Fix Lock Handler Routes (3 tasks)

### Task 1.1: Update lock handler route registration
[x] In `lock/handler.go`, replace method-less `HandleFunc` calls with explicit method-prefixed routes using Go 1.22+ path parameters:
- `GET /-/api/locks` → list
- `POST /-/api/locks/{scope}/acquire` → acquire
- `POST /-/api/locks/{scope}/heartbeat` → heartbeat
- `DELETE /-/api/locks/{scope}` → release

### Task 1.2: Update lock handler methods
[x] Refactor `handleAcquire`, `handleHeartbeat`, `handleRelease` to accept `http.Request` directly and extract `scope` from `r.PathValue("scope")` instead of manual path parsing. Remove `handleLockAction` dispatcher.

### Task 1.3: Update lock handler tests
[x] Tests pass without changes — existing test paths match the new route patterns.

## Phase 2: Verify (2 tasks)

### Task 2.1: Run tests
[x] Run `go test ./internal/adapter/lock/...` and `go test ./...` — all pass.

### Task 2.2: Manual verification
[x] Build succeeds. Full test suite passes (all 17 packages).

---

**Total: 2 phases, 5 tasks — all complete**
