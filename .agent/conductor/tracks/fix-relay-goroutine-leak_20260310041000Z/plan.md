# Implementation Plan: Fix Agent Relay Goroutine Leak and Double-Close Panic

**Track ID:** fix-relay-goroutine-leak_20260310041000Z

## Phase 1: Fix SDKSession Double-Close

### Task 1.1: Add sync.Once to SDKSession.Close()
- Add `closeOnce sync.Once` field to `SDKSession` struct in `sdk_client.go`
- Wrap `close(s.output)` and `close(s.done)` in `s.closeOnce.Do()`
- Ensure `Close()` is idempotent and safe for concurrent calls

### Task 1.2: Add tests for concurrent Close()
- Test calling `Close()` from two goroutines simultaneously — no panic
- Test calling `Close()` twice sequentially — no panic
- Verify channels are properly closed after `Close()`

### Task 1.3: Verify Phase 1
- `go test ./internal/adapter/agent/... -race` passes

## Phase 2: Fix Relay Goroutine Leak on Resume

### Task 2.1: Add relay cancellation tracking
- Add `cancelRelay context.CancelFunc` field to `InteractiveAgent` struct
- In `StartStructuredRelay()`, create a derived context with cancel
- Store the cancel func on the `InteractiveAgent`

### Task 2.2: Stop previous relay before starting new one
- In `ResumeAgent()` handler (api_handler.go), call `cancelRelay()` before starting new relay
- Ensure `StartStructuredRelay()` goroutine exits when its context is cancelled
- Verify the relay select loop checks for context cancellation

### Task 2.3: Add tests for resume relay replacement
- Test: spawn → stop → resume → verify only one relay goroutine active
- Test: spawn → resume (without stop) → verify old relay cancelled
- Use race detector to verify no concurrent channel reads

### Task 2.4: Verify Phase 2
- `go test ./internal/adapter/rest/... -race` passes
- `go test ./internal/adapter/agent/... -race` passes

## Phase 3: Fix Stale WebSocket Broadcast

### Task 3.1: Add context check in BroadcastToAgent
- In `ws/session.go` `BroadcastToAgent()`, check `session.ctx.Err()` before writing
- Skip sessions with cancelled contexts
- Optionally remove stale sessions from the map during broadcast

### Task 3.2: Verify Phase 3
- All tests pass with race detector
- Manual verification: disconnect WS client, verify no errors in broadcast
