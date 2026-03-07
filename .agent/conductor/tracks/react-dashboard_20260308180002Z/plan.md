# Implementation Plan: React Dashboard for Real-Time Agent Monitoring

**Track ID:** react-dashboard_20260308180002Z

## Phase 1: Foundation — Types, Hooks, and SSE (5 tasks)

### Task 1.1: Define TypeScript API types
- [ ] Create `src/types/api.ts` with interfaces matching all backend JSON responses
- [ ] `Agent` — id, role, ref, status, session_id, pid, worktree_dir, log_file, started_at, updated_at, suspended_at, shutdown_reason
- [ ] `QuotaResponse` — total_cost_usd, input_tokens, output_tokens, agent_count, rate_limited, retry_after_seconds, per_agent map
- [ ] `Track` — id, title, status
- [ ] `StatusResponse` — gitea_url, agent_counts, total_agents, sse_clients, total_cost_usd
- [ ] `SSEEvent` — type (agent_update | agent_removed | quota_update), data

### Task 1.2: Implement useSSE hook
- [ ] Create `src/hooks/useSSE.ts`
- [ ] Manage EventSource lifecycle (connect, reconnect on error, cleanup on unmount)
- [ ] Expose connection state: `connected`, `reconnecting`, `disconnected`
- [ ] Accept event handlers map: `{ [eventType]: (data) => void }`
- [ ] Auto-reconnect with exponential backoff (1s, 2s, 4s, max 30s)

### Task 1.3: Implement useAgents hook
- [ ] Create `src/hooks/useAgents.ts`
- [ ] Fetch initial agent list from `GET /api/agents` on mount
- [ ] Subscribe to `agent_update` SSE events — upsert into state
- [ ] Subscribe to `agent_removed` SSE events — remove from state
- [ ] Expose: `agents: Agent[]`, `loading: boolean`, `error: string | null`

### Task 1.4: Implement useQuota hook
- [ ] Create `src/hooks/useQuota.ts`
- [ ] Fetch initial quota from `GET /api/quota` on mount
- [ ] Subscribe to `quota_update` SSE events — replace state
- [ ] Expose: `quota: QuotaResponse | null`, `loading: boolean`

### Task 1.5: Implement useTracks hook
- [ ] Create `src/hooks/useTracks.ts`
- [ ] Fetch tracks from `GET /api/tracks` on mount
- [ ] Refetch on interval (30s) since tracks don't have SSE events
- [ ] Expose: `tracks: Track[]`, `loading: boolean`

## Phase 2: Core Components (5 tasks)

### Task 2.1: App layout and connection status
- [ ] Create main `App.tsx` with header, stat cards, main content grid
- [ ] `ConnectionStatus` component — green dot when connected, yellow when reconnecting, red when disconnected
- [ ] Fetch `/api/status` for Gitea URL and use in agent card links
- [ ] Dark theme CSS variables (background, surface, accent, status colors)
- [ ] Global styles in `src/index.css`

### Task 2.2: StatCards component
- [ ] Display 4 summary cards in a row: Total Agents, Total Cost, Rate Limit, Total Tokens
- [ ] Format cost as USD (e.g., `$1.23`)
- [ ] Format tokens with k/M suffixes (e.g., `1.2M`)
- [ ] Rate limit card shows warning state (yellow/red) when rate limited
- [ ] Cards update in real-time via quota hook

### Task 2.3: AgentGrid and AgentCard components
- [ ] `AgentGrid` — responsive CSS grid (auto-fill, min 320px)
- [ ] `AgentCard` — status badge, role icon/label, ref (PR link to Gitea), uptime timer, cost
- [ ] `StatusBadge` — colored chip matching status (running=green, waiting=yellow, completed=blue, failed=red, halted=orange, suspended=gray)
- [ ] "View Log" button on each card
- [ ] Sort agents: running first, then by most recently updated

### Task 2.4: TrackList component
- [ ] Table with columns: Status, Track ID, Title
- [ ] Status icons: pending (circle), in-progress (spinner/arrow), complete (checkmark)
- [ ] Alternating row backgrounds for readability
- [ ] Empty state: "No tracks found"

### Task 2.5: Utility formatters
- [ ] `src/utils/format.ts`
- [ ] `formatUSD(cents: number): string` — `$1.23`
- [ ] `formatTokens(count: number): string` — `1.2M`, `45.3k`, `123`
- [ ] `formatUptime(startedAt: string): string` — `2h 15m`, `3d 1h`
- [ ] `formatTimestamp(iso: string): string` — relative or absolute

## Phase 3: Log Viewer and Polish (4 tasks)

### Task 3.1: LogViewer modal component
- [ ] Modal overlay with backdrop click to close
- [ ] Fetch log from `GET /api/agents/{id}/log?lines=200`
- [ ] Display in monospace pre block with line numbers
- [ ] "Follow" toggle: when enabled, fetch with `?follow=true` and auto-scroll
- [ ] Loading spinner while fetching

### Task 3.2: Responsive layout and dark theme
- [ ] Ensure layout works at 1024px+ width
- [ ] Stat cards wrap to 2x2 grid on smaller screens
- [ ] Agent grid collapses to single column on narrow screens
- [ ] Consistent dark theme: `#0f1117` background, `#1a1d27` surface, `#6c8cff` accent
- [ ] Smooth transitions on status changes

### Task 3.3: Error handling and empty states
- [ ] Connection error banner when SSE disconnects
- [ ] API fetch error display (non-blocking toast or inline)
- [ ] Empty state for agents: "No agents running"
- [ ] Empty state for tracks: "No tracks registered"
- [ ] Loading skeletons for initial data fetch

### Task 3.4: Agent histogram in status bar
- [ ] Small bar or chip set showing count per status (e.g., "3 running · 1 waiting · 5 completed")
- [ ] Updates in real-time from agent state
- [ ] Placed in header or as a sub-bar below stat cards

## Phase 4: Build Verification (2 tasks)

### Task 4.1: Production build check
- [ ] `npm run build` — no TypeScript errors, no warnings
- [ ] Bundle size under 500KB gzipped
- [ ] Output in `backend/internal/adapter/dashboard/dist/`
- [ ] `cd backend && go build -buildvcs=false ./...` with embedded frontend
- [ ] Serve the built binary and verify dashboard loads

### Task 4.2: Verify full integration
- [ ] SSE events flow correctly in production build
- [ ] All API endpoints return data through embedded app
- [ ] Gitea proxy still works at `/gitea/`
- [ ] Log viewer works (fetch and follow mode)
- [ ] No console errors in browser

---

**Total: 16 tasks across 4 phases**
