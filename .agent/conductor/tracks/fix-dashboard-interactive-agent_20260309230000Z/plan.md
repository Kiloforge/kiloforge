# Implementation Plan: Fix Interactive Agent Wiring in Dashboard Command

**Track ID:** fix-dashboard-interactive-agent_20260309230000Z

## Phase 1: Wire Interactive Agent Dependencies

- [x] Task 1.1: Add missing imports to `dashboard.go` — `wsAdapter`, `agent` spawner, event bus, git sync, trace store, board service, consent, project manager
- [x] Task 1.2: Create `agent.Spawner` with config needed for interactive agents
- [x] Task 1.3: Create `wsAdapter.SessionManager`
- [x] Task 1.4: Create event bus, git sync adapter, trace store, board service, consent manager, project manager instances
- [x] Task 1.5: Wire all missing fields into `APIHandlerOpts`

## Phase 2: WebSocket Handler

- [x] Task 2.1: Register WebSocket handler on dashboard mux for `/ws` endpoint (mirror `server.go` pattern)

## Phase 3: Verification

- [x] Task 3.1: `make test` passes
- [x] Task 3.2: `kf dashboard` starts without errors
- [x] Task 3.3: Interactive agent spawn from dashboard UI works (no 500)
- [x] Task 3.4: Track generation from dashboard works (no 500)
