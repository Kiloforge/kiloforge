# Implementation Plan: Agent Completion Callback and Dry-Run Mode

**Track ID:** agent-completion-callback_20260309112000Z

## Phase 1: Completion Callback Infrastructure

- [ ] Task 1.1: Add `OnCompletion func(agentID, ref, status string)` field to `Spawner` struct in `spawner.go`
- [ ] Task 1.2: Call `OnCompletion` at the end of `monitorAgent()` after status update, passing agent ID, ref (track ID), and final status
- [ ] Task 1.3: Add `SetCompletionCallback(fn)` method to `Spawner` for clean injection

## Phase 2: Wire Callback in CLI

- [ ] Task 2.1: In `implement.go`, create completion callback closure that captures `boardSvc`, `pool`, and `proj.Slug`
- [ ] Task 2.2: Callback on "completed": call `boardSvc.MoveCard(slug, ref, domain.ColumnDone)` and `pool.ReturnByTrackID(ref)`
- [ ] Task 2.3: Callback on "failed": call `pool.ReturnByTrackID(ref)` (return worktree, leave board state for user to decide)
- [ ] Task 2.4: Pass callback to spawner via `SetCompletionCallback()` before `SpawnDeveloper()`

## Phase 3: Wire Callback in REST Server

- [ ] Task 3.1: In `server.go`, create completion callback for webhook-spawned reviewer agents
- [ ] Task 3.2: Ensure callback does not conflict with existing `handleReviewApproved` path (guard against double board-move)

## Phase 4: Dry-Run Mode

- [ ] Task 4.1: Add `--dry-run` flag to `implementCmd` in `implement.go`
- [ ] Task 4.2: Implement dry-run path: validate track, move board card to Done, mark track `[x]` in tracks.md, commit, print summary
- [ ] Task 4.3: Dry-run should NOT acquire a worktree or spawn an agent
- [ ] Task 4.4: Add track-marking utility: read `tracks.md`, change `[ ]`/`[~]` to `[x]` for the given track ID, update metadata.json status to "complete", commit

## Phase 5: Tests & Verification

- [ ] Task 5.1: Test `monitorAgent` calls `OnCompletion` on process exit (success case)
- [ ] Task 5.2: Test `monitorAgent` calls `OnCompletion` on process exit (failure case)
- [ ] Task 5.3: Test `OnCompletion` is nil-safe (no callback set → no panic)
- [ ] Task 5.4: Test dry-run path: verify no agent spawned, board moved, track marked
- [ ] Task 5.5: Verify `go test ./...` passes
- [ ] Task 5.6: Verify `make build` succeeds
