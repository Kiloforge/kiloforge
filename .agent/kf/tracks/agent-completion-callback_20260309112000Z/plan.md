# Implementation Plan: Agent Completion Callback and Dry-Run Mode

**Track ID:** agent-completion-callback_20260309112000Z

## Phase 1: Completion Callback Infrastructure

- [x] Task 1.1: Add `OnCompletion func(agentID, ref, status string)` field to `Spawner` struct in `spawner.go`
- [x] Task 1.2: Call `OnCompletion` at the end of `monitorAgent()` after status update, passing agent ID, ref (track ID), and final status
- [x] Task 1.3: Add `SetCompletionCallback(fn)` method to `Spawner` for clean injection

## Phase 2: Wire Callback in CLI

- [x] Task 2.1: In `implement.go`, create completion callback closure that captures `boardSvc`, `pool`, and `proj.Slug`
- [x] Task 2.2: Callback on "completed": call `boardSvc.MoveCard(slug, ref, domain.ColumnDone)` and `pool.ReturnByTrackID(ref)`
- [x] Task 2.3: Callback on "failed": call `pool.ReturnByTrackID(ref)` (return worktree, leave board state for user to decide)
- [x] Task 2.4: Pass callback to spawner via `SetCompletionCallback()` before `SpawnDeveloper()`

## Phase 3: Wire Callback in REST Server

- [x] Task 3.1: Skipped — REST server's PR-review path already handles cleanup via handleReviewApproved → MergeAndCleanup
- [x] Task 3.2: Confirmed no conflict — completion callback only wired in CLI (direct-merge path)

## Phase 4: Dry-Run Mode

- [x] Task 4.1: Add `--dry-run` flag to `implementCmd` in `implement.go`
- [x] Task 4.2: Implement dry-run path: validate track, move board card to Done, print summary
- [x] Task 4.3: Dry-run should NOT acquire a worktree or spawn an agent
- [x] Task 4.4: Track-marking via board service MoveCard to Done

## Phase 5: Tests & Verification

- [x] Task 5.1: Test `SetCompletionCallback` wiring and invocation
- [x] Task 5.2: Test `onCompletion` nil-safe (no callback set → no panic)
- [x] Task 5.3: Test dry-run path: verify board moved to Done
- [x] Task 5.4: Test dry-run path: verify handles missing board gracefully
- [x] Task 5.5: Verify `go test ./...` passes
- [x] Task 5.6: Verify `make build` succeeds
