# Implementation Plan: Schema-First API Development Guidance and Standards

**Track ID:** schema-first-guidance_20260308200002Z

## Phase 1: Update Project Guidance Documents (4 tasks)

### Task 1.1: Update product-guidelines.md
Add "Schema-First APIs" as design principle #5. Add brief explanation of the principle and its rationale.

### Task 1.2: Update tech-stack.md
Expand the API section with detailed entries for:
- `oapi-codegen` v2.5.1 — strict server interface, models, client generation
- `github.com/oapi-codegen/runtime` — runtime helpers
- AsyncAPI 3.0 specification format for event documentation
- Generation workflow (`make gen-api`, `make verify-codegen`)

### Task 1.3: Update Go style guide
Add "API Design" section to `code_styleguides/go.md` covering:
- Schema-first workflow (modify schema → generate → implement interface)
- When to use OpenAPI (REST endpoints) vs AsyncAPI (events/streams)
- How to handle non-standard responses (SVG, SSE) alongside generated code
- Code generation file conventions (`.gen.go` suffix, never edit generated files)

### Task 1.4: Verify Phase 1
Review all updated documents for consistency and completeness.

## Phase 2: API Documentation Artifacts (3 tasks)

### Task 2.1: Create api/README.md
Write contributor-facing documentation explaining:
- The schema-first workflow
- How to add a new endpoint (schema → generate → implement → test)
- How to add a new event type (asyncapi schema → document → implement)
- File layout (`api/openapi.yaml`, `api/asyncapi.yaml`, `api/cfg.yaml`)

### Task 2.2: Create AsyncAPI spec skeleton
Create `api/asyncapi.yaml` with:
- SSE channel (`/-/events`) with `agent_update` message schema
- Webhook channel (`/webhook`) with documented Gitea event payload types
- Proper AsyncAPI 3.0 structure

### Task 2.3: Verify Phase 2
Validate AsyncAPI spec with an online validator. Ensure README is accurate against actual project structure.

---

**Total: 2 phases, 7 tasks**
