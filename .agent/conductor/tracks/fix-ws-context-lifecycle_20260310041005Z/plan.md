# Implementation Plan: Fix WebSocket Session Context and Graceful Shutdown

**Track ID:** fix-ws-context-lifecycle_20260310041005Z

## Phase 1: Fix Session Context and Relay Tracking

### Task 1.1: Inherit request context in WebSocket sessions [x]
- Change `NewSession()` to accept `context.Context` (derived from request)
- Use this context for the session's lifetime
- Session operations check context cancellation

### Task 1.2: Track OutputRelay goroutine [x]
- Add `cancelRelay context.CancelFunc` to Session struct
- OutputRelay goroutine selects on `ctx.Done()` alongside channel reads
- Session.Close() calls cancelRelay()

### Task 1.3: Add stale session cleanup [x]
- In `BroadcastToAgent()`, remove sessions where `ctx.Err() != nil`
- Or add periodic cleanup sweep in session manager
- Ensure thread-safe map modification during iteration

### Task 1.4: Verify Phase 1 [x]
- `go test ./internal/adapter/ws/... -race` passes

## Phase 2: Graceful Shutdown

### Task 2.1: Add server shutdown coordination [x]
- Pass server-level context to WebSocket handler
- When server context cancels, close all active sessions
- Sessions drain pending writes before closing

### Task 2.2: Add tests for lifecycle scenarios [~]
- Test: client disconnect → session cleaned up, relay stopped
- Test: server shutdown → all sessions closed
- Test: stale session doesn't receive broadcasts

### Task 2.3: Verify Phase 2
- Full test suite passes: `make test`
- No goroutine leaks in session lifecycle tests
