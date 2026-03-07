# Implementation Plan: PR Merge, Worktree Cleanup, and Agent Teardown

**Track ID:** merge-cleanup_20260307125003Z

## Phase 1: Merge via API

### Task 1.1: Add merge API to Gitea client
- Add `MergePR(ctx, repo, prNum, method)` to `internal/gitea/client.go`
- `POST /api/v1/repos/{owner}/{repo}/pulls/{num}/merge` with `Do` field
- Support methods: merge, rebase, squash
- Add `GetPRStatus(ctx, repo, prNum)` to check if merged
- Tests: verify API call structure

### Task 1.2: Implement merge logic in developer resume flow
- When developer is resumed after approval:
  - Call `MergePR()` via Gitea API
  - Poll PR status to confirm merge succeeded
  - Handle merge failure (conflicts): log error, mark for human intervention
- Tests: verify merge call and confirmation

### Task 1.3: Post final comment on merged PR
- After successful merge, post comment:
  "Merge complete. Implementation branch cleaned up. Track `<track-id>` done."
- Add developer and reviewer session IDs to comment for audit trail
- Tests: verify comment posted

### Verification 1
- [ ] PR merged via API
- [ ] Merge confirmation verified
- [ ] Final comment posted
- [ ] Tests pass

## Phase 2: Worktree and Branch Cleanup

### Task 2.1: Implement worktree cleanup after merge
- After merge confirmed:
  - `git checkout <pool-branch>` in worktree
  - `git reset --hard main` (get latest with merged changes)
  - `git branch -D <track-id>` (delete local implementation branch)
- Tests: verify branch cleanup commands

### Task 2.2: Delete remote implementation branch
- `git push gitea --delete <track-id>`
- Handle case where branch already deleted (not an error)
- Tests: verify remote branch deletion

### Task 2.3: Return worktree to pool
- Call `pool.Return(worktree)` to mark as idle
- Verify pool state updated
- Tests: verify pool state after return

### Verification 2
- [ ] Worktree reset to main
- [ ] Local and remote branches deleted
- [ ] Worktree returned to pool
- [ ] Tests pass

## Phase 3: Agent Teardown and State Cleanup

### Task 3.1: Terminate agent processes
- After cleanup, terminate developer Claude Code process (SIGTERM → SIGKILL)
- Verify reviewer was already terminated (from review-cycle track)
- Clean up PID references in agent state
- Tests: verify process termination

### Task 3.2: Update all tracking state
- PR tracking: status=merged
- Developer agent: status=completed
- Track status: update conductor tracks.md to mark track complete
- Pool state: worktree idle
- Tests: verify all state transitions

### Task 3.3: Update docs and final verification
- Document merge behavior and cleanup in README
- Document configurable merge method
- Full end-to-end test: implement → review cycle → approve → merge → cleanup
- Verify worktree is reusable after cleanup

### Verification 3
- [ ] All agents terminated
- [ ] All state updated consistently
- [ ] Worktree reusable for next track
- [ ] Docs updated
- [ ] Build and tests pass
