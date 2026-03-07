# Implementation Plan: Board-Driven Agent Lifecycle Control

**Track ID:** board-agent-lifecycle_20260308180000Z

## Phase 1: Agent Resolution by Track (3 tasks)

### Task 1.1: Add FindByRef to AgentStore
- [ ] Add `FindByRef(ref string) *AgentInfo` method to `port.AgentStore` interface
- [ ] Implement in `jsonfile.AgentStore` — iterate agents, match on `Ref` field
- [ ] Return most recent matching agent (by `StartedAt`) if multiple exist
- [ ] Add unit test for FindByRef with single match, multiple matches, and no match

### Task 1.2: Add issue-to-agent resolution helper
- [ ] Create `resolveAgentForIssue(issueNumber int, slug string) (*domain.AgentInfo, error)` on Server
- [ ] Load TrackIssue mapping (from BoardService, provided by impl-track-board-sync)
- [ ] Reverse-lookup: issue number → track ID
- [ ] Call `store.FindByRef(trackID)` to get the developer agent
- [ ] Return nil if no agent found (track not yet implemented)

### Task 1.3: Add PR tracking lookup for reviewer agent
- [ ] Create `resolveReviewerForTrack(trackID, slug string) (*domain.AgentInfo, string)` on Server
- [ ] Load PRTracking for the project slug
- [ ] If PRTracking exists and `TrackID` matches, return reviewer agent info + session
- [ ] Return nil if no PR exists yet for this track

## Phase 2: Backward Movement Handlers (4 tasks)

### Task 2.1: Detect backward label transitions
- [ ] In `handleIssues()` for `label_updated` action, extract current labels from payload
- [ ] Define column ordering: Suggested(0) < Approved(1) < InProgress(2) < InReview(3) < Completed(4)
- [ ] Determine current column from labels (highest-priority `status:*` label)
- [ ] Compare against agent state to detect backward movement
- [ ] Skip if sender matches `config.GiteaAdminUser` (loop prevention)

### Task 2.2: Handle In Progress → Approved/Suggested (halt developer)
- [ ] Resolve agent via `resolveAgentForIssue()`
- [ ] If agent status is `running` or `waiting`, call `store.HaltAgent(agent.ID)`
- [ ] Update agent status to `halted`
- [ ] Set `agent.ShutdownReason = "board-demotion"`
- [ ] Save agent store
- [ ] Post comment on issue: "Developer agent halted — track moved back to {column}"
- [ ] Log the halt action

### Task 2.3: Handle In Review → backward (halt reviewer + developer)
- [ ] Resolve both developer and reviewer agents
- [ ] Halt reviewer agent if running (SIGINT → `halted`)
- [ ] Halt developer agent if running/waiting (SIGINT → `halted`)
- [ ] Update PRTracking status to reflect demotion
- [ ] Post comment on issue: "Review paused — track moved back to {column}"
- [ ] Save all state changes

### Task 2.4: Handle edge cases for backward moves
- [ ] Agent already completed → skip halt, log info
- [ ] Agent already halted → skip halt, idempotent
- [ ] Agent not found (track never implemented) → skip, no-op
- [ ] Process already dead (PID stale) → mark `halted` without SIGINT, log warning

## Phase 3: Forward Re-promotion (3 tasks)

### Task 3.1: Handle re-promotion to In Progress (resume developer)
- [ ] In `label_updated` handler, detect forward move from Approved/Suggested → In Progress
- [ ] Check if a halted developer agent exists for this track
- [ ] Validate: SessionID not empty, WorktreeDir exists on disk
- [ ] Call `spawner.ResumeDeveloper(ctx, sessionID, workDir)`
- [ ] Update agent status to `running`, clear `ShutdownReason`
- [ ] Save agent store
- [ ] Post comment on issue: "Developer agent resumed — implementation continuing"

### Task 3.2: Handle resume failures
- [ ] If SessionID is empty → mark `resume-failed`, reason: "no session to resume"
- [ ] If WorktreeDir missing → mark `resume-failed`, reason: "worktree not found"
- [ ] If `ResumeDeveloper()` returns error → mark `resume-failed`, record error message
- [ ] Post comment on issue: "Could not resume agent: {reason}. Use `crelay implement {trackID}` to restart."

### Task 3.3: Handle re-promotion to In Review
- [ ] Detect forward move to In Review for a halted track
- [ ] Resume reviewer agent if halted (similar to developer resume)
- [ ] If no reviewer exists, log and skip (PR may need to be re-opened)

## Phase 4: Track Rejection and Cancellation (3 tasks)

### Task 4.1: Handle issue closed without merge (rejection)
- [ ] In `handleIssues()` for `closed` action, resolve agent for the issue
- [ ] Check if associated PR was merged (load PRTracking, check status)
- [ ] If NOT merged (rejection): halt agent with SIGINT if running
- [ ] Update agent status to `stopped`, set `ShutdownReason = "track-rejected"`
- [ ] Return worktree to pool via `pool.ReturnByTrackID(trackID)`
- [ ] Post comment: "Track rejected — agent terminated, worktree returned to pool"

### Task 4.2: Handle `rejected` label
- [ ] In `label_updated` handler, check for `rejected` label
- [ ] If present: same termination flow as Task 4.1
- [ ] Close the issue via Gitea API (`UpdateIssue(state: "closed")`)
- [ ] Remove other status labels, keep only `rejected`

### Task 4.3: Cleanup edge cases
- [ ] If agent already stopped/completed → skip termination, just clean up worktree
- [ ] If no worktree assigned (track never started) → skip pool return
- [ ] If PRTracking exists with open PR → post comment on PR: "Associated issue rejected"

## Phase 5: Tests (3 tasks)

### Task 5.1: Unit tests for backward movement handlers
- [ ] Test In Progress → Approved halts developer agent
- [ ] Test In Review → Approved halts both agents
- [ ] Test backward move with already-halted agent (idempotent)
- [ ] Test backward move with no agent (no-op)
- [ ] Test admin user events are skipped

### Task 5.2: Unit tests for re-promotion and resume
- [ ] Test Approved → In Progress resumes halted developer
- [ ] Test resume with missing session → resume-failed
- [ ] Test resume with missing worktree → resume-failed
- [ ] Test re-promotion of non-halted agent (no-op)

### Task 5.3: Unit tests for rejection flow
- [ ] Test issue closed without merge → agent stopped, worktree returned
- [ ] Test `rejected` label → agent stopped, issue closed
- [ ] Test rejection with no active agent (no-op)
- [ ] Full build: `go build -buildvcs=false ./...`
- [ ] Full test: `go test -buildvcs=false -race ./...`

---

**Total: 16 tasks across 5 phases**
