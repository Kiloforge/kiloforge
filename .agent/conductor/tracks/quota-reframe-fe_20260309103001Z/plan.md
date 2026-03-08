# Implementation Plan: Reframe Quota System — Frontend

**Track ID:** quota-reframe-fe_20260309103001Z

## Phase 1: Type Definitions & Utilities

- [x] Task 1.1: Update `frontend/src/types/api.ts` — rename `total_cost_usd` to `estimated_cost_usd` in `QuotaResponse`, add `cache_read_tokens` and `cache_creation_tokens` as required fields
- [x] Task 1.2: Update `QuotaAgent` type — rename `cost_usd` to `estimated_cost_usd`, add `cache_read_tokens` and `cache_creation_tokens`
- [x] Task 1.3: Update `Agent` type — rename `cost_usd` to `estimated_cost_usd`
- [x] Task 1.4: Update `StatusResponse` type — rename `total_cost_usd` to `estimated_cost_usd` if present

## Phase 2: StatCards — Token-Primary Display

- [x] Task 2.1: Reorder StatCards — tokens become primary metric card, cost moves to last position as "Est. API Cost"
- [x] Task 2.2: Enhance token card — show input/output tokens prominently, show cache tokens (read/creation) when non-zero
- [x] Task 2.3: Update cost card label from "Total Cost" to "Est. API Cost" with informational styling
- [x] Task 2.4: Update StatCards.module.css if needed for token-primary layout emphasis

## Phase 3: AgentCard — Token Breakdown

- [x] Task 3.1: Update AgentCard to display per-agent token breakdown (input/output) as primary info
- [x] Task 3.2: Show cache token counts per agent when non-zero
- [x] Task 3.3: Display estimated cost per agent as secondary/dimmed info (rename from `cost_usd` to `estimated_cost_usd`)

## Phase 4: Verification

- [x] Task 4.1: Verify `useQuota` hook works with new field names (SSE `quota_update` events)
- [x] Task 4.2: Verify all TypeScript compilation passes (`npm run build` or equivalent)
- [x] Task 4.3: Manual smoke test — dashboard loads, StatCards render, AgentCards show token data
