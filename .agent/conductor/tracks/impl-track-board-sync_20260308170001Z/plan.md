# Implementation Plan: Track-to-Gitea Board Sync Service

**Track ID:** impl-track-board-sync_20260308170001Z

## Phase 1: Domain Types and Persistence (3 tasks)

### Task 1.1: Define BoardConfig domain type
- [x] `domain.BoardConfig` struct: ProjectBoardID, column IDs map (suggested/approved/in-progress/in-review/completed), label ID map
- [x] `domain.TrackIssue` struct: TrackID, IssueNumber, CardID, Column, LastSynced
- [x] `domain.LabelDef` struct: Name, Color, ID (for tracking created label IDs)

### Task 1.2: Implement BoardStore persistence
- [x] `jsonfile.BoardStore` — load/save `board.json` per project directory
- [x] `GetBoardConfig(projectDir) → (*BoardConfig, error)`
- [x] `SaveBoardConfig(projectDir, config) → error`
- [x] `GetTrackIssue(projectDir, trackID) → (*TrackIssue, error)`
- [x] `SaveTrackIssue(projectDir, mapping) → error`
- [x] `ListTrackIssues(projectDir) → ([]TrackIssue, error)`

### Task 1.3: Extend TrackEntry with new states
- [x] Add `StatusSuggested = "suggested"` and `StatusApproved = "approved"` constants
- [x] Update `parseTrackLine()` to recognize `[!]` as approved status
- [x] Keep `[ ]` as suggested (rename from "pending" — backward compatible)
- [x] Add `StatusInReview = "in-review"` for completeness

## Phase 2: Board Setup Service (3 tasks)

### Task 2.1: Implement BoardService.SetupBoard
- [x] Define `BoardService` struct with Gitea client and board store dependencies
- [x] `SetupBoard(ctx, project) → (*BoardConfig, error)`
- [x] Create standard labels via `EnsureLabels`: type:feature (blue), type:bug (red), type:refactor (yellow), type:chore (gray), status:suggested (white), status:approved (green), status:in-progress (orange), status:in-review (purple)
- [x] Create project board: "Tracks" with description
- [x] Create 5 columns in order: Suggested, Approved, In Progress, In Review, Completed
- [x] Save BoardConfig to disk
- [x] Idempotent — skip if board already exists

### Task 2.2: Integrate board setup into `crelay add`
- [x] After creating Gitea repo and webhook, call `BoardService.SetupBoard()`
- [x] Store BoardConfig in project's data directory
- [x] Log board creation URL

### Task 2.3: Add `crelay board` CLI command
- [x] Show board status for a project (board URL, column counts)
- [x] `--setup` flag to re-run board setup if needed
- [x] Default to current project if `--project` not specified

## Phase 3: Track Publishing (5 tasks)

### Task 3.1: Implement PublishTrack
- [x] `BoardService.PublishTrack(ctx, project, track, specContent) → error`
- [x] Create Gitea issue: title = track.Title, body = specContent (markdown)
- [x] Add labels: type label based on track type, status label based on track state
- [x] Create card in appropriate column based on track status
- [x] Save TrackIssue mapping to board store
- [x] Skip if track already published (idempotent)

### Task 3.2: Implement SyncTracks
- [x] `BoardService.SyncTracks(ctx, project, tracks []TrackEntry) → SyncResult`
- [x] Load existing TrackIssue mappings
- [x] Diff: identify new tracks (not published), changed tracks (status changed), stale mappings
- [x] Publish new tracks
- [x] Update existing issues: move card to correct column, update labels
- [x] Return SyncResult with counts (created, updated, unchanged)

### Task 3.3: Read track spec content for issue body
- [x] Helper: `ReadTrackSpec(projectDir, trackID) → (string, error)`
- [x] Read from `.agent/conductor/tracks/{trackID}/spec.md`
- [x] Fall back to `.agent/conductor/tracks/_archive/{trackID}/spec.md`
- [x] If no spec found, use track title as body

### Task 3.4: Implement `crelay sync` CLI command
- [x] `crelay sync [--project slug]` — sync tracks to Gitea board
- [x] Discover tracks from project directory
- [x] Call SyncTracks
- [x] Print sync results (created N, updated N, unchanged N)

### Task 3.5: Track status ↔ column mapping
- [x] Define mapping function: `StatusToColumn(status) → columnName`
- [x] Define reverse: `ColumnToStatus(column) → status`
- [x] suggested → Suggested, approved → Approved, in-progress → In Progress, in-review → In Review, complete → Completed

## Phase 4: Tests (3 tasks)

### Task 4.1: Unit tests for BoardService
- [x] Test SetupBoard with mock Gitea client (labels created, board created, columns created)
- [x] Test SetupBoard idempotent (board already exists)
- [x] Test PublishTrack (issue created, card placed, mapping saved)
- [x] Test SyncTracks (new tracks published, changed tracks updated)

### Task 4.2: Unit tests for BoardStore persistence
- [x] Round-trip: save → load board config
- [x] Round-trip: save → load track issue mappings
- [x] Missing file → empty state
- [x] Corrupt file → graceful error

### Task 4.3: Build and test verification
- [x] `go build -buildvcs=false ./...`
- [x] `go test -buildvcs=false -race ./...`
- [x] No regressions

---

**Total: 14 tasks across 4 phases**
