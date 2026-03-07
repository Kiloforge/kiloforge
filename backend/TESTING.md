# Testing Guide

## Running Tests

```bash
# Unit tests only (fast, no infrastructure needed)
make test

# Smoke tests (binary builds, route registration, CLI parsing)
make test-smoke

# Integration tests (starts real HTTP servers on random ports)
make test-integration

# All tests (unit + integration)
make test-all

# Unit tests with coverage report
make test-coverage
```

## Test Organization

### Build Tags

| Tag | Purpose | Dependencies |
|-----|---------|-------------|
| (none) | Unit tests — always run | None |
| `integration` | Server integration tests | Temp dirs, real HTTP |

Unit tests run by default with `go test ./...`. Integration tests require `-tags=integration`.

### File Naming

```
*_test.go                     — Unit tests (default build tag)
integration_test.go           — Integration tests (//go:build integration)
lock_integration_test.go      — Scoped integration tests (//go:build integration)
```

### Test Locations

```
cmd/crelay/main_test.go                          — Binary build smoke test
internal/adapter/cli/cli_test.go                 — CLI command registration and help
internal/adapter/rest/routes_test.go             — Route registration smoke tests
internal/adapter/rest/server_test.go             — Webhook handler unit tests
internal/adapter/rest/api_handler_test.go        — API handler unit tests
internal/adapter/rest/integration_test.go        — Server integration tests
internal/adapter/rest/lock_integration_test.go   — Lock lifecycle integration tests
internal/core/testutil/mocks.go                  — Shared test doubles
```

## Adding Tests

### New Unit Test

Add `*_test.go` in the same package. Use `t.Parallel()` where safe.

### New Integration Test

1. Add `//go:build integration` at the top of the file
2. Use `startTestServer(t)` helper from `integration_test.go`
3. Make real HTTP requests against the returned URL
4. The server cleans up automatically via `t.Cleanup`

### Test Doubles

Use `internal/core/testutil/mocks.go` for shared mocks. For one-off stubs, define them locally in the test file. See `server_test.go` for examples of `fakeSpawner`, `stubAgentLister`.

## Smoke Tests

Smoke tests catch critical failures that unit tests miss:

- **Binary build test**: Verifies the binary compiles (catches import cycles)
- **Route registration test**: Exercises `ServeMux` setup (catches route conflicts)
- **CLI registration test**: Verifies all commands are wired (catches missing `AddCommand`)
- **CLI help test**: Verifies `--help` doesn't panic for any command
