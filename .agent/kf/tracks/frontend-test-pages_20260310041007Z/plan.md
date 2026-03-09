# Implementation Plan: Frontend Test Coverage — Pages and Components

**Track ID:** frontend-test-pages_20260310041007Z

## Phase 1: Page Tests

### Task 1.1: Test OverviewPage
- [x] Create `src/pages/OverviewPage.test.tsx`
- [x] Test: renders stat cards with agent counts
- [x] Test: renders agent grid with mock agents
- [x] Test: renders quota information
- [x] Test: handles loading state
- [x] Test: handles error state

### Task 1.2: Test ProjectPage
- [x] Create `src/pages/ProjectPage.test.tsx`
- [x] Test: renders project list
- [x] Test: add project modal flow (open, fill, submit)
- [x] Test: remove project confirmation dialog
- [x] Test: board tab renders KanbanBoard
- [x] Test: sync panel renders with status
- [x] Test: handles empty project list

### Task 1.3: Test AgentDetailPage
- [x] Create `src/pages/AgentDetailPage.test.tsx`
- [x] Test: renders agent metadata (name, role, status, timing)
- [x] Test: renders log viewer with mock logs
- [x] Test: handles agent not found
- [x] Test: stop/resume/delete buttons appear based on agent status

### Task 1.4: Verify Phase 1
- [x] `npm test` passes

## Phase 2: Component Tests

### Task 2.1: Test KanbanBoard
- [x] Create `src/components/KanbanBoard.test.tsx`
- [x] Test: renders columns with correct cards
- [x] Test: move card triggers mutation
- [x] Test: confirmation dialog appears for state transitions
- [x] Test: handles empty board state

### Task 2.2: Test AgentCard
- [x] Create `src/components/AgentCard.test.tsx`
- [x] Test: renders agent name, role, status badge
- [x] Test: links to correct agent detail page
- [x] Test: shows track/project association if present
- [x] Test: handles missing optional fields

### Task 2.3: Add any missing test utilities
- [x] Consistent mock patterns used across all test files (inline mocks)

### Task 2.4: Verify Phase 2
- [x] `npm test` passes
- [x] `make build` passes
