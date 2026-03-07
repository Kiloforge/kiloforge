# Implementation Plan: Graceful Agent Shutdown and Auto-Recovery on Restart

**Track ID:** impl-graceful-shutdown-recovery_20260307160002Z

## Phase 1: State Model Extensions (3 tasks)

### Task 1.1: Extend AgentInfo with shutdown/resume fields
- [ ] Add `ShutdownAt`, `ShutdownReason`, `ResumeError` fields to `AgentInfo`
- [ ] Add new status values: `suspended`, `suspending`, `force-killed`, `resume-failed`
- [ ] Ensure backwards compatibility: existing state.json without new fields loads correctly

### Task 1.2: Add batch agent operations to Store
- [ ] `RunningAgents() []AgentInfo` — filter agents with status "running" or "suspending"
- [ ] `SuspendedAgents() []AgentInfo` — filter agents with status "suspended"
- [ ] `BulkUpdateStatus(ids []string, status string)` — update multiple agents atomically

### Task 1.3: Write state model tests
- [ ] Test new fields serialize/deserialize correctly
- [ ] Test backwards compatibility with old state.json format
- [ ] Test batch operations with concurrent access

## Phase 2: Graceful Shutdown (4 tasks)

### Task 2.1: Implement shutdown manager
- [ ] Create `internal/agent/shutdown.go` with `ShutdownManager` struct
- [ ] `ShutdownAll(timeout time.Duration) ShutdownResult` — SIGINT all, wait, SIGKILL stragglers
- [ ] Track per-agent shutdown outcome

### Task 2.2: Integrate shutdown into relay lifecycle
- [ ] In `cli/up.go`: register shutdown hook that calls `ShutdownAll()` before server stops
- [ ] In `cli/down.go`: call `ShutdownAll()` before stopping Docker compose
- [ ] Set `shutting_down` flag to reject new spawn requests during shutdown

### Task 2.3: Update agent statuses on shutdown
- [ ] Agents that exit cleanly after SIGINT → status `suspended`
- [ ] Agents force-killed after timeout → status `force-killed`
- [ ] Save state.json after all agents handled

### Task 2.4: Write shutdown tests
- [ ] Test: all agents exit cleanly within timeout → all `suspended`
- [ ] Test: some agents hang past timeout → those get `force-killed`
- [ ] Test: no running agents → no-op shutdown
- [ ] Test: shutdown during spawn → spawn rejected

## Phase 3: Auto-Recovery on Startup (4 tasks)

### Task 3.1: Implement recovery manager
- [ ] Create `internal/agent/recovery.go` with `RecoveryManager` struct
- [ ] `RecoverAll(ctx context.Context) RecoveryResult`
- [ ] Pre-resume validation: worktree exists, session ID non-empty, branch intact

### Task 3.2: Resume agents with proper context
- [ ] Developer agents: resume in worktree dir
- [ ] Reviewer agents: resume in project dir
- [ ] Update PID, status to `running`, clear shutdown fields on success
- [ ] On failure: set `resume-failed` with error reason

### Task 3.3: Integrate recovery into relay startup
- [ ] In `cli/up.go`: after relay server starts, call `RecoverAll()`
- [ ] Print user-facing summary: "Restored 3/4 agents, 1 failed: session expired"
- [ ] `crelay status` shows `resume-failed` agents with reason

### Task 3.4: Write recovery tests
- [ ] Test: all suspended agents resume successfully
- [ ] Test: some agents fail (expired session, missing worktree)
- [ ] Test: no suspended agents → no-op
- [ ] Test: mixed states → only suspended attempted

## Phase 4: Verification (2 tasks)

### Task 4.1: End-to-end shutdown/recovery test
- [ ] Spawn mock agents → shutdown → verify suspended → startup → verify resumed
- [ ] Race detector: `go test -race ./internal/agent/...`

### Task 4.2: Full build and test
- [ ] `go build ./...`
- [ ] `go test -race ./...`
- [ ] Verify no regressions

---

**Total: 13 tasks across 4 phases**
