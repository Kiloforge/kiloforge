# Implementation Plan: Web UI Server with Real-Time Agent Monitoring

**Track ID:** impl-webui-server_20260308140000Z

## Phase 1: Dashboard Server Scaffold (3 tasks)

### Task 1.1: Create dashboard package and server
- [ ] Create `internal/dashboard/server.go`
- [ ] `New()` constructor accepting store, tracker, gitea URL, port
- [ ] `Run(ctx)` starts HTTP server, shuts down on context cancellation
- [ ] Register mux with placeholder routes

### Task 1.2: Create SSE hub
- [ ] Create `internal/dashboard/sse.go` with `SSEHub` struct
- [ ] `Broadcast(event)` — fan-out to all connected clients
- [ ] `Subscribe() / Unsubscribe()` — client management
- [ ] SSE handler: `GET /events` with proper headers (`text/event-stream`, no buffering)

### Task 1.3: Create state watcher
- [ ] Create `internal/dashboard/watcher.go`
- [ ] Poll state store and tracker on interval (2-3 seconds)
- [ ] Detect changes (agent status, quota updates, new/removed agents)
- [ ] Broadcast deltas via SSE hub

## Phase 2: REST API Endpoints (4 tasks)

### Task 2.1: Agent endpoints
- [ ] `GET /api/agents` — list all agents with status, role, track, PID, uptime
- [ ] `GET /api/agents/:id` — single agent detail
- [ ] JSON response format with consistent error handling

### Task 2.2: Log streaming endpoint
- [ ] `GET /api/agents/:id/log` — tail agent log file
- [ ] Support `?follow=true` for SSE-style log streaming
- [ ] Support `?lines=N` for last N lines
- [ ] Handle missing/unreadable log files gracefully

### Task 2.3: Quota and status endpoints
- [ ] `GET /api/quota` — aggregate usage, per-agent breakdown, rate limit status
- [ ] `GET /api/status` — overall system status (gitea health, relay health, agent counts)
- [ ] `GET /api/tracks` — parse tracks.md for progress summary

### Task 2.4: Write API handler tests
- [ ] Table-driven tests for each endpoint
- [ ] Test with empty state (no agents, no quota data)
- [ ] Test with various agent states (running, suspended, failed)
- [ ] Test error cases (invalid agent ID, missing log file)

## Phase 3: Embedded Frontend (4 tasks)

### Task 3.1: Create HTML dashboard layout
- [ ] Create `internal/dashboard/static/index.html`
- [ ] Header: crelay logo/name, Gitea link, system status indicator
- [ ] Main area: agent cards grid, quota bar, track progress
- [ ] Footer: connection status (SSE connected/disconnected)

### Task 3.2: Create CSS styling
- [ ] Create `internal/dashboard/static/style.css`
- [ ] Clean, minimal design with status colors (green/yellow/red)
- [ ] Agent cards with role icon, status badge, cost display
- [ ] Quota bar with threshold indicators
- [ ] Log viewer with monospace font and auto-scroll

### Task 3.3: Create JavaScript client
- [ ] Create `internal/dashboard/static/app.js`
- [ ] Fetch initial state from REST endpoints on load
- [ ] Connect to `/events` SSE and update DOM on events
- [ ] Log viewer: fetch + follow mode with auto-scroll
- [ ] Reconnect logic for SSE disconnections

### Task 3.4: Embed static files
- [ ] Use `go:embed static/*` directive
- [ ] Serve via `http.FileServer(http.FS(...))`
- [ ] `GET /` serves index.html
- [ ] Verify embedded assets work in built binary

## Phase 4: Verification (3 tasks)

### Task 4.1: Integration test
- [ ] Start dashboard with mock state and tracker
- [ ] Verify API responses match expected format
- [ ] Verify SSE events fire on state changes

### Task 4.2: Manual smoke test
- [ ] Build binary, start dashboard, open in browser
- [ ] Verify layout renders correctly
- [ ] Verify SSE updates in real-time

### Task 4.3: Full build and test
- [ ] `go build ./...`
- [ ] `go test -race ./...`
- [ ] Verify no regressions

---

**Total: 14 tasks across 4 phases**
