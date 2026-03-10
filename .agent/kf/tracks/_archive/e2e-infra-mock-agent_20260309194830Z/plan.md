# Implementation Plan: E2E Test Infrastructure and Mock Agent Binary

**Track ID:** e2e-infra-mock-agent_20260309194830Z

## Phase 1: Playwright Setup

- [ ] Task 1.1: Install Playwright and dependencies — `npm init playwright@latest` in `frontend/`, configure `playwright.config.ts` with headless/headed toggle via `HEADED` env var, set base URL to be dynamic
- [ ] Task 1.2: Create base test fixture in `frontend/e2e/fixtures.ts` — extend Playwright `test` with custom `serverURL` fixture that reads from environment, add `apiClient` helper for direct REST calls
- [ ] Task 1.3: Add Playwright to `.gitignore` (test-results/, playwright-report/) and verify `npx playwright test` runs with a trivial smoke test

## Phase 2: Mock Agent Binary

- [ ] Task 2.1: Create `backend/internal/adapter/agent/testdata/mock-agent/main.go` — parse CLI flags (`--output-format`, `--model`, `--verbose`, `--print`, `-p`), read env vars (`MOCK_AGENT_EVENTS`, `MOCK_AGENT_DELAY`, `MOCK_AGENT_EXIT_CODE`, `MOCK_AGENT_INTERACTIVE`, `MOCK_AGENT_FAIL_AFTER`)
- [ ] Task 2.2: Implement default stream-JSON output — emit `init`, `content_block_delta`, and `result` events with configurable delays between them
- [ ] Task 2.3: Implement custom event sequence — when `MOCK_AGENT_EVENTS` is set, parse JSON array and emit those events instead of defaults
- [ ] Task 2.4: Implement interactive mode — when `MOCK_AGENT_INTERACTIVE=true`, emit `init`, then read stdin lines and echo as `content_block_delta` events, emit `result` on EOF
- [ ] Task 2.5: Write unit tests for mock agent — verify default output, custom events, interactive mode, exit codes, and fail-after behavior; tests go in `backend/internal/adapter/agent/testdata/mock-agent/main_test.go`

## Phase 3: E2E Server Helpers

- [ ] Task 3.1: Create `backend/internal/adapter/rest/e2e_helpers_test.go` with `//go:build e2e` tag — implement `startE2EServer()` that builds mock agent binary to temp dir, creates temp SQLite DB, boots Fiber server with mock agent path override, returns server URL + cleanup func
- [ ] Task 3.2: Implement `seedTestData()` helper — create test project, test tracks (various statuses), and test agents via direct service calls or REST API
- [ ] Task 3.3: Implement `cleanupTestData()` helper — reset SQLite database between tests, kill any spawned mock agent processes
- [ ] Task 3.4: Write health check smoke test — `TestE2E_HealthCheck` starts server, hits `GET /api/health`, verifies 200 response with expected body structure

## Phase 4: Integration and Makefile

- [ ] Task 4.1: Add `test-e2e` target to `Makefile` — build mock agent, start test server, run `npx playwright test`, tear down; use `//go:build e2e` tag so `go test ./...` skips E2E by default
- [ ] Task 4.2: Create `frontend/e2e/README.md` documenting E2E conventions — mock agent usage, env vars, how to run headed vs headless, how to add new E2E tests, test data seeding
- [ ] Task 4.3: Run full verification — execute `make test-e2e`, confirm health check smoke test passes end-to-end with Playwright driving the browser, fix any issues
