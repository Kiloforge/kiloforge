# Implementation Plan: Dashboard Root Routing (Frontend)

**Track ID:** dashboard-root-routing-fe_20260309130001Z

## Phase 1: Path Prefix Updates

- [ ] Task 1.1: Update `frontend/src/main.tsx` — change BrowserRouter basename from `"/-/"` to `"/"`
- [ ] Task 1.2: Update all hooks — replace `/-/api/` with `/api/` in fetch URLs: `useAgents.ts`, `useQuota.ts`, `useTraces.ts`, `useTracks.ts`, `useBoard.ts`, `useProjects.ts`, `useSkillsStatus.ts`
- [ ] Task 1.3: Update components — replace `/-/api/` in `LogViewer.tsx`, `App.tsx`
- [ ] Task 1.4: Update SSE endpoint — change `/-/events` to `/events` in `App.tsx` or `useSSE.ts`
- [ ] Task 1.5: Update navigation links — remove `/-/` prefix from internal navigation in `TracePage.tsx` and other pages

## Phase 2: Gitea Link Verification

- [ ] Task 2.1: Verify `AgentCard.tsx` Gitea PR links work with the new `/gitea/` subpath — the `giteaURL` from status API should reflect the new path
- [ ] Task 2.2: Update any hardcoded Gitea URL patterns if needed

## Phase 3: Verification

- [ ] Task 3.1: Verify `npm run build` succeeds
- [ ] Task 3.2: Verify dashboard loads at `localhost:4001/` with correct client-side routing
- [ ] Task 3.3: Verify all API calls work (agents, quota, tracks, board, projects, traces)
- [ ] Task 3.4: Verify SSE events stream correctly
