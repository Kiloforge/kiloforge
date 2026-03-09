# Specification: Project-Scoped Dashboard Tracks

**Track ID:** project-scoped-dashboard_20260308220001Z
**Type:** Bug
**Created:** 2026-03-08T22:00:01Z
**Status:** Draft

## Summary

Fix the dashboard to read tracks from registered projects (via `projects.json`) instead of from `os.Getwd()`. After `kf destroy`, the dashboard should show no tracks. Tracks should be grouped by project.

## Context

The dashboard currently gets its `projectDir` from `os.Getwd()` in `serve.go:85`. This means it reads `.agent/conductor/tracks.md` from whatever directory `kf up` was run in — typically the kiloforge repo itself. This shows kiloforge's own conductor development tracks, not the tracks of projects registered via `kf add`. After a fresh `kf destroy && kf init`, the dashboard still shows tracks because it reads from CWD.

## Codebase Analysis

- **`cli/serve.go:85`** — `projectDir, _ := os.Getwd()` — root cause, passes CWD to dashboard
- **`cli/serve.go:86`** — `rest.WithDashboard(store, tracker, "/", projectDir)` — single projectDir
- **`dashboard/server.go:34`** — `projectDir string` field on Server struct — single project only
- **`dashboard/server.go:40`** — `New(port, agents, quota, giteaURL, projectDir)` — takes single projectDir
- **`dashboard/handlers.go:222-226`** — `handleTracks()` calls `service.DiscoverTracks(s.projectDir)` — single project
- **`core/service/track_service.go:54`** — `DiscoverTracks(projectDir)` reads `.agent/conductor/tracks.md`
- **`persistence/jsonfile/project_store.go`** — `ProjectStore` has `List()` returning all registered projects with `ProjectDir` paths
- **`rest/api_handler.go`** — `APIHandler` has separate `ListTracks` implementation via OpenAPI gen
- **Frontend `useTracks.ts`** — fetches `/-/api/tracks`, expects array of track objects

## Acceptance Criteria

- [ ] Dashboard reads tracks from all registered projects in `projects.json`
- [ ] Tracks are returned with a `project` field identifying which project they belong to
- [ ] After `kf destroy && kf init` (no projects), dashboard shows no tracks
- [ ] After `kf add <remote>`, dashboard shows that project's tracks (if any)
- [ ] Multiple registered projects each show their own tracks
- [ ] The `/-/api/tracks` endpoint response includes project slug per track
- [ ] The `/-/api/projects` endpoint returns list of registered projects
- [ ] Frontend uses React Router with URL-based state for navigation
- [ ] `/-/` shows overview (agents, stats) with projects list
- [ ] `/-/projects/:slug` shows a project's tracks with status
- [ ] Browser back/forward navigation works correctly
- [ ] URLs are shareable — navigating directly to `/-/projects/myapp` works (SPA fallback already handled by backend)
- [ ] Frontend displays a Projects section; each project shows its tracks with status
- [ ] `os.Getwd()` is no longer used for track discovery

## Dependencies

None (but benefits from add-local-ssh-identity_20260308220000Z being done first so projects have proper repo dirs)

## Out of Scope

- Real-time track change detection (polling on 30s interval is fine)
- Track editing from the dashboard
- Project management from the dashboard (add/remove projects)

## Technical Notes

**Backend changes:**

1. **Schema-first: update OpenAPI spec** — add `GET /-/api/projects` endpoint, add `project` field to `Track` schema, add `project` query param to `listTracks`. All new endpoints MUST be defined in `openapi.yaml` first and implemented via the generated strict handler (`api_handler.go`), per schema-first guidelines.
2. **Remove duplicate hand-rolled API handlers** — `dashboard/handlers.go` has hand-written `handleAgents`, `handleTracks`, `handleQuota`, `handleStatus` that duplicate the OpenAPI-generated handlers. The `RegisterRoutes()` method mounts all of these. When the unified server uses `RegisterNonAPIRoutes()`, the generated handlers take precedence, but the duplicates should be removed to avoid confusion. Keep only `RegisterNonAPIRoutes()` (SSE, HTML pages, SPA static) — all JSON API routes go through OpenAPI codegen.
3. Replace `projectDir string` on dashboard Server with `projects ProjectLister` interface
4. Update `ListTracks` in `api_handler.go` to iterate registered projects via `ProjectLister`, annotate tracks with project slug
5. Add `ListProjects` to `api_handler.go` via generated interface
6. Update `serve.go` to pass the already-loaded `reg` (project store) instead of `os.Getwd()`

**Frontend changes:**

1. Add `react-router-dom` dependency
2. Set up `BrowserRouter` with `basename="/-/"` in `main.tsx`
3. Routes:
   - `/` — Overview page: agents, stats, projects list with track summary counts
   - `/projects/:slug` — Project detail: full track list with status for that project
4. Add `project` field to track type
5. Add `/-/api/projects` fetch hook for the projects list
6. Dashboard layout: overview shows a "Projects" section listing all registered projects with track counts. Clicking a project navigates to `/projects/:slug` showing full track details.
7. Handle empty state (no projects → "No projects registered — run `kf add <remote>`" message)
8. SPA fallback already handled by backend's `spaFileServer` — direct URL navigation works out of the box

**API response change:**
```json
[
  {
    "id": "track-001",
    "title": "Implement auth",
    "status": "in-progress",
    "project": "my-app"
  }
]
```

---

_Generated by conductor-track-generator from prompt: "Project-scoped dashboard tracks"_
