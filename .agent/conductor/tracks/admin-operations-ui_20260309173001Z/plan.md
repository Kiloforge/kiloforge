# Implementation Plan: Admin Operations UI

**Track ID:** admin-operations-ui_20260309173001Z

## Phase 1: Backend — Admin Run Endpoint

- [ ] Task 1.1: Add `POST /api/admin/run` to OpenAPI spec — request schema with `operation` enum and optional `project`, response with `agent_id` and `ws_url`
- [ ] Task 1.2: Implement `RunAdminOperation` handler — map operation to skill prompt, spawn interactive agent, create WebSocket bridge, return agent info
- [ ] Task 1.3: Add board auto-sync on completion for archive operations (follow `GenerateTracks` pattern)
- [ ] Task 1.4: Add concurrency guard — reject if another admin operation is already running (412 response)

## Phase 2: Frontend — Admin Panel Component

- [ ] Task 2.1: Create `AdminPanel` component — row of operation buttons (Bulk Archive, Compact Archive, Generate Report)
- [ ] Task 2.2: Wire button click → `POST /api/admin/run` → open AgentTerminal with WebSocket URL
- [ ] Task 2.3: Add disabled state while operation is running, success/error feedback on completion
- [ ] Task 2.4: Integrate AdminPanel into ProjectPage below the board section
- [ ] Task 2.5: Auto-refresh board after archive operations complete

## Phase 3: Verification

- [ ] Task 3.1: Verify `go test ./...` passes
- [ ] Task 3.2: Verify frontend builds without errors
- [ ] Task 3.3: Manual verification — trigger each operation from UI, confirm terminal output and board refresh
