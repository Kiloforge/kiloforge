# Implementation Plan: Fix Stale Frontend Build — Rebuild dist with Correct Base Path

**Track ID:** fix-stale-frontend-build_20260309210000Z

## Phase 1: Rebuild Frontend

- [x] Task 1.1: Run `npm install` in `frontend/` to ensure dependencies are current
- [x] Task 1.2: Run `npm run build` to regenerate `dist/` with `base: '/'` paths
- [x] Task 1.3: Verify `dist/index.html` references `/assets/...` (not `/-/assets/...`)
- [x] Task 1.4: Verify `index.html` title is "Kiloforge" (not "crelay dashboard")

## Phase 2: Verification

- [x] Task 2.1: `go build ./...` succeeds (embedded assets compile)
- [x] Task 2.2: Start server, confirm dashboard loads at `http://localhost:<port>/`
- [x] Task 2.3: Confirm JS/CSS assets return correct MIME types (not text/html)
- [x] Task 2.4: Commit rebuilt dist artifacts
