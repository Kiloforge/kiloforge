# Tech Stack

## Language

- **Go 1.24** (latest stable)

## Frameworks & Libraries

### CLI
- **Cobra** — CLI framework (`github.com/spf13/cobra`)

### API
- **Fiber** — HTTP framework for the relay server API
- **OpenAPI** — API-first design with code generation (strict types mode)

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
- **golangci-lint** — Go linter
