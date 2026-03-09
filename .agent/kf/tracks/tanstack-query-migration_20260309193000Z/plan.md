# Implementation Plan: Migrate Frontend Data Fetching to TanStack Query

**Track ID:** tanstack-query-migration_20260309193000Z

## Phase 1: Setup and Infrastructure

- [x] Task 1.1: Install `@tanstack/react-query` and `@tanstack/react-query-devtools`
- [x] Task 1.2: Create `QueryClientProvider` wrapper in `main.tsx` with default options
- [x] Task 1.3: Create `frontend/src/api/queryKeys.ts` — query key factory for all endpoints
- [x] Task 1.4: Create `frontend/src/api/fetcher.ts` — shared fetch wrapper with error handling (throw on non-2xx)

## Phase 2: Migrate Simple GET Hooks

- [x] Task 2.1: Migrate `useAgents` — `useQuery` for agent list, keep SSE handler updating cache via `setQueryData`
- [x] Task 2.2: Migrate `useQuota` — `useQuery` with SSE cache update
- [x] Task 2.3: Migrate `useTraces` — `useQuery` with SSE cache update
- [x] Task 2.4: Migrate `useTracks` — `useQuery` with project param in key, SSE cache update
- [x] Task 2.5: Migrate `useConfig` (read) — `useQuery` for GET

## Phase 3: Migrate GET + Mutation Hooks

- [x] Task 3.1: Migrate `useConfig` (write) — `useMutation` for PUT with invalidation
- [x] Task 3.2: Migrate `useProjects` — `useQuery` for list, `useMutation` for POST/DELETE with invalidation, SSE cache update
- [x] Task 3.3: Migrate `useSkillsStatus` — `useQuery` with `refetchInterval: 60_000`, `useMutation` for update
- [x] Task 3.4: Migrate `useOriginSync` — `useQuery` for sync-status, `useMutation` for push/pull

## Phase 4: Migrate Complex Hooks and Components

- [x] Task 4.1: Migrate `useBoard` — `useQuery` with `refetchInterval: 300_000`, optimistic `useMutation` for card moves, SSE cache update
- [x] Task 4.2: Migrate `App.tsx` — `useQuery` for status, `useMutation` for interactive agent spawn
- [x] Task 4.3: Migrate `ProjectPage.tsx` — `useMutation` for track generation and deletion
- [x] Task 4.4: Migrate `TracePage.tsx` — `useQuery` for trace detail
- [x] Task 4.5: Migrate `AdminPanel.tsx` — `useMutation` for admin operations
- [x] Task 4.6: Migrate `OverviewPage.tsx` sync badges — `useQuery` for per-project sync status

## Phase 5: SSE Integration Refactor

- [x] Task 5.1: Refactor `useSSE` — instead of passing setState callbacks, use `queryClient.setQueryData()` / `invalidateQueries()` for all event types
- [x] Task 5.2: Remove manual state merging from all migrated hooks — SSE now updates the Query cache directly

## Phase 6: Cleanup and Verification

- [x] Task 6.1: Remove all unused `useState` for loading/error/data in migrated hooks
- [x] Task 6.2: Verify no raw `fetch()` calls remain for REST endpoints (grep)
- [x] Task 6.3: Verify `npm run build` succeeds
- [x] Task 6.4: Manual verification — all pages load, SSE updates work, board drag-drop works, mutations succeed
