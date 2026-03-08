# Implementation Plan: Agent List with Role/Track Links and Monitoring View

**Track ID:** agent-list-monitoring-ui_20260309180000Z

## Phase 1: Agent Detail Page

- [ ] Task 1.1: Add `/agents/:id` route in `App.tsx`
- [ ] Task 1.2: Create `AgentDetailPage` component — fetch agent via `GET /api/agents/{id}`, display metadata (role, ref, status, model, uptime, PID, worktree, tokens, cost)
- [ ] Task 1.3: Embed `LogViewer` in detail page (always visible, not modal) with follow mode
- [ ] Task 1.4: Embed `AgentTerminal` in detail page for interactive agents (conditionally rendered based on role)

## Phase 2: Enhance Agent Grid

- [ ] Task 2.1: Make agent card clickable — link agent ID or entire card to `/agents/:id`
- [ ] Task 2.2: Make ref field a secondary link — clicking ref navigates to agent detail page
- [ ] Task 2.3: Add "View on Board" link in agent card when ref matches a track ID — link to `/projects/:slug` with the track highlighted
- [ ] Task 2.4: Add role/status filter chips above the agent grid in OverviewPage — client-side filtering on fetched agent list

## Phase 3: Verification

- [ ] Task 3.1: Verify frontend builds without errors (`npm run build`)
- [ ] Task 3.2: Manual verification — navigate agent grid → detail page → log → terminal → back to board
