# Implementation Plan: Stop Committing Frontend Dist

**Track ID:** gitignore-frontend-dist_20260310000000Z

## Phase 1: Remove and Gitignore

- [ ] Task 1.1: Add `backend/internal/adapter/dashboard/dist/` and `backend/internal/adapter/rest/state.json` to `.gitignore`
- [ ] Task 1.2: Update `.gitignore` comment (remove "committed so //go:embed always has correct assets")
- [ ] Task 1.3: Remove dist from git tracking: `git rm --cached -r backend/internal/adapter/dashboard/dist/`

## Phase 2: Verification

- [ ] Task 2.1: `make build` works (frontend builds, backend embeds dist)
- [ ] Task 2.2: `make test` works (ensure-dist creates placeholder)
- [ ] Task 2.3: `git status` is clean after build
