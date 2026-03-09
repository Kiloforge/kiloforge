# Specification: E2E Tests — Real-Time SSE Updates: Connection, Events, and Reconnection

**Track ID:** e2e-sse-realtime_20260309194837Z
**Type:** Chore
**Created:** 2026-03-09T19:48:37Z
**Status:** Draft

## Summary

Comprehensive E2E tests for the Server-Sent Events stream: establishing connection, receiving typed events (agent, quota, track, board, project, lock, trace), UI reactivity to events, and reconnection behavior, covering happy path, edge cases, and expected failures.

## Context

Kiloforge's dashboard uses Server-Sent Events (SSE) at the `/events` endpoint for real-time updates. The SSE stream carries typed events that drive UI reactivity without polling:

| Event Type | Trigger | UI Effect |
|---|---|---|
| `agent_update` | Agent status change | Refresh agent list/grid |
| `agent_removed` | Agent deleted | Remove agent from list |
| `quota_update` | Token/cost usage change | Update stat cards |
| `track_update` | Track status change | Update track list |
| `board_update` | Card moved on board | Move kanban card |
| `trace_update` | New trace span | Update trace timeline |
| `project_update` | Project modified | Refresh project list |
| `project_removed` | Project deleted | Remove from project list |
| `lock_update` | Merge lock acquired | Show lock indicator |
| `lock_released` | Merge lock released | Clear lock indicator |

The dashboard shows a connection indicator (connected/disconnected) and auto-reconnects on SSE connection loss. These tests verify the SSE transport layer and UI reactivity independently of the features that generate events.

## Codebase Analysis

### SSE endpoint (`backend/internal/adapter/rest/`)

The `/events` endpoint uses Fiber's SSE support to maintain a long-lived connection. Events are published to an internal event bus, which fans out to all connected SSE clients. Each event has a `type` field and a JSON `data` payload.

### Event bus (`backend/internal/core/service/` or `adapter/rest/`)

The event bus is an in-memory pub/sub system. Services publish events (e.g., agent store publishes `agent_update` when an agent's status changes), and the SSE handler subscribes and forwards to connected clients.

### Frontend SSE client

The React app connects to `/events` using the `EventSource` API (or a custom SSE client). Event handlers dispatch to TanStack Query cache invalidation or direct state updates. A connection indicator component shows the current SSE connection state.

### Event format

```
event: agent_update
data: {"agent_id":"abc123","status":"running","role":"developer"}

event: quota_update
data: {"input_tokens":1500,"output_tokens":750,"total_cost":0.015}

event: lock_update
data: {"scope":"project-1","holder":"agent-abc","ttl_seconds":120}
```

## Acceptance Criteria

- [ ] SSE connection established on dashboard load — connection indicator shows connected
- [ ] `agent_update` event triggers agent list refresh in UI
- [ ] `agent_removed` event removes agent from list
- [ ] `quota_update` event updates stat cards
- [ ] `track_update` event updates track list
- [ ] `board_update` event moves kanban card
- [ ] `project_update` event refreshes project list
- [ ] `lock_update` / `lock_released` events update lock display
- [ ] Connection indicator shows disconnected on SSE drop
- [ ] Auto-reconnection after SSE connection loss
- [ ] Edge cases: rapid event burst, malformed events
- [ ] Failure cases: SSE connection refused, server restart during connection

## Dependencies

- `e2e-infra-mock-agent_20260309194830Z` — provides Playwright config, test server helpers, and seed data utilities

## Blockers

None.

## Conflict Risk

- LOW — adds new E2E test files only, no production code changes.

## Out of Scope

- The specific features that trigger events (agent lifecycle, kanban moves, etc.) — this track tests SSE transport and UI reactivity only
- WebSocket protocol for interactive agents (covered in `e2e-interactive-terminal_20260309194835Z`)
- Testing the event bus internals (unit test concern, not E2E)

## Technical Notes

### Test file organization

```
e2e/
  sse-realtime/
    connection_test.go     — SSE connects, indicator, headers
    agent_events_test.go   — agent_update, agent_removed, quota_update
    track_events_test.go   — track_update, board_update, track removal
    project_events_test.go — project_update, project_removed, lock events, trace events
    reconnect_test.go      — auto-reconnect, event burst, malformed events, server restart
```

### Triggering SSE events in tests

Tests should trigger events by performing backend actions (via API calls), then asserting the UI updates. For example:
1. Spawn an agent via API -> verify `agent_update` causes the agent to appear in the UI list
2. Delete an agent via API -> verify `agent_removed` causes the agent to disappear from the UI list
3. Update quota via a mock agent completing -> verify stat cards update

For direct event injection (testing transport without feature dependencies), the test helper can publish events directly to the event bus if the test server exposes it.

### Connection indicator testing

```typescript
// Verify connected state
await expect(page.locator('[data-testid="sse-indicator"]')).toHaveAttribute('data-status', 'connected');

// Simulate disconnect (stop SSE endpoint or kill server)
// Verify disconnected state
await expect(page.locator('[data-testid="sse-indicator"]')).toHaveAttribute('data-status', 'disconnected');
```

### Event burst testing

Send 50+ events in rapid succession (< 100ms apart), verify:
- All events are received (no drops)
- UI updates are batched/debounced (no flicker)
- Final UI state is correct

### Malformed event handling

Inject an event with invalid JSON data, verify:
- No JavaScript errors in console
- Subsequent valid events still processed
- Connection remains open

### Auto-reconnect testing

1. Establish SSE connection
2. Kill the test server's SSE endpoint (or restart server)
3. Verify connection indicator shows "disconnected"
4. Restore the endpoint
5. Verify auto-reconnect within timeout (typically 3-5 seconds)
6. Verify new events are received after reconnect

### Developer agent instructions

When building this track, use the Playwright MCP skill to verify E2E tests work in the browser. Run tests in headed mode during development for visual verification.

---

_Generated by conductor-track-generator for E2E SSE real-time tests_
