# Implementation Plan: Remove Tracing Toggle UI (Frontend)

**Track ID:** tracing-always-on-fe_20260309231001Z

## Phase 1: Remove Toggle Component

- [x] Task 1.1: Delete `frontend/src/components/TracingToggle.tsx` and `TracingToggle.module.css`
- [x] Task 1.2: Update `TraceList.tsx` — remove TracingToggle import, render, and "enable tracing" conditional message
- [x] Task 1.3: Update `OverviewPage.tsx` — remove config hook usage and config-related props passed to TraceList

## Phase 2: Update Types

- [x] Task 2.1: Remove `tracing_enabled` from `ConfigResponse` and `UpdateConfigRequest` in `types/api.ts`
- [x] Task 2.2: Update `useConfig` hook if it references `tracing_enabled`

## Phase 3: Verification

- [x] Task 3.1: `npm run build` succeeds
- [x] Task 3.2: Dashboard loads without errors, TraceList renders without toggle
