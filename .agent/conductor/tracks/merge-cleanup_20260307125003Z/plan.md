# Implementation Plan: PR Merge, Worktree Cleanup, and Agent Teardown

**Track ID:** merge-cleanup_20260307125003Z

## Phase 1: Merge via API

### Task 1.1: Add merge API to Gitea client
- [x] Add `MergePR(ctx, repo, prNum, method)` to `internal/gitea/client.go`
- [x] `POST /api/v1/repos/{owner}/{repo}/pulls/{num}/merge` with `Do` field
- [x] Support methods: merge, rebase, squash
- [x] Add `DeleteBranch(ctx, repo, branch)` for remote cleanup
- [x] Tests: verify API call structure

### Task 1.2: Implement merge logic in approval flow
- [x] When review approved, call `MergeAndCleanup()` orchestration
- [x] Handle merge via Gitea API with configurable method
- [x] Tests: verify merge call with mock Gitea server

### Task 1.3: Post final comment on merged PR
- [x] After successful merge, post comment with track ID and session IDs
- [x] Tests: verify comment posted

### Verification 1
- [x] PR merged via API
- [x] Merge confirmation verified
- [x] Final comment posted
- [x] Tests pass

## Phase 2: Worktree and Branch Cleanup

### Task 2.1: Implement worktree cleanup after merge
- [x] Pool.Return resets worktree (checkout pool branch, reset to main, delete track branch)
- [x] Tests: verify via existing pool tests

### Task 2.2: Delete remote implementation branch
- [x] DeleteBranch via Gitea API (best effort)
- [x] Tests: verify remote branch deletion

### Task 2.3: Return worktree to pool
- [x] poolReturnerAdapter wraps pool.Pool for interface compatibility
- [x] ReturnByTrackID finds and returns worktree, saves pool state
- [x] Tests: verify pool state after return

### Verification 2
- [x] Worktree reset to main
- [x] Local and remote branches deleted
- [x] Worktree returned to pool
- [x] Tests pass

## Phase 3: Agent Teardown and State Cleanup

### Task 3.1: Terminate agent processes
- [x] HaltAgent sends SIGINT to developer/reviewer processes (best effort)
- [x] Called before updating status to "completed"

### Task 3.2: Update all tracking state
- [x] PR tracking: status=merged
- [x] Developer agent: status=completed
- [x] Reviewer agent: status=completed
- [x] Pool state: worktree idle (via PoolReturner)
- [x] Tests: verify all state transitions

### Task 3.3: Final verification
- [x] Full flow test in cleanup_test.go
- [x] Relay integration test with mock Gitea server
- [x] Default merge method test
- [x] Build and tests pass

### Verification 3
- [x] All agents terminated
- [x] All state updated consistently
- [x] Worktree reusable for next track
- [x] Build and tests pass
