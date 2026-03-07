# Implementation Plan: Testing Audit — Integration, E2E, and Smoke Tests

**Track ID:** testing-audit-e2e_20260308201000Z

## Phase 1: Fix Broken Test Suite (3 tasks)

### Task 1.1: Fix dist/ embed issue for test builds
Ensure `backend/internal/adapter/dashboard/dist/` exists with at least a `.gitkeep` so `//go:embed all:dist` doesn't break test compilation. Update Makefile `test` target to create the directory if missing before running tests.

### Task 1.2: Verify `make test` passes
Run `make test` and fix any other build/test failures. All existing unit tests should pass.

### Task 1.3: Add coverage reporting
Update Makefile `test` target to output coverage: `go test -race -coverprofile=coverage.out ./...`. Add `make test-coverage` target that opens HTML report.

## Phase 2: Smoke Tests (4 tasks)

### Task 2.1: Add binary build smoke test
Create `backend/cmd/crelay/main_test.go` that verifies the binary compiles without error. This catches import cycles, missing dependencies, and compilation errors.

### Task 2.2: Add route registration smoke test
Create `backend/internal/adapter/rest/routes_test.go` that:
- Creates a `Server` with all options enabled (dashboard, Gitea proxy, board sync)
- Calls the route registration path (same as `Run()` but without `ListenAndServe`)
- Verifies no panic occurs
- Verifies key routes are registered by making requests to the mux

### Task 2.3: Add CLI command parsing smoke tests
Create `backend/internal/adapter/cli/cli_test.go` that:
- Verifies all registered commands exist on the root command
- Verifies `--help` doesn't panic for each command
- Verifies commands that require init fail gracefully without infrastructure

### Task 2.4: Verify Phase 2
Run all smoke tests. Intentionally break a route registration to verify the test catches it.

## Phase 3: Integration Test Infrastructure (3 tasks)

### Task 3.1: Set up integration test build tag
Create `backend/internal/adapter/rest/integration_test.go` with `//go:build integration` tag. Set up a test helper that:
- Creates a temp data directory
- Initializes a `Config` with test values
- Creates a real `Server` (not mocked)
- Starts it on a random port
- Returns a cleanup function

### Task 3.2: Add server integration tests
Using the test helper from 3.1, write integration tests that:
- Start the full server
- Hit `/health` and verify 200 response
- Hit `/-/api/agents` and verify empty JSON array
- Hit `/-/api/status` and verify response shape
- Hit `/-/api/locks` and verify empty list
- Hit `/-/api/badges/track/nonexistent` and verify SVG response with "pending"
- Hit an unknown route and verify it doesn't 404 (Gitea proxy catch-all)

### Task 3.3: Add lock service integration test
Integration test that exercises the full lock lifecycle through HTTP:
- POST `/-/api/locks/{scope}/acquire` — acquire a lock
- POST `/-/api/locks/{scope}/heartbeat` — extend TTL
- GET `/-/api/locks` — verify lock appears in list
- DELETE `/-/api/locks/{scope}` — release
- GET `/-/api/locks` — verify empty

## Phase 4: Makefile & Test Organization (3 tasks)

### Task 4.1: Add Makefile test targets
```makefile
test:              # Unit tests only (fast, no infra needed)
test-integration:  # Integration tests (needs temp dirs, real HTTP)
test-all:          # Both unit and integration
test-smoke:        # Just smoke tests (binary builds, routes register)
```

### Task 4.2: Add test README
Create `backend/TESTING.md` documenting:
- How to run tests (`make test`, `make test-integration`)
- Test file naming conventions
- When to use build tags
- How to add new integration tests
- Mock patterns and testutil usage

### Task 4.3: Final verification
Run `make test-all`. Verify:
- All unit tests pass
- All smoke tests pass
- All integration tests pass
- A deliberately broken route causes test failure

---

**Total: 4 phases, 13 tasks**
