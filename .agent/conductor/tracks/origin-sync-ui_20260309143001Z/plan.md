# Implementation Plan: Origin Push/Pull Dashboard UI

**Track ID:** origin-sync-ui_20260309143001Z

## Phase 1: Hook and Types

- [ ] Task 1.1: Add sync-related types to `frontend/src/types/api.ts` (SyncStatus, PushRequest, PullRequest, PushResponse, PullResponse)
- [ ] Task 1.2: Create `frontend/src/hooks/useOriginSync.ts` — fetch sync status, push(), pull() methods

## Phase 2: Project List Integration

- [ ] Task 2.1: Add sync status badge to project rows in `OverviewPage.tsx`
- [ ] Task 2.2: Add push/pull action buttons to project rows

## Phase 3: Project Detail Controls

- [ ] Task 3.1: Create sync status panel component — shows ahead/behind, last fetch, status
- [ ] Task 3.2: Add push control with remote branch input (default `kf/main`)
- [ ] Task 3.3: Add pull control with confirmation
- [ ] Task 3.4: Integrate sync panel into `ProjectPage.tsx`

## Phase 4: Verification

- [ ] Task 4.1: Verify `npm run build` succeeds
- [ ] Task 4.2: Verify sync status displays correctly
- [ ] Task 4.3: Verify push/pull operations trigger API calls and update UI
