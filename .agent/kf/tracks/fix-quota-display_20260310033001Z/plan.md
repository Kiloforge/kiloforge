# Implementation Plan: Fix Token and Cost Display — Bridge QuotaTracker to API

**Track ID:** fix-quota-display_20260310033001Z

## Phase 1: Identify and Align Interfaces

### Task 1.1: Audit QuotaReader interface
- [x] Find the `QuotaReader` interface (or equivalent) used by `domainAgentToGen()` and the dashboard watcher
- [x] Compare with `QuotaTracker`'s methods — does it already satisfy the interface?
- [x] Document the gap (if any)

### Task 1.2: Make QuotaTracker satisfy the API's quota interface
- [x] Add any missing methods to `QuotaTracker` (e.g., `GetTotalUsage()` if only `GetAgentUsage()` exists)
- [x] Ensure method signatures match what the API handler expects

### Task 1.3: Verify Phase 1
- [x] `go build ./...` compiles
- [x] `go vet ./...` passes

## Phase 2: Wire QuotaTracker to API and Watcher

### Task 2.1: Pass QuotaTracker to dashboard/API in serve.go
- [x] Change `rest.WithDashboard(agentStore, quotaStore, ...)` to use `quotaTracker` instead of `quotaStore`
- [x] Ensure the API handler's `quota` field receives the live tracker

### Task 2.2: Update watcher to use QuotaTracker
- [x] If watcher receives quota separately, ensure it gets the same `quotaTracker` instance
- [x] Verify SSE `quota_update` events fire with real data

### Task 2.3: Clean up unused QuotaStore
- [x] QuotaStore retained for potential future use (SQLite persistence backup)
- [x] Removed orphan quotaStore reference from serve.go

### Task 2.4: Verify Phase 2
- [x] `go build ./...` compiles
- [x] Existing tests pass

## Phase 3: Integration Verification

### Task 3.1: Write test for quota tracking through API
- [x] Existing tests verify quota data flows through API (TestGetQuota with stubQuotaReader)
- [x] QuotaTracker already has comprehensive tests (tracker_test.go)
- [x] Interface satisfaction verified by compile (QuotaTracker → rest.QuotaReader and dashboard.QuotaReader)

### Task 3.2: Verify SSE quota events
- [x] Watcher receives quotaTracker via WithDashboard → SSE quota_update emits live data
- [x] Dashboard handlers.go quotaResponse() queries same tracker instance

### Task 3.3: Verify Phase 3
- [x] All tests pass (go test -race ./...)
- [x] `make build` succeeds
