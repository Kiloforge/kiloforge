# Implementation Plan: E2E Tests — Kanban Board: View, Move, Sync, and Column Transitions

**Track ID:** e2e-kanban-board_20260309194836Z

## Phase 1: Board Display Tests

- [ ] Task 1.1: Board loads columns — navigate to project board page, verify all four columns render with correct headers: Backlog, In Progress, In Review, Done
- [ ] Task 1.2: Cards render correctly — seed tracks with known statuses, verify each track card appears in the correct column with visible title and status indicator
- [ ] Task 1.3: Empty board state — navigate to board for a project with no tracks, verify empty state message or placeholder is shown in each column

## Phase 2: Card Movement Tests

- [ ] Task 2.1: Move card forward — move a card from Backlog to In Progress (via drag-drop or button), verify it appears in the target column and is removed from the source
- [ ] Task 2.2: Move card backward — move a card from In Review back to In Progress, verify the card moves correctly (regression: ensure backward moves are allowed)
- [ ] Task 2.3: Move API call verification — after moving a card, call `GET /api/projects/{id}/board` and verify backend state matches the UI column positions
- [ ] Task 2.4: Column count updates — verify that column card count badges update after each card movement (source decrements, target increments)

## Phase 3: Board Sync Tests

- [ ] Task 3.1: Sync button refreshes board — change a track's status via API (not UI), click board sync button, verify the card moves to the correct column reflecting the new status
- [ ] Task 3.2: Sync after external change — modify track statuses via direct API calls, trigger sync, verify all cards reflect current backend state
- [ ] Task 3.3: Sync during page load — seed data, navigate to board, verify initial load performs a sync and cards are in correct positions without manual sync

## Phase 4: Card Content Tests

- [ ] Task 4.1: Track title and ID display — verify each card shows the track title and track ID (or abbreviated ID)
- [ ] Task 4.2: Status indicators — verify cards display a visual status indicator (color dot, badge, or icon) matching their current status
- [ ] Task 4.3: Card click navigates to detail — click a card, verify navigation to the track detail page with correct track information

## Phase 5: Edge and Failure Cases

- [ ] Task 5.1: Rapid card moves — move the same card between columns in rapid succession, verify final state is consistent and no ghost cards appear
- [ ] Task 5.2: API error handling — simulate API failure on card move (e.g., network error), verify error toast/message appears and card reverts to original column
- [ ] Task 5.3: Stale board state recovery — open board, wait for backend state to change externally, attempt a move, verify conflict is handled gracefully (sync or error)
