# Implementation Plan: Origin Push/Pull Dashboard UI

**Track ID:** origin-sync-ui_20260309143001Z

## Phase 1: Hook and Types

- [x] Task 1.1: Add sync-related types to `frontend/src/types/api.ts` (SyncStatus, PushRequest, PullRequest, PushResponse, PullResponse)
- [x] Task 1.2: Create `frontend/src/hooks/useOriginSync.ts` — fetch sync status, push(), pull() methods

## Phase 2: Project List Integration

- [x] Task 2.1: Add sync status badge to project rows in `OverviewPage.tsx`
- [x] Task 2.2: Add sync column header and per-project badge display

## Phase 3: Project Detail Controls

- [x] Task 3.1: Create SyncPanel component — shows ahead/behind, status dot, refresh button
- [x] Task 3.2: Add push control with remote branch input (default `kf/main`)
- [x] Task 3.3: Add pull control with loading/error states
- [x] Task 3.4: Integrate sync panel into `ProjectPage.tsx`

## Phase 4: Verification

- [x] Task 4.1: `npm run build` succeeds (via `make build`)
- [x] Task 4.2: `make test` passes
- [x] Task 4.3: All sync API endpoints wired correctly (/api/projects/{slug}/sync-status, push, pull)
