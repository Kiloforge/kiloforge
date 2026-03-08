# Implementation Plan: Disable Agent Actions Until Setup Complete (Frontend)

**Track ID:** setup-gate-fe_20260310234711Z

## Phase 1: ProjectPage Button Gating

- [ ] Task 1.1: Derive `setupIncomplete` boolean from `setupStatus` query result
- [ ] Task 1.2: Disable "Generate Tracks" button when `setupIncomplete` — add `disabled` attr and tooltip, prevent prompt form from opening
- [ ] Task 1.3: Disable "Sync" board button when `setupIncomplete` — add `disabled` attr and tooltip

## Phase 2: AdminPanel Gating and Error Handling

- [ ] Task 2.1: Add `disabled` prop to AdminPanel component — disable all operation buttons when true
- [ ] Task 2.2: Pass `setupIncomplete` as `disabled` from ProjectPage to AdminPanel
- [ ] Task 2.3: Add 428 error handler to AdminPanel mutation — show SetupRequiredDialog via callback
- [ ] Task 2.4: Add 412 error handler to AdminPanel mutation — show SkillsInstallDialog via callback

## Phase 3: Verification

- [ ] Task 3.1: `npm run build` succeeds with no errors
