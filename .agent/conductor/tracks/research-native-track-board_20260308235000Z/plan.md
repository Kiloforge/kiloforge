# Implementation Plan: Research — Native Track Board in crelay Dashboard

## Phase 1: State Machine and Storage Design (3 tasks)

### Task 1.1: Define board state machine
- Document all valid states and transitions
- Map each transition to agent lifecycle action
- Document guard conditions and edge cases

### Task 1.2: Evaluate board state storage options
- JSON file, SQLite, or tracks.md augmentation
- Assess persistence, query patterns, multi-writer safety

### Task 1.3: Design API endpoints
- Draft OpenAPI schema fragments for board operations
- Define SSE events for real-time board updates

## Phase 2: UI and Integration Design (3 tasks)

### Task 2.1: Evaluate kanban UI libraries
- Compare @hello-pangea/dnd, dnd-kit, HTML5 drag API
- Recommend with rationale

### Task 2.2: Map Gitea board code to keep vs remove
- Catalog all board-related code, classify as remove/keep/modify

### Task 2.3: Define migration strategy
- Bootstrap native board from tracks.md
- Coexistence during transition

## Phase 3: Decision Document (1 task)

### Task 3.1: Write decision document
- Summarize findings and recommendations
