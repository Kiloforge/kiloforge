# Implementation Plan: Research: Architectural Review and Alignment Audit

**Track ID:** arch-review_20260310040000Z

## Phase 1: Backend Core Layer Review

### Task 1.1: Audit dependency direction in core/
- Verify `domain/` has zero imports from `internal/`
- Verify `port/` imports only `domain/`
- Verify `service/` imports only `domain/` and `port/`, never adapters
- Check for any indirect violations (e.g., domain types referencing adapter-specific libraries)
- Document all violations with file:line references

### Task 1.2: Audit service layer patterns
- Verify all 7 services use constructor injection (`NewXxx(...)`)
- Check that services receive port interfaces, not concrete adapter types
- Verify optional dependencies use setter methods (not constructor bloat)
- Check for business logic consistency — no duplicate logic across services
- Verify error wrapping patterns (fmt.Errorf + %w with context)

### Task 1.3: Audit domain layer completeness
- Check sentinel errors in `domain/errors.go` — are all domain error cases covered?
- Verify entity types are pure Go (no I/O, no external deps)
- Check if RBAC activity model is actually used or dead code
- Assess whether domain types cover all concepts in the system or if some live in adapters

### Task 1.4: Verify Phase 1
- Phase 1 findings documented in `findings-backend-core.md`

## Phase 2: Backend Adapter Layer Review

### Task 2.1: Audit adapter consistency patterns
- Check all 19 adapter packages for consistent error handling
- Verify port.Logger usage vs direct fmt.Println/log calls
- Check context.Context as first parameter convention
- Verify adapters don't call other adapters directly (should go through service layer)
- Check for adapter-level business logic that should be in services

### Task 2.2: Audit CLI thin-adapter compliance
- Review all 34 CLI command files for business logic violations
- Verify CLI commands construct services and delegate (not implement logic directly)
- Check flag/arg parsing consistency
- Identify any CLI commands that bypass the service layer

### Task 2.3: Audit schema-first API compliance
- Compare all routes in `server.go` / `api_handler.go` against `openapi.yaml`
- Check for endpoints not defined in the OpenAPI spec
- Verify generated `.gen.go` files haven't been manually modified
- Check AsyncAPI spec covers SSE channels and webhook payloads
- Verify HTTP error responses don't leak internal details

### Task 2.4: Audit persistence and data layer
- Review SQLite adapter for parameterized queries (no string interpolation)
- Check migration patterns and ordering
- Verify store implementations return domain sentinel errors correctly
- Check for any remaining JSON file persistence (should be migrated to SQLite)
- Assess quota store / quota tracker alignment

### Task 2.5: Audit concurrency and lifecycle patterns
- Review agent package for goroutine ownership documentation
- Check context cancellation handling in long-lived operations
- Verify WebSocket session cleanup on disconnect
- Check SSE connection lifecycle management
- Review merge lock implementation for edge cases

### Task 2.6: Verify Phase 2
- Phase 2 findings documented in `findings-backend-adapters.md`

## Phase 3: Frontend Review

### Task 3.1: Audit frontend architecture patterns
- Check component organization — presentational vs container patterns
- Verify hook patterns — consistent data fetching, cleanup, error handling
- Check TanStack Query usage — cache keys, invalidation, error/loading states
- Verify SSE integration pattern consistency across hooks
- Check for prop drilling vs appropriate state management

### Task 3.2: Audit frontend type safety and API alignment
- Compare `types/api.ts` against OpenAPI spec for drift
- Check for `any` type usage or type assertions that bypass safety
- Verify consistent error handling in API calls (FetchError pattern)
- Check for missing null/undefined guards in component rendering

### Task 3.3: Audit frontend test coverage
- Identify untested components (only 5 test files currently)
- Prioritize which components/hooks most need tests
- Check existing tests for quality — do they test behavior or implementation?
- Note any components with complex logic that are test-critical

### Task 3.4: Verify Phase 3
- Phase 3 findings documented in `findings-frontend.md`

## Phase 4: Cross-Cutting Review and Synthesis

### Task 4.1: Audit naming conventions project-wide
- Check Go package names against style guide (short, lowercase, single-word)
- Check exported/unexported naming conventions
- Check interface naming (-er suffix convention)
- Check frontend file/component naming consistency
- Check CSS module class naming patterns

### Task 4.2: Produce prioritized findings summary
- Aggregate all findings from Phases 1-3
- Assign severity levels (Critical / High / Medium / Low)
- Group by category (Architecture, Testing, Consistency, Naming, Technical Debt)
- Rank by impact and effort
- Generate follow-up track recommendations (title + 1-line scope for each)

### Task 4.3: Verify Phase 4
- Final `findings-summary.md` produced with all findings, severities, and track recommendations
- All findings files cross-referenced and consistent
