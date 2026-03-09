# Implementation Plan: Testing Audit — Integration, E2E, and Smoke Tests

**Track ID:** testing-audit-e2e_20260308201000Z

## Phase 1: Fix Broken Test Suite (3 tasks)

### Task 1.1: Fix dist/ embed issue for test builds
- [x] Ensure `dist/` exists with `.gitkeep` and Makefile creates it before tests

### Task 1.2: Verify `make test` passes
- [x] All existing unit tests pass with `make test`

### Task 1.3: Add coverage reporting
- [x] `make test-coverage` target with coverage output

## Phase 2: Smoke Tests (4 tasks)

### Task 2.1: Add binary build smoke test
- [x] `cmd/kiloforge/main_test.go` verifies binary compiles

### Task 2.2: Add route registration smoke test
- [x] `rest/routes_test.go` exercises full ServeMux setup with all options
- [x] Verifies key routes respond with correct status codes
- [x] Badge route test verifies SVG content type

### Task 2.3: Add CLI command parsing smoke tests
- [x] Verifies all 19 subcommands are registered
- [x] Verifies `--help` doesn't panic for any command

### Task 2.4: Verify Phase 2
- [x] All smoke tests pass alongside existing unit tests

## Phase 3: Integration Test Infrastructure (3 tasks)

### Task 3.1: Set up integration test build tag
- [x] `integration_test.go` with `//go:build integration`
- [x] `startTestServer(t)` helper creates full server on random port

### Task 3.2: Add server integration tests
- [x] Health, agents, status, quota, locks, badge endpoints tested end-to-end

### Task 3.3: Add lock service integration test
- [x] Full lifecycle: acquire → heartbeat → list → release → verify empty
- [x] Conflict test: 409 on double acquire with current_holder

## Phase 4: Makefile & Test Organization (3 tasks)

### Task 4.1: Add Makefile test targets
- [x] `make test` (unit), `make test-smoke`, `make test-integration`, `make test-all`

### Task 4.2: Add test README
- [x] `backend/TESTING.md` with full testing guide

### Task 4.3: Final verification
- [x] `make test-all` passes (unit + integration)

---

**Total: 4 phases, 13 tasks**
