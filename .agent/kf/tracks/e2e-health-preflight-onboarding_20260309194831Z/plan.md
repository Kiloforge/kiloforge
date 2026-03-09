# Implementation Plan: E2E Tests: Health Check, Preflight Validation, and Onboarding Flow

**Track ID:** e2e-health-preflight-onboarding_20260309194831Z

## Phase 1: Health and Status Tests

- [ ] Task 1.1: Create `frontend/e2e/health-preflight.spec.ts` — test `GET /api/health` returns 200 with `{"status":"ok"}`, verify response headers and content type
- [ ] Task 1.2: Test status endpoint — `GET /api/status` returns system info including version string, uptime, and configuration summary
- [ ] Task 1.3: Test health endpoint failure — stop the backend, verify Playwright detects connection refused, verify UI shows offline/error state when navigating to dashboard

## Phase 2: Preflight Tests

- [ ] Task 2.1: Test all-clear preflight — seed server with consent granted, skills installed, project setup complete; hit `GET /api/preflight`, verify all checks pass, verify dashboard loads without any blocking dialogs
- [ ] Task 2.2: Test missing skills preflight — seed server with consent granted but skills not installed; verify preflight returns skills-missing status, verify SkillsInstallDialog appears in UI
- [ ] Task 2.3: Test missing consent preflight — seed server with skills installed but consent not granted; verify preflight returns consent-missing status, verify ConsentDialog appears in UI
- [ ] Task 2.4: Test missing setup preflight — seed server with consent and skills but no project configured; verify preflight returns setup-incomplete status, verify SetupRequiredDialog appears in UI

## Phase 3: Consent Flow Tests

- [ ] Task 3.1: Test grant consent happy path — navigate to dashboard, verify ConsentDialog appears, click accept button, verify dialog dismisses, verify consent state persisted via `GET /api/consent`
- [ ] Task 3.2: Test consent dialog UI interaction — verify dialog shows correct permission descriptions, verify cancel/dismiss behavior, verify keyboard accessibility (Escape key)
- [ ] Task 3.3: Test consent state persistence — grant consent, reload page, verify ConsentDialog does not reappear, verify `GET /api/consent` returns granted state

## Phase 4: Tour Tests

- [ ] Task 4.1: Create `frontend/e2e/onboarding-tour.spec.ts` — test tour auto-starts on first visit when tour state is `pending`, verify first step content is displayed, verify tour overlay/highlight is visible
- [ ] Task 4.2: Test tour step navigation — advance through all tour steps using next button, verify each step shows correct content and highlights the correct UI element, test previous button navigates back
- [ ] Task 4.3: Test tour completion — advance through all steps to final step, click complete/finish button, verify tour state changes to `completed` via `GET /api/tour`, verify tour UI dismissed
- [ ] Task 4.4: Test tour state persistence across reload — start tour, advance to step 3, reload page, verify tour resumes at step 3 (not step 1), verify tour state shows `active` with `current_step: 3`

## Phase 5: Edge and Failure Cases

- [ ] Task 5.1: Test concurrent preflight calls — fire 5 simultaneous `GET /api/preflight` requests using `Promise.all`, verify all return consistent results, verify no 500 errors or data corruption
- [ ] Task 5.2: Test server errors during onboarding — mock API error on preflight endpoint, verify UI shows a user-friendly error message or retry option rather than a blank screen or unhandled exception
- [ ] Task 5.3: Test partial failure scenarios — skills install endpoint returns partial success (some skills installed, some failed), verify UI shows which skills failed, verify retry is possible for failed skills only
