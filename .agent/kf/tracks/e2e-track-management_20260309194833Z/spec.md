# Specification: E2E Tests: Track Management — List, Detail, Generate, and Delete

**Track ID:** e2e-track-management_20260309194833Z
**Type:** Chore
**Created:** 2026-03-09T19:48:33Z
**Status:** Draft

## Summary

Comprehensive E2E tests for track lifecycle: listing tracks, viewing track details, generating tracks via mock agent, and deleting tracks, covering happy path, edge cases, and expected failures.

## Context

Tracks are the core work unit in Kiloforge. Each track represents a development task with a spec, plan, and metadata. The track lifecycle includes:

1. **List tracks** — view all tracks for a project with status, type, and timestamps
2. **Track detail** — view spec, plan, and metadata for a specific track
3. **Generate tracks** — spawn a mock agent that produces track artifacts via stream-JSON
4. **Delete tracks** — remove a track and its artifacts

Track generation is the most complex flow: it spawns an agent (via `exec.CommandContext` calling the claude CLI), streams output in real-time via WebSocket/SSE, and creates track artifacts on completion. The mock agent binary from Track 1 simulates this without needing a real Claude API key.

## Codebase Analysis

### Existing patterns

- **Track list API** — `GET /api/projects/{projectId}/tracks` returns all tracks with status, type, timestamps
- **Track detail API** — `GET /api/tracks/{trackId}` returns full track including spec, plan, metadata
- **Track generation API** — `POST /api/projects/{projectId}/tracks/generate` spawns an agent to generate tracks
- **Track delete API** — `DELETE /api/tracks/{trackId}` removes track and artifacts
- **SSE events** — `track_update` event fires when track status changes
- **Agent spawner** — `exec.CommandContext` runs claude CLI with `--output-format stream-json`
- **WebSocket** — `/ws/agents/{agentId}` streams agent output in real-time (message types: `output`, `status`, `error`)

### Frontend components

- `TrackList` — table/list view with filtering, sorting, status badges
- `TrackDetail` — tabbed view showing spec, plan, metadata
- `TrackGenerateDialog` — form to configure and trigger track generation
- `AgentOutputStream` — real-time stream display during generation

### Mock agent integration

Track generation spawns the mock agent binary. The mock agent outputs stream-JSON events that the spawner parses. The default event sequence (init, content_block_delta, result) simulates a successful generation. Custom `MOCK_AGENT_EVENTS` can simulate failures or partial output.

## Acceptance Criteria

- [ ] Track list displays all tracks with correct status, project, timestamps
- [ ] Track detail page shows spec, plan, and metadata
- [ ] Track generation spawns mock agent, streams output in real-time
- [ ] Generated tracks appear in list and on kanban board
- [ ] Track deletion removes track and updates list
- [ ] Track filtering/sorting works correctly
- [ ] Edge cases: empty track list, very long track titles, rapid generation
- [ ] Failure cases: generate with mock agent failure, delete nonexistent track, API timeout

## Dependencies

- `e2e-infra-mock-agent_20260309194830Z` — E2E test infrastructure and mock agent binary
- `e2e-project-management_20260309194832Z` — needs a project to exist for track operations

## Blockers

None.

## Conflict Risk

- LOW — adds test files only, no production code changes.

## Out of Scope

- Kanban board drag-and-drop interactions (covered by `e2e-kanban-board_20260309194836Z`)
- Agent lifecycle beyond track generation (covered by `e2e-agent-lifecycle_20260309194834Z`)
- Track editing/updating (not currently a feature)

## Technical Notes

### Test file structure

```
frontend/e2e/
  track-list.spec.ts          — track listing, filtering, sorting tests
  track-detail.spec.ts        — track detail view tests
  track-generate.spec.ts      — track generation with mock agent tests
  track-delete.spec.ts        — track deletion tests
```

### Track list test scenarios

| Scenario | Seed Data | Expected |
|---|---|---|
| Multiple tracks | 5 tracks, various statuses | All displayed with correct badges |
| Empty list | No tracks | Empty state message shown |
| Status filter | 5 tracks, filter by "pending" | Only pending tracks shown |
| Sort by date | 5 tracks, different dates | Sorted correctly ascending/descending |

### Track generation test scenarios

1. **Happy path**: trigger generation, verify mock agent spawned, verify stream output displayed in real-time, verify track created on completion
2. **Stream display**: verify content_block_delta events render as text in the output stream panel
3. **Completion**: verify result event triggers track creation, verify track appears in list with correct metadata
4. **Failure**: set `MOCK_AGENT_EXIT_CODE=1`, trigger generation, verify error displayed, verify no partial track created

### Track detail test scenarios

- Navigate from list to detail via click
- Verify spec tab shows markdown content
- Verify plan tab shows task checkboxes
- Verify metadata tab shows JSON or formatted metadata
- Navigate to nonexistent track ID, verify 404 page

### Track deletion test scenarios

- Delete with confirmation dialog
- Cancel delete — track still exists
- Delete last track — list shows empty state
- Delete nonexistent track (API returns 404) — error toast

### Real-time streaming verification

Track generation streams output via WebSocket. Tests should:
1. Trigger generation
2. Wait for WebSocket connection (Playwright can intercept WebSocket frames)
3. Verify stream output updates in the UI character by character or chunk by chunk
4. Verify completion state (success badge, agent stopped)

### Developer agent instructions

Use the Playwright MCP skill to verify all UI interactions. Key areas to visually verify:
- Track list rendering with various statuses and badge colors
- Real-time stream output during generation (text appearing progressively)
- Tab navigation in track detail view
- Confirmation dialog for deletion
- Toast notifications for errors
- Empty state displays

---

_Generated by conductor-track-generator for E2E track management tests_
