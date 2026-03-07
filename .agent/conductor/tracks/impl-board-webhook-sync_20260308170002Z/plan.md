# Implementation Plan: Webhook-Driven Board State Synchronization

**Track ID:** impl-board-webhook-sync_20260308170002Z

## Phase 1: Issue Event Handlers (4 tasks)

### Task 1.1: Load board config in relay server
- [ ] Load BoardConfig per project in relay server startup (or lazily on first event)
- [ ] Store board service reference in Server struct
- [ ] Pass board service to webhook handlers

### Task 1.2: Handle label_updated events
- [ ] In `handleIssues()`, detect `label_updated` action
- [ ] Extract labels from issue payload
- [ ] If status label changed (e.g., `status:approved` added), determine target column
- [ ] Move card to matching column via BoardService
- [ ] Skip if event triggered by admin user (prevent loops)

### Task 1.3: Handle issue closed and assigned events
- [ ] `closed` action → move card to Completed column, update TrackIssue mapping
- [ ] `assigned` action → if issue is in Suggested/Approved column, move to In Progress
- [ ] Update status labels to match new column
- [ ] Skip if event triggered by admin user

### Task 1.4: Event loop prevention
- [ ] Check `sender` field in webhook payload against `config.GiteaAdminUser`
- [ ] If sender matches admin user, skip processing (system-triggered event)
- [ ] Log skipped events for debugging

## Phase 2: Bidirectional Sync (3 tasks)

### Task 2.1: Update implement command for board sync
- [ ] After spawning developer in `crelay implement`, look up TrackIssue
- [ ] Move card to "In Progress" column
- [ ] Update labels: add `status:in-progress`, remove `status:approved`/`status:suggested`
- [ ] Post comment on issue: "Developer agent spawned — implementation started"

### Task 2.2: PR creation → In Review sync
- [ ] In `handlePullRequest()` on `opened` action, look up TrackIssue by branch name (track ID)
- [ ] If found, move card to "In Review" column
- [ ] Update labels: add `status:in-review`, remove `status:in-progress`
- [ ] Post comment on issue: "PR #{number} opened — under review"

### Task 2.3: PR merge → Completed sync
- [ ] In `handleReviewApproved()` after successful merge, look up TrackIssue
- [ ] Move card to "Completed" column
- [ ] Close the Gitea issue via UpdateIssue(state: "closed")
- [ ] Update labels: add `status:completed`
- [ ] Post comment on issue: "PR #{number} merged — track complete"

## Phase 3: Tests and Verification (3 tasks)

### Task 3.1: Unit tests for issue event handlers
- [ ] Test label_updated with status label change → card moves
- [ ] Test closed → card moves to Completed
- [ ] Test assigned → card moves to In Progress
- [ ] Test admin user events are skipped (no loop)

### Task 3.2: Unit tests for bidirectional sync
- [ ] Test implement → card moves to In Progress
- [ ] Test PR opened → card moves to In Review
- [ ] Test PR merged → card moves to Completed, issue closed

### Task 3.3: Full build and test
- [ ] `go build -buildvcs=false ./...`
- [ ] `go test -buildvcs=false -race ./...`
- [ ] No regressions across all packages

---

**Total: 10 tasks across 3 phases**
