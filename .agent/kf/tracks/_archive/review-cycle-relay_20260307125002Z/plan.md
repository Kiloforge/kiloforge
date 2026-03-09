# Implementation Plan: Developer-Reviewer Relay Cycle

**Track ID:** review-cycle-relay_20260307125002Z

## Phase 1: Reviewer Spawning and PR Tracking

### Task 1.1: Spawn reviewer on PR opened [x]
- Update `handlePullRequest()` in relay server
- On `opened`/`reopened`: look up PR tracking record, spawn reviewer agent
- Reviewer prompt: `/conductor-reviewer <pr-url>`
- Record reviewer agent ID and session ID in PR tracking
- Post comment on PR with reviewer session info
- Tests: verify reviewer spawned on PR opened webhook

### Task 1.2: Add Gitea API methods for PR management [x]
- `AddLabel(ctx, repo, prNum, label)` — create label if needed, add to PR
- `CommentOnPR(ctx, repo, prNum, body)` — post comment
- `GetPRReviews(ctx, repo, prNum)` — fetch review comments for developer context
- Tests: verify API call structure

### Task 1.3: Track review cycle state [x]
- Extend PR tracking with `ReviewCycleCount` counter
- Increment on each `changes_requested` → developer resume cycle
- Load/save with existing PR tracking persistence
- Tests: verify cycle counting

### Verification 1
- [x] Reviewer spawned on PR opened
- [x] PR tracking records reviewer info
- [x] Cycle counting works
- [x] Tests pass

## Phase 2: Review Response Handling

### Task 2.1: Handle review approved [x]
- On `pull_request_review` with state `approved`:
  - Look up developer agent from PR tracking
  - Resume developer with `claude --resume <session-id>`
  - Inject context: "PR approved, proceed to merge"
  - Update PR tracking: status=approved
  - Update developer agent: status=running
- Tests: verify developer resumed on approval

### Task 2.2: Handle changes requested [x]
- On `pull_request_review` with state `changes_requested`:
  - Check cycle count vs max
  - If under limit: resume developer with review comments context
  - Update PR tracking: cycle count++, status=changes-requested
  - Update developer agent: status=running
- Tests: verify developer resumed with cycle increment

### Task 2.3: Handle developer push (synchronize) [x]
- On `pull_request.synchronize`:
  - Developer pushed revisions
  - Update developer agent: status=waiting-review
  - Spawn new reviewer agent for re-review
- Tests: verify re-review cycle

### Task 2.4: Implement escalation [x]
- When cycle count >= max:
  - Label PR `needs-human-review`
  - Post comment explaining cycle limit reached
  - Stop all agents for this PR
  - Update PR tracking: status=escalated
- Tests: verify escalation triggers at limit

### Verification 2
- [x] Approval resumes developer
- [x] Changes requested resumes developer with feedback
- [x] Push triggers re-review
- [x] Escalation at cycle limit
- [x] Tests pass

## Phase 3: CLI and Integration

### Task 3.1: Implement `kf escalated` command [x]
- Create `internal/cli/escalated.go`
- Shows PRs that hit review cycle limit
- Table: `PROJECT  PR#  TRACK  CYCLES  ESCALATED_AT`
- Register in root.go

### Task 3.2: End-to-end integration test [x]
- Full cycle: implement → PR → review → changes requested → revise → re-review → approve
- Verify all state transitions
- Verify escalation at cycle limit

### Task 3.3: Update docs [x]
- Document review cycle in README
- Document escalation behavior
- Update architecture diagram with review cycle flow

### Verification 3
- [x] `kf escalated` shows escalated PRs
- [x] Full review cycle works end-to-end
- [x] Docs updated
- [x] Build and tests pass
