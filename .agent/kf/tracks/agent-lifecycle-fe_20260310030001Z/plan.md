# Implementation Plan: Agent Lifecycle Management — Stop, Resume, Delete (Frontend)

**Track ID:** agent-lifecycle-fe_20260310030001Z

## Phase 1: API Integration & Hooks

### Task 1.1: Add mutation hooks for agent lifecycle actions
- [x] Create `useAgentActions` hook (or add to existing patterns) with:
  - `stopAgent(id)` → `POST /api/agents/{id}/stop`
  - `resumeAgent(id)` → `POST /api/agents/{id}/resume`
  - `deleteAgent(id)` → `DELETE /api/agents/{id}`
- [x] Invalidate relevant query keys on success
- [x] Handle error responses (409 conflict, 404 not found)

### Task 1.2: Add helper for action visibility logic
- [x] `canStop(agent)` → status is "running" or "waiting"
- [x] `canResume(agent)` → status is "stopped", "completed", or "failed" AND role is "interactive"
- [x] `canDelete(agent)` → status is NOT "running" and NOT "waiting"

### Task 1.3: Verify Phase 1
- [x] TypeScript compiles (`npx tsc --noEmit`)
- [x] Hooks are importable

## Phase 2: AgentCard & AgentDetailPage Actions

### Task 2.1: Add action buttons to AgentCard
- [x] Add Stop button (visible when `canStop`)
- [x] Add Resume button (visible when `canResume`)
- [x] Add Delete button (visible when `canDelete`) with confirmation
- [x] Show loading spinner during mutations
- [x] Resume auto-triggers `onAttach` callback on success

### Task 2.2: Add action toolbar to AgentDetailPage
- [x] Add action buttons below the title bar or in the meta section
- [x] Stop/Resume/Delete with same visibility logic
- [x] Delete navigates to `/` on success
- [x] Resume shows terminal section on success

### Task 2.3: Add actions column to AgentHistoryPage
- [x] Add "Actions" column to the table
- [x] Compact button group: Stop / Resume / Delete icons or text
- [x] Same visibility and confirmation logic

### Task 2.4: Verify Phase 2
- [x] TypeScript compiles
- [x] Visual inspection of button states

## Phase 3: Polish & Edge Cases

### Task 3.1: Error handling and feedback
- [x] Show toast on stop/resume/delete failure
- [x] Handle 409 responses with clear message ("Agent is still running", "Agent is already running")
- [x] Disable buttons during pending mutations to prevent double-clicks

### Task 3.2: Confirmation dialog for delete
- [x] Show agent name in confirmation message
- [x] Cancel returns to previous state
- [x] OK triggers delete mutation

### Task 3.3: Verify Phase 3
- [x] Full frontend type check passes
- [x] Manual test: stop → resume → delete flow in UI
