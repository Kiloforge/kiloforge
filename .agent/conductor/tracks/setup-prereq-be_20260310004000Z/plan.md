# Implementation Plan: Setup Prerequisite Check (Backend)

**Track ID:** setup-prereq-be_20260310004000Z

## Phase 1: API Contract

- [x] Task 1.1: Add `setup_required` boolean to `PreflightResponse` in `openapi.yaml`
- [x] Task 1.2: Add `GET /api/projects/{slug}/setup-status` endpoint with `SetupStatusResponse` schema
- [x] Task 1.3: Add `POST /api/projects/{slug}/setup` endpoint (request: empty, response: agent info)
- [x] Task 1.4: Add 428 response schema (`SetupRequiredResponse`) to `tracks/generate` and `agents/interactive`
- [x] Task 1.5: Run `make generate` to regenerate server stubs

## Phase 2: Implementation

- [x] Task 2.1: Add `checkSetup(projectSlug string)` helper to `api_handler.go`
- [x] Task 2.2: Add `setup_required` to `GetPreflight` — check active project's conductor artifacts
- [x] Task 2.3: Implement `GetProjectSetupStatus` handler
- [x] Task 2.4: Implement `StartProjectSetup` handler — spawn interactive agent with `/kf-setup` prompt
- [x] Task 2.5: Add 428 check to `GenerateTracks` — return `SetupRequiredResponse` when setup incomplete
- [x] Task 2.6: Add 428 check to `SpawnInteractiveAgent` — same pattern
- [x] Task 2.7: Remove the auto-chain `/kf-setup` hack from `GenerateTracks`

## Phase 3: Verification

- [x] Task 3.1: `make test` passes
- [x] Task 3.2: `make generate` produces no diff (API contract matches implementation)
