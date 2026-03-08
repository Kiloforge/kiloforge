# Implementation Plan: Fix VCS Stamping in Git Worktrees

**Track ID:** fix-buildvcs-worktree_20260309114000Z

## Phase 1: Fix Build

- [ ] Task 1.1: Update `Makefile` `build-backend` target — set `GIT_DIR=$(git rev-parse --git-common-dir)` and `GIT_WORK_TREE` env vars, remove `-buildvcs=false`
- [ ] Task 1.2: Update `backend/cmd/kf/main_test.go` `TestBinaryBuilds` — set same env vars on the exec.Command, remove `-buildvcs=false`

## Phase 2: Verification

- [ ] Task 2.1: Verify `make build-backend` succeeds in worktree
- [ ] Task 2.2: Verify built binary has VCS metadata: `go version -m .build/kf | grep vcs`
- [ ] Task 2.3: Verify `go test ./...` passes
- [ ] Task 2.4: Grep codebase for any remaining `-buildvcs=false` references
