# Specification: Webhook-Driven Board State Synchronization

**Track ID:** impl-board-webhook-sync_20260308170002Z
**Type:** Feature
**Created:** 2026-03-08T17:00:02Z
**Status:** Draft

## Summary

Handle Gitea issue and label webhook events to keep the kanban board in sync with track state. When a user moves a card on the board, approves a track via label, or when the system creates/merges a PR, the corresponding track state and board position update automatically.

## Context

Track 2 establishes one-way sync: tracks → Gitea issues on a kanban board. This track closes the loop with bidirectional sync:

1. **Board → Track**: User moves a card from "Suggested" to "Approved" on the Gitea board → track state updates
2. **System → Board**: `crelay implement` spawns a developer → issue moves to "In Progress"; PR created → "In Review"; PR merged → "Completed"

The relay server already handles issue webhook events but only logs them. This track adds meaningful actions to those handlers.

## Codebase Analysis

- **Webhook handlers**: `internal/adapter/rest/server.go` — `handleIssues()` logs action/number/title but takes no action
- **Issue events subscribed**: `issues` (opened, edited, closed, label_updated, assigned) — already in webhook config
- **PR lifecycle**: `handlePullRequest()` and `handlePullRequestReview()` — already handle PR open/merge/review
- **Board service**: Will exist after impl-track-board-sync (provides `BoardService`, `BoardConfig`, `TrackIssue` types)
- **Track-to-branch mapping**: PRTracking already links `TrackID` (branch name) to PR number
- **Label-based state**: When `status:approved` label is added to an issue, we can detect that via `label_updated` event

### Event → Action mapping

| Gitea Event | Action | Board Effect |
|-------------|--------|-------------|
| `issues` → `label_updated` (status:approved added) | Update track state | Move card to Approved column |
| `issues` → `closed` | Mark track complete | Move card to Completed column |
| `issues` → `assigned` | Mark track in-progress | Move card to In Progress column |
| `crelay implement <track-id>` | System action | Move issue to In Progress, assign |
| PR opened (linked to track issue) | System action | Move issue to In Review |
| PR merged (review approved) | System action | Move issue to Completed, close |

## Acceptance Criteria

- [ ] `handleIssues()` label_updated → detect status label changes, move card to matching column
- [ ] `handleIssues()` closed → move card to Completed column
- [ ] `handleIssues()` assigned → move card to In Progress column (if currently in Suggested/Approved)
- [ ] `crelay implement` updates Gitea issue — moves to In Progress, updates labels
- [ ] PR creation links to track issue — moves to In Review column
- [ ] PR merge closes track issue — moves to Completed column
- [ ] Board config loaded in relay server startup
- [ ] Guard against event loops — system-triggered updates should not re-trigger webhooks, or be detected and ignored
- [ ] Unit tests for all event handlers with mock board service
- [ ] All existing tests pass, build succeeds

## Dependencies

- `impl-gitea-issue-api_20260308170000Z` — provides Gitea API methods
- `impl-track-board-sync_20260308170001Z` — provides BoardService, domain types, persistence

## Blockers

None

## Conflict Risk

- **MEDIUM** — Modifies `rest/server.go` webhook handlers (handleIssues, handlePullRequest). Also modifies `cli/implement.go` and PR lifecycle code. These are existing files with active logic.

## Out of Scope

- Updating tracks.md from webhook events (filesystem writes from relay would conflict with agent worktrees)
- Creating tracks from Gitea issues (tracks are created by conductor agents)
- Multi-user permissions on the board
- Drag-and-drop column ordering in Gitea (Gitea handles this natively)

## Technical Notes

### Event loop prevention

When the relay moves a card or updates labels (system-triggered), Gitea fires a new webhook. To prevent infinite loops:

```go
// Option 1: Track "last actor" — ignore events from our own API token
if eventUser == config.GiteaAdminUser {
    // System-triggered event, skip
    return
}

// Option 2: Track pending operations with short TTL
type pendingOp struct {
    issueNum int
    action   string
    expires  time.Time
}
```

Option 1 is simpler — check if the event was triggered by the admin user (our bot).

### Linking PR to track issue

When a developer creates a PR, the PR body or branch name contains the track ID. The relay already extracts this via `head.ref` in the PR webhook. To link:

```go
// In handlePullRequest "opened":
// 1. Look up TrackIssue by branch name (track ID)
// 2. If found, move issue to "In Review" column
// 3. Add comment to issue: "PR #N opened for this track"
```

### Implement command integration

```go
// In crelay implement:
// After spawning developer agent:
// 1. Look up TrackIssue for this track ID
// 2. Move card to "In Progress" column
// 3. Update labels: remove status:approved, add status:in-progress
// 4. Assign issue to developer agent (or admin user)
```

---

_Generated by conductor-track-generator_
