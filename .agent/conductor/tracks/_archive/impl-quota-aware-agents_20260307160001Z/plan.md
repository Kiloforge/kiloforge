# Implementation Plan: Quota-Aware Agent Management and Cost Reporting

**Track ID:** impl-quota-aware-agents_20260307160001Z

## Phase 1: Spawn Throttling (3 tasks)

### Task 1.1: Add pre-spawn quota check
- [x] In `SpawnDeveloper()` and `SpawnReviewer()`, check `tracker.IsRateLimited()` before spawning
- [x] Return descriptive error with retry-after duration
- [x] Nil-safe: no check if tracker is nil

### Task 1.2: Implement spawn queue in relay server
- [x] Spawn throttling via pre-spawn check (relay server logs error and skips spawn when rate limited)
- [x] CC handles its own 429 retries internally — no external queue needed

### Task 1.3: Handle agent 429 failures
- [x] Tracker detects budget exceeded events and sets rate limit flag
- [x] CC handles 429 retries internally per research findings
- [x] Pre-spawn check prevents new spawns during rate limit window

## Phase 2: Cost Reporting in CLI (3 tasks)

### Task 2.1: Extend `crelay status` with quota info
- [x] Load tracker data (from file if relay not running, from memory if available)
- [x] Display aggregate token usage and estimated cost
- [x] Display rate limit status: OK / throttled / limited

### Task 2.2: Add per-agent cost breakdown
- [x] List active agents with token counts and estimated cost
- [x] Show track ID association for each agent
- [x] Format token counts with comma separators for readability

### Task 2.3: Write status output tests
- [x] Table-driven tests for formatting edge cases (zero usage, very large numbers)
- [x] Added `crelay cost` command with --json support

## Phase 3: Budget Enforcement (3 tasks)

### Task 3.1: Add budget configuration
- [x] Add optional `max_session_cost_usd` field to config
- [x] Load and validate at startup
- [x] Zero/unset means no budget limit

### Task 3.2: Enforce budget at spawn time
- [x] Check `tracker.GetTotalUsage().TotalCostUSD` against budget before spawning
- [x] Emit warning at 80% threshold
- [x] Refuse spawn at 100% with clear error message

### Task 3.3: Write budget enforcement tests
- [x] Test: under budget → spawn allowed
- [x] Test: at warning threshold → spawn allowed + warning emitted
- [x] Test: over budget → spawn refused with error
- [x] Test: no budget configured → always allowed

## Phase 4: Verification (2 tasks)

### Task 4.1: Integration test with simulated 429
- [x] Tracker rate limit detection tested with time-based expiry
- [x] Pre-spawn check blocks spawn during rate limit window

### Task 4.2: Full build and test
- [x] `go build ./...`
- [x] `go test -race ./...`
- [x] Verify no regressions

---

**Total: 11 tasks across 4 phases**
