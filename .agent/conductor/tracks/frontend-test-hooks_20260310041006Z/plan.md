# Implementation Plan: Frontend Test Coverage — Critical Hooks

**Track ID:** frontend-test-hooks_20260310041006Z

## Phase 1: WebSocket and SSE Hook Tests

### Task 1.1: Create test helpers for WebSocket and SSE mocks [x]
- Create `src/test/mocks/websocket.ts` with MockWebSocket class
- Create `src/test/mocks/eventsource.ts` with MockEventSource class
- Create `src/test/helpers.tsx` with QueryClient wrapper for `renderHook()`

### Task 1.2: Test useAgentWebSocket [x]
- Create `src/hooks/useAgentWebSocket.test.ts`
- Test: connects to correct URL with agent ID
- Test: parses incoming messages and updates state
- Test: reconnects with exponential backoff on close (non-1000 code)
- Test: does not reconnect on clean close (code 1000)
- Test: cleans up WebSocket and timeout on unmount
- Test: handles malformed messages gracefully

### Task 1.3: Test useSSE [x]
- Create `src/hooks/useSSE.test.ts`
- Test: creates EventSource with correct URL
- Test: dispatches events to registered handlers
- Test: reconnects on error
- Test: cleans up EventSource on unmount
- Test: handles connection state transitions

### Task 1.4: Verify Phase 1 [x]
- `npm test` passes, all new tests green

## Phase 2: Data Mutation Hook Tests

### Task 2.1: Test useProjects [x]
- Create `src/hooks/useProjects.test.ts`
- Test: fetches project list on mount
- Test: addProject mutation sends correct request, invalidates cache
- Test: removeProject mutation sends correct request, invalidates cache
- Test: SSE event handler updates query data
- Test: error handling shows toast

### Task 2.2: Test useBoard [x]
- Create `src/hooks/useBoard.test.ts`
- Test: fetches board state for project
- Test: moveCard optimistic update applies immediately
- Test: moveCard rollback on server error
- Test: syncBoard mutation triggers refetch

### Task 2.3: Test useOriginSync [x]
- Create `src/hooks/useOriginSync.test.ts`
- Test: fetchSyncStatus returns ahead/behind counts
- Test: push mutation sends correct request
- Test: pull mutation sends correct request
- Test: error handling for push/pull failures

### Task 2.4: Verify Phase 2 [x]
- `npm test` passes, all new tests green
- `make test-frontend` passes
