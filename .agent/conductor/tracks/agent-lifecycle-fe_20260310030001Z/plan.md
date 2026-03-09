# Implementation Plan: Agent Lifecycle Management — Stop, Resume, Delete (Frontend)

**Track ID:** agent-lifecycle-fe_20260310030001Z

## Phase 1: API Integration & Hooks

### Task 1.1: Add mutation hooks for agent lifecycle actions
- Create `useAgentActions` hook (or add to existing patterns) with:
  - `stopAgent(id)` → `POST /api/agents/{id}/stop`
  - `resumeAgent(id)` → `POST /api/agents/{id}/resume`
  - `deleteAgent(id)` → `DELETE /api/agents/{id}`
- Invalidate relevant query keys on success
- Handle error responses (409 conflict, 404 not found)

### Task 1.2: Add helper for action visibility logic
- `canStop(agent)` → status is "running" or "waiting"
- `canResume(agent)` → status is "stopped", "completed", or "failed" AND role is "interactive"
- `canDelete(agent)` → status is NOT "running" and NOT "waiting"

### Task 1.3: Verify Phase 1
- TypeScript compiles (`npx tsc --noEmit`)
- Hooks are importable

## Phase 2: AgentCard & AgentDetailPage Actions

### Task 2.1: Add action buttons to AgentCard
- Add Stop button (visible when `canStop`)
- Add Resume button (visible when `canResume`)
- Add Delete button (visible when `canDelete`) with confirmation
- Show loading spinner during mutations
- Resume auto-triggers `onAttach` callback on success

### Task 2.2: Add action toolbar to AgentDetailPage
- Add action buttons below the title bar or in the meta section
- Stop/Resume/Delete with same visibility logic
- Delete navigates to `/` on success
- Resume shows terminal section on success

### Task 2.3: Add actions column to AgentHistoryPage
- Add "Actions" column to the table
- Compact button group: Stop / Resume / Delete icons or text
- Same visibility and confirmation logic

### Task 2.4: Verify Phase 2
- TypeScript compiles
- Visual inspection of button states

## Phase 3: Polish & Edge Cases

### Task 3.1: Error handling and feedback
- Show toast on stop/resume/delete failure
- Handle 409 responses with clear message ("Agent is still running", "Agent is already running")
- Disable buttons during pending mutations to prevent double-clicks

### Task 3.2: Confirmation dialog for delete
- Show agent name in confirmation message
- Cancel returns to previous state
- OK triggers delete mutation

### Task 3.3: Verify Phase 3
- Full frontend type check passes
- Manual test: stop → resume → delete flow in UI
