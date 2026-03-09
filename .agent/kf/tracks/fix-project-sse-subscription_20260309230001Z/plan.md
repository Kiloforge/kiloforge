# Implementation Plan: Wire Project SSE Events in Dashboard Frontend

**Track ID:** fix-project-sse-subscription_20260309230001Z

## Phase 1: Wire SSE Handlers

- [x] Task 1.1: Import `useProjects` in `App.tsx` and destructure `handleProjectUpdate` and `handleProjectRemoved`
- [x] Task 1.2: Add `project_update` and `project_removed` to `sseHandlers` object and `useMemo` dependency array
- [x] Task 1.3: Verify `OverviewPage` still works (TanStack Query deduplicates the shared query key)

## Phase 2: Verification

- [x] Task 2.1: Build passes (`cd frontend && npm run build`)
- [x] Task 2.2: Adding a project live-updates the project list without page refresh
- [x] Task 2.3: Removing a project live-updates the project list without page refresh
