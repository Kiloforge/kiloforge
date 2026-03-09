# Implementation Plan: Track Detail View UI

**Track ID:** track-detail-view-fe_20260309001727Z

## Phase 1: Types and API Client

- [ ] Task 1.1: Add `TrackDetail` interface to `types/api.ts` — id, title, status, type, spec, plan, phases (total/completed), tasks (total/completed), created_at, updated_at
- [ ] Task 1.2: Add `fetchTrackDetail(trackId: string, project: string)` to API client
- [ ] Task 1.3: Add `trackDetail` query key factory to `queryKeys.ts`
- [ ] Task 1.4: Add `useTrackDetail(trackId, project)` hook using TanStack Query

## Phase 2: Track Detail Page

- [ ] Task 2.1: Create `TrackDetailPage` component — fetch track detail, display loading/error states
- [ ] Task 2.2: Render track header: title, status badge, type badge, dates, phase/task progress bar
- [ ] Task 2.3: Render spec.md content — use markdown rendering if library available, otherwise `<pre>` with whitespace preservation
- [ ] Task 2.4: Render plan.md content — same approach as spec
- [ ] Task 2.5: Add back-link to project page and show agent/trace links if available
- [ ] Task 2.6: Add route `/projects/:slug/tracks/:trackId` to App.tsx

## Phase 3: Navigation from Board and List

- [ ] Task 3.1: Make KanbanBoard cards clickable — add click handler or Link to navigate to track detail route
- [ ] Task 3.2: Make TrackList items clickable — wrap items in Link to track detail route

## Phase 4: Verification

- [ ] Task 4.1: `npm run build` succeeds with no errors
