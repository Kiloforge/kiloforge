# Implementation Plan: E2E Tests — Real-Time SSE Updates: Connection, Events, and Reconnection

**Track ID:** e2e-sse-realtime_20260309194837Z

## Phase 1: Connection Tests

- [ ] Task 1.1: SSE connects on dashboard load — navigate to dashboard, verify the SSE connection is established (check connection indicator shows "connected")
- [ ] Task 1.2: Connection indicator UI — verify the connection indicator component renders with correct visual state (green/connected), and verify it changes to disconnected state when SSE drops
- [ ] Task 1.3: Connection headers — verify the SSE request includes correct headers (`Accept: text/event-stream`) and the response has correct content type

## Phase 2: Agent Event Tests

- [ ] Task 2.1: `agent_update` UI reaction — spawn an agent via API, verify the agent list in the dashboard updates to show the new agent without a page refresh
- [ ] Task 2.2: `agent_removed` UI reaction — delete an agent via API, verify the agent disappears from the dashboard agent list without a page refresh
- [ ] Task 2.3: `quota_update` stat cards — trigger a quota change (mock agent completes with token usage), verify the stat cards on the dashboard update with new token/cost values

## Phase 3: Track and Board Event Tests

- [ ] Task 3.1: `track_update` list refresh — change a track's status via API, verify the track list in the UI updates to reflect the new status without page refresh
- [ ] Task 3.2: `board_update` card move — move a kanban card via API, verify the board UI moves the card to the new column without page refresh
- [ ] Task 3.3: Track removal — remove a track via API, verify the track disappears from the track list UI without page refresh

## Phase 4: Project and Lock Event Tests

- [ ] Task 4.1: `project_update` and `project_removed` — update a project via API, verify project list refreshes; delete a project, verify it disappears from the list
- [ ] Task 4.2: `lock_update` and `lock_released` — acquire a merge lock via API, verify lock indicator appears in the UI; release the lock, verify the indicator clears
- [ ] Task 4.3: `trace_update` — trigger a trace event, verify the trace timeline or trace indicator updates in the UI

## Phase 5: Reconnection and Failure Tests

- [ ] Task 5.1: Auto-reconnect after disconnect — establish SSE connection, simulate server-side disconnect, verify connection indicator shows "disconnected", then verify auto-reconnect restores the connection and new events are received
- [ ] Task 5.2: Event burst handling — send 50+ events in rapid succession via API actions, verify all events are processed, UI reaches correct final state, and no events are dropped
- [ ] Task 5.3: Malformed event handling — inject an event with invalid JSON data (via test helper), verify no JavaScript console errors, connection remains open, and subsequent valid events are still processed correctly
- [ ] Task 5.4: Server restart during connection — restart the test server while SSE is connected, verify disconnection is detected, auto-reconnect occurs after server is back, and dashboard state is refreshed
