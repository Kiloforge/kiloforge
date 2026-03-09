# Implementation Plan: Fix Skill Prerequisite Chain — Frontend Proactive Gating

**Track ID:** fix-skill-prereq-fe_20260310000645Z

## Phase 1: Preflight Query

- [x] Task 1.1: Add `preflight` query key to `queryKeys.ts`
- [x] Task 1.2: Add preflight query to ProjectPage — fetch `GET /api/preflight`
- [x] Task 1.3: Derive `skillsMissing`, `setupIncomplete`, `actionsDisabled`, and `disabledReason` from preflight + setupStatus

## Phase 2: Button Gating Update

- [x] Task 2.1: Update "Generate Tracks" and "Sync" button disabled state to use `actionsDisabled` instead of `setupIncomplete`
- [x] Task 2.2: Update button tooltips to use `disabledReason` instead of hardcoded "Run kiloforge setup first"
- [x] Task 2.3: Update setup banner to show skills-missing state vs setup-incomplete state with appropriate action buttons
- [x] Task 2.4: Add `disabledReason` prop to AdminPanel — replace hardcoded tooltip string

## Phase 3: Cache Invalidation

- [x] Task 3.1: Invalidate `preflight` query after skills install completes
- [x] Task 3.2: Invalidate `setupStatus` query after setup completes (already done — verify still works)

## Phase 4: Verification

- [x] Task 4.1: `npm run build` succeeds with no errors
