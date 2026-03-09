# Specification: Reframe Quota System — Tokens and Rate Limits as Primary Metrics (Backend)

**Track ID:** quota-reframe-be_20260309103000Z
**Type:** Refactor
**Created:** 2026-03-09T10:30:00Z
**Status:** Draft

## Summary

Reframe the quota tracking system to treat token usage and rate limits as the primary metrics, with USD cost as secondary informational data ("estimated API-equivalent cost"). Users are on Claude Code subscriptions — the real constraints are token consumption and rate limits, not dollar cost.

## Context

The current quota system was built around USD cost as the primary metric (`TotalCostUSD`, `MaxSessionCostUSD`, `cost_usd` per agent). This framing is misleading for subscription users — they don't pay per-token, so "Total Cost: $4.23" implies a billing event that doesn't exist. The real value proposition is showing:

1. **Token consumption** — how many tokens each agent is consuming (input/output/cache)
2. **Rate limit status** — whether agents are being throttled by the subscription quota
3. **Estimated API cost** — informational: "this work would cost $X on the API" (nice marketing for the subscription value)

## Codebase Analysis

### Files to modify

**Core tracker:**
- `backend/internal/adapter/agent/tracker.go` — `AgentUsage.TotalCostUSD` and `TotalUsage.TotalCostUSD` stay but become secondary. Add model tracking.
- `backend/internal/adapter/agent/parser.go` — `StreamEvent.CostUSD` already parsed from Claude Code output. Add model field parsing if available.
- `backend/internal/adapter/agent/spawner.go` — `checkQuota()` uses `MaxSessionCostUSD` budget check. Reframe to token-based or remove (subscription handles limits).

**API spec & handlers:**
- `backend/api/openapi.yaml` — `QuotaInfo` schema: rename/reframe `total_cost_usd` to `estimated_cost_usd`, add token summary fields as top-level, add `cache_read_tokens`/`cache_creation_tokens`.
- `backend/internal/adapter/rest/api_handler.go` — `GetQuota()` handler builds response. Update field mapping.
- `backend/internal/adapter/rest/gen/` — regenerate from updated OpenAPI spec.

**SSE & dashboard:**
- `backend/api/asyncapi.yaml` — `QuotaPayload` schema: reframe `total_cost_usd`.
- `backend/internal/adapter/dashboard/watcher.go` — `watcherState` tracks `totalCost` for delta detection. Add token totals.
- `backend/internal/adapter/dashboard/handlers.go` — `quotaResponse()` builds SSE payload. Update fields.

**Config:**
- `backend/internal/adapter/config/config.go` — `MaxSessionCostUSD` field. Consider deprecating in favor of rate-limit-only approach (subscription enforces limits naturally).

## Acceptance Criteria

- [ ] `QuotaInfo` API response includes `estimated_cost_usd` (renamed from `total_cost_usd`) clearly marked as informational
- [ ] Token fields (`input_tokens`, `output_tokens`, `cache_read_tokens`, `cache_creation_tokens`) are top-level non-optional fields in the quota response
- [ ] Per-agent usage includes `estimated_cost_usd` (renamed from `cost_usd`) and all token breakdown fields
- [ ] Rate limit status remains the primary constraint indicator
- [ ] `MaxSessionCostUSD` config is deprecated — rate limiting is handled by Claude Code subscription naturally
- [ ] `checkQuota()` in spawner only checks rate-limit status, removes USD budget enforcement
- [ ] OpenAPI spec regenerated with updated schema
- [ ] SSE `quota_update` payload reflects the reframed fields
- [ ] All existing tests updated to match new field names
- [ ] Backward-compatible: old `MaxSessionCostUSD` config field is silently ignored (not a hard error)

## Dependencies

None.

## Blockers

- **quota-reframe-fe_20260309103001Z** — depends on this track for updated API schema.

## Conflict Risk

- **sse-event-bus_20260309091500Z** — low risk. That track refactors the SSE hub infrastructure, this track changes quota event payloads. Different concerns but both touch `watcher.go`.
- **sse-entity-subscriptions_20260309091501Z** — low risk. Same reasoning.

## Out of Scope

- Adding subscription plan details or tier information
- Fetching actual subscription limits from Anthropic API
- Token budgeting per-track or per-project
- Frontend changes (separate track)

## Technical Notes

### API Schema Changes

**Before:**
```yaml
QuotaInfo:
  total_cost_usd: number (required)
  input_tokens: integer (optional)
  output_tokens: integer (optional)
  rate_limited: boolean
```

**After:**
```yaml
QuotaInfo:
  input_tokens: integer (required)
  output_tokens: integer (required)
  cache_read_tokens: integer (required)
  cache_creation_tokens: integer (required)
  rate_limited: boolean (required)
  retry_after_seconds: integer (optional)
  estimated_cost_usd: number (required)  # informational — API-equivalent cost
  agent_count: integer (required)
  agents: QuotaAgentUsage[] (optional)
```

### Spawner Budget Check

Remove the `MaxSessionCostUSD` budget enforcement. The Claude Code subscription has its own rate limiting which is already detected via the `error_max_budget_usd` stream event. The spawner should only check `IsRateLimited()`.

---

_Generated by conductor-track-generator from prompt: "reframe quota from cost to tokens and rate limits"_
