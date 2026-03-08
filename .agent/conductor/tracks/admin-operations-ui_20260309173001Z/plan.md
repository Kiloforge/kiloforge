# Implementation Plan: Admin Operations UI

**Track ID:** admin-operations-ui_20260309173001Z

## Phase 1: Backend — Admin Run Endpoint

- [x] Task 1.1: Add `POST /api/admin/run` to OpenAPI spec — request schema with `operation` enum and optional `project`, response with `agent_id` and `ws_url`
- [x] Task 1.2: Implement `RunAdminOperation` handler — map operation to skill prompt, spawn interactive agent, create WebSocket bridge, return agent info
- [x] Task 1.3: Add board auto-sync on completion for archive operations (follow `GenerateTracks` pattern)
- [x] Task 1.4: Add concurrency guard — reject if another admin operation is already running (412 response)

## Phase 2: Frontend — Admin Panel Component

- [x] Task 2.1: Create `AdminPanel` component — row of operation buttons (Bulk Archive, Compact Archive, Generate Report)
- [x] Task 2.2: Wire button click → `POST /api/admin/run` → open AgentTerminal with WebSocket URL
- [x] Task 2.3: Add disabled state while operation is running, success/error feedback on completion
- [x] Task 2.4: Integrate AdminPanel into ProjectPage below the board section
- [x] Task 2.5: Auto-refresh board after archive operations complete

## Phase 3: Verification

- [x] Task 3.1: Verify `go test ./...` passes
- [x] Task 3.2: Verify frontend builds without errors
- [x] Task 3.3: Manual verification — trigger each operation from UI, confirm terminal output and board refresh
