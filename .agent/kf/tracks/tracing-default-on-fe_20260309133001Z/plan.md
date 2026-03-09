# Implementation Plan: Tracing Toggle in Dashboard UI (Frontend)

**Track ID:** tracing-default-on-fe_20260309133001Z

## Phase 1: Config Hook

- [x] Task 1.1: Create `frontend/src/hooks/useConfig.ts` — fetch `GET /api/config`, expose `updateConfig()` via `PUT /api/config`
- [x] Task 1.2: Define TypeScript types for config response (`TracingEnabled`, `DashboardEnabled`)

## Phase 2: Toggle Component

- [x] Task 2.1: Create tracing toggle component — switch with label, calls `updateConfig` on change
- [x] Task 2.2: Integrate toggle into TraceList header area or traces tab
- [x] Task 2.3: Update TraceList empty state — replace text-only message with toggle + explanation
- [x] Task 2.4: Add note: "Changes take effect on next restart"

## Phase 3: Verification

- [x] Task 3.1: Verify `npm run build` succeeds
- [x] Task 3.2: Verify toggle reflects current config state on page load
- [x] Task 3.3: Verify toggle persists config change via API
