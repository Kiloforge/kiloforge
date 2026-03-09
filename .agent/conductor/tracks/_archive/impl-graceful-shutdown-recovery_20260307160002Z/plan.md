# Implementation Plan: Graceful Agent Shutdown and Auto-Recovery on Restart

**Track ID:** impl-graceful-shutdown-recovery_20260307160002Z

## Phase 1: State Model Extensions (3 tasks)

### Task 1.1: Extend AgentInfo with shutdown/resume fields
- [x] Add `SuspendedAt`, `ShutdownReason`, `ResumeError` fields to `AgentInfo`
- [x] Add new status values: `suspended`, `suspending`, `force-killed`, `resume-failed`
- [x] Ensure backwards compatibility: existing state.json without new fields loads correctly

### Task 1.2: Add batch agent operations to Store
- [x] `AgentsByStatus(statuses ...string) []AgentInfo` — filter agents by status
- [x] Add to port.AgentStore interface
- [x] Implement in jsonfile.AgentStore and testutil.MockAgentStore

### Task 1.3: Write state model tests
- [x] Test new fields serialize/deserialize correctly
- [x] Test backwards compatibility with old state.json format
- [x] Test AgentsByStatus filtering

## Phase 2: Graceful Shutdown (4 tasks)

### Task 2.1: Implement shutdown manager
- [x] Create `internal/agent/shutdown.go` with `ShutdownManager` struct
- [x] `ShutdownAll(timeout time.Duration) ShutdownResult` — SIGINT all, wait, SIGKILL stragglers
- [x] Track per-agent shutdown outcome

### Task 2.2: Integrate shutdown into relay lifecycle
- [x] In `relay/server.go`: shutdown agents after HTTP server stops
- [x] In `cli/down.go`: call `ShutdownAll()` before stopping Docker compose
- [x] In `cli/destroy.go`: call `ShutdownAll()` before destroying

### Task 2.3: Update agent statuses on shutdown
- [x] Agents that exit cleanly after SIGINT → status `suspended`
- [x] Agents force-killed after timeout → status `force-killed`
- [x] Save state.json after all agents handled

### Task 2.4: Write shutdown tests
- [x] Test: no running agents → no-op shutdown
- [x] Test: agent with no PID → suspended
- [x] Test: dead process → already dead, marked suspended
- [x] Test: mixed states → only running/waiting touched

## Phase 3: Auto-Recovery on Startup (4 tasks)

### Task 3.1: Implement recovery manager
- [x] Create `internal/agent/recovery.go` with `RecoveryManager` struct
- [x] `RecoverAll(ctx context.Context) RecoveryResult`
- [x] Pre-resume validation: worktree exists, session ID non-empty

### Task 3.2: Resume agents with proper context
- [x] Developer agents: resume in worktree dir (developers prioritized first)
- [x] Reviewer agents: resume in project dir
- [x] Update PID, status to `running`, clear shutdown fields on success
- [x] On failure: set `resume-failed` with error reason

### Task 3.3: Integrate recovery into relay startup
- [x] In `cli/up.go`: before starting relay, call `RecoverAll()`
- [x] Print user-facing summary
- [x] `kf agents` shows `resume-failed` agents with reason in INFO column

### Task 3.4: Write recovery tests
- [x] Test: all suspended agents resume successfully
- [x] Test: missing session ID → resume-failed
- [x] Test: missing worktree → resume-failed
- [x] Test: start fails → resume-failed with error
- [x] Test: stale running agents → detected and resumed
- [x] Test: developers resumed before reviewers

## Phase 4: Verification (2 tasks)

### Task 4.1: Race detector verification
- [x] All agent tests pass with -race

### Task 4.2: Full build and test
- [x] `go build -buildvcs=false ./...` passes
- [x] `go test -buildvcs=false -race ./...` passes
- [x] No regressions

---

**Total: 13 tasks across 4 phases — ALL COMPLETE**
