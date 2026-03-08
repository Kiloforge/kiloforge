# Implementation Plan: Fix Stale Frontend Build — Rebuild dist with Correct Base Path

**Track ID:** fix-stale-frontend-build_20260309210000Z

## Phase 1: Rebuild Frontend

- [ ] Task 1.1: Run `npm install` in `frontend/` to ensure dependencies are current
- [ ] Task 1.2: Run `npm run build` to regenerate `dist/` with `base: '/'` paths
- [ ] Task 1.3: Verify `dist/index.html` references `/assets/...` (not `/-/assets/...`)
- [ ] Task 1.4: Verify `index.html` title is "Kiloforge" (not "crelay dashboard")

## Phase 2: Verification

- [ ] Task 2.1: `go build ./...` succeeds (embedded assets compile)
- [ ] Task 2.2: Start server, confirm dashboard loads at `http://localhost:<port>/`
- [ ] Task 2.3: Confirm JS/CSS assets return correct MIME types (not text/html)
- [ ] Task 2.4: Commit rebuilt dist artifacts
