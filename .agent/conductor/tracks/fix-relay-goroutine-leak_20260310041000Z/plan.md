# Implementation Plan: Fix Agent Relay Goroutine Leak and Double-Close Panic

**Track ID:** fix-relay-goroutine-leak_20260310041000Z

## Phase 1: Fix SDKSession Double-Close

### [x] Task 1.1: Add sync.Once to SDKSession.Close()
### [x] Task 1.2: Add tests for concurrent Close()
### [x] Task 1.3: Verify Phase 1

## Phase 2: Fix Relay Goroutine Leak on Resume

### [x] Task 2.1: Add relay cancellation tracking
### [x] Task 2.2: Stop previous relay before starting new one
### [x] Task 2.3: Add tests for resume relay replacement
### [x] Task 2.4: Verify Phase 2

## Phase 3: Fix Stale WebSocket Broadcast

### [x] Task 3.1: Add context check in BroadcastToAgent
### [x] Task 3.2: Verify Phase 3
