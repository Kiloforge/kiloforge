# Specification: Universal Live Agent Status Badges

**Track ID:** live-status-badges_20260308190000Z
**Type:** Feature
**Created:** 2026-03-08T19:00:00Z
**Status:** Draft

## Summary

Serve dynamic SVG status badges from the crelay backend that reflect real-time agent state. Badges work universally — anywhere an agent is referenced (track issues, PRs, comments, dashboards). Each badge type resolves to the relevant agent(s) and links to a frontend detail page with live SSE updates.

## Context

When tracks are published to Gitea as issues (via `impl-track-board-sync`), the issue body can include a badge image URL like:

```markdown
[![agent status](http://localhost:3001/api/badges/track/{trackId})](http://localhost:3001/tracks/{trackId})
```

Every time someone views the issue, Gitea fetches the badge URL and renders the current agent state (e.g., "running", "halted", "completed"). Clicking the badge navigates to a dedicated frontend page with full agent details and live SSE updates.

## Codebase Analysis

- **Badge endpoint location:** New routes on the relay server mux (`/api/badges/track/`, `/api/badges/pr/`, `/api/badges/agent/`), alongside existing `/api/agents`, `/api/tracks`, etc.
- **Agent lookup by track:** `AgentInfo.Ref` stores track ID. Need `FindByRef()` (added by `board-agent-lifecycle` track) or iterate `Agents()` filtering on `Ref`.
- **Agent lookup by PR:** `PRTracking` links `DeveloperAgentID` and `ReviewerAgentID` to a PR number and slug. Loaded via `jsonfile.LoadPRTracking(projectDir)`.
- **Agent lookup by ID:** `AgentStore.FindAgent(idPrefix)` already exists.
- **SVG generation:** No external library needed — SVG badges are simple XML templates with text and colored rectangles (similar to shields.io format).
- **Frontend detail pages:** New routes in React app (`/tracks/:trackId`, `/pr/:slug/:prNumber`) showing agent cards, log viewer, and metadata. Uses existing hooks (`useAgents`, `useSSE`).
- **Integration points:** Track issues (body), PR descriptions (body), reviewer spawn comments, and any future agent-referencing context.
- **Cache control:** Badge responses must include `Cache-Control: no-cache` so Gitea (and browsers) always fetch fresh state.

## Acceptance Criteria

- [ ] `GET /api/badges/track/{trackId}` returns SVG badge showing developer agent status for a track
- [ ] `GET /api/badges/pr/{slug}/{prNumber}` returns SVG badge showing reviewer + developer status for a PR
- [ ] `GET /api/badges/agent/{agentId}` returns SVG badge for any specific agent
- [ ] Badge colors match status: green (running), yellow (waiting), blue (completed), red (failed), orange (halted), gray (pending/no agent)
- [ ] Badge includes label on left (track ID / PR# / agent role), status on right (shields.io style)
- [ ] Response includes `Cache-Control: no-cache, no-store` and `Content-Type: image/svg+xml`
- [ ] Frontend route `/tracks/{trackId}` shows agent details with live SSE updates
- [ ] Frontend route `/pr/{slug}/{prNumber}` shows PR agent pair (developer + reviewer) with live updates
- [ ] Badge links to the appropriate frontend detail page when clicked
- [ ] Track board sync embeds track badge markdown in created Gitea issues
- [ ] PR creation embeds PR badge markdown in PR description
- [ ] Reviewer spawn posts a comment with reviewer badge on the PR
- [ ] Badge gracefully handles: no agent yet (shows "pending"), unknown entity (shows "unknown")

## Dependencies

- **board-agent-lifecycle_20260308180000Z** — Provides `FindByRef()` on AgentStore (or can be implemented independently by iterating agents)
- **react-dashboard_20260308180002Z** — Provides the React app where the detail page lives
- **impl-track-board-sync_20260308170001Z** — Creates Gitea issues where badges will be embedded

## Out of Scope

- External badge service (shields.io) integration — we serve our own
- Badge caching/CDN layer
- Authentication on badge endpoint (must be publicly accessible within the network)
- Badges for non-agent entities (quota, system health, etc.)

## Technical Notes

### SVG Badge Template

Shields.io-style flat badge:

```svg
<svg xmlns="http://www.w3.org/2000/svg" width="{totalWidth}" height="20">
  <rect width="{labelWidth}" height="20" fill="#555"/>
  <rect x="{labelWidth}" width="{statusWidth}" height="20" fill="{statusColor}"/>
  <text x="{labelCenter}" y="14" fill="#fff" font-family="Verdana,sans-serif"
        font-size="11" text-anchor="middle">{trackId}</text>
  <text x="{statusCenter}" y="14" fill="#fff" font-family="Verdana,sans-serif"
        font-size="11" text-anchor="middle">{status}</text>
</svg>
```

### Status Color Map

| Status | Color | Hex |
|--------|-------|-----|
| running | green | #4c1 |
| waiting | yellow | #dfb317 |
| completed | blue | #007ec6 |
| failed | red | #e05d44 |
| halted | orange | #fe7d37 |
| stopped | gray | #9f9f9f |
| suspended | gray | #9f9f9f |
| pending (no agent) | light gray | #lightgrey |

### Badge Endpoints

| Endpoint | Label | Status Source | Links To |
|----------|-------|---------------|----------|
| `/api/badges/track/{trackId}` | Track ID (short) | Developer agent via `Ref` match | `/tracks/{trackId}` |
| `/api/badges/pr/{slug}/{prNum}` | `PR #{prNum}` | PRTracking → developer + reviewer | `/pr/{slug}/{prNum}` |
| `/api/badges/agent/{agentId}` | Agent role | Direct agent lookup | `/tracks/{trackId}` or `/pr/...` |

For PR badges, show a combined badge: `PR #42 | dev: running · rev: waiting`

### Frontend Detail Pages

**Route: `/tracks/:trackId`**
- Track title and ID
- Agent status badge (large)
- Agent card (reuse `AgentCard` component from dashboard)
- Log viewer (inline, not modal)
- Track spec summary (if available via API)
- Back link to main dashboard

**Route: `/pr/:slug/:prNumber`**
- PR title and number (linked to Gitea)
- Developer agent card + status
- Reviewer agent card + status
- Review cycle count and max
- Log viewer for selected agent
- Back link to main dashboard

### Gitea Integration Points

**Track issues** — when `impl-track-board-sync` creates an issue:
```markdown
[![status]({relayURL}/api/badges/track/{trackId})]({relayURL}/tracks/{trackId})
```

**PR descriptions** — when developer creates a PR (via relay tracking):
```markdown
[![agents]({relayURL}/api/badges/pr/{slug}/{prNum})]({relayURL}/pr/{slug}/{prNum})
```

**Reviewer comments** — when reviewer is spawned, post a comment:
```markdown
Reviewer agent spawned. [![reviewer]({relayURL}/api/badges/agent/{agentId})]({relayURL}/pr/{slug}/{prNum})
```

---

_Generated by conductor-track-generator from prompt: "live status badges in Gitea issues linking to frontend detail page"_
