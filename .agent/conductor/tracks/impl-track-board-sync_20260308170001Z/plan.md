# Implementation Plan: Track-to-Gitea Board Sync Service

**Track ID:** impl-track-board-sync_20260308170001Z

## Phase 1: Domain Types and Persistence (3 tasks)

### Task 1.1: Define BoardConfig domain type
- [ ] `domain.BoardConfig` struct: ProjectBoardID, column IDs map (suggested/approved/in-progress/in-review/completed), label ID map
- [ ] `domain.TrackIssue` struct: TrackID, IssueNumber, CardID, Column, LastSynced
- [ ] `domain.LabelDef` struct: Name, Color, ID (for tracking created label IDs)

### Task 1.2: Implement BoardStore persistence
- [ ] `jsonfile.BoardStore` — load/save `board.json` per project directory
- [ ] `GetBoardConfig(projectDir) → (*BoardConfig, error)`
- [ ] `SaveBoardConfig(projectDir, config) → error`
- [ ] `GetTrackIssue(projectDir, trackID) → (*TrackIssue, error)`
- [ ] `SaveTrackIssue(projectDir, mapping) → error`
- [ ] `ListTrackIssues(projectDir) → ([]TrackIssue, error)`

### Task 1.3: Extend TrackEntry with new states
- [ ] Add `StatusSuggested = "suggested"` and `StatusApproved = "approved"` constants
- [ ] Update `parseTrackLine()` to recognize `[!]` as approved status
- [ ] Keep `[ ]` as suggested (rename from "pending" — backward compatible)
- [ ] Add `StatusInReview = "in-review"` for completeness

## Phase 2: Board Setup Service (3 tasks)

### Task 2.1: Implement BoardService.SetupBoard
- [ ] Define `BoardService` struct with Gitea client and board store dependencies
- [ ] `SetupBoard(ctx, project) → (*BoardConfig, error)`
- [ ] Create standard labels via `EnsureLabels`: type:feature (blue), type:bug (red), type:refactor (yellow), type:chore (gray), status:suggested (white), status:approved (green), status:in-progress (orange), status:in-review (purple)
- [ ] Create project board: "Tracks" with description
- [ ] Create 5 columns in order: Suggested, Approved, In Progress, In Review, Completed
- [ ] Save BoardConfig to disk
- [ ] Idempotent — skip if board already exists

### Task 2.2: Integrate board setup into `crelay add`
- [ ] After creating Gitea repo and webhook, call `BoardService.SetupBoard()`
- [ ] Store BoardConfig in project's data directory
- [ ] Log board creation URL

### Task 2.3: Add `crelay board` CLI command
- [ ] Show board status for a project (board URL, column counts)
- [ ] `--setup` flag to re-run board setup if needed
- [ ] Default to current project if `--project` not specified

## Phase 3: Track Publishing (5 tasks)

### Task 3.1: Implement PublishTrack
- [ ] `BoardService.PublishTrack(ctx, project, track, specContent) → error`
- [ ] Create Gitea issue: title = track.Title, body = specContent (markdown)
- [ ] Add labels: type label based on track type, status label based on track state
- [ ] Create card in appropriate column based on track status
- [ ] Save TrackIssue mapping to board store
- [ ] Skip if track already published (idempotent)

### Task 3.2: Implement SyncTracks
- [ ] `BoardService.SyncTracks(ctx, project, tracks []TrackEntry) → SyncResult`
- [ ] Load existing TrackIssue mappings
- [ ] Diff: identify new tracks (not published), changed tracks (status changed), stale mappings
- [ ] Publish new tracks
- [ ] Update existing issues: move card to correct column, update labels
- [ ] Return SyncResult with counts (created, updated, unchanged)

### Task 3.3: Read track spec content for issue body
- [ ] Helper: `ReadTrackSpec(projectDir, trackID) → (string, error)`
- [ ] Read from `.agent/conductor/tracks/{trackID}/spec.md`
- [ ] Fall back to `.agent/conductor/tracks/_archive/{trackID}/spec.md`
- [ ] If no spec found, use track title as body

### Task 3.4: Implement `crelay sync` CLI command
- [ ] `crelay sync [--project slug]` — sync tracks to Gitea board
- [ ] Discover tracks from project directory
- [ ] Call SyncTracks
- [ ] Print sync results (created N, updated N, unchanged N)

### Task 3.5: Track status ↔ column mapping
- [ ] Define mapping function: `StatusToColumn(status) → columnName`
- [ ] Define reverse: `ColumnToStatus(column) → status`
- [ ] suggested → Suggested, approved → Approved, in-progress → In Progress, in-review → In Review, complete → Completed

## Phase 4: Tests (3 tasks)

### Task 4.1: Unit tests for BoardService
- [ ] Test SetupBoard with mock Gitea client (labels created, board created, columns created)
- [ ] Test SetupBoard idempotent (board already exists)
- [ ] Test PublishTrack (issue created, card placed, mapping saved)
- [ ] Test SyncTracks (new tracks published, changed tracks updated)

### Task 4.2: Unit tests for BoardStore persistence
- [ ] Round-trip: save → load board config
- [ ] Round-trip: save → load track issue mappings
- [ ] Missing file → empty state
- [ ] Corrupt file → graceful error

### Task 4.3: Build and test verification
- [ ] `go build -buildvcs=false ./...`
- [ ] `go test -buildvcs=false -race ./...`
- [ ] No regressions

---

**Total: 14 tasks across 4 phases**
