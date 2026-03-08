# Implementation Plan: Enable Tracing by Default with Config API (Backend)

**Track ID:** tracing-default-on-be_20260309133000Z

## Phase 1: Default and Env Var

- [ ] Task 1.1: Update `backend/internal/adapter/config/defaults.go` — set `TracingEnabled` to `boolPtr(true)`
- [ ] Task 1.2: Add `KF_TRACING_ENABLED` to `backend/internal/adapter/config/env_adapter.go`
- [ ] Task 1.3: Update tests in `defaults_test.go` and `env_adapter_test.go` to verify new default and env var

## Phase 2: Config API Endpoint

- [ ] Task 2.1: Add `GET /api/config` and `PUT /api/config` to `backend/api/openapi.yaml` with request/response schemas
- [ ] Task 2.2: Regenerate server code (`oapi-codegen`)
- [ ] Task 2.3: Implement `GetConfig()` handler in `api_handler.go` — return current `tracing_enabled` and `dashboard_enabled`
- [ ] Task 2.4: Implement `UpdateConfig()` handler in `api_handler.go` — merge partial config, persist via JSON adapter, return updated config
- [ ] Task 2.5: Add config persistence helper — load config.json, merge fields, save
- [ ] Task 2.6: Add tests for GET/PUT config endpoints

## Phase 3: Verification

- [ ] Task 3.1: Verify `go test ./...` passes
- [ ] Task 3.2: Verify `make build` succeeds
