# Implementation Plan: Fix Token and Cost Display — Bridge QuotaTracker to API

**Track ID:** fix-quota-display_20260310033001Z

## Phase 1: Identify and Align Interfaces

### Task 1.1: Audit QuotaReader interface
- Find the `QuotaReader` interface (or equivalent) used by `domainAgentToGen()` and the dashboard watcher
- Compare with `QuotaTracker`'s methods — does it already satisfy the interface?
- Document the gap (if any)

### Task 1.2: Make QuotaTracker satisfy the API's quota interface
- Add any missing methods to `QuotaTracker` (e.g., `GetTotalUsage()` if only `GetAgentUsage()` exists)
- Ensure method signatures match what the API handler expects

### Task 1.3: Verify Phase 1
- `go build ./...` compiles
- `go vet ./...` passes

## Phase 2: Wire QuotaTracker to API and Watcher

### Task 2.1: Pass QuotaTracker to dashboard/API in serve.go
- Change `rest.WithDashboard(agentStore, quotaStore, ...)` to use `quotaTracker` instead of `quotaStore`
- Ensure the API handler's `quota` field receives the live tracker

### Task 2.2: Update watcher to use QuotaTracker
- If watcher receives quota separately, ensure it gets the same `quotaTracker` instance
- Verify SSE `quota_update` events fire with real data

### Task 2.3: Clean up unused QuotaStore
- If QuotaStore is now fully unused, remove it or mark as deprecated
- Remove the orphan `RecordUsage()` call site (if any)

### Task 2.4: Verify Phase 2
- `go build ./...` compiles
- Existing tests pass

## Phase 3: Integration Verification

### Task 3.1: Write test for quota tracking through API
- Use MockSession with a scenario that has cost/usage data
- Spawn interactive agent, wait for turn completion
- Query agent via API, verify non-zero tokens and cost in response

### Task 3.2: Verify SSE quota events
- Check that the watcher emits `quota_update` after agent turn
- Verify the event payload has real token/cost data

### Task 3.3: Verify Phase 3
- All tests pass
- `make build` succeeds
