# Implementation Plan: Interactive Agent Sessions via WebSocket (Backend)

**Track ID:** interactive-agent-be_20260309150000Z

## Phase 1: WebSocket Infrastructure

- [x] Task 1.1: Add `nhooyr.io/websocket` to `go.mod`
- [x] Task 1.2: Create `backend/internal/adapter/ws/handler.go` — WebSocket upgrade handler, message types, read/write loops
- [x] Task 1.3: Create `backend/internal/adapter/ws/session.go` — manages active WebSocket sessions per agent, handles reconnection and output buffering (ring buffer)
- [x] Task 1.4: Add tests for WebSocket handler and session management

## Phase 2: Interactive Spawner

- [x] Task 2.1: Add `SpawnInteractive()` to `agent/spawner.go` — creates agent with `cmd.StdinPipe()` + `cmd.StdoutPipe()`, returns `InteractiveAgent` with IO handles
- [x] Task 2.2: Add `InteractiveAgent` struct — wraps agent ID, stdin writer, stdout reader, command handle
- [x] Task 2.3: Create output parser goroutine — reads stream-json lines, extracts text content for UI, forwards raw lines to quota tracker and log file
- [x] Task 2.4: Create input relay goroutine — reads from a channel (fed by WebSocket), writes to agent stdin
- [x] Task 2.5: Add tests for interactive spawn and IO relay

## Phase 3: WebSocket-Agent Bridge

- [x] Task 3.1: Create `backend/internal/adapter/ws/bridge.go` — connects WebSocket session to InteractiveAgent IO (stdin channel ← WebSocket, stdout → WebSocket)
- [x] Task 3.2: Handle agent lifecycle events — send status messages on agent start/pause/complete/error
- [x] Task 3.3: Implement output ring buffer — stores last 500 lines for reconnecting clients
- [x] Task 3.4: Handle multiple observers — first client is read-write, additional clients are read-only

## Phase 4: Server Wiring

- [x] Task 4.1: Add `GET /ws/agent/{id}` route to `rest/server.go`
- [x] Task 4.2: Add `POST /api/agents/interactive` endpoint to spawn an interactive agent (OpenAPI spec + handler)
- [x] Task 4.3: Wire WebSocket handler and interactive spawner into server startup
- [x] Task 4.4: Update agent store to track interactive vs non-interactive agents

## Phase 5: Verification

- [x] Task 5.1: Verify `go test ./...` passes
- [x] Task 5.2: Verify `make build` succeeds
- [x] Task 5.3: Manual test: spawn interactive agent, send input via WebSocket client (wscat or similar), receive output
