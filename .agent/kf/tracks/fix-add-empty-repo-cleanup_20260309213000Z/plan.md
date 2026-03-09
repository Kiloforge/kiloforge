# Implementation Plan: Fix kf add for Empty Repos and Add Rollback on Failure

**Track ID:** fix-add-empty-repo-cleanup_20260309213000Z

## Phase 1: Empty Repo Handling

- [x] Task 1.1: Add `hasCommits()` helper — runs `git rev-parse HEAD` in clone dir, returns bool
- [x] Task 1.2: In CLI `add.go` — after clone, check `hasCommits()`; if false, skip push and warn user
- [x] Task 1.3: In service `project_service.go` — same empty repo check and skip push logic

## Phase 2: Rollback on Failure

- [x] Task 2.1: Track whether Gitea repo was newly created (vs. pre-existing 409) with a `giteaRepoCreated` flag
- [x] Task 2.2: In CLI `add.go` — if push or any subsequent step fails, delete Gitea repo (if newly created) and remove clone dir
- [x] Task 2.3: In service `project_service.go` — same rollback logic for REST API path

## Phase 3: Orphan Cleanup on Retry

- [x] Task 3.1: Before cloning, check if clone directory exists but project is NOT in registry → remove orphaned directory
- [x] Task 3.2: Apply orphan check in both CLI and service paths

## Phase 4: Verification

- [x] Task 4.1: `go test ./...` passes
- [x] Task 4.2: Manual test — add empty repo succeeds with warning, retry after failure succeeds
