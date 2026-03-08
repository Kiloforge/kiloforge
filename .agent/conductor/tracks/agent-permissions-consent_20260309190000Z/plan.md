# Implementation Plan: Agent Permissions Flag and User Consent

**Track ID:** agent-permissions-consent_20260309190000Z

## Phase 1: Add Permissions Flag to Spawner

- [ ] Task 1.1: Append `--dangerously-skip-permissions` to args in `SpawnReviewer()`, `SpawnDeveloper()`, and `SpawnInteractive()`
- [ ] Task 1.2: Update spawner tests if they assert on args

## Phase 2: Consent Storage

- [ ] Task 2.1: Add consent check/store functions using the SQLite config table — `HasAgentPermissionsConsent(db)` and `RecordAgentPermissionsConsent(db)`
- [ ] Task 2.2: Unit test for consent read/write round-trip

## Phase 3: CLI Consent Flow

- [ ] Task 3.1: Add consent check to `kf implement` — before spawning, check DB; if not consented, display warning and prompt; on "y", store consent and proceed; on "n", abort
- [ ] Task 3.2: Verify consent is checked only once — second `kf implement` skips prompt

## Phase 4: REST API Consent

- [ ] Task 4.1: Add `GET /api/consent/agent-permissions` endpoint — returns `{"consented": bool, "consented_at": string}`
- [ ] Task 4.2: Add `POST /api/consent/agent-permissions` endpoint — records consent, returns 200
- [ ] Task 4.3: Add consent guard to all agent spawn REST handlers — return 403 if not consented
- [ ] Task 4.4: Update OpenAPI spec with consent endpoints and 403 responses

## Phase 5: Dashboard Consent Dialog

- [ ] Task 5.1: Add consent check before agent spawn in dashboard — on 403, show confirmation dialog with warning text
- [ ] Task 5.2: On user confirm, call `POST /api/consent/agent-permissions`, then retry the spawn

## Phase 6: Verification

- [ ] Task 6.1: Verify `go test ./...` passes
- [ ] Task 6.2: Verify frontend builds without errors
- [ ] Task 6.3: Verify `kf implement` prompts on first run, skips on second
- [ ] Task 6.4: Verify dashboard consent dialog flow
