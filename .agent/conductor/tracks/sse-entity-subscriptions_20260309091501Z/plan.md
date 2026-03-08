# Implementation Plan: SSE Entity Subscriptions

**Track ID:** sse-entity-subscriptions_20260309091501Z

## Phase 1: Backend — Watcher-Driven Events (Tracks & Traces)

- [x] Task 1.1: Extend `watcherState` struct with `tracks map[string]string` (id→status) and `traceCount int`
- [x] Task 1.2: Add track delta detection to `checkAndBroadcast` — discover tracks via `service.DiscoverTracks()`, compare with previous state, emit `track_update` for new/changed tracks and `track_removed` for deleted tracks
- [x] Task 1.3: Inject `ProjectLister` into the watcher so it can iterate project dirs for track discovery
- [x] Task 1.4: Add trace delta detection — compare `traceStore.ListTraces()` count/IDs with previous state, emit `trace_update` for new traces

## Phase 2: Backend — Mutation-Driven Events (Board, Projects, Locks)

- [x] Task 2.1: Emit `board_update` event from `MoveCard` and `SyncBoard` REST handlers after successful operations
- [x] Task 2.2: Emit `lock_update` event from `AcquireLock` and `HeartbeatLock` handlers; emit `lock_released` from `ReleaseLock` handler
- [x] Task 2.3: Emit `project_update` event from project add handler (if exists, or prepare hook for project-manage-api track)
- [x] Task 2.4: Unit tests — verify event bus receives correct event types when mutations execute

## Phase 3: Frontend — SSE Event Handlers

- [x] Task 3.1: Update `useTracks` hook — add SSE handler for `track_update` (upsert) and `track_removed` (filter), remove `setInterval` polling
- [x] Task 3.2: Update `useBoard` hook — add SSE handler for `board_update` (replace board state), remove `setInterval` polling
- [x] Task 3.3: Update `useTraces` hook — add SSE handler for `trace_update` (prepend new trace), remove `setInterval` polling
- [x] Task 3.4: Update `useProjects` hook — add SSE handler for `project_update` (upsert) and `project_removed` (filter)
- [x] Task 3.5: Update `App.tsx` — register new SSE event handlers in `sseHandlers` memo, pass handlers to hooks or lift state

## Phase 4: Frontend — Hook Refactor for SSE Integration

- [x] Task 4.1: Refactor hooks to accept SSE handler callbacks (same pattern as `useAgents` — return handler functions, wire in App.tsx)
- [x] Task 4.2: Add optional long-interval background sync (5min) as drift protection fallback for tracks and board
- [x] Task 4.3: Update TypeScript types in `types/api.ts` if new event payload shapes are needed

## Phase 5: Verification

- [x] Task 5.1: Verify `cd frontend && npm run build` succeeds
- [x] Task 5.2: Verify `make build` succeeds (full build with embed)
- [x] Task 5.3: Verify `go test ./...` passes
- [x] Task 5.4: Manual verification — open dashboard, trigger track/board/project changes, confirm real-time updates without page refresh
