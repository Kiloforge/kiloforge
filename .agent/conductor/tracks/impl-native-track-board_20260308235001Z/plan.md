# Implementation Plan: Native Track Board with Dashboard Kanban and Agent Lifecycle

## Phase 1: Native Board Service (4 tasks)

### Task 1.1: Define board domain types
- Replace Gitea-specific BoardConfig with native board state
- BoardState, BoardCard domain types

### Task 1.2: Create native board service
- GetBoard, MoveCard, RejectTrack, SyncFromTracks
- Calls LifecycleService for agent actions on transitions
- Emits SSE board_update events

### Task 1.3: Update board persistence
- Simplify storage (no Gitea IDs)
- JSON file per project

### Task 1.4: Test board service
- Valid/invalid transitions, sync, reject

## Phase 2: Board API (3 tasks)

### Task 2.1: Add board endpoints to OpenAPI spec
- GET /-/api/board/{project}, POST move, POST reject

### Task 2.2: Implement board API handler
- Generated strict handler via oapi-codegen

### Task 2.3: Add board_update SSE event
- Real-time board state broadcast

## Phase 3: Dashboard Kanban UI (4 tasks)

### Task 3.1: Add kanban UI dependency
### Task 3.2: Create KanbanBoard component with drag-and-drop
### Task 3.3: Create board data hook (useBoard)
### Task 3.4: Integrate kanban into project page

## Phase 4: Remove Gitea Board Code (3 tasks)

### Task 4.1: Remove boardSyncer and label-webhook handling
### Task 4.2: Simplify Gitea integration (keep issue/PR, remove board)
### Task 4.3: Update tests

## Phase 5: Verification (2 tasks)

### Task 5.1: Integration test (end-to-end board → agent lifecycle)
### Task 5.2: Documentation
