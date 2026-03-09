# Implementation Plan: Fix Board Sync and SSE Events (Backend)

**Track ID:** fix-board-sync-ux-be_20260310012000Z

## Phase 1: Auto-Sync on Empty Board

- [x] Task 1.1: In `GetBoard` handler, check if board has zero cards after fetch
- [x] Task 1.2: If empty, discover tracks from project directory and call `SyncFromTracks()`
- [x] Task 1.3: Re-read board after sync and return the populated result
- [x] Task 1.4: Publish `board_update` SSE event after auto-sync

## Phase 2: Consistent SSE Events

- [x] Task 2.1: Verify `POST /api/board/{project}/sync` publishes `board_update` SSE event
- [x] Task 2.2: Verify `MoveCard` publishes `board_update` SSE event
- [x] Task 2.3: Verify `DeleteCard` (if exists) publishes `board_update` SSE event
- [x] Task 2.4: Add `board_update` SSE event to any board mutation that's missing it

## Phase 3: Verification

- [x] Task 3.1: `make test` passes
- [x] Task 3.2: `GET /api/board/{project}` returns populated board on first load when tracks exist
