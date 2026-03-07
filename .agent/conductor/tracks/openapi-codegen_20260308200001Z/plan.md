# Implementation Plan: OpenAPI Code Generation for Relay API

**Track ID:** openapi-codegen_20260308200001Z

## Phase 1: OpenAPI Spec Authoring (4 tasks)

### Task 1.1: Create OpenAPI 3.0 spec skeleton
Create `api/openapi.yaml` with info, servers, and path stubs for all current API endpoints: agents (list, get, log), quota, tracks, status, locks (list, acquire, release, delete).

### Task 1.2: Define model schemas
Add `components/schemas` for all request/response types: `AgentInfo`, `AgentLog`, `QuotaInfo`, `TrackInfo`, `StatusInfo`, `LockInfo`, `LockAcquireRequest`, `ErrorResponse`.

### Task 1.3: Complete path operations
Fill in all path operations with request parameters, request bodies, and response schemas. Include proper HTTP status codes and error responses.

### Task 1.4: Validate spec
Use `oapi-codegen` dry run or an OpenAPI linter to validate the spec is well-formed.

## Phase 2: Code Generation Setup (4 tasks)

### Task 2.1: Add oapi-codegen dependency
Add `github.com/oapi-codegen/oapi-codegen/v2` and `github.com/oapi-codegen/runtime` to `go.mod`.

### Task 2.2: Create generation config
Create `api/cfg.yaml` with strict server, models, and client generation targeting `internal/adapter/rest/gen/`.

### Task 2.3: Generate initial code
Run `oapi-codegen` to produce `server.gen.go` in `internal/adapter/rest/gen/`. Verify it compiles.

### Task 2.4: Add Makefile targets
Add `gen-api` and `verify-codegen` targets to `Makefile`.

## Phase 3: Implement Strict Server Interface (5 tasks)

### Task 3.1: Create handler struct
Create `internal/adapter/rest/api_handler.go` implementing the generated `StrictServerInterface`. Start with stub methods returning not-implemented errors.

### Task 3.2: Migrate agent endpoints
Implement `ListAgents`, `GetAgent`, `GetAgentLog` methods using existing handler logic from dashboard and rest packages.

### Task 3.3: Migrate status endpoints
Implement `GetQuota`, `ListTracks`, `GetStatus` methods.

### Task 3.4: Migrate lock endpoints
Implement `ListLocks`, `AcquireLock`, `ReleaseLock`, `DeleteLock` methods using existing lock handler logic.

### Task 3.5: Write tests for API handler
Test all interface methods with mock dependencies. Verify request/response types match OpenAPI spec.

## Phase 4: Router Integration (3 tasks)

### Task 4.1: Wire generated router into server
In `rest/server.go`, mount the generated strict handler router alongside manual routes (webhook, SSE, badges).

### Task 4.2: Remove old hand-written handlers
Delete old `HandleFunc` registrations and handler functions that are now covered by the generated interface.

### Task 4.3: Update all tests
Update integration tests in `rest/`, `dashboard/`, and `lock/` packages to use new API handler. Run full test suite.

## Phase 5: Client Generation & Verification (2 tasks)

### Task 5.1: Generate and verify client package
Ensure the generated client code compiles and can be used by internal consumers (e.g., for inter-service communication or CLI).

### Task 5.2: Final verification
Run `make verify-codegen` to ensure generated code matches spec. Run full test suite. Verify all API endpoints respond correctly with `curl` tests.

---

**Total: 5 phases, 18 tasks**
