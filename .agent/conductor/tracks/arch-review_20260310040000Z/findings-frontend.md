# Frontend Review Findings

**Track:** arch-review_20260310040000Z
**Phase:** 3 — Frontend Review

## 1. Architecture Patterns

### 1.1 Component Organization — GOOD
- Clear container/presenter separation in most places
- Pages act as containers (data fetching), components are presentational
- CSS Modules prevent style collisions

### 1.2 Issues Found

| # | Severity | File | Issue |
|---|----------|------|-------|
| F1 | Medium | `pages/ProjectPage.tsx` (361 lines) | Overloaded container: 10+ hooks, multiple modals, board + sync + tracks + setup + admin. Should be split into sub-containers |
| F2 | Low | `components/AgentCard.tsx:17` | Calls `useTracks()` internally — presentational component doing data fetching. Should receive track/project via props |
| F3 | Low | `pages/AgentDetailPage.tsx:47-87` | Mixed data fetching: raw `fetch()` for logs alongside TanStack Query for agent data. Comment justifies it (streaming not cache-friendly) but inconsistent |

### 1.3 Prop Drilling — MINIMAL
- Most prop chains are 2-3 levels (acceptable)
- SyncPanel receives 8 props from ProjectPage — could be a sub-container instead

## 2. Hook Patterns — EXCELLENT

### 2.1 TanStack Query — Consistent
- All 9 data hooks use `useQuery` + `useMutation` correctly
- Query keys centralized in `api/queryKeys.ts`
- Cache invalidation consistent: `queryClient.invalidateQueries()` after mutations
- Optimistic updates implemented for board moves (`useBoard`)

### 2.2 Cleanup — Excellent
- All `useEffect` hooks with subscriptions (EventSource, WebSocket) have proper cleanup
- `useSSE.ts` clears timeout + closes EventSource on unmount
- `useAgentWebSocket.ts` clears retry timeout + closes WebSocket

### 2.3 Error Handling — Good
- Consistent `FetchError` pattern across mutations
- `formatMutationError()` utility for toast notifications
- Minor inconsistency: `useSetupPrompt` uses direct JSON parsing vs standard approach

### 2.4 Prompt Hook Duplication

| # | Severity | File | Issue |
|---|----------|------|-------|
| F4 | Low | `useConsent.ts`, `useSkillsPrompt.ts`, `useSetupPrompt.ts` | All three use identical retryRef + setState pattern. Could use a shared hook factory |

## 3. Type Safety — EXCELLENT

### 3.1 Strengths
- **Zero `any` types** across entire frontend codebase
- 21 comprehensive interfaces in `types/api.ts`
- Consistent use of optional chaining (`?.`) and nullish coalescing (`??`)
- Compile-time assertion via `gen.StrictServerInterface` ensures backend alignment

### 3.2 Minor Issues

| # | Severity | File | Issue |
|---|----------|------|-------|
| F5 | Low | `types/api.ts` | No type for API error response body (inferred as `body?.error`) |
| F6 | Low | `hooks/useAgents.ts:28`, `hooks/useBoard.ts:31` | Type assertions (`as`) at SSE boundary — acceptable but could validate structure more explicitly |

## 4. Test Coverage — SIGNIFICANT GAP

### 4.1 Current State
- **5 test files** out of **57 source files** (8.8% coverage)
- Tested: `fetcher.test.ts`, `queryKeys.test.ts`, `AddProjectForm.test.tsx`, `useTour.test.ts`, `format.test.ts`
- Test quality is good: behavior-focused, proper React Testing Library usage

### 4.2 Critical Untested Areas

| Priority | File | Lines | Risk |
|----------|------|-------|------|
| Critical | `hooks/useAgentWebSocket.ts` | 200 | WebSocket reconnection, exponential backoff, message parsing. Bugs invisible without tests |
| Critical | `pages/ProjectPage.tsx` | 361 | Main orchestration page. 10+ hooks, cascading failures |
| Critical | `components/KanbanBoard.tsx` | 204 | Drag-and-drop, state machine. Interaction-heavy |
| High | `hooks/useProjects.ts` | 142 | Add/remove mutations + SSE handlers |
| High | `hooks/useBoard.ts` | 113 | Board state + optimistic updates |
| High | `pages/AgentDetailPage.tsx` | 316 | Log streaming + WebSocket |
| High | `hooks/useSSE.ts` | 69 | EventSource lifecycle + retry logic |
| High | `pages/OverviewPage.tsx` | 251 | Dashboard rendering |

### 4.3 Untested Summary by Category
- **0/6 pages** tested
- **1/17 hooks** tested (useTour only)
- **1/26 components** tested (AddProjectForm only)
- **2/7 utilities** tested (fetcher, queryKeys)

## 5. Error Toast Architecture

| # | Severity | File | Issue |
|---|----------|------|-------|
| F7 | Low | `api/errorToast.ts` | Uses mutable global state (`let _addToast = null`) with imperative initialization. Works but fragile — could integrate with QueryClient's MutationCache directly |

## Summary

| Category | Critical | High | Medium | Low |
|----------|----------|------|--------|-----|
| Architecture | 0 | 0 | 1 | 2 |
| Hooks/Data | 0 | 0 | 0 | 2 |
| Type Safety | 0 | 0 | 0 | 2 |
| Test Coverage | 3 | 5 | 0 | 0 |
| **Total** | **3** | **5** | **1** | **6** |

All Critical/High items are test coverage gaps, not code quality issues. The frontend architecture is well-structured.

### Track Recommendations
1. **frontend-test-coverage-hooks** — Tests for useAgentWebSocket, useSSE, useProjects, useBoard, useOriginSync (critical hooks with complex logic)
2. **frontend-test-coverage-pages** — Tests for ProjectPage, AgentDetailPage, OverviewPage (critical orchestration pages)
3. **refactor-project-page** — Split ProjectPage (361 lines) into sub-containers: BoardContainer, SyncContainer, AdminContainer
4. **frontend-type-hardening** — Add API error response type, validate SSE event structure with discriminated unions
