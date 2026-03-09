# Implementation Plan: Frontend Test Infrastructure and Makefile Integration

**Track ID:** frontend-test-infra_20260309225000Z

## Phase 1: Test Framework Setup

- [x] Task 1.1: Install vitest, @testing-library/react, @testing-library/jest-dom, @testing-library/user-event, jsdom, msw as dev dependencies
- [x] Task 1.2: Add vitest config to `vite.config.ts` (test environment: jsdom, globals: true)
- [x] Task 1.3: Create `frontend/src/test/setup.ts` — import jest-dom matchers, configure MSW server
- [x] Task 1.4: Add `"test": "vitest"` script to `package.json`

## Phase 2: Initial Tests

- [x] Task 2.1: Create `frontend/src/api/queryKeys.test.ts` — verify all key factory functions
- [x] Task 2.2: Create `frontend/src/hooks/useTour.test.ts` — verify mutation sends correct action-based body
- [x] Task 2.3: Create `frontend/src/components/AddProjectForm.test.tsx` — URL validation, form submission
- [x] Task 2.4: Create `frontend/src/utils/format.test.ts` — verify formatUSD, formatTokens, formatUptime
- [x] Task 2.5: Create `frontend/src/api/fetcher.test.ts` — verify fetcher success, error, non-JSON error

## Phase 3: Makefile Integration

- [x] Task 3.1: Update Makefile `test` target to also run `cd frontend && npm test -- --run`
- [x] Task 3.2: Update Makefile `test-all` target similarly

## Phase 4: Verification

- [x] Task 4.1: `cd frontend && npm test -- --run` passes
- [x] Task 4.2: `make test` runs both backend and frontend tests successfully
