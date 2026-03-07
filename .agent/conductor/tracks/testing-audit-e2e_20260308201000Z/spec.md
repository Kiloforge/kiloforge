# Specification: Testing Audit — Integration, E2E, and Smoke Tests

**Track ID:** testing-audit-e2e_20260308201000Z
**Type:** Chore
**Created:** 2026-03-08T20:10:00Z
**Status:** Draft

## Summary

Audit and fix the project's testing infrastructure, then add integration/smoke tests that catch startup failures, route registration panics, and cross-package regressions. The recent `ServeMux` route conflict panic on `crelay up` went completely undetected — no test exercises server startup or route registration.

## Context

The project has 42 test files and 296 unit tests, but every test uses mocks in isolation. There are zero integration, e2e, or smoke tests. The `make test` target itself is broken because the dashboard `dist/` embed directory is missing. This means:

1. The test suite doesn't even run cleanly
2. No test verifies that the application can build and start
3. No test catches route registration conflicts (the exact bug that crashed `crelay up`)
4. 20+ CLI commands have zero test coverage

## Codebase Analysis — Test Audit Results

### What's well-tested (unit level)
- Service layer: lifecycle, board, PR, track, cleanup — all with mocks and table-driven tests
- Config system: 8 test files covering defaults → JSON → env → flags precedence
- HTTP handlers: individual handler functions tested via `httptest`
- Persistence: JSON file stores thoroughly tested
- Lock service: 27 tests including concurrency and reaper
- Gitea client: 26 tests with fake HTTP backends

### Critical gaps

| Gap | Risk | Impact |
|-----|------|--------|
| **`go test ./...` is broken** | Critical | CI/local test suite fails due to missing `dist/` embed |
| **No server startup test** | Critical | Route registration panics go undetected |
| **No CLI command tests** | High | 20+ commands with zero coverage |
| **No integration tests** | High | Cross-package bugs only found manually |
| **No build/smoke test** | High | Compilation failures not caught |
| **Dashboard embed breaks dependent packages** | High | rest, cli packages can't be tested |

### Packages without tests
- `internal/core/port/` — interfaces only (acceptable)
- `internal/core/testutil/` — test helpers (acceptable)
- `cmd/crelay/` — entry point (not acceptable — needs smoke test)

### Test infrastructure
- Mock library: `internal/core/testutil/mocks.go` — good, comprehensive
- Makefile: `make test` runs `go test -race ./...` but currently fails
- No CI pipeline (local-only project)
- No coverage reporting
- No test build tags for integration vs unit separation

## Acceptance Criteria

- [ ] `make test` passes cleanly (fix `dist/` embed issue for test builds)
- [ ] Smoke test: verify `go build` succeeds and binary starts without panic
- [ ] Route registration test: exercise `Server.Run()` and verify all routes register without conflict
- [ ] Server integration test: start full HTTP server, hit key endpoints, verify responses
- [ ] CLI smoke tests: verify commands parse flags correctly and fail gracefully without infrastructure
- [ ] Build tag separation: `go test ./...` runs unit tests; `go test -tags=integration ./...` runs integration tests
- [ ] Makefile targets: `make test` (unit), `make test-integration` (integration), `make test-all` (both)
- [ ] Test that adding a new route with a conflict causes a test failure (regression prevention)

## Dependencies

- **fix-route-conflict_20260308200100Z**: Already completed. Routes are now correct.

## Blockers

None.

## Conflict Risk

- **openapi-codegen_20260308200001Z**: Medium — when routes migrate to generated code, integration tests will need updating. But the test infrastructure itself is durable.

## Out of Scope

- Full coverage metrics and enforcement (future track)
- Performance/load testing
- Fuzzing
- External dependency testing (real Docker, real Gitea)

## Technical Notes

### Fix dist/ embed for tests
The `//go:embed all:dist` in `dashboard/embed.go` breaks any test that transitively imports the dashboard package (including `rest`, `cli`). Options:
1. **Build tag guard**: Use `//go:build !testing` on embed, provide test stub
2. **Ensure dist/ always exists**: `make test` first runs `mkdir -p backend/internal/adapter/dashboard/dist && touch backend/internal/adapter/dashboard/dist/.gitkeep`
3. **Extract embed to separate package**: Isolate the embed so it doesn't infect test builds

Option 2 is simplest; option 1 is most correct.

### Smoke test approach
```go
// backend/cmd/crelay/main_test.go
func TestBinaryBuilds(t *testing.T) {
    cmd := exec.Command("go", "build", "-o", "/dev/null", ".")
    cmd.Dir = "../../cmd/crelay"
    if err := cmd.Run(); err != nil {
        t.Fatalf("binary build failed: %v", err)
    }
}
```

### Route registration test
```go
// backend/internal/adapter/rest/routes_test.go
func TestServerRouteRegistration(t *testing.T) {
    // Create server with all options enabled
    // Call Run() in a goroutine with a context that cancels immediately
    // Verify no panic occurred
}
```

### Integration test pattern
```go
//go:build integration

func TestServerIntegration(t *testing.T) {
    srv := startTestServer(t)
    defer srv.Close()

    // Hit /health
    resp, _ := http.Get(srv.URL + "/health")
    assert(resp.StatusCode, 200)

    // Hit /-/api/agents
    resp, _ = http.Get(srv.URL + "/-/api/agents")
    assert(resp.StatusCode, 200)
}
```

### CLI smoke test pattern
```go
func TestUpCommand_RequiresInit(t *testing.T) {
    cmd := rootCmd
    cmd.SetArgs([]string{"up"})
    err := cmd.Execute()
    // Should fail gracefully with "not initialized" message, not panic
}
```

---

_Generated by conductor-track-generator from prompt: "testing audit with e2e/integration tests — application fails to start without any test detecting it"_
