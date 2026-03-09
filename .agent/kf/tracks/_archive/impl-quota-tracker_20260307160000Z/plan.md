# Implementation Plan: CC Stream-JSON Parser and Centralized Quota Tracker

**Track ID:** impl-quota-tracker_20260307160000Z

## Phase 1: Stream-JSON Parser (3 tasks)

### Task 1.1: Define parser types and interface
- [x] Create `internal/agent/parser.go` with `StreamEvent`, `UsageData`, `ErrorData` types
- [x] Implement `ParseStreamLine(line string) (StreamEvent, error)`
- [x] Handle: result events with usage, error events with status codes, unknown event types

### Task 1.2: Write parser tests
- [x] Table-driven tests for known stream-json event formats
- [x] Test malformed JSON (graceful skip, no crash)
- [x] Test events without usage data (message events, system events)
- [x] Test 429/529 error events with retry-after

### Task 1.3: Verify against real CC output
- [x] Capture sample stream-json output from a real CC session
- [x] Ensure parser handles all observed event types
- [x] Adjust parser if real output differs from documentation

## Phase 2: Quota Tracker (4 tasks)

### Task 2.1: Implement in-memory tracker
- [x] Create `internal/agent/tracker.go` with `QuotaTracker` struct
- [x] Thread-safe via `sync.RWMutex`
- [x] `RecordEvent(agentID, event)` — update per-agent and aggregate counters
- [x] `GetAgentUsage(id)`, `GetTotalUsage()` — read accessors

### Task 2.2: Add rate limit detection
- [x] `IsRateLimited()` — returns true if any agent received 429 recently
- [x] `RetryAfter()` — returns max retry-after across all agents
- [x] Track rolling window of errors (last N minutes)

### Task 2.3: Add file persistence
- [x] `Save()` — write `quota-usage.json` to DataDir
- [x] `Load()` — restore on startup
- [x] Auto-save on interval (e.g., every 30 seconds) or on significant events

### Task 2.4: Write tracker tests
- [x] Concurrent RecordEvent from multiple goroutines (race detector)
- [x] Aggregation correctness across multiple agents
- [x] Persistence round-trip (save + load)
- [x] Rate limit detection with time-based expiry
- [x] `t.Parallel()` for all independent tests

## Phase 3: Spawner Integration (3 tasks)

### Task 3.1: Inject tracker into spawner
- [x] Add `QuotaTracker` field to `Spawner` struct
- [x] Pass tracker via `NewSpawner()` constructor
- [x] Nil-safe: if tracker is nil, spawner works as before (no parsing)

### Task 3.2: Parse stream output in spawner goroutines
- [x] In both `SpawnReviewer` and `SpawnDeveloper` goroutines:
  - Parse each line with `ParseStreamLine()`
  - On successful parse, call `tracker.RecordEvent(agentID, event)`
  - Always write line to log file (unchanged behavior)
- [x] Associate agent ID with track ID in tracker

### Task 3.3: Integration test
- [x] Test spawner with mock CC process that emits stream-json
- [x] Verify tracker receives events
- [x] Verify log file still written correctly

## Phase 4: Verification (2 tasks)

### Task 4.1: Race detector pass
- [x] `go test -race ./internal/agent/...`
- [x] Fix any races found

### Task 4.2: Full build and test
- [x] `go build ./...`
- [x] `go test ./...`
- [x] Verify no regressions

---

**Total: 12 tasks across 4 phases**
