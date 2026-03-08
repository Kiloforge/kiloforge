# Implementation Plan: Remove Tracing Toggle UI (Frontend)

**Track ID:** tracing-always-on-fe_20260309231001Z

## Phase 1: Remove Toggle Component

- [ ] Task 1.1: Delete `frontend/src/components/TracingToggle.tsx` and `TracingToggle.module.css`
- [ ] Task 1.2: Update `TraceList.tsx` — remove TracingToggle import, render, and "enable tracing" conditional message
- [ ] Task 1.3: Update `OverviewPage.tsx` — remove config hook usage and config-related props passed to TraceList

## Phase 2: Update Types

- [ ] Task 2.1: Remove `tracing_enabled` from `ConfigResponse` and `UpdateConfigRequest` in `types/api.ts`
- [ ] Task 2.2: Update `useConfig` hook if it references `tracing_enabled`

## Phase 3: Verification

- [ ] Task 3.1: `npm run build` succeeds
- [ ] Task 3.2: Dashboard loads without errors, TraceList renders without toggle
