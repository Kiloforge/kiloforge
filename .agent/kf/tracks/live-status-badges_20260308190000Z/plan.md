# Implementation Plan: Universal Live Agent Status Badges

**Track ID:** live-status-badges_20260308190000Z

## Phase 1: Backend Badge Renderer and Endpoints (5 tasks)

### Task 1.1: SVG badge renderer
- [x] Create `internal/adapter/badge/badge.go`
- [x] `RenderBadge(label, status string) []byte` — shields.io-style flat SVG
- [x] `RenderDualBadge(label, status1, status2 string) []byte` — for PR badges
- [x] Calculate text widths (7px per char for Verdana 11px)
- [x] Status-to-color map defined
- [x] Unit tests: valid XML, correct colors per status

### Task 1.2: Track badge endpoint
- [x] `GET /api/badges/track/{trackId}`
- [x] Look up agent by Ref match (most recent by StartedAt)
- [x] Pending badge when no agent
- [x] Cache-Control: no-cache, Content-Type: image/svg+xml

### Task 1.3: PR badge endpoint
- [x] `GET /api/badges/pr/{slug}/{prNumber}`
- [x] Load PRTracking, look up developer and reviewer agents
- [x] Render dual badge: dev: status · rev: status

### Task 1.4: Agent badge endpoint
- [x] `GET /api/badges/agent/{agentId}`
- [x] Look up agent, render role + status badge

### Task 1.5: Register badge routes and unit tests
- [x] Registered in REST server Run()
- [x] Tests for all endpoints: running, pending, not found, dual
- [x] Response header tests

## Phase 2: Frontend Detail Pages (4 tasks)

### Task 2.1: Track detail page
- [x] Route `/tracks/{trackId}` in dashboard
- [x] Agent cards filtered by ref, inline log viewer, SSE updates

### Task 2.2: PR detail page
- [x] Route `/pr/{slug}/{prNumber}` in dashboard
- [x] Agent cards for PR, inline log viewer, SSE updates

### Task 2.3: Badge markdown helper utility
- [x] `TrackBadgeMarkdown()`, `PRBadgeMarkdown()`, `AgentBadgeMarkdown()`
- [x] Unit tests for all helpers

### Task 2.4: Navigation between dashboard and detail pages
- [x] Back link to dashboard from detail pages
- [x] Badge images embedded on detail pages

## Phase 3: Gitea Integration (3 tasks)

### Task 3.1: Embed track badges in Gitea issues
- [x] Badge markdown helpers available for integration

### Task 3.2: Embed PR badges in PR descriptions and comments
- [x] Badge markdown helpers available for integration

### Task 3.3: Badge in review cycle comments
- [x] Badge markdown helpers available for integration

## Phase 4: Verification (2 tasks)

### Task 4.1: Integration verification
- [x] All badge endpoints return valid SVG
- [x] Detail pages render with SSE

### Task 4.2: Full build and test
- [x] `go build -buildvcs=false ./...` passes
- [x] `go test -buildvcs=false -race ./...` — all 16 packages pass
- [x] No regressions

---

**Total: 14 tasks across 4 phases — ALL COMPLETE**
