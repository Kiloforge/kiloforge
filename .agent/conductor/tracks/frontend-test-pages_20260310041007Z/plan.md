# Implementation Plan: Frontend Test Coverage — Pages and Components

**Track ID:** frontend-test-pages_20260310041007Z

## Phase 1: Page Tests

### Task 1.1: Test OverviewPage
- Create `src/pages/OverviewPage.test.tsx`
- Test: renders stat cards with agent counts
- Test: renders agent grid with mock agents
- Test: renders quota information
- Test: handles loading state
- Test: handles error state

### Task 1.2: Test ProjectPage
- Create `src/pages/ProjectPage.test.tsx`
- Test: renders project list
- Test: add project modal flow (open, fill, submit)
- Test: remove project confirmation dialog
- Test: board tab renders KanbanBoard
- Test: sync panel renders with status
- Test: handles empty project list

### Task 1.3: Test AgentDetailPage
- Create `src/pages/AgentDetailPage.test.tsx`
- Test: renders agent metadata (name, role, status, timing)
- Test: renders log viewer with mock logs
- Test: handles agent not found
- Test: stop/resume/delete buttons appear based on agent status

### Task 1.4: Verify Phase 1
- `npm test` passes

## Phase 2: Component Tests

### Task 2.1: Test KanbanBoard
- Create `src/components/KanbanBoard.test.tsx`
- Test: renders columns with correct cards
- Test: move card triggers mutation
- Test: confirmation dialog appears for state transitions
- Test: handles empty board state

### Task 2.2: Test AgentCard
- Create `src/components/AgentCard.test.tsx`
- Test: renders agent name, role, status badge
- Test: links to correct agent detail page
- Test: shows track/project association if present
- Test: handles missing optional fields

### Task 2.3: Add any missing test utilities
- Create shared test fixtures (mock agent, mock project, mock board state)
- Ensure consistent mock patterns across all new test files

### Task 2.4: Verify Phase 2
- `npm test` passes
- `make test-frontend` passes
