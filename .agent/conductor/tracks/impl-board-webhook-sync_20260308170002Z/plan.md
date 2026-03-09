# Implementation Plan: Webhook-Driven Board State Synchronization

**Track ID:** impl-board-webhook-sync_20260308170002Z

## Phase 1: Issue Event Handlers (4 tasks)

### Task 1.1: Load board config in relay server
- [x] Load BoardConfig per project in relay server startup (or lazily on first event)
- [x] Store board service reference in Server struct
- [x] Pass board service to webhook handlers

### Task 1.2: Handle label_updated events
- [x] In `handleIssues()`, detect `label_updated` action
- [x] Extract labels from issue payload
- [x] If status label changed (e.g., `status:approved` added), determine target column
- [x] Move card to matching column via BoardService
- [x] Skip if event triggered by admin user (prevent loops)

### Task 1.3: Handle issue closed and assigned events
- [x] `closed` action → move card to Completed column, update TrackIssue mapping
- [x] `assigned` action → if issue is in Suggested/Approved column, move to In Progress
- [x] Update status labels to match new column
- [x] Skip if event triggered by admin user

### Task 1.4: Event loop prevention
- [x] Check `sender` field in webhook payload against `config.GiteaAdminUser`
- [x] If sender matches admin user, skip processing (system-triggered event)
- [x] Log skipped events for debugging

## Phase 2: Bidirectional Sync (3 tasks)

### Task 2.1: Update implement command for board sync
- [x] After spawning developer in `kf implement`, look up TrackIssue
- [x] Move card to "In Progress" column
- [x] Update labels: add `status:in-progress`, remove `status:approved`/`status:suggested`
- [x] Post comment on issue: "Developer agent spawned — implementation started"

### Task 2.2: PR creation → In Review sync
- [x] In `handlePullRequest()` on `opened` action, look up TrackIssue by branch name (track ID)
- [x] If found, move card to "In Review" column
- [x] Update labels: add `status:in-review`, remove `status:in-progress`
- [x] Post comment on issue: "PR #{number} opened — under review"

### Task 2.3: PR merge → Completed sync
- [x] In `handleReviewApproved()` after successful merge, look up TrackIssue
- [x] Move card to "Completed" column
- [x] Close the Gitea issue via UpdateIssue(state: "closed")
- [x] Update labels: add `status:completed`
- [x] Post comment on issue: "PR #{number} merged — track complete"

## Phase 3: Tests and Verification (3 tasks)

### Task 3.1: Unit tests for issue event handlers
- [x] Test label_updated with status label change → card moves
- [x] Test closed → card moves to Completed
- [x] Test assigned → card moves to In Progress
- [x] Test admin user events are skipped (no loop)

### Task 3.2: Unit tests for bidirectional sync
- [x] Test implement → card moves to In Progress
- [x] Test PR opened → card moves to In Review
- [x] Test PR merged → card moves to Completed, issue closed

### Task 3.3: Full build and test
- [x] `go build -buildvcs=false ./...`
- [x] `go test -buildvcs=false -race ./...`
- [x] No regressions across all packages

---

**Total: 10 tasks across 3 phases**
