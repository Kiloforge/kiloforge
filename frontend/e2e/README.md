# E2E Testing

End-to-end tests using Playwright against a real Kiloforge server with SQLite and a mock agent binary.

## Quick Start

```bash
# From repo root:
make test-e2e

# Or manually:
cd frontend && npx playwright test
```

## Architecture

- **Backend E2E helpers** (`backend/internal/adapter/rest/e2e_helpers_test.go`): `startE2EServer()` boots a real HTTP server on a random port with SQLite. Uses `//go:build e2e` tag so `go test ./...` skips them by default.
- **Mock agent binary** (`backend/internal/adapter/agent/testdata/mock-agent/`): Standalone Go binary that simulates Claude CLI stream-JSON output. Built automatically by `startE2EServer()`.
- **Playwright tests** (`frontend/e2e/`): Browser tests using custom fixtures (`fixtures.ts`).

## Running Tests

### Headless (default)

```bash
make test-e2e
```

### Headed (visual debugging)

```bash
HEADED=1 npx playwright test --project=chromium
```

### Specific test file

```bash
npx playwright test e2e/smoke.spec.ts
```

## Mock Agent Binary

The mock agent simulates Claude CLI's `--output-format stream-json` protocol.

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `MOCK_AGENT_EVENTS` | JSON array of stream-JSON events to emit | Default init + content + result |
| `MOCK_AGENT_DELAY` | Milliseconds between events | `100` |
| `MOCK_AGENT_EXIT_CODE` | Process exit code | `0` |
| `MOCK_AGENT_INTERACTIVE` | Enable stdin echo mode | `false` |
| `MOCK_AGENT_FAIL_AFTER` | Emit N events then crash | Disabled |

### Default Output

```json
{"type":"init","session_id":"mock-session-001"}
{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello from mock agent"}}
{"type":"result","usage":{"input_tokens":100,"output_tokens":50},"cost":{"input_cost":0.001,"output_cost":0.0005}}
```

## Custom Fixtures

Import from `./fixtures` instead of `@playwright/test`:

```typescript
import { test, expect } from "./fixtures";

test("example", async ({ serverURL, apiClient }) => {
  // serverURL: base URL of the test server
  // apiClient: helper for direct REST calls (get, post, put, del)
  const resp = await apiClient.get("/health");
  expect(resp.ok).toBe(true);
});
```

## Adding New E2E Tests

1. Create a new `.spec.ts` file in `frontend/e2e/`
2. Import fixtures: `import { test, expect } from "./fixtures";`
3. Use `apiClient` for REST calls, `page` for browser interaction
4. For test data, use `seedTestData()` in backend E2E helpers
5. Never use a real Claude API key — use the mock agent binary for agent flows

## Test Data Seeding

Backend E2E helpers provide:

- `startE2EServer(t)` — boots server, builds mock agent, returns server handle
- `seedTestData(t, srv)` — creates test project and sample agents
- `cleanupTestData(t, srv)` — removes all agents between tests
