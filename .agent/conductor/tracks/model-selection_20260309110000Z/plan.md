# Implementation Plan: Configurable Model Selection with Opus Default

**Track ID:** model-selection_20260309110000Z

## Phase 1: Config & Domain

- [x] Task 1.1: Add `Model string` field to `Config` struct in `config.go` with JSON tag `"model,omitempty"`
- [x] Task 1.2: Add `Model: "opus"` default in `defaults.go`
- [x] Task 1.3: Add `KF_MODEL` env var support in `env_adapter.go`
- [x] Task 1.4: Add `Model string` field to `AgentInfo` in `domain/agent.go`

## Phase 2: Spawner & Recovery

- [x] Task 2.1: Add `Model string` to `SpawnDeveloperOpts` struct
- [x] Task 2.2: Add `Model string` to `ReviewerOpts` in `port/agent_spawner.go`
- [x] Task 2.3: Update `SpawnDeveloper()` — pass `--model` flag to `claude` command, record model in `AgentInfo`
- [x] Task 2.4: Update `SpawnReviewer()` — pass `--model` flag to `claude` command, record model in `AgentInfo`
- [x] Task 2.5: Update `recovery.go` resume command — pass `--model` flag

## Phase 3: CLI & API

- [x] Task 3.1: Update `implement.go` — pass `cfg.Model` to `SpawnDeveloperOpts`
- [x] Task 3.2: Update `openapi.yaml` — add `model` field to `Agent` schema
- [x] Task 3.3: Regenerate server code: `oapi-codegen` → `backend/internal/adapter/rest/gen/`
- [x] Task 3.4: Update `api_handler.go` — map `Model` field in agent response (`domainAgentToGen`)
- [x] Task 3.5: Update REST server `SpawnReviewer` call — pass model from config

## Phase 4: Frontend & Tests

- [x] Task 4.1: Add `model?: string` to `Agent` interface in `frontend/src/types/api.ts`
- [x] Task 4.2: Display model name on `AgentCard` (small label, e.g., "opus")
- [x] Task 4.3: Update config tests — verify model default and env override
- [x] Task 4.4: Verify `go test ./...` passes
- [x] Task 4.5: Verify `make build` succeeds
