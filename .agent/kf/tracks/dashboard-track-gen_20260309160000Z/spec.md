# Specification: Dashboard-Driven Track Generation with Interactive Agent

**Track ID:** dashboard-track-gen_20260309160000Z
**Type:** Feature
**Created:** 2026-03-09T16:00:00Z
**Status:** Draft

## Summary

Add a dashboard workflow for generating tracks: user provides a prompt, an interactive track-generator agent researches the codebase and generates tracks (asking the user for clarification when needed via WebSocket), and the resulting tracks land in the board's `backlog` column for human approval or rejection.

## Context

Track generation is currently CLI-only via the `/kf-track-generator` skill running in a dedicated worktree. The dashboard has no way to create tracks — it only displays and moves existing board cards. Users want to:

1. Enter a feature description in the dashboard
2. Watch the agent research and plan in real-time
3. Answer clarifying questions when the agent needs more context
4. Review generated tracks on the board and approve/reject them

This builds on the interactive agent WebSocket infrastructure (`interactive-agent-be/fe` tracks) for the bidirectional conversation, and the existing board system for approval workflow.

## Codebase Analysis

### Existing board flow
- Board columns: `backlog` → `approved` → `in_progress` → `in_review` → `done`
- `POST /api/board/{project}/sync` discovers tracks from conductor artifacts and creates board cards
- `POST /api/board/{project}/move` moves cards between columns
- `KanbanBoard.tsx` supports drag-and-drop between columns

### Existing track-generator skill (`skills/kf-track-generator/SKILL.md`)
- Researches codebase, generates specs/plans, creates track directories
- Merges track artifacts to main
- Currently runs in a worktree (`track-generator-*` branch)
- Presents tracks for review and waits for approval before writing files

### Interactive agent infra (pending tracks)
- `interactive-agent-be` — WebSocket endpoint, stdin/stdout pipes, output buffering
- `interactive-agent-fe` — chat terminal component
- `POST /api/agents/interactive` — spawn interactive agent

### What needs to change
The track-generator skill currently assumes a CLI environment with direct user interaction. For dashboard use, we need:
- A way to spawn it via API with an initial prompt
- The agent's "approval" step to map to board backlog instead of terminal input
- Generated tracks to auto-sync to the board after merge

## Acceptance Criteria

- [ ] "Generate Tracks" button in dashboard (project page or global action)
- [ ] Prompt input form — text area for feature description
- [ ] Spawns interactive track-generator agent with the prompt
- [ ] Agent conversation visible in chat terminal (reuses interactive-agent-fe component)
- [ ] Agent can ask clarifying questions — user responds via chat
- [ ] Generated tracks appear in board `backlog` column after agent completes
- [ ] Board cards in `backlog` have "Approve" and "Reject" actions
- [ ] "Approve" moves card to `approved` column (available for developer agents)
- [ ] "Reject" removes the card and optionally deletes the track artifacts
- [ ] Board auto-syncs after track generation completes
- [ ] `go test ./...` passes
- [ ] `npm run build` succeeds

## Dependencies

- **interactive-agent-be_20260309150000Z** — MUST complete first (provides WebSocket agent infra)
- **interactive-agent-fe_20260309150001Z** — MUST complete first (provides chat terminal component)

## Blockers

None.

## Conflict Risk

- **release-process_20260309153000Z** — NONE. Completely independent subsystems.
- **rebrand-historical-records_20260309063900Z** — LOW. Both touch conductor artifacts but different areas.

## Out of Scope

- Auto-assigning developer agents to approved tracks (future enhancement)
- Track editing in the dashboard (users can only approve/reject, not modify specs)
- Multiple simultaneous track-generator agents (one at a time for now)
- Review mode toggle (separate track)

## Technical Notes

### Spawn flow
```
User clicks "Generate Tracks" → prompt input form
  → POST /api/tracks/generate
    body: {
      "prompt": "Add OAuth2 authentication with Google provider",
      "project": "myapp"
    }
  → Backend prefixes the user prompt:
    "I would like to generate one or more tracks and the specifications are the following: <user prompt>"
  → Spawns: claude -p "/kf-track-generator <prefixed prompt>" --session-id <uuid> --output-format stream-json
  → Returns agent ID
  → Frontend opens chat terminal connected to ws://host/ws/agent/{id}
```

### Prompt prefixing
The backend MUST prefix the raw user prompt with a standard preamble before passing it to the track-generator skill:

```
I would like to generate one or more tracks and the specifications are the following: <user's prompt text>
```

This ensures the track-generator skill receives clear intent regardless of how the user phrases their input. The prefix is applied server-side in the generate handler — the frontend sends only the raw user text.

### Agent clarification flow
The track-generator skill already has a pattern for asking questions:
```
What feature, change, or improvement would you like to generate tracks for?
```

In interactive mode, this becomes a WebSocket message. The agent writes to stdout (question), user reads via WebSocket, types response, which goes to stdin.

### Track approval on board
When the agent completes and tracks are merged to main:
1. Dashboard triggers `POST /api/board/{project}/sync`
2. New tracks appear as cards in `backlog` column
3. User drags to `approved` (or clicks Approve button)
4. Reject action: `DELETE /api/tracks/{trackId}` (new endpoint) removes track artifacts

### New API endpoints needed
```
POST /api/tracks/generate
  → Spawns track-generator agent with prompt
  → Returns: { agent_id, session_url: "/ws/agent/{id}" }

DELETE /api/tracks/{trackId}
  → Removes track artifacts from conductor directory
  → Removes board card
```

### Board card actions
Add action buttons to board cards in `backlog` column:
- "Approve" → `POST /api/board/{project}/move` with column=approved
- "Reject" → `DELETE /api/tracks/{trackId}` + remove card

These actions are only shown for cards in the `backlog` column.

---

_Generated by conductor-track-generator from prompt: "dashboard-driven track generation with interactive agent and board approval"_
