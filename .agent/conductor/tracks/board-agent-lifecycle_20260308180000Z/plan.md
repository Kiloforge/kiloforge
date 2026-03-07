# Implementation Plan: Board-Driven Agent Lifecycle Control

**Track ID:** board-agent-lifecycle_20260308180000Z

## Phase 1: Agent Resolution by Track (3 tasks)

### Task 1.1: Add FindByRef to AgentStore
- [x] Add `FindByRef(ref string) *AgentInfo` method to `port.AgentStore` interface
- [x] Implement in `jsonfile.AgentStore` — iterate agents, match on `Ref` field
- [x] Return most recent matching agent (by `StartedAt`) if multiple exist
- [x] Add unit test for FindByRef with single match, multiple matches, and no match

### Task 1.2: Add issue-to-agent resolution helper
- [x] Reverse-lookup: issue number → track ID via TrackIssue mapping
- [x] Call `store.FindByRef(trackID)` to get the developer agent
- [x] Return nil if no agent found (track not yet implemented)

### Task 1.3: Add PR tracking lookup for reviewer agent
- [x] Load PRTracking for the project slug via prLoader function
- [x] If PRTracking exists and `TrackID` matches, use for reviewer resolution
- [x] Return nil if no PR exists yet for this track

## Phase 2: Backward Movement Handlers (4 tasks)

### Task 2.1: Detect backward label transitions
- [x] Define column ordering: Suggested(0) < Approved(1) < InProgress(2) < InReview(3) < Completed(4)
- [x] Compare previous column against new column to detect backward movement
- [x] Skip if sender matches `config.GiteaAdminUser` (loop prevention via isSelfTriggered)

### Task 2.2: Handle In Progress → Approved/Suggested (halt developer)
- [x] Resolve agent via FindByRef(trackID)
- [x] If agent status is `running` or `waiting`, call `store.HaltAgent(agent.ID)`
- [x] Update agent status to `halted`, set ShutdownReason = "board-demotion"
- [x] Post comment on issue, log the halt action

### Task 2.3: Handle In Review → backward (halt reviewer + developer)
- [x] Resolve both developer and reviewer agents
- [x] Halt both agents if running

### Task 2.4: Handle edge cases for backward moves
- [x] Agent already completed/stopped → skip halt, log info
- [x] Agent already halted → skip halt, idempotent
- [x] Agent not found → skip, no-op
- [x] Process already dead → mark halted anyway, log warning

## Phase 3: Forward Re-promotion (3 tasks)

### Task 3.1: Handle re-promotion to In Progress (resume developer)
- [x] Detect forward move, check for halted developer agent
- [x] Validate SessionID not empty, WorktreeDir exists
- [x] Call spawner.ResumeDeveloper, update status to running

### Task 3.2: Handle resume failures
- [x] Missing session → resume-failed
- [x] Missing worktree → resume-failed
- [x] Spawn error → resume-failed with ResumeError

### Task 3.3: Handle re-promotion to In Review
- [x] Log and skip (reviewer resume not directly supported)

## Phase 4: Track Rejection and Cancellation (3 tasks)

### Task 4.1: Handle issue closed without merge (rejection)
- [x] Detect closed without merged PR
- [x] Halt agent, set status to stopped, return worktree to pool

### Task 4.2: Handle `rejected` label
- [x] Detect rejected label in handleLabelUpdated
- [x] Terminate agent, close issue via Gitea API

### Task 4.3: Cleanup edge cases
- [x] Already stopped/completed → skip termination
- [x] No worktree assigned → skip pool return (nil pool check)

## Phase 5: Tests (3 tasks)

### Task 5.1: Unit tests for backward movement handlers
- [x] All backward move scenarios tested

### Task 5.2: Unit tests for re-promotion and resume
- [x] All resume scenarios tested

### Task 5.3: Unit tests for rejection flow
- [x] All rejection scenarios tested
- [x] Full build: `go build -buildvcs=false ./...`
- [x] Full test: `go test -buildvcs=false -race ./...`

---

**Total: 16 tasks across 5 phases**
