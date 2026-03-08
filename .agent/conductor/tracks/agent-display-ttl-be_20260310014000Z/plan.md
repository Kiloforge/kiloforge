# Implementation Plan: Agent Display TTL and History API (Backend)

**Track ID:** agent-display-ttl-be_20260310014000Z

## Phase 1: Domain and Storage

- [x] Task 1.1: Add `FinishedAt *time.Time` to `domain.AgentInfo`
- [x] Task 1.2: Add `IsActive() bool` helper method to `domain.AgentInfo`
- [x] Task 1.3: SQLite migration — add `finished_at` column to agents table
- [x] Task 1.4: Update SQLite agent store to read/write `finished_at`
- [x] Task 1.5: Set `FinishedAt` in spawner when agent reaches terminal status

## Phase 2: API

- [x] Task 2.1: Update `openapi.yaml` — add `finished_at` to Agent schema, add `active` query param to `GET /api/agents`
- [x] Task 2.2: Run `make generate` to regenerate stubs
- [x] Task 2.3: Implement `?active=true` filter in `ListAgents` handler — filter to active + recently finished (30 min)
- [x] Task 2.4: Update `GET /api/status` — `agent_counts` only includes active statuses, add `active_agents` and `total_agents` fields
- [x] Task 2.5: Include `finished_at` in `domainAgentToGen` response conversion

## Phase 3: Verification

- [x] Task 3.1: `make test` passes
- [x] Task 3.2: `make generate` produces no diff
