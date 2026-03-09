# Specification: E2E Tests: Health Check, Preflight Validation, and Onboarding Flow

**Track ID:** e2e-health-preflight-onboarding_20260309194831Z
**Type:** Chore
**Created:** 2026-03-09T19:48:31Z
**Status:** Draft

## Summary

Comprehensive E2E tests for health check, preflight validation (Claude auth, skills, consent, setup), and the guided tour onboarding flow, covering happy path, edge cases, and expected failures.

## Context

The Kiloforge application has several startup and validation flows that must work correctly for users:

1. **Health check** (`GET /api/health`) — basic server liveness
2. **Status endpoint** — returns system info including version
3. **Preflight validation** (`GET /api/preflight`) — checks Claude CLI auth, installed skills, consent state, and project setup
4. **Consent dialog** — appears when agent permissions have not been granted
5. **Skills install dialog** — appears when required skills are missing
6. **Setup required dialog** — appears when no project is configured
7. **Guided tour** — onboarding walkthrough for first-time users with step-by-step navigation and persistent state

These flows are critical to the user experience and must be tested with both browser UI interactions (via Playwright) and direct API validation.

## Codebase Analysis

### Existing patterns

- **Health endpoint** — `GET /api/health` returns `{"status": "ok"}` in `backend/internal/adapter/rest/api_handler.go`
- **Preflight endpoint** — `GET /api/preflight` checks auth, skills, consent, and setup status
- **Consent API** — `GET /api/consent` and `PUT /api/consent` for agent permissions consent
- **Skills API** — `GET /api/skills` and `POST /api/skills/install` for skill management
- **Tour API** — `GET /api/tour`, `PUT /api/tour`, `GET /api/tour/demo-board` for guided tour state
- **Frontend dialogs** — `ConsentDialog`, `SkillsInstallDialog`, `SetupRequiredDialog` components
- **Tour component** — guided tour with step navigation, localStorage/API persistence

### Test infrastructure dependency

All tests in this track rely on the E2E infrastructure from `e2e-infra-mock-agent_20260309194830Z`:
- `startE2EServer()` for booting a test server with mock agent
- `seedTestData()` for populating initial state
- Playwright fixtures for browser-driven tests

## Acceptance Criteria

- [ ] Health endpoint returns 200 with expected body
- [ ] Status endpoint returns system info including version
- [ ] Preflight endpoint validates all prerequisites (auth, skills, consent, setup)
- [ ] ConsentDialog appears when permissions not granted, can be accepted
- [ ] SkillsInstallDialog appears when skills missing
- [ ] SetupRequiredDialog appears when project not setup
- [ ] Guided tour starts on first visit, advances through steps, completes
- [ ] Tour state persists across page reloads
- [ ] Edge cases: concurrent preflight calls, rapid consent toggle
- [ ] Failure cases: server down during preflight, partial skill installation failure

## Dependencies

- `e2e-infra-mock-agent_20260309194830Z` — E2E test infrastructure and mock agent binary

## Blockers

None.

## Conflict Risk

- LOW — adds test files only, no production code changes.

## Out of Scope

- Project CRUD operations (covered by `e2e-project-management_20260309194832Z`)
- Agent spawning and lifecycle (covered by `e2e-agent-lifecycle_20260309194834Z`)
- Track management (covered by `e2e-track-management_20260309194833Z`)

## Technical Notes

### Test file structure

```
frontend/e2e/
  health-preflight.spec.ts    — health, status, preflight API tests
  consent-flow.spec.ts        — consent dialog UI tests
  onboarding-tour.spec.ts     — guided tour UI tests
```

### Preflight test scenarios

| Scenario | Consent | Skills | Setup | Expected |
|---|---|---|---|---|
| All clear | granted | installed | complete | No dialogs, dashboard loads |
| Missing consent | not granted | installed | complete | ConsentDialog shown |
| Missing skills | granted | missing | complete | SkillsInstallDialog shown |
| Missing setup | granted | installed | incomplete | SetupRequiredDialog shown |
| Fresh install | not granted | missing | incomplete | ConsentDialog first, then skills, then setup |

### Tour test scenarios

- Tour starts automatically on first visit (tour state = `pending`)
- Tour step navigation: next, previous, skip to end
- Tour completion sets state to `completed`
- Tour dismissal sets state to `dismissed`
- Reloading page resumes tour at current step (state = `active`)
- Completed/dismissed tour does not restart on reload

### Edge case handling

- Concurrent preflight: fire multiple `/api/preflight` requests simultaneously, verify no race conditions
- Rapid consent toggle: grant and revoke consent rapidly, verify final state is consistent
- Server error during preflight: mock server error response, verify UI shows appropriate error state

### Developer agent instructions

Use the Playwright MCP skill to verify all UI interactions. Run tests in headed mode during development to visually confirm dialog appearances, tour step transitions, and state persistence across reloads.

---

_Generated by conductor-track-generator for E2E health/preflight/onboarding tests_
