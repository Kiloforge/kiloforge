# Decision: Native Track Board in kiloforge Dashboard

**Track ID:** research-native-track-board_20260308235000Z
**Date:** 2026-03-08
**Status:** Accepted

---

## Context

kiloforge currently uses Gitea's project board as the kanban UI for track lifecycle. Column transitions on the Gitea board trigger webhooks, which the relay processes to drive agent lifecycle (spawn, halt, resume, terminate). This works but introduces architectural coupling:

1. **Gitea dependency** — The board is only available when Gitea is running
2. **Indirect state** — Track state lives in tracks.md, board state lives in Gitea, and the mapping lives in board.json — three sources of truth
3. **Webhook latency** — Board → webhook → relay → agent lifecycle adds unnecessary round-trips for what should be a local operation
4. **Limited UI** — Gitea's project board is generic; it can't show kiloforge-specific info (agent status, cost, session IDs)

The goal: own the board in kiloforge's dashboard, make column transitions drive agent lifecycle directly via API calls, and simplify the architecture.

---

## 1. Board State Machine

### States (Columns)

| Column | tracks.md | Description |
|--------|-----------|-------------|
| Backlog | `[ ]` | Track defined but not approved for work |
| Approved | `[!]` | Approved for implementation, waiting for worker |
| In Progress | `[~]` | Developer agent actively working |
| In Review | `[r]` | PR created, reviewer agent working |
| Done | `[x]` | Merged and complete |

### Valid Transitions and Actions

| From → To | Action | Guard |
|-----------|--------|-------|
| Backlog → Approved | None (human decision) | — |
| Approved → In Progress | Spawn developer agent | Worktree available |
| In Progress → In Review | Spawn reviewer agent | PR exists |
| In Review → Done | Close issue, return worktree | PR merged |
| In Progress → Backlog | Halt developer agent | — |
| In Review → In Progress | Halt reviewer, resume developer | — |
| In Review → Backlog | Halt reviewer, halt developer | — |
| Any → Rejected | Terminate all agents, return worktree | — |

### Guard Conditions

- **Approved → In Progress**: Must have a free worktree in the pool (or auto-create one)
- **In Progress → In Review**: Requires a PR number/URL to assign to the reviewer
- **In Review → Done**: Only valid when PR status is "merged"

### Edge Cases

- **Concurrent moves**: Lock the board state per-track during transitions (reuse existing lock service)
- **Stale agents**: If agent PID is dead but status shows "running", detect and mark as failed before allowing transitions
- **Re-promotion**: Moving a track back to In Progress when the developer agent is halted should attempt `claude --resume`

---

## 2. Board State Storage

### Option A: Augment tracks.md (Rejected)

Tracks.md is a markdown file parsed by both kiloforge and conductor skills. Adding board-specific fields (assignee, column history, timestamps) would complicate the format and create merge conflicts in multi-worker scenarios.

### Option B: SQLite (Rejected for now)

SQLite would be the right choice at scale, but kiloforge currently uses JSON file persistence for all state (agents, board config, projects). Introducing SQLite just for the board creates an inconsistent persistence story. Revisit when migrating all state to SQLite.

### Option C: JSON file per project (Recommended)

Extend the existing `board.json` persistence to store native board state instead of Gitea board mappings. This is consistent with the current architecture.

**Schema:**

```json
{
  "columns": ["backlog", "approved", "in_progress", "in_review", "done"],
  "cards": {
    "track-id-1": {
      "track_id": "track-id-1",
      "column": "in_progress",
      "position": 0,
      "agent_id": "abc123",
      "assigned_worker": "developer-1",
      "pr_number": null,
      "moved_at": "2026-03-08T12:00:00Z",
      "created_at": "2026-03-08T10:00:00Z"
    }
  }
}
```

**Multi-writer safety**: Use file-level locking (already patterned in jsonfile package). For the native board, only the relay server writes; dashboard reads via API. No concurrent writer problem.

**Bootstrap**: On first load, scan tracks.md and create cards for all tracks in their current status column.

---

## 3. API Design

### New Endpoints

```yaml
# Board state
GET    /-/api/board                    # Get board state (all columns + cards)
GET    /-/api/board?project={slug}     # Filter by project

# Card operations
POST   /-/api/board/cards/{trackId}/move   # Move card to new column
  Body: { "column": "approved" }
  Response: 200 { "track_id": "...", "column": "approved", "action_taken": "agent_spawned" }
  Response: 409 { "error": "no worktree available" }

POST   /-/api/board/cards/{trackId}/reject  # Reject a track
  Response: 200 { "track_id": "...", "action_taken": "agent_terminated" }

# Bulk operations
POST   /-/api/board/sync               # Re-sync board from tracks.md
  Response: 200 { "created": 3, "updated": 1, "unchanged": 20 }
```

### OpenAPI Schema Fragment

```yaml
/board:
  get:
    operationId: getBoardState
    parameters:
      - name: project
        in: query
        schema: { type: string }
    responses:
      '200':
        content:
          application/json:
            schema: { $ref: '#/components/schemas/BoardState' }

/board/cards/{trackId}/move:
  post:
    operationId: moveCard
    parameters:
      - name: trackId
        in: path
        required: true
        schema: { type: string }
    requestBody:
      content:
        application/json:
          schema:
            type: object
            required: [column]
            properties:
              column: { type: string, enum: [backlog, approved, in_progress, in_review, done] }
    responses:
      '200':
        content:
          application/json:
            schema: { $ref: '#/components/schemas/MoveResult' }
      '409':
        content:
          application/json:
            schema: { $ref: '#/components/schemas/Error' }

schemas:
  BoardState:
    type: object
    properties:
      columns:
        type: array
        items: { type: string }
      cards:
        type: array
        items: { $ref: '#/components/schemas/BoardCard' }

  BoardCard:
    type: object
    properties:
      track_id: { type: string }
      title: { type: string }
      type: { type: string }
      column: { type: string }
      position: { type: integer }
      agent_id: { type: string, nullable: true }
      agent_status: { type: string, nullable: true }
      assigned_worker: { type: string, nullable: true }
      pr_number: { type: integer, nullable: true }
      moved_at: { type: string, format: date-time }

  MoveResult:
    type: object
    properties:
      track_id: { type: string }
      column: { type: string }
      action_taken: { type: string, nullable: true }
```

### SSE Events for Board Updates

Add to the existing SSE stream (`/-/events`):

| Event Type | Payload | Trigger |
|------------|---------|---------|
| `board_card_moved` | `{ track_id, from_column, to_column, agent_action }` | Card moved via API or webhook |
| `board_card_added` | `{ track_id, column }` | New track synced to board |
| `board_card_removed` | `{ track_id }` | Track archived/deleted |

These integrate with the existing `SSEHub.Broadcast()` mechanism.

---

## 4. Kanban UI Library Evaluation

### Option A: @hello-pangea/dnd (Recommended)

- **Pros**: Fork of react-beautiful-dnd (battle-tested), active maintenance, React 19 compatible, declarative API, accessible by default, smooth animations
- **Cons**: Slightly larger bundle (~30KB gzipped)
- **Fit**: Best match for a traditional kanban with columns and cards

### Option B: @dnd-kit/core

- **Pros**: Modular, tree-shakeable, framework-agnostic core, collision detection strategies
- **Cons**: More low-level — requires more boilerplate for kanban, less opinionated about accessibility
- **Fit**: Better for custom drag interactions, overkill for a standard kanban

### Option C: HTML5 Drag API (native)

- **Pros**: Zero bundle cost, no dependency
- **Cons**: Poor mobile support, no animation, accessibility requires manual implementation, inconsistent across browsers
- **Fit**: Only viable for desktop-only tool (which kiloforge is), but UX would feel dated

### Recommendation: **@hello-pangea/dnd**

It provides the best out-of-box kanban experience with minimal code. The declarative `<DragDropContext>`, `<Droppable>`, `<Draggable>` API maps directly to our board/column/card model.

---

## 5. Gitea Board Code: Keep vs Remove

### Code to Remove

| File/Component | Reason |
|----------------|--------|
| `service/board_service.go` — `SetupBoard()`, `PublishTrack()`, `SyncTracks()` | Gitea project board creation/sync no longer needed |
| `adapter/gitea/board_adapter.go` — all board methods | Gitea board API calls replaced by native board |
| `adapter/gitea/projects.go` — project board API calls | No longer needed |
| `port/board_client.go` — `BoardGiteaClient` interface | Port for Gitea board operations |
| `domain/board.go` — `BoardConfig`, `TrackIssue` | Replace with native board domain types |
| `persistence/jsonfile/board_store.go` — Gitea board mappings | Replace with native board store |
| `rest/board_sync.go` — webhook-driven label/column sync | Replace with API-driven transitions |
| `cli/board.go` — `board --setup` command | Replace with auto-bootstrap on first API call |
| Standard labels (`status:suggested`, etc.) | No longer synced to Gitea |

### Code to Keep

| File/Component | Reason |
|----------------|--------|
| `service/lifecycle_service.go` | Core agent lifecycle logic is board-agnostic — reuse directly |
| `service/track_service.go` | tracks.md parsing stays (source of truth for track definitions) |
| `adapter/agent/*` | Agent spawning, recovery, shutdown — unchanged |
| `rest/server.go` — webhook handler | Still needed for PR events (merge detection) from Gitea |
| Board sync for PR events only | `handlePROpened`, `handlePRMerged` can trigger native board moves |

### Code to Modify

| File/Component | Change |
|----------------|--------|
| `service/board_service.go` | Rewrite as native board service (reads tracks.md, manages JSON state, calls lifecycle service) |
| `persistence/jsonfile/board_store.go` | Rewrite to store native board cards instead of Gitea mappings |
| `dashboard/watcher.go` | Add board state change detection and `board_*` SSE events |
| `dashboard/server.go` | No change needed — new API endpoints go through the generated REST server |

---

## 6. Migration Path

### Phase 1: Build native board (no Gitea removal)

1. Add native board domain types and persistence
2. Add board API endpoints to OpenAPI schema and implement
3. Build kanban UI component in the dashboard
4. Bootstrap board state from tracks.md on first load
5. Board API moves call lifecycle service directly (same as webhook handler does today)

### Phase 2: Wire SSE events

1. Add `board_card_moved`, `board_card_added`, `board_card_removed` to SSE hub
2. Kanban UI subscribes to these events for real-time updates
3. Webhook handler for PR events triggers native board moves (not Gitea board moves)

### Phase 3: Remove Gitea board code

1. Remove `board --setup` CLI command
2. Remove Gitea board adapter, port, and domain types
3. Remove label-based webhook handling (keep PR event handling)
4. Remove board config from project persistence
5. Update `kf add` to skip board setup

### Coexistence

During Phase 1-2, both Gitea board and native board can coexist. The native board is the primary UI; Gitea board is read-only legacy. No sync between them — they're independent views of the same tracks.md state.

---

## 7. Multi-Project Boards

### Per-Project Board (Primary)

Each registered project gets its own board state file (`projects/{slug}/board.json`). The `GET /-/api/board?project={slug}` endpoint returns a single project's board.

### Global View (Secondary)

`GET /-/api/board` (no project filter) returns a merged view of all projects' boards. Cards include a `project` field so the UI can group or filter. This is a read-only aggregation — moves must target a specific project.

The dashboard landing page shows the global view; the project detail page shows the per-project board.

---

## 8. Implementation Tracks (Suggested)

Based on this research, the implementation should be split into:

1. **impl-native-board-backend** — Domain types, persistence, board service, API endpoints (OpenAPI schema-first)
2. **impl-native-board-frontend** — Kanban UI component with @hello-pangea/dnd, SSE integration
3. **impl-remove-gitea-board** — Remove Gitea board code, update CLI, clean up persistence

Tracks 1 and 2 can run in parallel. Track 3 depends on both.

---

## Summary of Recommendations

| Question | Recommendation |
|----------|---------------|
| State machine | 5 columns: Backlog → Approved → In Progress → In Review → Done |
| Storage | JSON file per project (extend existing pattern) |
| API | REST endpoints with OpenAPI schema-first design |
| UI library | @hello-pangea/dnd |
| Gitea board removal | Phased: build native first, then remove Gitea board code |
| Migration | Bootstrap from tracks.md, no Gitea board sync needed |
| Multi-project | Per-project boards with global aggregation view |
| Real-time | SSE events via existing hub |
