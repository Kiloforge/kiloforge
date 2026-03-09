# Implementation Plan: Agent Lifecycle Management — Stop, Resume, Delete (Backend)

**Track ID:** agent-lifecycle-be_20260310030000Z

## Phase 1: Store & Interface Extensions

### Task 1.1: Add RemoveAgent to AgentStore interface
- [x] Add `RemoveAgent(id string) error` to `port.AgentStore`
- [x] Update all interface implementations

### Task 1.2: Implement RemoveAgent in SQLite store
- [x] Add `DELETE FROM agents WHERE id = ?` implementation
- [x] Return error if agent not found

### Task 1.3: Add active session registry to Spawner
- [x] Add `activeAgents map[string]*InteractiveAgent` field with mutex
- [x] Register in `SpawnInteractive()` after successful spawn
- [x] Deregister in `monitorSDKSession()` on completion
- [x] Add `GetActiveAgent(id string) (*InteractiveAgent, bool)` method

### Task 1.4: Fix UnregisterBridge memory leak
- [x] Call `SessionEndCallback` in `monitorSDKSession()` after session ends
- [x] Add `SetSessionEndCallback` to spawner for bridge cleanup

### Task 1.5: Verify Phase 1
- [x] `go build ./...`
- [x] Existing tests pass

## Phase 2: Spawner Stop & Resume Methods

### Task 2.1: Add StopAgent method to Spawner
- [x] Look up active agent from registry
- [x] Call `session.Close()` to terminate subprocess
- [x] Update status to `"stopped"`, set `ShutdownReason: "user_stopped"`, set `FinishedAt`
- [x] Remove from active registry
- [x] Return error if agent not found or not running

### Task 2.2: Add ResumeAgent method to Spawner
- [x] Accept agent ID; look up from store
- [x] Verify status is stopped/completed/failed (not running)
- [x] Create new SDKSession with `opts.WithResume(agent.SessionID)`
- [x] Connect, create bridge, start relay, start monitor
- [x] Update status to `"running"`, clear `FinishedAt`
- [x] Register in active agents map
- [x] Return new `InteractiveAgent`

### Task 2.3: Write tests for StopAgent
- [x] Test stop running agent (MockSession)
- [x] Test stop already-stopped agent returns error
- [x] Test stop non-existent agent returns error

### Task 2.4: Write tests for ResumeAgent
- [x] Test resume already-running agent returns error
- [x] Test resume non-existent agent returns error

### Task 2.5: Verify Phase 2
- [x] All new and existing tests pass

## Phase 3: OpenAPI Schema & REST Endpoints

### Task 3.1: Add endpoints to OpenAPI spec
- [x] `POST /api/agents/{id}/stop` — 200 returns Agent, 404 not found, 409 not running
- [x] `POST /api/agents/{id}/resume` — 200 returns Agent with ws_url, 404 not found, 409 already running
- [x] `DELETE /api/agents/{id}` — 204 no content, 404 not found, 409 still running
- [x] Run oapi-codegen to regenerate Go code

### Task 3.2: Implement StopAgent handler
- [x] Call `spawner.StopAgent(id)`
- [x] Call `wsSessions.UnregisterBridge(id)`
- [x] Return updated agent

### Task 3.3: Implement ResumeAgent handler
- [x] Call `spawner.ResumeAgent(ctx, id)`
- [x] Create new bridge, register with WS session manager
- [x] Start structured relay
- [x] Return updated agent with ws_url

### Task 3.4: Implement DeleteAgent handler
- [x] Verify agent is not running (check active registry)
- [x] Call `store.RemoveAgent(id)`
- [x] Delete log file if exists
- [x] Return 204

### Task 3.5: Verify Phase 3
- [x] oapi-codegen succeeds
- [x] `go build ./...`
- [x] All tests pass

## Phase 4: Integration & Polish

### Task 4.1: Wire UnregisterBridge callback
- Ensure spawner has access to WS session manager for bridge cleanup
- Pass as callback or direct reference in constructor

### Task 4.2: End-to-end test with MockSession
- Spawn interactive agent with mock → verify running
- Stop agent → verify stopped status, bridge removed
- Resume agent → verify running again, new bridge registered
- Stop again → delete → verify removed from store

### Task 4.3: Verify Phase 4
- Full test suite passes
- `go vet ./...`
- `make build`
