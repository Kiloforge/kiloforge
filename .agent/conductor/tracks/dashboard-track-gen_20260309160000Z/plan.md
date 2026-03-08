# Implementation Plan: Dashboard-Driven Track Generation with Interactive Agent

**Track ID:** dashboard-track-gen_20260309160000Z

## Phase 1: Track Generation API

- [x] Task 1.1: Add `POST /api/tracks/generate` to OpenAPI spec — accepts prompt text and optional project slug, returns agent ID and WebSocket URL
- [x] Task 1.2: Implement handler — spawns interactive track-generator agent (`claude -p "/kf-track-generator <prompt>"`) in a track-generator worktree
- [x] Task 1.3: Add `DELETE /api/tracks/{trackId}` to OpenAPI spec — removes track artifacts and board card
- [x] Task 1.4: Implement delete handler — removes track directory from conductor artifacts, updates tracks.md and index.md, removes board card
- [x] Task 1.5: Add auto-sync trigger — after track-generator agent completes, automatically run board sync to surface new tracks

## Phase 2: Board Approval Actions

- [x] Task 2.1: Add "Approve" action to board cards in `backlog` column — calls move endpoint with column=approved
- [x] Task 2.2: Add "Reject" action to board cards in `backlog` — calls DELETE /api/tracks/{trackId}, shows confirmation dialog
- [x] Task 2.3: Style backlog cards with approve/reject action buttons (green checkmark, red X)
- [x] Task 2.4: Add visual distinction for backlog cards (pending human review indicator)

## Phase 3: Track Generation UI Flow

- [x] Task 3.1: Add "Generate Tracks" button to project page (or global nav action)
- [x] Task 3.2: Create prompt input form — text area with description, optional context fields, "Generate" submit button
- [x] Task 3.3: On submit, call `POST /api/tracks/generate`, open chat terminal with returned agent WebSocket URL
- [x] Task 3.4: Show agent progress in chat terminal (reuse AgentTerminal component from interactive-agent-fe)
- [x] Task 3.5: On agent completion, auto-refresh board to show new backlog cards
- [x] Task 3.6: Show success summary — list of generated tracks with links to board

## Phase 4: Verification

- [x] Task 4.1: Verify `go test ./...` passes
- [x] Task 4.2: Verify `npm run build` succeeds
- [x] Task 4.3: End-to-end: enter prompt → agent generates tracks → tracks appear in backlog → approve → moves to approved
- [x] Task 4.4: End-to-end: reject a track → artifacts deleted, card removed
