# Specification: Agent Completion Callback and Dry-Run Mode

**Track ID:** agent-completion-callback_20260309112000Z
**Type:** Feature
**Created:** 2026-03-09T11:20:00Z
**Status:** Draft

## Summary

Add a post-completion callback to the spawner so that when an agent exits, the orchestrator automatically performs cleanup (move board card to Done, return worktree to pool). Also add `kf implement --dry-run` mode that skips agent spawning and immediately runs the completion callback, useful for testing the notification pipeline and manually advancing tracks.

## Context

Currently, when an agent process exits, `monitorAgent` only sets the agent status to "completed" or "failed" in the store. No further orchestrator-level actions occur:

- **Board card stays stuck in "In Progress"** on direct merge (no PR). Only the webhook-driven PR path (`handleReviewApproved`) moves cards to Done.
- **Worktree is not returned to the pool** on direct merge. Only `MergeAndCleanup` (webhook path) returns worktrees.
- **No notification emitted** beyond the agent status change (which the dashboard watcher picks up via polling).

The PR-review path works correctly because Gitea webhooks trigger `handleReviewApproved` → `MergeAndCleanup` → board move + worktree return. But the direct-merge path (default, no `--with-review`) has no equivalent post-completion handler.

Additionally, users need a way to test the completion pipeline or manually mark tracks as done without running an actual agent.

## Codebase Analysis

### Current completion flow (direct merge, broken)

```
Developer skill exits (process exit code 0)
  → monitorAgent (spawner.go:258) calls UpdateStatus(agentID, "completed")
  → AgentStore.Save()
  → Dashboard watcher detects status change (2s poll)
  → SSE broadcasts agent_update
  ✗ Board card NOT moved to Done
  ✗ Worktree NOT returned to pool
```

### Current completion flow (PR path, works)

```
Gitea webhook: review approved
  → handleReviewApproved (server.go:533)
  → MergeAndCleanup (cleanup_service.go:22)
    → Merge PR, post comment, delete branch
    → ReturnByTrackID (pool return)
    → HaltAgent + UpdateStatus("completed")
  → MoveCard(slug, trackID, ColumnDone)
```

### Files to modify

**Spawner (completion callback):**
- `backend/internal/adapter/agent/spawner.go` — `monitorAgent()` needs a completion callback. Add a `CompletionCallback` function field to `Spawner` that's called after process exit with agent ID, status, and ref (track ID).

**CLI (wire callback + dry-run):**
- `backend/internal/adapter/cli/implement.go` — wire up the completion callback (board move + worktree return). Add `--dry-run` flag.

**Orchestrator server (wire callback):**
- `backend/internal/adapter/rest/server.go` — when spawning agents via API, provide the same completion callback.

### Existing patterns
- `MergeAndCleanup` in `cleanup_service.go` handles the PR path cleanup
- `LifecycleService` in `lifecycle_service.go` handles board-driven agent control (halt, resume, reject)
- `NativeBoardService.MoveCard()` moves cards between columns
- `Pool.ReturnByTrackID()` returns worktrees

### Completion callback design

```go
type CompletionCallback func(agentID, ref, status string)
```

The callback should:
1. If status is "completed" and ref looks like a track ID:
   - Move board card to `ColumnDone` for the agent's project
   - Return worktree to pool via `ReturnByTrackID`
2. If status is "failed":
   - Move board card back to `ColumnBacklog` (or leave in progress — TBD)
   - Return worktree to pool

### Dry-run design

`kf implement --dry-run <track-id>`:
1. Validate track exists and is pending
2. Move board card: Backlog → In Progress → Done
3. Mark track as `[x]` in `tracks.md` (commit)
4. Return immediately — no agent spawned, no worktree acquired
5. Print completion summary

This exercises the same board-move and notification pipeline without burning tokens.

## Acceptance Criteria

- [ ] When an agent completes (direct merge, no PR), the board card is moved to Done
- [ ] When an agent completes (direct merge), the worktree is returned to the pool
- [ ] When an agent fails, the worktree is returned to the pool
- [ ] Completion callback is wired in both CLI (`kf implement`) and REST server agent spawning
- [ ] `kf implement --dry-run <track-id>` skips agent spawning and immediately completes the track
- [ ] Dry-run moves the board card through the expected states (→ Done)
- [ ] Dry-run marks the track `[x]` in `tracks.md` and commits
- [ ] Dry-run prints a summary of what it did
- [ ] Existing PR-review path is not affected
- [ ] Tests cover completion callback for both success and failure
- [ ] `go test ./...` passes
- [ ] `make build` succeeds

## Dependencies

None.

## Blockers

None.

## Conflict Risk

- **sse-event-bus_20260309091500Z** — low risk. That track refactors SSE infrastructure; this track adds post-completion logic in the spawner. Different concerns.
- **quota-reframe-be_20260309103000Z** — low risk. Both touch `spawner.go` but in different sections.
- **model-selection_20260309110000Z** — low risk. Both touch `spawner.go` but model selection adds a flag, this adds a callback.

## Out of Scope

- Changing the PR-review path (already works via webhooks)
- Adding retry logic for failed agents
- Agent restart/resume from dry-run
- Frontend changes (board updates flow through existing SSE)

## Technical Notes

### CompletionCallback Integration

The callback needs access to board service and pool — neither of which the spawner currently has. Two approaches:

**Option A (preferred): Closure injection.** The caller (implement.go, server.go) creates a closure with captured dependencies and passes it to the spawner:

```go
spawner.OnCompletion = func(agentID, ref, status string) {
    if status == "completed" {
        boardSvc.MoveCard(slug, ref, domain.ColumnDone)
    }
    pool.ReturnByTrackID(ref)
}
```

**Option B: Event-based.** Spawner emits an event, orchestrator subscribes. More complex, better for future extensibility. Overkill for now.

### Dry-Run Flow

```
$ kf implement --dry-run my-track-id

Dry run: skipping agent spawn for track "my-track-id"
  Board:     Backlog → Done
  Track:     marked [x] in tracks.md
  Worktree:  not acquired (dry run)
  Agent:     not spawned (dry run)

Done. Track "my-track-id" marked complete.
```

---

_Generated by conductor-track-generator from prompt: "agent completion callback and dry-run mode"_
