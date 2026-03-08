# Implementation Plan: Native Track Board with Dashboard Kanban and Agent Lifecycle

## Phase 1: Native Board Service (4 tasks)

### Task 1.1: Define board domain types [x]
- Replace Gitea-specific BoardConfig with native board state
- BoardState, BoardCard domain types

### Task 1.2: Create native board service [x]
- GetBoard, MoveCard, SyncFromTracks, UpdateCardAgent
- Uses NativeBoardStore interface for persistence

### Task 1.3: Update board persistence [x]
- Simplify storage (no Gitea IDs)
- JSON file per project

### Task 1.4: Test board service [x]
- Valid/invalid transitions, sync, same-column moves, track not found

## Phase 2: Board API (3 tasks)

### Task 2.1: Add board endpoints to OpenAPI spec [x]
- GET /-/api/board/{project}, POST move, POST sync

### Task 2.2: Implement board API handler [x]
- Generated strict handler via oapi-codegen
- Wired via WithBoardService server option

### Task 2.3: Add board updates in webhook handlers [x]
- PR open → move card to in_review
- PR merged → move card to done

## Phase 3: Dashboard Kanban UI (4 tasks)

### Task 3.1: Add BoardState/BoardCard types [x]
### Task 3.2: Create KanbanBoard component with HTML5 drag-and-drop [x]
### Task 3.3: Create useBoard hook with optimistic updates [x]
### Task 3.4: Integrate kanban into project page [x]

## Phase 4: Remove Gitea Board Code (3 tasks)

### Task 4.1: Remove boardSyncer, WithBoardSync, and label-webhook handling [x]
### Task 4.2: Remove Gitea board adapter, projects API, board CLI command [x]
### Task 4.3: Update tests and fix lifecycle_service column names [x]

## Phase 5: Verification (2 tasks)

### Task 5.1: Full test suite and build verification [x]
### Task 5.2: Track completion [x]
