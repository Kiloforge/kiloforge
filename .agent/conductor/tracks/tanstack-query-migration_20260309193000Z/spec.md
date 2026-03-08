# Specification: Migrate Frontend Data Fetching to TanStack Query

**Track ID:** tanstack-query-migration_20260309193000Z
**Type:** Refactor
**Created:** 2026-03-09T19:30:00Z
**Status:** Draft

## Summary

Replace all vanilla `fetch()` + `useState` data fetching patterns with `@tanstack/react-query`. This gives us automatic caching, request deduplication, background refetching, optimistic updates, and consistent loading/error states. SSE (`useSSE`) and WebSocket (`useAgentWebSocket`) hooks stay as-is but integrate with Query via cache invalidation.

## Context

The frontend currently uses 26 raw `fetch()` calls across 10 hooks, 3 pages, and 6 components. Each hook manually manages `loading`, `error`, and `data` state. There's no request deduplication, no normalized cache, and manual refresh functions everywhere. TanStack Query replaces all of this with a declarative, cache-first approach.

## Codebase Analysis

### Current patterns

- **10 data-fetching hooks** — `useAgents`, `useBoard`, `useConfig`, `useProjects`, `useQuota`, `useSkillsStatus`, `useTraces`, `useTracks`, `useOriginSync`, `useSSE`
- **Direct fetch in components** — `App.tsx` (status), `ProjectPage.tsx` (generate, delete), `TracePage.tsx` (trace detail), `LogViewer.tsx` (log fetch), `AdminPanel.tsx` (admin run), `OverviewPage.tsx` (sync badges)
- **SSE integration** — Central EventSource at `/events` with typed event handlers that merge into component state
- **WebSocket** — `useAgentWebSocket` for agent terminal I/O
- **Optimistic updates** — `useBoard.moveCard()` does optimistic card move with rollback on failure

### Migration scope

| Category | Count | Migration approach |
|----------|-------|--------------------|
| Simple GET hooks | 5 | `useQuery` with query key |
| GET + POST/DELETE hooks | 4 | `useQuery` + `useMutation` with invalidation |
| Optimistic update hooks | 1 | `useMutation` with `onMutate`/`onError` rollback |
| Direct fetch in components | 6 | Extract to hooks or inline `useQuery`/`useMutation` |
| SSE hook | 1 | Keep as-is, fire `queryClient.invalidateQueries()` on events |
| WebSocket hook | 1 | Keep as-is (not HTTP) |

### Not migrating

- `useSSE.ts` — Stays as EventSource wrapper, but SSE handlers call `queryClient.setQueryData()` or `invalidateQueries()` instead of `setState`
- `useAgentWebSocket.ts` — WebSocket, not HTTP. Stays as-is.
- `LogViewer.tsx` EventSource follow mode — Streaming, not cacheable. Stays as-is.

## Acceptance Criteria

- [ ] `@tanstack/react-query` added as dependency
- [ ] `QueryClientProvider` wrapping app in `main.tsx` or `App.tsx`
- [ ] All 10 data-fetching hooks migrated to `useQuery`/`useMutation`
- [ ] All direct `fetch()` calls in pages/components migrated
- [ ] SSE events invalidate/update TanStack Query cache (not local state)
- [ ] Board optimistic card move uses TanStack Query optimistic update pattern
- [ ] Polling hooks (skills status 60s, board 5m) use Query's `refetchInterval`
- [ ] No manual `loading`/`error` state management — use Query's built-in states
- [ ] All existing functionality preserved (no regressions)
- [ ] Frontend builds without errors
- [ ] No raw `fetch()` calls remaining for REST API endpoints (SSE/WS excluded)

## Dependencies

None.

## Blockers

None.

## Conflict Risk

- HIGH against `agent-list-monitoring-ui_20260309180000Z` — both touch frontend extensively. This track should ideally land first (or after), not concurrently.
- MEDIUM against other pending FE tracks.

## Out of Scope

- Migrating SSE or WebSocket to a different library
- Adding React Suspense boundaries (can be a follow-up)
- Server-side rendering or prefetching
- Changing API response formats

## Technical Notes

### Setup

```tsx
// main.tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,      // 30s before refetch
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

// Wrap <App /> with <QueryClientProvider client={queryClient}>
```

### Query key convention

```ts
// Consistent key factory
export const queryKeys = {
  agents: ['agents'] as const,
  agent: (id: string) => ['agents', id] as const,
  board: (project: string) => ['board', project] as const,
  config: ['config'] as const,
  projects: ['projects'] as const,
  quota: ['quota'] as const,
  skills: ['skills'] as const,
  traces: ['traces'] as const,
  trace: (id: string) => ['traces', id] as const,
  tracks: (project?: string) => ['tracks', project] as const,
  syncStatus: (slug: string) => ['syncStatus', slug] as const,
  status: ['status'] as const,
  sshKeys: ['sshKeys'] as const,
};
```

### SSE → Query cache integration

```ts
// In useSSE handler setup
const queryClient = useQueryClient();

onEvent('agent_update', (agent) => {
  queryClient.setQueryData(queryKeys.agents, (old) =>
    old?.map(a => a.id === agent.id ? agent : a) ?? [agent]
  );
});

onEvent('agent_removed', (id) => {
  queryClient.setQueryData(queryKeys.agents, (old) =>
    old?.filter(a => a.id !== id) ?? []
  );
});
```

### Optimistic board move

```ts
const moveCard = useMutation({
  mutationFn: ({ trackId, column }) =>
    fetch(`/api/board/${project}/move`, { method: 'POST', body: ... }),
  onMutate: async ({ trackId, column }) => {
    await queryClient.cancelQueries({ queryKey: queryKeys.board(project) });
    const previous = queryClient.getQueryData(queryKeys.board(project));
    queryClient.setQueryData(queryKeys.board(project), (old) => /* move card */);
    return { previous };
  },
  onError: (_err, _vars, context) => {
    queryClient.setQueryData(queryKeys.board(project), context?.previous);
  },
  onSettled: () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.board(project) });
  },
});
```

---

_Generated by conductor-track-generator from prompt: "migrate frontend to tanstack query"_
