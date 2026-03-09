# Specification: E2E Tests — Agent Lifecycle: Spawn, Monitor, Stop, Resume, Delete

**Track ID:** e2e-agent-lifecycle_20260309194834Z
**Type:** Chore
**Created:** 2026-03-09T19:48:34Z
**Status:** Draft

## Summary

Comprehensive E2E tests for the full agent lifecycle: spawning developer/reviewer agents via the mock agent binary, monitoring status transitions, viewing logs, stopping/resuming/deleting agents, covering happy path, edge cases, and expected failures.

## Context

Kiloforge manages Claude Code agents through a full lifecycle: spawn, monitor, stop, resume, and delete. Each agent has a role (developer, reviewer, interactive), belongs to a project, and transitions through well-defined statuses (running, waiting, halted, stopped, completed, failed, suspended, suspending, force-killed, resume-failed). The dashboard displays agents in a grid/list with real-time status updates via SSE.

These tests verify that the entire lifecycle works end-to-end: from the user clicking "Spawn Agent" through the UI updating as the mock agent runs and completes, to stopping/resuming/deleting agents and verifying the UI reflects each change correctly.

## Codebase Analysis

### Agent spawner (`backend/internal/adapter/agent/`)

The spawner uses `exec.CommandContext` to launch the agent binary (overridden to mock agent in E2E). It monitors stdout for stream-JSON events and updates agent status in the store. Terminal statuses: `stopped`, `completed`, `failed`, `force-killed`, `resume-failed`.

### API endpoints

- `POST /api/agents` — spawn a new agent (role, project ID, track ref, prompt)
- `GET /api/agents` — list agents (supports `?active=true` filter)
- `GET /api/agents/{agentId}` — agent detail (role, track ref, session ID, timestamps)
- `POST /api/agents/{agentId}/stop` — send SIGINT to agent
- `POST /api/agents/{agentId}/resume` — resume a suspended agent
- `DELETE /api/agents/{agentId}` — remove agent from store

### Frontend pages

- Agent list/grid on dashboard — shows status, role, project
- Agent detail page — shows full agent info, log viewer with streamed output
- Agent history page — filterable list of all agents

### Mock agent

The mock agent binary (from `e2e-infra-mock-agent`) produces stream-JSON events and supports configurable behavior via environment variables (`MOCK_AGENT_DELAY`, `MOCK_AGENT_EXIT_CODE`, `MOCK_AGENT_FAIL_AFTER`, etc.).

## Acceptance Criteria

- [ ] Spawn developer agent with mock — appears in agent list as "running"
- [ ] Spawn reviewer agent with mock — appears in agent list as "running"
- [ ] Agent status transitions: running -> completed (success), running -> failed (error)
- [ ] Agent detail page shows role, track ref, session ID, timestamps
- [ ] Agent log viewer shows streamed output from mock agent
- [ ] Stop agent sends SIGINT, status becomes "stopped"
- [ ] Resume suspended agent, status returns to "running"
- [ ] Delete agent removes from list
- [ ] Agent history page shows all agents with filtering by status
- [ ] Edge cases: stop already-stopped agent, resume non-suspended agent, rapid spawn/stop
- [ ] Failure cases: spawn with missing project, agent crash (non-zero exit), resume failure

## Dependencies

- `e2e-infra-mock-agent_20260309194830Z` — provides mock agent binary, Playwright config, and test helpers

## Blockers

None.

## Conflict Risk

- LOW — adds new E2E test files only, no production code changes.

## Out of Scope

- Interactive terminal testing (covered in `e2e-interactive-terminal_20260309194835Z`)
- WebSocket protocol details (covered in `e2e-interactive-terminal_20260309194835Z`)
- SSE event transport testing (covered in `e2e-sse-realtime_20260309194837Z`)
- Kanban board card movement from agent status changes (covered in `e2e-kanban-board_20260309194836Z`)

## Technical Notes

### Test file organization

```
e2e/
  agent-lifecycle/
    spawn_test.go          — spawn happy path and validation tests
    monitoring_test.go     — agent list, detail page, log viewer tests
    status_test.go         — status transition tests
    lifecycle_test.go      — stop, resume, delete action tests
    history_test.go        — history page and filtering tests
```

### Mock agent configurations for each test scenario

| Scenario | Mock Config |
|---|---|
| Happy path (completed) | Default events, exit code 0 |
| Agent failure | `MOCK_AGENT_EXIT_CODE=1` |
| Agent crash mid-run | `MOCK_AGENT_FAIL_AFTER=2` |
| Long-running agent (for stop test) | `MOCK_AGENT_DELAY=5000` |
| Resume failure | Mock that exits non-zero on resume signal |

### Playwright assertions

Each test should:
1. Navigate to the relevant page
2. Perform the action (spawn, stop, resume, delete)
3. Assert UI updates within a reasonable timeout (5s for status changes)
4. Verify API state matches UI state via fetch assertions

### Developer agent instructions

When building this track, use the Playwright MCP skill to verify E2E tests work in the browser. Run tests in headed mode during development for visual verification.

---

_Generated by conductor-track-generator for E2E agent lifecycle tests_
