# Implementation Plan: Remove Optional Tracing — Always-On (Backend)

**Track ID:** tracing-always-on-be_20260309231000Z

## Phase 1: Remove Config Field

- [x] Task 1.1: Remove `TracingEnabled *bool` from config struct and `IsTracingEnabled()` method in `config.go`
- [x] Task 1.2: Remove `KF_TRACING_ENABLED` parsing from `env_adapter.go`
- [x] Task 1.3: Remove `TracingEnabled` overlay from `merger.go`
- [x] Task 1.4: Update/remove related tests in `config_test.go` and `env_adapter_test.go`

## Phase 2: Always Initialize Tracing

- [x] Task 2.1: Remove `if cfg.IsTracingEnabled()` guards in `serve.go` and `implement.go` — always call `tracing.Init()` and append `WithTracer`

## Phase 3: Update Config API

- [x] Task 3.1: Remove `tracing_enabled` from `ConfigResponse` and `UpdateConfigRequest` in `openapi.yaml`
- [x] Task 3.2: Regenerate API code (`oapi-codegen`)
- [x] Task 3.3: Update `GetConfig`/`UpdateConfig` handlers to not reference tracing

## Phase 4: Verification

- [x] Task 4.1: `make test` passes
- [x] Task 4.2: `make build` succeeds
