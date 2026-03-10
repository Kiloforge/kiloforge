# Tech Stack

## Language

- **Go 1.24** (latest stable)

## Frameworks & Libraries

### CLI
- **Cobra** — CLI framework (`github.com/spf13/cobra`)

### API
- **Fiber** — HTTP framework for the relay server API
- **OpenAPI 3.1** — Schema-first REST API design
  - `oapi-codegen` v2.5.1 — Go server interface, models, and client generation (strict server mode)
  - `github.com/oapi-codegen/runtime` — Runtime helpers for generated code
  - Schema location: `backend/api/openapi.yaml`
  - Config: `backend/api/cfg.yaml`
  - Generated files use `.gen.go` suffix — never edit manually
- **AsyncAPI 3.0** — Schema-first event/stream documentation
  - Documents SSE channels (`/-/events`), webhook payloads (`/webhook`)
  - Schema location: `backend/api/asyncapi.yaml`
  - Code generation TBD — currently documentation-only

### Database
- **SQLite** — Embedded database for agent/session tracking and state persistence

## Key Dependencies

- `github.com/google/uuid` — UUID generation for agent and session IDs
- `github.com/spf13/cobra` — CLI command framework

## Frontend

- **React 19** — UI framework
- **Vite 7** — Build tool and dev server
- **TypeScript 5.9** — Type safety
- Frontend lives in `frontend/`, builds to `backend/internal/adapter/dashboard/dist/` for Go embed

## Infrastructure

- **Local machine only** — runs on the developer's workstation
- **Docker** — required for running the Gitea instance
- **Gitea** — local git forge for PR management and webhooks

## Build & Tooling

- **Makefile** — build, test, and lint targets
- **go vet + staticcheck** — Primary Go linters (CI and `make lint`)
- **gofmt + goimports** — Format checking (CI and `make lint`)
- **golangci-lint** — Thorough Go linting (`make lint-full`, local use)
- **Code generation workflow**:
  - `make gen-api` — regenerate Go code from OpenAPI schema
  - `make verify-codegen` — verify generated code matches schema (CI gate)
