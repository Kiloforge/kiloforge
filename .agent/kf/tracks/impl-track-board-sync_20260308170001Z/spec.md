# Specification: Track-to-Gitea Board Sync Service

**Track ID:** impl-track-board-sync_20260308170001Z
**Type:** Feature
**Created:** 2026-03-08T17:00:01Z
**Status:** Draft

## Summary

Domain types for track-issue mapping, kanban board setup on project registration, publishing conductor tracks as Gitea issues on a project board, and CLI commands for syncing. Tracks flow through kanban columns: Suggested → Approved → In Progress → In Review → Completed.

## Context

Conductor tracks currently live only in the filesystem (`.agent/conductor/tracks.md`). There is no visual representation of track state beyond the CLI. By syncing tracks to Gitea issues on a project kanban board, users get a visual overview of all work across states, can approve tracks by moving cards, and have a single place to monitor progress alongside PRs and code review.

## Codebase Analysis

- **Track service**: `internal/core/service/track_service.go` — `ParseTracks()`, `DiscoverTracks()`, `FilterByStatus()` parse tracks.md
- **TrackEntry**: `{ID, Title, Status}` — simple struct, status values: `complete`, `pending`, `in-progress`
- **Project registry**: `internal/adapter/persistence/jsonfile/project_store.go` — per-project metadata
- **Project domain**: `internal/core/domain/project.go` — Slug, RepoName, ProjectDir, OriginRemote
- **CLI add command**: `internal/adapter/cli/add.go` — registers project, creates Gitea repo, adds remote, pushes, creates webhook
- **Gitea client**: Will have issue/board APIs after impl-gitea-issue-api track

### New state model

Current tracks.md only has 3 states. We need 5 to match the kanban:

| State | tracks.md | Kanban Column | Meaning |
|-------|-----------|---------------|---------|
| Suggested | `[ ]` | Suggested | Track generated, awaiting approval |
| Approved | `[!]` | Approved | User approved, ready for a developer |
| In Progress | `[~]` | In Progress | Developer working on it |
| In Review | `[r]` | In Review | PR created, under review |
| Completed | `[x]` | Completed | PR merged, track done |

Note: We can keep `[ ]` for suggested (backward compatible) and introduce `[!]` for approved. The `[r]` marker is optional — in-review is a sub-state tracked via board position, not necessarily in tracks.md.

## Acceptance Criteria

- [ ] `domain.BoardConfig` — stores project board ID, column IDs, label definitions per project
- [ ] `domain.TrackIssue` — maps trackID ↔ Gitea issue number with sync metadata
- [ ] `jsonfile.BoardStore` — persistence for board config and track-issue mappings
- [ ] `service.BoardService.SetupBoard()` — create 5-column kanban + standard labels on Gitea
- [ ] Board setup integrated into `kf add` — automatic on project registration
- [ ] `service.BoardService.PublishTrack()` — create Gitea issue from track, place in correct column
- [ ] `service.BoardService.SyncTracks()` — diff local tracks vs published, create/update as needed
- [ ] `kf sync [--project slug]` CLI command — publish tracks to board
- [ ] `kf board [--project slug]` CLI command — show board status
- [ ] Track spec content used as issue body (markdown)
- [ ] Track type mapped to label (type:feature, type:bug, etc.)
- [ ] Track status mapped to kanban column position
- [ ] Unit tests for BoardService, BoardStore
- [ ] All existing tests pass, build succeeds

## Dependencies

- `impl-gitea-issue-api_20260308170000Z` — provides issue/board API methods on Gitea client

## Blockers

- **impl-board-webhook-sync_20260308170002Z** — depends on this track for board service and domain types

## Conflict Risk

- **MEDIUM** — Modifies `cli/add.go` to add board setup step. Adds new domain types, service, persistence, and CLI commands. Touches `service/track_service.go` to add new status parsing.

## Out of Scope

- Webhook-driven state sync (that's track 3)
- Bidirectional sync from Gitea → tracks.md (track 3)
- Authentication/permissions on the board
- Multi-project board aggregation (each project gets its own board)

## Technical Notes

### Board setup flow

```go
func (s *BoardService) SetupBoard(ctx context.Context, project domain.Project) (*domain.BoardConfig, error) {
    // 1. Create labels: type:feature, type:bug, type:refactor, type:chore,
    //    status:suggested, status:approved, status:in-progress, status:in-review
    // 2. Create project board: "Track Board"
    // 3. Create columns: Suggested, Approved, In Progress, In Review, Completed
    // 4. Save BoardConfig to disk
}
```

### Track publishing flow

```go
func (s *BoardService) PublishTrack(ctx context.Context, project domain.Project, track service.TrackEntry, specContent string) error {
    // 1. Check if already published (lookup in TrackIssue store)
    // 2. Create Gitea issue: title=track.Title, body=specContent, labels=[type:X, status:Y]
    // 3. Create card in appropriate column based on track status
    // 4. Save TrackIssue mapping
}
```

### Sync command flow

```go
// kiloforge sync
func runSync(cmd *cobra.Command, args []string) error {
    // 1. Load project registry
    // 2. For each project (or specified --project):
    //    a. DiscoverTracks(projectDir)
    //    b. Load existing TrackIssue mappings
    //    c. Diff: new tracks, changed status, removed tracks
    //    d. PublishTrack for new ones
    //    e. Update issue state/column for changed ones
}
```

### BoardConfig persistence

```json
// ~/.kiloforge/projects/<slug>/board.json
{
  "project_id": 1,
  "columns": {
    "suggested": 1,
    "approved": 2,
    "in_progress": 3,
    "in_review": 4,
    "completed": 5
  },
  "labels": {
    "type:feature": 1,
    "type:bug": 2,
    "status:suggested": 3
  },
  "track_issues": {
    "impl-foo_20260308Z": {
      "issue_number": 42,
      "card_id": 15,
      "column": "suggested",
      "last_synced": "2026-03-08T17:00:00Z"
    }
  }
}
```

---

_Generated by conductor-track-generator_
