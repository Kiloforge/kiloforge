# Implementation Plan: React Dashboard for Real-Time Agent Monitoring

**Track ID:** react-dashboard_20260308180002Z

## Phase 1: Foundation — Types, Hooks, and SSE (5 tasks)

### Task 1.1: Define TypeScript API types
- [x] Create `src/types/api.ts` with interfaces matching all backend JSON responses
- [x] `Agent`, `QuotaResponse`, `Track`, `StatusResponse`, `SSEEventData`, `LogResponse`

### Task 1.2: Implement useSSE hook
- [x] EventSource lifecycle with auto-reconnect (exponential backoff, max 30s)
- [x] Expose connection state: connected, reconnecting, disconnected

### Task 1.3: Implement useAgents hook
- [x] Fetch initial + SSE upsert/remove handlers

### Task 1.4: Implement useQuota hook
- [x] Fetch initial + SSE update handler

### Task 1.5: Implement useTracks hook
- [x] Fetch with 30s polling interval

## Phase 2: Core Components (5 tasks)

### Task 2.1–2.5: All components implemented
- [x] App.tsx with single SSE connection feeding all hooks
- [x] ConnectionStatus, StatCards, AgentGrid, AgentCard, StatusBadge, TrackList
- [x] AgentHistogram in header bar
- [x] Utility formatters (formatUSD, formatTokens, formatUptime)
- [x] Dark theme CSS variables, responsive grid, CSS modules

## Phase 3: Log Viewer and Polish (4 tasks)

### Task 3.1–3.4: All polish tasks complete
- [x] LogViewer modal with backdrop close, follow mode (SSE streaming)
- [x] Responsive layout (auto-fit grids)
- [x] Empty states for agents and tracks
- [x] Agent histogram chips in header

## Phase 4: Build Verification (2 tasks)

### Task 4.1: Production build check
- [x] `npm run build` — 0 errors, 0 warnings
- [x] Bundle: ~66KB gzipped (under 500KB limit)
- [x] Output in `backend/internal/adapter/dashboard/dist/`
- [x] `cd backend && go build -buildvcs=false ./...` succeeds
- [x] All Go tests pass

### Task 4.2: Integration verified
- [x] Go embed works with new dist assets
- [x] detail.go templates use inline styles (no broken CSS refs)
- [x] TypeScript strict mode, verbatimModuleSyntax compatible

---

**Total: 16 tasks across 4 phases — ALL COMPLETE**
