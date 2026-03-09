# Specification: Agent Lifecycle Management — Stop, Resume, Delete (Backend)

**Track ID:** agent-lifecycle-be_20260310030000Z
**Type:** Feature
**Created:** 2026-03-10T03:00:00Z
**Status:** Draft

## Summary

Add REST API endpoints and backend service logic for stopping running agents, resuming stopped sessions via the SDK's `WithResume(sessionID)`, and deleting agent records from the store. Also fix the `UnregisterBridge` memory leak where bridges are never cleaned up after agent completion.

## Context

Currently there is no way to terminate a running agent from the dashboard, resume a stopped agent session, or clean up completed/failed agent records. The only stop mechanism is `HaltAgent()` which sends SIGINT via PID — but SDK-based agents need `SDKSession.Close()` instead. The SDK supports session resume via `WithResume(sessionID)` which passes `--resume <sessionID>` to the Claude CLI. Agent records persist indefinitely in SQLite with no delete capability. Additionally, `UnregisterBridge()` is defined but never called, causing a memory leak for completed agent bridges.

## Codebase Analysis

- **`port.AgentStore` interface** (`internal/core/port/agent_store.go`): Has `HaltAgent(idPrefix)`, `UpdateStatus()`, but no `RemoveAgent()` or `DeleteAgent()` method
- **SQLite agent store** (`internal/adapter/persistence/sqlite/agent_store.go`): `HaltAgent()` sends SIGINT via PID; no delete; terminal statuses: stopped, completed, failed, force-killed, resume-failed
- **`SDKSession`** (`internal/adapter/agent/sdk_client.go`): `Close()` cancels context, calls `client.Close()` (which terminates subprocess), closes channels
- **`Spawner`** (`internal/adapter/agent/spawner.go`): `InteractiveAgent` has `session AgentSession` field; `monitorSDKSession()` waits for `Done` but never calls `UnregisterBridge()`; no stop or resume methods
- **SDK** (`github.com/schlunsen/claude-agent-sdk-go` v0.5.1): `types.ClaudeAgentOptions.WithResume(sessionID)` passes `--resume <sessionID>` to CLI subprocess; `Client.Close()` terminates subprocess gracefully
- **WS session manager** (`internal/adapter/ws/session.go`): `UnregisterBridge()` defined but never called — bridges persist in memory after agent exits
- **OpenAPI spec** (`api/openapi.yaml`): Only has `GET /api/agents`, `POST /api/agents/interactive`, `GET /api/agents/{id}`, `GET /api/agents/{id}/log` — no stop/delete/resume endpoints
- **`AgentInfo.SessionID`** stored in SQLite — used for resume; `FinishedAt` field for terminal status tracking

## Acceptance Criteria

- [ ] `POST /api/agents/{id}/stop` terminates a running agent (SDK Close + WS bridge cleanup), returns updated agent
- [ ] `POST /api/agents/{id}/resume` re-spawns a stopped/completed agent using `WithResume(sessionID)`, returns updated agent with new WS URL
- [ ] `DELETE /api/agents/{id}` removes agent record from SQLite store permanently
- [ ] `AgentStore` interface extended with `RemoveAgent(id string) error`
- [ ] Spawner gains `StopAgent(agentID)` that closes the SDK session and cleans up WS bridge
- [ ] Spawner gains `ResumeAgent()` that creates a new SDK session with `WithResume(sessionID)`
- [ ] `UnregisterBridge()` called in `monitorSDKSession()` on agent completion (fixes memory leak)
- [ ] Running interactive agent sessions tracked by ID in spawner for stop access
- [ ] Stop returns 409 if agent is not running; Resume returns 409 if agent is already running
- [ ] Delete returns 409 if agent is still running (must stop first)
- [ ] All new endpoints documented in OpenAPI spec with generated code
- [ ] Tests for stop, resume, and delete flows using MockSession

## Dependencies

None

## Out of Scope

- Frontend UI (separate track: `agent-lifecycle-fe_20260310030001Z`)
- Bulk stop/delete operations
- Auto-cleanup of old agents (TTL-based garbage collection)
- Agent process recovery on server restart (existing `LifecycleService` handles this separately)

## Technical Notes

### Stop Flow
1. Look up active `InteractiveAgent` by ID in a new spawner registry (map)
2. Call `session.Close()` → cancels context, terminates subprocess, closes channels
3. Call `wsSessions.UnregisterBridge(agentID)` to clean up WS bridge
4. Update agent status to `"stopped"` with `ShutdownReason: "user_stopped"`
5. SSE event for agent_update

### Resume Flow
1. Look up agent info from store; verify status is stopped/completed/failed
2. Create new `SDKSession` with `opts.WithResume(agent.SessionID)`
3. Connect and register new bridge, start structured relay
4. Update status to `"running"`, clear `FinishedAt`
5. Return updated agent with WS URL for frontend to attach

### Delete Flow
1. Verify agent is not running (status not "running"/"waiting")
2. Remove from SQLite via new `RemoveAgent(id)` method
3. Optionally delete log file
4. SSE event for agent_removed

### Session Registry
The spawner currently loses track of `InteractiveAgent` after returning it from `SpawnInteractive()`. Add `activeAgents map[string]*InteractiveAgent` to `Spawner` so we can look up sessions for stop. Remove entries in `monitorSDKSession()` when agent completes.

---

_Generated by kf-architect from prompt: "Add agent lifecycle management: stop/kill, resume, and delete capabilities"_
