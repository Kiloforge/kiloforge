# Implementation Plan: Fix Board UX — SSE Handler, Sync Button, Empty Columns (Frontend)

**Track ID:** fix-board-sync-ux-fe_20260310012001Z

## Phase 1: SSE and Data Layer

- [x] Task 1.1: Add `board_update` SSE handler in `App.tsx` — invalidate board query cache
- [x] Task 1.2: Add `syncBoard` mutation to `useBoard.ts` — calls `POST /api/board/{project}/sync`
- [x] Task 1.3: Export `handleBoardUpdate` from `useBoard.ts` for SSE handler wiring

## Phase 2: Board UI

- [x] Task 2.1: Always render `KanbanBoard` in `ProjectPage.tsx` — remove the empty-cards conditional
- [x] Task 2.2: Ensure `KanbanBoard` renders all 5 columns with human-readable labels even when empty
- [x] Task 2.3: Add "Sync" button in the board section header, wired to `syncBoard` mutation
- [x] Task 2.4: Show loading/disabled state on sync button during mutation

## Phase 3: Verification

- [x] Task 3.1: Frontend builds without errors (`npm run build`)
- [x] Task 3.2: Board shows 5 empty columns on a project with no tracks
- [x] Task 3.3: Board populates with cards after sync on a project with tracks
- [x] Task 3.4: Board updates in real-time when tracks change (via SSE)
