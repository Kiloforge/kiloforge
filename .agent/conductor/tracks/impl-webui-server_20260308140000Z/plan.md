# Implementation Plan: Web UI Server with Real-Time Agent Monitoring

**Track ID:** impl-webui-server_20260308140000Z

## Phase 1: Dashboard Server Scaffold (3 tasks)

### Task 1.1: Create dashboard package and server
- [x] Create `internal/dashboard/server.go`
- [x] `New()` constructor accepting store, tracker, gitea URL, port
- [x] `Run(ctx)` starts HTTP server, shuts down on context cancellation
- [x] Register mux with placeholder routes

### Task 1.2: Create SSE hub
- [x] Create `internal/dashboard/sse.go` with `SSEHub` struct
- [x] `Broadcast(event)` — fan-out to all connected clients
- [x] `Subscribe() / Unsubscribe()` — client management
- [x] SSE handler: `GET /events` with proper headers (`text/event-stream`, no buffering)

### Task 1.3: Create state watcher
- [x] Create `internal/dashboard/watcher.go`
- [x] Poll state store and tracker on interval (2-3 seconds)
- [x] Detect changes (agent status, quota updates, new/removed agents)
- [x] Broadcast deltas via SSE hub

## Phase 2: REST API Endpoints (4 tasks)

### Task 2.1: Agent endpoints
- [x] `GET /api/agents` — list all agents with status, role, track, PID, uptime
- [x] `GET /api/agents/:id` — single agent detail
- [x] JSON response format with consistent error handling

### Task 2.2: Log streaming endpoint
- [x] `GET /api/agents/:id/log` — tail agent log file
- [x] Support `?follow=true` for SSE-style log streaming
- [x] Support `?lines=N` for last N lines
- [x] Handle missing/unreadable log files gracefully

### Task 2.3: Quota and status endpoints
- [x] `GET /api/quota` — aggregate usage, per-agent breakdown, rate limit status
- [x] `GET /api/status` — overall system status (gitea health, relay health, agent counts)
- [x] `GET /api/tracks` — parse tracks.md for progress summary

### Task 2.4: Write API handler tests
- [x] Table-driven tests for each endpoint
- [x] Test with empty state (no agents, no quota data)
- [x] Test with various agent states (running, suspended, failed)
- [x] Test error cases (invalid agent ID, missing log file)

## Phase 3: Embedded Frontend (4 tasks)

### Task 3.1: Create HTML dashboard layout
- [x] Create `internal/dashboard/static/index.html`
- [x] Header: crelay logo/name, Gitea link, system status indicator
- [x] Main area: agent cards grid, quota bar, track progress
- [x] Footer: connection status (SSE connected/disconnected)

### Task 3.2: Create CSS styling
- [x] Create `internal/dashboard/static/style.css`
- [x] Clean, minimal design with status colors (green/yellow/red)
- [x] Agent cards with role icon, status badge, cost display
- [x] Quota bar with threshold indicators
- [x] Log viewer with monospace font and auto-scroll

### Task 3.3: Create JavaScript client
- [x] Create `internal/dashboard/static/app.js`
- [x] Fetch initial state from REST endpoints on load
- [x] Connect to `/events` SSE and update DOM on events
- [x] Log viewer: fetch + follow mode with auto-scroll
- [x] Reconnect logic for SSE disconnections

### Task 3.4: Embed static files
- [x] Use `go:embed static/*` directive
- [x] Serve via `http.FileServer(http.FS(...))`
- [x] `GET /` serves index.html
- [x] Verify embedded assets work in built binary

## Phase 4: Verification (3 tasks)

### Task 4.1: Integration test
- [x] Start dashboard with mock state and tracker
- [x] Verify API responses match expected format
- [x] Verify SSE events fire on state changes

### Task 4.2: Manual smoke test
- [x] SKIPPED — requires browser interaction, not automatable in CI

### Task 4.3: Full build and test
- [x] `go build ./...`
- [x] `go test -race ./...`
- [x] Verify no regressions

---

**Total: 14 tasks across 4 phases — ALL COMPLETE (1 skipped as not automatable)**
