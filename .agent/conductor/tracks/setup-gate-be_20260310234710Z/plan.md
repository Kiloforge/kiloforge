# Implementation Plan: Gate All Agent Endpoints Behind Setup Check (Backend)

**Track ID:** setup-gate-be_20260310234710Z

## Phase 1: API Schema and Handler

- [ ] Task 1.1: Update `openapi.yaml` — add 428 response to `POST /api/admin/run` using existing `SetupRequiredResponse` schema
- [ ] Task 1.2: Run `make generate` to regenerate server stubs
- [ ] Task 1.3: Add 428 setup check to `RunAdminOperation` handler — after skills check, before concurrency guard

## Phase 2: Verification

- [ ] Task 2.1: `make generate` produces no diff
- [ ] Task 2.2: `make test` passes
