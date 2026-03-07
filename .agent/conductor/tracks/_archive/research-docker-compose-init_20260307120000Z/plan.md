# Implementation Plan: Research: Docker Compose for Init with v1/v2 CLI Compatibility

**Track ID:** research-docker-compose-init_20260307120000Z

## Phase 1: CLI Variant Detection Research

### Task 1.1: Investigate docker compose v1 vs v2 CLI differences [x]
- Compare `docker-compose` (standalone Python/Go binary) vs `docker compose` (Docker CLI plugin)
- Document command syntax differences, if any
- Test on macOS with Docker Desktop vs Colima
- **Output:** Section in research doc on CLI variants

### Task 1.2: Design CLI detection strategy [x]
- Propose detection logic: try v2 first, fall back to v1
- Handle edge cases: neither installed, broken installs, version requirements
- Consider minimum compose file version needed
- **Output:** Proposed detection code/pseudocode

### Task 1.3: Research Colima-specific considerations [x]
- `host.docker.internal` availability and workarounds
- Volume mount behavior differences
- Network mode differences
- Known issues with compose on Colima
- **Output:** Colima compatibility notes

### Verification 1
- [x] CLI variant differences documented
- [x] Detection strategy proposed
- [x] Colima notes captured

## Phase 2: Compose File Design

### Task 2.1: Draft docker-compose.yml for Gitea [x]
- Translate current `docker run` flags to compose format
- Define service, ports, volumes, environment, restart policy
- Consider health check definition in compose
- **Output:** Proposed docker-compose.yml

### Task 2.2: Evaluate compose file management strategy [x]
- Option A: Ship a template, copy to data dir at init
- Option B: Generate programmatically with Go (template/text or struct)
- Option C: Embed in binary via `embed` package
- Evaluate tradeoffs: flexibility, maintainability, user visibility
- **Output:** Recommended approach with rationale

### Verification 2
- [x] Compose file drafted and validated
- [x] File management strategy chosen

## Phase 3: Impact Analysis & Recommendations

### Task 3.1: Map changes needed in existing code [x]
- `internal/gitea/manager.go` — Replace docker run/inspect/rm with compose up/down/ps
- `internal/cli/init.go` — Update orchestration flow
- `internal/cli/destroy.go` — Replace docker rm with compose down
- `internal/cli/status.go` — Replace docker inspect with compose ps
- **Output:** Change map with affected functions

### Task 3.2: Write final research document [x]
- Compile all findings into `docs/research-docker-compose.md`
- Include: recommendation, tradeoffs, proposed compose file, code change map, Colima notes
- **Output:** Complete research document

### Verification 3
- [x] Impact analysis complete
- [x] Research document written and committed
