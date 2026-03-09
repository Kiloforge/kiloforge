# Specification: Research: Architectural Review and Alignment Audit

**Track ID:** arch-review_20260310040000Z
**Type:** Research
**Created:** 2026-03-10T04:00:00Z
**Status:** Draft

## Summary

Perform a thorough architectural review of the entire Kiloforge codebase (backend Go + frontend React) to identify pattern inconsistencies, clean architecture violations, test coverage gaps, naming drift, technical debt, and areas that have diverged from the intended design. Produce prioritized findings that inform follow-up improvement tracks.

## Context

Kiloforge has grown rapidly — from a simple CLI relay to a full-stack platform with 19 adapter packages, 7 service modules, 35 REST API endpoints, WebSocket/SSE real-time communication, SQLite persistence, and a React dashboard. This velocity introduces risk of architectural drift: patterns that worked at small scale may not hold, conventions established early may have been forgotten or violated, and cross-cutting concerns (error handling, logging, context propagation) may be inconsistent across packages added at different times.

The project has strong documented conventions (Go style guide, build conventions, product guidelines, schema-first API design) but no systematic verification that the codebase adheres to them. This review bridges that gap.

## Codebase Analysis

Key areas identified for review based on codebase research:

### Backend (Go)
- **19 adapter packages** — high surface area for inconsistency in error handling, logging patterns, and dependency injection
- **7 service modules** — need to verify they follow the "services depend only on ports" rule consistently
- **34 CLI command files** — verify they are truly "thin adapters" per product guidelines (no business logic in CLI layer)
- **SQLite persistence** (19 files) — verify parameterized queries, migration patterns, error wrapping
- **OpenAPI codegen** — verify all 35 endpoints have schema-first implementations, no hand-written routes bypassing the spec
- **Agent package** (14 files) — complex lifecycle management, concurrency patterns need scrutiny
- **Dashboard adapter** — SSE, file watching, embedded assets — verify clean separation
- **WebSocket package** — session management, ring buffer, message handling patterns

### Frontend (React/TypeScript)
- **21+ components** — verify consistent patterns for state management, error handling, loading states
- **16 hooks** — verify consistent patterns for data fetching, SSE integration, cleanup
- **CSS Modules** (30 files) — verify consistent naming, no style leaks, responsive patterns
- **API types** — single `api.ts` file — verify types match OpenAPI spec, no drift
- **Test coverage** — only 5 test files on frontend vs 73 on backend — significant gap
- **TanStack Query usage** — verify consistent cache invalidation, error handling, loading patterns

### Cross-Cutting Concerns
- **Error handling** — domain sentinel errors, error wrapping consistency, HTTP error responses
- **Logging** — port.Logger usage vs direct fmt/log calls
- **Context propagation** — context.Context as first param, cancellation handling
- **Naming conventions** — PascalCase exports, camelCase unexported, package naming
- **Test patterns** — table-driven tests, t.Parallel(), t.Helper(), interface compliance checks

## Acceptance Criteria

- [ ] Backend clean architecture audit complete — dependency violations documented
- [ ] Backend adapter consistency audit complete — error handling, logging, DI patterns
- [ ] Backend test coverage gap analysis complete — packages/functions lacking tests identified
- [ ] Frontend architecture audit complete — component patterns, hook patterns, type safety
- [ ] Frontend test coverage gap analysis complete — untested components/hooks identified
- [ ] Cross-cutting concern audit complete — error handling, logging, context, naming
- [ ] Schema-first compliance audit complete — endpoints vs OpenAPI spec alignment
- [ ] CLI thin-adapter audit complete — business logic in CLI layer identified
- [ ] Prioritized findings document produced with severity levels
- [ ] Follow-up track recommendations generated (titles + brief scope)

## Dependencies

None — this is a standalone research track.

## Out of Scope

- **Making code changes** — this track only produces findings documents
- **Writing new tests** — test gaps are identified, not filled
- **Refactoring** — recommendations are documented for follow-up tracks
- **Performance profiling** — this is a structural/design review, not a performance audit
- **Security audit** — security concerns are noted if found, but this is not a dedicated security review

## Technical Notes

### Review Methodology

The review should systematically examine each layer of the architecture:

1. **Top-down**: Start from the style guide and product guidelines, check each rule against the actual codebase
2. **Bottom-up**: Walk through each package looking for local inconsistencies
3. **Cross-cutting**: Check horizontal concerns (errors, logging, context) across all packages

### Output Format

Findings should be structured as:
- **Category** (e.g., "Clean Architecture Violation", "Test Gap", "Naming Drift")
- **Severity** (Critical / High / Medium / Low)
- **Location** (file:line or package)
- **Description** (what's wrong)
- **Recommendation** (how to fix)
- **Track Suggestion** (title for a follow-up improvement track, if warranted)

### Review Checklist (derived from project conventions)

**Clean Architecture:**
- [ ] domain/ imports nothing from internal/
- [ ] port/ imports only domain/
- [ ] service/ imports only domain/ and port/
- [ ] Adapters never call other adapters directly
- [ ] No business logic in CLI or REST handlers

**Schema-First API:**
- [ ] Every REST endpoint exists in openapi.yaml
- [ ] No hand-written routes bypass generated server interface
- [ ] Generated .gen.go files are not manually edited
- [ ] AsyncAPI schema covers SSE and webhook channels

**Error Handling:**
- [ ] Sentinel errors defined in domain/errors.go
- [ ] Errors wrapped with context (fmt.Errorf + %w)
- [ ] HTTP handlers don't leak internal error details
- [ ] Consistent error response format across endpoints

**Testing:**
- [ ] Every service has tests
- [ ] Error paths tested, not just happy paths
- [ ] Table-driven test pattern used consistently
- [ ] Interface compliance checks (var _ port.X = (*Y)(nil))
- [ ] t.Parallel() used where safe

**Naming:**
- [ ] Package names: short, lowercase, single-word
- [ ] Exported: PascalCase; Unexported: camelCase
- [ ] Interfaces named by behavior (-er suffix)
- [ ] Acronyms: all caps exported, all lowercase unexported

---

_Generated by kf-architect from prompt: "Thorough architectural review to gather improvement tracks and ensure broad alignment across the project"_
