# Implementation Plan: Reframe Quota System — Backend

**Track ID:** quota-reframe-be_20260309103000Z

## Phase 1: OpenAPI Spec & Code Generation

- [x] Task 1.1: Update `backend/api/openapi.yaml` — reframe `QuotaInfo` schema: rename `total_cost_usd` to `estimated_cost_usd`, promote token fields to required, add `cache_read_tokens` and `cache_creation_tokens`
- [x] Task 1.2: Update `QuotaAgentUsage` schema — rename `cost_usd` to `estimated_cost_usd`, add `cache_read_tokens` and `cache_creation_tokens`
- [x] Task 1.3: Update `Agent` schema — rename `cost_usd` to `estimated_cost_usd`
- [x] Task 1.4: Regenerate server code: `oapi-codegen` → `backend/internal/adapter/rest/gen/`

## Phase 2: Backend Tracker & Handler Updates

- [x] Task 2.1: Update `api_handler.go` `GetQuota()` — map to new field names (`estimated_cost_usd`), include cache token fields
- [x] Task 2.2: Update `domainAgentToGen()` in `api_handler.go` — rename `CostUsd` to `EstimatedCostUsd`, add cache tokens
- [x] Task 2.3: Deprecate `MaxSessionCostUSD` — remove budget enforcement from `spawner.go` `checkQuota()`, keep only `IsRateLimited()` check
- [x] Task 2.4: Update `config.go` — keep `MaxSessionCostUSD` field for backward compat but mark as deprecated (no functional effect)

## Phase 3: SSE & AsyncAPI Updates

- [x] Task 3.1: Update `backend/api/asyncapi.yaml` — reframe `QuotaPayload` with `estimated_cost_usd` and token fields
- [x] Task 3.2: Update `dashboard/handlers.go` `quotaResponse()` — map to new field names
- [x] Task 3.3: Update `dashboard/watcher.go` — add token totals to `watcherState` for delta detection

## Phase 4: Tests & Verification

- [x] Task 4.1: Update `tracker_test.go` — verify field names and values in AgentUsage/TotalUsage
- [x] Task 4.2: Update `api_handler_test.go` — verify quota response uses new field names
- [x] Task 4.3: Update `spawner.go` tests — verify budget check removed, only rate-limit check remains
- [x] Task 4.4: Verify `go test ./...` passes
- [x] Task 4.5: Verify `make build` succeeds
