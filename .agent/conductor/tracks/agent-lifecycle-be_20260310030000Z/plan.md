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
- Look up active agent from registry
- Call `session.Close()` to terminate subprocess
- Update status to `"stopped"`, set `ShutdownReason: "user_stopped"`, set `FinishedAt`
- Remove from active registry
- Return error if agent not found or not running

### Task 2.2: Add ResumeAgent method to Spawner
- Accept agent ID; look up from store
- Verify status is stopped/completed/failed (not running)
- Create new SDKSession with `opts.WithResume(agent.SessionID)`
- Connect, create bridge, start relay, start monitor
- Update status to `"running"`, clear `FinishedAt`
- Register in active agents map
- Return new `InteractiveAgent`

### Task 2.3: Write tests for StopAgent
- Test stop running agent (MockSession)
- Test stop already-stopped agent returns error
- Test stop non-existent agent returns error

### Task 2.4: Write tests for ResumeAgent
- Test resume stopped agent
- Test resume already-running agent returns error
- Test resume non-existent agent returns error

### Task 2.5: Verify Phase 2
- All new and existing tests pass

## Phase 3: OpenAPI Schema & REST Endpoints

### Task 3.1: Add endpoints to OpenAPI spec
- `POST /api/agents/{id}/stop` — 200 returns Agent, 404 not found, 409 not running
- `POST /api/agents/{id}/resume` — 200 returns Agent with ws_url, 404 not found, 409 already running
- `DELETE /api/agents/{id}` — 204 no content, 404 not found, 409 still running
- Run `make gen-api` to regenerate Go code

### Task 3.2: Implement StopAgent handler
- Call `spawner.StopAgent(id)`
- Call `wsSessions.UnregisterBridge(id)`
- Broadcast SSE agent_update event
- Return updated agent

### Task 3.3: Implement ResumeAgent handler
- Call `spawner.ResumeAgent(ctx, id)`
- Create new bridge, register with WS session manager
- Start structured relay
- Broadcast SSE agent_update event
- Return updated agent with ws_url

### Task 3.4: Implement DeleteAgent handler
- Verify agent is not running (check active registry)
- Call `store.RemoveAgent(id)`
- Delete log file if exists
- Broadcast SSE agent_removed event
- Return 204

### Task 3.5: Verify Phase 3
- `make gen-api` succeeds
- `go build ./...`
- All tests pass

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
