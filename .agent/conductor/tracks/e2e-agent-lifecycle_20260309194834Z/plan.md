# Implementation Plan: E2E Tests — Agent Lifecycle: Spawn, Monitor, Stop, Resume, Delete

**Track ID:** e2e-agent-lifecycle_20260309194834Z

## Phase 1: Spawn Tests

- [ ] Task 1.1: Spawn developer agent — navigate to dashboard, click spawn, select developer role, verify agent appears in list with "running" status
- [ ] Task 1.2: Spawn reviewer agent — spawn with reviewer role, verify agent appears with correct role badge and "running" status
- [ ] Task 1.3: Spawn with mock agent failure — configure mock to exit non-zero immediately, verify agent transitions to "failed" status in UI
- [ ] Task 1.4: Spawn validation errors — attempt spawn with missing project, missing prompt, invalid role; verify error messages display correctly

## Phase 2: Monitoring Tests

- [ ] Task 2.1: Agent list display — spawn multiple agents, verify all appear in the agent list/grid with correct status, role, and project info
- [ ] Task 2.2: Agent detail page — click an agent in the list, verify detail page shows role, track ref, session ID, started-at timestamp, and current status
- [ ] Task 2.3: Log viewer — navigate to agent detail, verify log viewer shows streamed output text from mock agent events (content_block_delta text)

## Phase 3: Status Transition Tests

- [ ] Task 3.1: Running to completed — spawn agent with default mock config, wait for completion, verify status changes from "running" to "completed" in UI
- [ ] Task 3.2: Running to failed — spawn agent with `MOCK_AGENT_EXIT_CODE=1`, verify status changes to "failed" with error indication
- [ ] Task 3.3: Mock agent exits with different codes — test exit codes 1, 2, 137 (SIGKILL); verify each produces "failed" status with appropriate indication
- [ ] Task 3.4: Status updates in real-time — spawn agent with slow delay (`MOCK_AGENT_DELAY=2000`), verify status badge updates without page refresh (SSE-driven)

## Phase 4: Lifecycle Action Tests

- [ ] Task 4.1: Stop agent — spawn a long-running agent, click stop button, verify status transitions to "stopped" and stop button is replaced by appropriate controls
- [ ] Task 4.2: Resume agent — suspend an agent, click resume, verify status returns to "running" and output resumes in log viewer
- [ ] Task 4.3: Delete agent — stop an agent, click delete, confirm deletion dialog, verify agent removed from list entirely
- [ ] Task 4.4: Lifecycle edge cases — stop an already-stopped agent (expect no-op or error toast), resume a non-suspended agent (expect error), rapid spawn then immediate stop

## Phase 5: History and Filtering Tests

- [ ] Task 5.1: Agent history page — navigate to history page, verify all agents (including completed/failed) are listed in reverse chronological order
- [ ] Task 5.2: Filter by status — use status filter dropdown, verify only agents with selected status appear (test with "completed", "failed", "running")
- [ ] Task 5.3: Filter by role — use role filter, verify only agents with selected role appear (test with "developer", "reviewer")
