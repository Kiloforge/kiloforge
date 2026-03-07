# Implementation Plan: Quota-Aware Agent Management and Cost Reporting

**Track ID:** impl-quota-aware-agents_20260307160001Z

## Phase 1: Spawn Throttling (3 tasks)

### Task 1.1: Add pre-spawn quota check
- [ ] In `SpawnDeveloper()` and `SpawnReviewer()`, check `tracker.IsRateLimited()` before spawning
- [ ] Return descriptive error with retry-after duration
- [ ] Nil-safe: no check if tracker is nil

### Task 1.2: Implement spawn queue in relay server
- [ ] Add `spawnQueue` to relay server struct
- [ ] Queue entries: spawn function + retry timer + max retries
- [ ] Background goroutine: drain queue when not rate limited
- [ ] Persist queue to disk for crash recovery

### Task 1.3: Handle agent 429 failures
- [ ] When tracker detects 429 error event for an agent, don't mark as "failed"
- [ ] Instead set status to "throttled" and allow CC to handle its own retry
- [ ] If agent exits due to 429, re-queue the spawn with backoff

## Phase 2: Cost Reporting in CLI (3 tasks)

### Task 2.1: Extend `crelay status` with quota info
- [ ] Load tracker data (from file if relay not running, from memory if available)
- [ ] Display aggregate token usage and estimated cost
- [ ] Display rate limit status: OK / throttled / limited

### Task 2.2: Add per-agent cost breakdown
- [ ] List active agents with token counts and estimated cost
- [ ] Show track ID association for each agent
- [ ] Format token counts with comma separators for readability

### Task 2.3: Write status output tests
- [ ] Table-driven tests for various status scenarios
- [ ] Test formatting edge cases (zero usage, very large numbers)

## Phase 3: Budget Enforcement (3 tasks)

### Task 3.1: Add budget configuration
- [ ] Add optional `max_session_cost_usd` field to config
- [ ] Load and validate at startup
- [ ] Zero/unset means no budget limit

### Task 3.2: Enforce budget at spawn time
- [ ] Check `tracker.GetTotalUsage().EstCostUSD` against budget before spawning
- [ ] Emit warning at 80% threshold
- [ ] Refuse spawn at 100% with clear error message

### Task 3.3: Write budget enforcement tests
- [ ] Test: under budget → spawn allowed
- [ ] Test: at warning threshold → spawn allowed + warning emitted
- [ ] Test: over budget → spawn refused with error
- [ ] Test: no budget configured → always allowed

## Phase 4: Verification (2 tasks)

### Task 4.1: Integration test with simulated 429
- [ ] Mock CC process that emits 429 error in stream-json
- [ ] Verify: spawn queue retries after backoff
- [ ] Verify: status shows "throttled" during backoff

### Task 4.2: Full build and test
- [ ] `go build ./...`
- [ ] `go test -race ./...`
- [ ] Verify no regressions

---

**Total: 11 tasks across 4 phases**
