# Implementation Plan: CC Stream-JSON Parser and Centralized Quota Tracker

**Track ID:** impl-quota-tracker_20260307160000Z

## Phase 1: Stream-JSON Parser (3 tasks)

### Task 1.1: Define parser types and interface
- [ ] Create `internal/agent/parser.go` with `StreamEvent`, `UsageData`, `ErrorData` types
- [ ] Implement `ParseStreamLine(line string) (StreamEvent, error)`
- [ ] Handle: result events with usage, error events with status codes, unknown event types

### Task 1.2: Write parser tests
- [ ] Table-driven tests for known stream-json event formats
- [ ] Test malformed JSON (graceful skip, no crash)
- [ ] Test events without usage data (message events, system events)
- [ ] Test 429/529 error events with retry-after

### Task 1.3: Verify against real CC output
- [ ] Capture sample stream-json output from a real CC session
- [ ] Ensure parser handles all observed event types
- [ ] Adjust parser if real output differs from documentation

## Phase 2: Quota Tracker (4 tasks)

### Task 2.1: Implement in-memory tracker
- [ ] Create `internal/agent/tracker.go` with `QuotaTracker` struct
- [ ] Thread-safe via `sync.RWMutex`
- [ ] `RecordEvent(agentID, event)` — update per-agent and aggregate counters
- [ ] `GetAgentUsage(id)`, `GetTotalUsage()` — read accessors

### Task 2.2: Add rate limit detection
- [ ] `IsRateLimited()` — returns true if any agent received 429 recently
- [ ] `RetryAfter()` — returns max retry-after across all agents
- [ ] Track rolling window of errors (last N minutes)

### Task 2.3: Add file persistence
- [ ] `Save()` — write `quota-usage.json` to DataDir
- [ ] `Load()` — restore on startup
- [ ] Auto-save on interval (e.g., every 30 seconds) or on significant events

### Task 2.4: Write tracker tests
- [ ] Concurrent RecordEvent from multiple goroutines (race detector)
- [ ] Aggregation correctness across multiple agents
- [ ] Persistence round-trip (save + load)
- [ ] Rate limit detection with time-based expiry
- [ ] `t.Parallel()` for all independent tests

## Phase 3: Spawner Integration (3 tasks)

### Task 3.1: Inject tracker into spawner
- [ ] Add `QuotaTracker` field to `Spawner` struct
- [ ] Pass tracker via `NewSpawner()` constructor
- [ ] Nil-safe: if tracker is nil, spawner works as before (no parsing)

### Task 3.2: Parse stream output in spawner goroutines
- [ ] In both `SpawnReviewer` and `SpawnDeveloper` goroutines:
  - Parse each line with `ParseStreamLine()`
  - On successful parse, call `tracker.RecordEvent(agentID, event)`
  - Always write line to log file (unchanged behavior)
- [ ] Associate agent ID with track ID in tracker

### Task 3.3: Integration test
- [ ] Test spawner with mock CC process that emits stream-json
- [ ] Verify tracker receives events
- [ ] Verify log file still written correctly

## Phase 4: Verification (2 tasks)

### Task 4.1: Race detector pass
- [ ] `go test -race ./internal/agent/...`
- [ ] Fix any races found

### Task 4.2: Full build and test
- [ ] `go build ./...`
- [ ] `go test ./...`
- [ ] Verify no regressions

---

**Total: 12 tasks across 4 phases**
