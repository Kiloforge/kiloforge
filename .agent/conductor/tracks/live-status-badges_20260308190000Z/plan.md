# Implementation Plan: Universal Live Agent Status Badges

**Track ID:** live-status-badges_20260308190000Z

## Phase 1: Backend Badge Renderer and Endpoints (5 tasks)

### Task 1.1: SVG badge renderer
- [ ] Create `internal/adapter/badge/badge.go`
- [ ] `RenderBadge(label, status, color string) []byte` — generates shields.io-style flat SVG
- [ ] `RenderDualBadge(label, status1, color1, status2, color2 string) []byte` — for PR badges showing two agent statuses
- [ ] Calculate text widths (approximate: 7px per character for Verdana 11px)
- [ ] Define status-to-color map: running=#4c1, waiting=#dfb317, completed=#007ec6, failed=#e05d44, halted=#fe7d37, stopped=#9f9f9f, pending=#lightgrey
- [ ] Unit test: verify SVG output is valid XML, correct colors per status

### Task 1.2: Track badge endpoint
- [ ] Add `handleBadgeTrack(w, r)` to relay server
- [ ] Route: `GET /api/badges/track/{trackId}`
- [ ] Look up agent by iterating `store.Agents()` matching `Ref == trackId` (most recent by StartedAt)
- [ ] If agent found: render badge with `{shortTrackId} | {status}`
- [ ] If no agent: render badge with `{shortTrackId} | pending`
- [ ] Set headers: `Content-Type: image/svg+xml`, `Cache-Control: no-cache, no-store, must-revalidate`

### Task 1.3: PR badge endpoint
- [ ] Add `handleBadgePR(w, r)` to relay server
- [ ] Route: `GET /api/badges/pr/{slug}/{prNumber}`
- [ ] Load PRTracking for the project slug
- [ ] Look up developer and reviewer agents from tracking
- [ ] Render dual badge: `PR #{num} | dev: {status} · rev: {status}`
- [ ] If no tracking found: render `PR #{num} | unknown`

### Task 1.4: Agent badge endpoint
- [ ] Add `handleBadgeAgent(w, r)` to relay server
- [ ] Route: `GET /api/badges/agent/{agentId}`
- [ ] Look up agent via `store.FindAgent(agentId)`
- [ ] Render badge with `{role} | {status}`
- [ ] If not found: render `agent | unknown`

### Task 1.5: Register badge routes and unit tests
- [ ] Register all 3 badge routes in `Server.Run()`
- [ ] Test track badge: running agent, pending (no agent), unknown track
- [ ] Test PR badge: both agents running, one halted, no tracking
- [ ] Test agent badge: found, not found
- [ ] Test response headers (Content-Type, Cache-Control)

## Phase 2: Frontend Detail Pages (4 tasks)

### Task 2.1: Track detail page
- [ ] Add route `/tracks/:trackId` in React Router
- [ ] Create `src/pages/TrackDetail.tsx`
- [ ] Fetch agent data from `/api/agents` filtered by `ref == trackId`
- [ ] Subscribe to SSE for real-time updates
- [ ] Display: track ID, large status badge, agent card, inline log viewer
- [ ] Handle: loading, no agent ("awaiting implementation"), error states
- [ ] Back link to dashboard

### Task 2.2: PR detail page
- [ ] Add route `/pr/:slug/:prNumber` in React Router
- [ ] Create `src/pages/PRDetail.tsx`
- [ ] Fetch PR tracking data (new endpoint or derive from agents)
- [ ] Display: PR number (linked to Gitea), developer agent card, reviewer agent card
- [ ] Show review cycle count and status
- [ ] Log viewer toggle between developer and reviewer logs
- [ ] Subscribe to SSE for live updates

### Task 2.3: Badge markdown helper utility
- [ ] Create `internal/adapter/badge/markdown.go`
- [ ] `TrackBadgeMarkdown(trackID, relayURL string) string` — returns `[![status](...)](...)`
- [ ] `PRBadgeMarkdown(slug string, prNum int, relayURL string) string`
- [ ] `AgentBadgeMarkdown(agentID, relayURL, linkURL string) string`
- [ ] Unit tests for all helpers

### Task 2.4: Navigation between dashboard and detail pages
- [ ] Agent cards on main dashboard link to appropriate detail page
- [ ] If agent role is "developer" and Ref is a track ID → link to `/tracks/{ref}`
- [ ] If agent role is "reviewer" and Ref is a PR → link to `/pr/{slug}/{prNum}`
- [ ] Breadcrumb navigation on detail pages

## Phase 3: Gitea Integration (3 tasks)

### Task 3.1: Embed track badges in Gitea issues
- [ ] In BoardService issue creation (from impl-track-board-sync), append track badge markdown to issue body
- [ ] Use configured relay URL from config
- [ ] Verify badge renders when viewing issue in Gitea

### Task 3.2: Embed PR badges in PR descriptions and comments
- [ ] In `createPRTracking()`, if PR body can be updated, add PR badge markdown
- [ ] In `spawnReviewerForPR()`, post a comment with reviewer agent badge
- [ ] Use `CommentOnPR()` (existing) to add badge comment

### Task 3.3: Badge in review cycle comments
- [ ] When changes are requested and developer is resumed, post comment with developer badge
- [ ] When review cycle escalates, post comment with both agent badges showing stopped status
- [ ] Consistent badge placement across all automated comments

## Phase 4: Verification (2 tasks)

### Task 4.1: Integration verification
- [ ] Badge endpoint → agent state change → badge reflects new state
- [ ] Verify track badge, PR badge, and agent badge all update correctly
- [ ] Verify frontend detail pages update via SSE
- [ ] Verify badges render correctly in Gitea markdown preview

### Task 4.2: Full build and test
- [ ] `go build -buildvcs=false ./...`
- [ ] `go test -buildvcs=false -race ./...`
- [ ] `npm run build` (frontend)
- [ ] No regressions

---

**Total: 14 tasks across 4 phases**
