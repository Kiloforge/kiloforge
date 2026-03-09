# Specification: E2E Test Infrastructure and Mock Agent Binary

**Track ID:** e2e-infra-mock-agent_20260309194830Z
**Type:** Chore
**Created:** 2026-03-09T19:48:30Z
**Status:** Draft

## Summary

Set up Playwright E2E test infrastructure with a mock agent binary that simulates Claude CLI stream-JSON output, enabling all subsequent E2E tracks to test against a fully wired server without requiring a real Claude API key.

## Context

Kiloforge has no E2E testing infrastructure. The application spans a Go backend (Fiber HTTP, SQLite, agent spawning via `exec.CommandContext`) and a React frontend (Vite, TanStack Query). The agent spawner invokes the `claude` CLI binary with `--output-format stream-json`, which produces events like `init`, `content_block_delta` (with `text_delta`), and `result` (with `usage`/`cost`). To test E2E flows without a real Claude API key, we need a mock agent binary that produces the same stream-JSON protocol.

The frontend communicates with agents via WebSocket (message types: `input`, `output`, `status`, `error`) and receives server-sent events at `/events` for real-time updates (`agent_update`, `agent_removed`, `quota_update`, `track_update`, `board_update`, etc.).

## Codebase Analysis

### Existing patterns

- **Integration tests** use `startTestServer()` helper that wires a real Fiber server on a random port with SQLite — see `backend/internal/adapter/rest/` test files.
- **Test mocks** in `backend/internal/core/testutil/mocks.go` provide mock implementations of port interfaces.
- **Agent spawner** in `backend/internal/adapter/agent/` uses `exec.CommandContext` to run the `claude` CLI binary.
- **Stream-JSON parsing** — the spawner reads stdout line by line, parsing JSON events with type discrimination.

### Mock agent approach

A small Go binary at `backend/internal/adapter/agent/testdata/mock-agent/main.go` that:
1. Accepts the same flags as `claude` CLI (at minimum `--output-format`, `--model`, `--verbose`)
2. Reads environment variables to configure behavior
3. Outputs stream-JSON events to stdout
4. Supports both non-interactive (fire-and-forget) and interactive (stdin/stdout echo) modes

## Acceptance Criteria

- [ ] Playwright installed and configured for the project (`playwright.config.ts`)
- [ ] Mock agent Go binary created that accepts claude CLI flags and outputs configurable stream-JSON events
- [ ] Mock agent supports both non-interactive (developer/reviewer) and interactive (stdin/stdout) modes
- [ ] Mock agent behavior is configurable via environment variables (delay, events, exit code, failure mode)
- [ ] E2E test helper: `startTestServer()` that boots a real server on random port with SQLite, using mock agent binary path
- [ ] E2E test helper: `seedTestData()` for populating projects, tracks, agents
- [ ] Makefile target `test-e2e` added with `//go:build e2e` tag convention
- [ ] Test that server starts, health check passes, and mock agent can be spawned and produces expected output
- [ ] Documentation in test directory explaining mock agent usage and E2E conventions
- [ ] All E2E tests use Playwright to drive the browser and verify UI behavior

## Dependencies

None (foundation track).

## Blockers

None.

## Conflict Risk

- LOW — adds new test infrastructure files only, no production code changes.

## Out of Scope

- Testing any specific application feature (that's for subsequent tracks)
- CI pipeline integration (can be a follow-up)
- Visual regression testing
- Performance/load testing

## Technical Notes

### Mock agent binary location

```
backend/internal/adapter/agent/testdata/mock-agent/main.go
```

Build the mock agent as part of test setup (not committed as a binary).

### Environment variable configuration

| Variable | Description | Default |
|---|---|---|
| `MOCK_AGENT_EVENTS` | JSON array of stream-JSON events to emit | Default init + content + result sequence |
| `MOCK_AGENT_DELAY` | Milliseconds between events | `100` |
| `MOCK_AGENT_EXIT_CODE` | Process exit code | `0` |
| `MOCK_AGENT_INTERACTIVE` | Enable stdin echo mode (`true`/`false`) | `false` |
| `MOCK_AGENT_FAIL_AFTER` | Emit N events then crash | Disabled |

### Default stream-JSON event sequence

```json
{"type":"init","session_id":"mock-session-001"}
{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello from mock agent"}}
{"type":"result","usage":{"input_tokens":100,"output_tokens":50},"cost":{"input_cost":0.001,"output_cost":0.0005}}
```

### Interactive mode

When `MOCK_AGENT_INTERACTIVE=true`, the mock agent:
1. Emits `init` event
2. Reads lines from stdin
3. For each line, emits a `content_block_delta` echoing the input
4. On EOF or empty line, emits `result` and exits

### Playwright configuration

- Support both headless (CI) and headed (debug) modes via `HEADED` env var
- Base URL configured to point at the E2E test server
- Global setup builds mock agent binary and starts test server
- Global teardown shuts down test server

### startTestServer wiring

The E2E `startTestServer()` should:
1. Build mock agent binary to a temp directory
2. Create a temp SQLite database
3. Boot the Fiber server with `AgentBinaryPath` overridden to mock agent
4. Return the server URL, cleanup function, and HTTP client

### Developer agent instructions

When building this track, use the Playwright MCP skill to verify E2E tests work in the browser. Run tests in headed mode during development for visual verification.

---

_Generated by conductor-track-generator for E2E test infrastructure_
