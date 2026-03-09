# Implementation Plan: Fix Skill Prerequisite Chain — Frontend Proactive Gating

**Track ID:** fix-skill-prereq-fe_20260310000645Z

## Phase 1: Preflight Query

- [ ] Task 1.1: Add `preflight` query key to `queryKeys.ts`
- [ ] Task 1.2: Add preflight query to ProjectPage — fetch `GET /api/preflight`
- [ ] Task 1.3: Derive `skillsMissing`, `setupIncomplete`, `actionsDisabled`, and `disabledReason` from preflight + setupStatus

## Phase 2: Button Gating Update

- [ ] Task 2.1: Update "Generate Tracks" and "Sync" button disabled state to use `actionsDisabled` instead of `setupIncomplete`
- [ ] Task 2.2: Update button tooltips to use `disabledReason` instead of hardcoded "Run kiloforge setup first"
- [ ] Task 2.3: Update setup banner to show skills-missing state vs setup-incomplete state with appropriate action buttons
- [ ] Task 2.4: Add `disabledReason` prop to AdminPanel — replace hardcoded tooltip string

## Phase 3: Cache Invalidation

- [ ] Task 3.1: Invalidate `preflight` query after skills install completes
- [ ] Task 3.2: Invalidate `setupStatus` query after setup completes (already done — verify still works)

## Phase 4: Verification

- [ ] Task 4.1: `npm run build` succeeds with no errors
