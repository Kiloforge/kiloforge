# Implementation Plan: OpenAPI Code Generation for Relay API

**Track ID:** openapi-codegen_20260308200001Z

## Phase 1: OpenAPI Spec Authoring (4 tasks)

### Task 1.1: Create OpenAPI 3.0 spec skeleton
- [x] Create `api/openapi.yaml` with info, servers, and path stubs for all current API endpoints

### Task 1.2: Define model schemas
- [x] Add `components/schemas` for all request/response types

### Task 1.3: Complete path operations
- [x] Fill in all path operations with request parameters, request bodies, and response schemas

### Task 1.4: Validate spec
- [x] Validated with Redocly CLI — valid with only expected warnings (no auth)

## Phase 2: Code Generation Setup (4 tasks)

### Task 2.1: Add oapi-codegen dependency
- [x] Added `oapi-codegen/v2` v2.6.0 and `runtime` v1.2.0

### Task 2.2: Create generation config
- [x] Created `api/cfg.yaml` (server+models) and `api/cfg-client.yaml` (client only)

### Task 2.3: Generate initial code
- [x] Generated `server.gen.go` (1322 lines) and `client.gen.go` — both compile

### Task 2.4: Add Makefile targets
- [x] Added `gen-api` and `verify-codegen` targets

## Phase 3: Implement Strict Server Interface (5 tasks)

### Task 3.1: Create handler struct
- [x] Created `api_handler.go` implementing `StrictServerInterface`

### Task 3.2: Migrate agent endpoints
- [x] Implemented `ListAgents`, `GetAgent`, `GetAgentLog`

### Task 3.3: Migrate status endpoints
- [x] Implemented `GetQuota`, `ListTracks`, `GetStatus`

### Task 3.4: Migrate lock endpoints
- [x] Implemented `ListLocks`, `AcquireLock`, `HeartbeatLock`, `ReleaseLock`

### Task 3.5: Write tests for API handler
- [x] Full test coverage for all interface methods with mocks

## Phase 4: Router Integration (3 tasks)

### Task 4.1: Wire generated router into server
- [x] Mounted via `gen.HandlerFromMux` alongside manual routes

### Task 4.2: Remove old hand-written handlers
- [x] Removed `handleHealth`, replaced lock handler and dashboard API registrations

### Task 4.3: Update all tests
- [x] Updated health test to use generated handler — full suite passes (17 packages)

## Phase 5: Client Generation & Verification (2 tasks)

### Task 5.1: Generate and verify client package
- [x] Client compiles and is importable

### Task 5.2: Final verification
- [x] `make verify-codegen` passes
- [x] `make test` passes with race detector (17 packages)

---

**Total: 5 phases, 18 tasks — all complete**
