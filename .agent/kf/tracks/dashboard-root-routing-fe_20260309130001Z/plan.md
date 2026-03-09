# Implementation Plan: Dashboard Root Routing (Frontend)

**Track ID:** dashboard-root-routing-fe_20260309130001Z

## Phase 1: Path Prefix Updates

- [x] Task 1.1: Update `frontend/src/main.tsx` — change BrowserRouter basename from `"/-/"` to `"/"`
- [x] Task 1.2: Update all hooks — replace `/-/api/` with `/api/` in fetch URLs: `useAgents.ts`, `useQuota.ts`, `useTraces.ts`, `useTracks.ts`, `useBoard.ts`, `useProjects.ts`, `useSkillsStatus.ts`, `useConfig.ts`
- [x] Task 1.3: Update components — replace `/-/api/` in `LogViewer.tsx`, `App.tsx`
- [x] Task 1.4: Update SSE endpoint — change `/-/events` to `/events` in `App.tsx`
- [x] Task 1.5: Update navigation links — remove `/-/` prefix from internal navigation in `TracePage.tsx`

## Phase 2: Gitea Link Verification

- [x] Task 2.1: Verify `AgentCard.tsx` Gitea PR links — removed broken `/-/pulls/` pattern (was never a valid Gitea URL)
- [x] Task 2.2: Updated header Gitea link to `/gitea/` (proxied path), cleaned up unused giteaURL prop cascade

## Phase 3: Verification

- [x] Task 3.1: `npm run build` succeeds (via `make build`)
- [x] Task 3.2: No `/-/` references remain in frontend source
- [x] Task 3.3: All API paths updated (agents, quota, tracks, board, projects, traces, config, skills, ssh-keys)
- [x] Task 3.4: SSE endpoint updated from `/-/events` to `/events`
