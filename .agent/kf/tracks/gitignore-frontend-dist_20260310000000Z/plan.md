# Implementation Plan: Stop Committing Frontend Dist

**Track ID:** gitignore-frontend-dist_20260310000000Z

## Phase 1: Remove and Gitignore

- [x] Task 1.1: Add `backend/internal/adapter/dashboard/dist/` and `backend/internal/adapter/rest/state.json` to `.gitignore`
- [x] Task 1.2: Update `.gitignore` comment (remove "committed so //go:embed always has correct assets")
- [x] Task 1.3: Remove dist from git tracking — already untracked, no action needed

## Phase 2: Verification

- [x] Task 2.1: `go build` works (backend embeds dist)
- [x] Task 2.2: `go test ./...` passes
- [x] Task 2.3: `git status` is clean after build (dist files properly ignored)
