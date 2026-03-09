# Product Guidelines

## Voice and Tone

Professional and technical. Documentation and CLI output should be precise, detailed, and assume technical competence from the user.

## Design Principles

1. **Simplicity over features** — One command does the right thing. Minimize required flags and configuration.
2. **Local-first and private** — Everything runs on the user's machine. No external services, no data leaves the workstation.
3. **Convention over configuration** — Sensible defaults for ports, paths, and behavior. Override only when needed.
4. **Reliability and observability** — Clear status output, structured logs, and actionable error messages. The user should always know what's happening.
5. **Schema-first APIs** — All inter-process and client-server communication is defined by machine-readable schemas (OpenAPI for REST/HTTP, AsyncAPI for events/streams) before implementation. Server stubs, client code, and models are generated from schemas, never hand-written. No new HTTP endpoint may be added without first updating the OpenAPI schema. No new event or message type may be added without first updating the AsyncAPI schema.
6. **Makefile as build entry point** — All builds, tests, and code generation go through the Makefile. See `code_styleguides/build.md` for conventions on frontend embedding, output directories, and VCS stamping.
7. **Thin adapters, shared domain logic** — CLI commands and REST handlers are thin adapters that convert input into domain commands/queries and dispatch to the service layer. Business logic lives exclusively in `core/service/`. Adapters never access stores directly — they receive services via constructor injection. This ensures consistent behavior regardless of entry point (CLI, API, webhook). When adding a new operation, implement it as a service method first, then wire it from both CLI and API adapters.

## Build Artifacts — Never Commit

**`backend/internal/adapter/dashboard/dist/`** contains frontend build output (compiled JS/CSS/HTML/assets). These files are:
- Built on demand via `make build-frontend`
- Embedded into the Go binary via `//go:embed dist/*`
- Listed in `.gitignore` — must NEVER be committed or staged

If `dist/` is missing, the Makefile's `ensure-dist` target creates a placeholder. Do not work around a missing `dist/` by committing build artifacts — run `make build-frontend` instead.

**`skills/`** (repo root) must not exist. Canonical skills live in `backend/internal/adapter/skills/embedded/`. Working copies are installed to `~/.claude/skills/`. Do not create or commit a `skills/` directory at the repo root.

## E2E Testing Requirements

All agents building UI functionality **must** develop automated E2E tests using the Playwright MCP plugin. E2E tests ensure the full stack (Go backend + React frontend + SQLite) works together correctly.

### Mandatory coverage

- **Happy path** — Primary user flows must be tested end-to-end via Playwright browser automation.
- **Edge cases** — Empty states, error responses, and boundary conditions must be covered.
- **Expected failures** — Verify error handling surfaces correctly in the UI.

### Mock agent binary

Agent-related flows must use the mock agent binary (`backend/internal/adapter/agent/testdata/mock-agent/`) instead of a real Claude API key. Configure mock behavior via environment variables (`MOCK_AGENT_EVENTS`, `MOCK_AGENT_DELAY`, `MOCK_AGENT_EXIT_CODE`, `MOCK_AGENT_INTERACTIVE`, `MOCK_AGENT_FAIL_AFTER`).

### Conventions

- E2E test files live in `frontend/e2e/` with `.spec.ts` extension.
- Import fixtures from `./fixtures` (not `@playwright/test` directly).
- Backend E2E helpers use `//go:build e2e` tag — skipped by `go test ./...`.
- Run via `make test-e2e`.
- See `frontend/e2e/README.md` for full documentation.
