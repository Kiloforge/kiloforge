# Implementation Plan: Project-Scoped Dashboard Tracks

## Phase 1: OpenAPI Schema Updates (3 tasks)

### Task 1.1: Add Project schema and projects endpoint to OpenAPI spec
- **File:** `backend/api/openapi.yaml`
- Add `Project` schema: slug, repo_name, origin_remote, active, registered_at
- Add `GET /-/api/projects` endpoint returning `Project[]`
- Add `project` query param to `GET /-/api/tracks` for filtering by project slug
- Add `project` field to `Track` schema (required)

### Task 1.2: Regenerate server and client code
- Run `make gen-api`
- Verify new `ListProjects` and updated `ListTracks` appear in generated interface

### Task 1.3: Define ProjectLister interface and update dashboard Server
- **File:** `backend/internal/adapter/dashboard/server.go`
- Add `ProjectLister` interface with `List() []domain.Project`
- Replace `projectDir string` field with `projects ProjectLister`
- Update `New()` constructor signature

## Phase 2: Backend — Implement Generated Handlers (4 tasks)

### Task 2.1: Implement ListProjects in api_handler.go
- **File:** `backend/internal/adapter/rest/api_handler.go`
- Add `Projects ProjectLister` to `APIHandlerOpts`
- Implement `ListProjects()` — return projects from store

### Task 2.2: Update ListTracks in api_handler.go
- **File:** `backend/internal/adapter/rest/api_handler.go`
- Iterate `Projects.List()`, call `service.DiscoverTracks(p.ProjectDir)` for each
- Annotate each track with project slug
- Support `?project=<slug>` query param filter
- Return empty array when no projects registered

### Task 2.3: Remove duplicate hand-rolled API handlers from dashboard
- **File:** `backend/internal/adapter/dashboard/handlers.go`
- Remove `handleAgents`, `handleAgent`, `handleAgentLog`, `handleQuota`, `handleTracks`, `handleStatus` — these duplicate the OpenAPI-generated handlers
- Remove `RegisterRoutes()` method (only keep `RegisterNonAPIRoutes()`)
- Keep only: SSE handler, HTML template handlers, SPA static, and helper functions used by watchers

### Task 2.4: Update serve.go to pass project store
- **File:** `backend/internal/adapter/cli/serve.go`
- Pass `reg` (ProjectStore) to `WithDashboard()` and `APIHandlerOpts`
- Remove `projectDir, _ := os.Getwd()` line
- Update `WithDashboard()` in rest/server.go to accept `ProjectLister`

## Phase 2: Backend — Tests (3 tasks)

### Task 2.1: Update dashboard handler tests
- **File:** `backend/internal/adapter/dashboard/server_test.go`
- Create mock `ProjectLister` that returns test projects
- Test: no projects → empty tracks array
- Test: one project with tracks → tracks returned with project slug
- Test: multiple projects → tracks from all projects, each tagged
- Test: `/-/api/projects` returns project list
- Test: `?project=slug` filter works

### Task 2.2: Update route registration tests
- **File:** `backend/internal/adapter/rest/routes_test.go`
- Update `buildMux()` and test helpers to pass ProjectLister instead of projectDir
- Verify all route tests still pass

### Task 2.3: Run full test suite
- `make test` — all pass
- `make test-smoke` — smoke tests pass

## Phase 3: Frontend — React Router Setup (3 tasks)

### Task 3.1: Add react-router-dom dependency
- `cd frontend && npm install react-router-dom`
- Update `package.json` and `package-lock.json`

### Task 3.2: Set up BrowserRouter in main.tsx
- **File:** `frontend/src/main.tsx`
- Wrap `<App />` with `<BrowserRouter basename="/-/">`
- The `/-/` basename ensures all frontend routes are under the dashboard prefix

### Task 3.3: Define routes in App.tsx
- **File:** `frontend/src/App.tsx`
- Add `<Routes>`:
  - `/` → `<OverviewPage />` — agents, stats, projects list with track count summaries
  - `/projects/:slug` → `<ProjectPage />` — full track list for that project
- Move existing agent/stat content into `OverviewPage` component
- Shared layout: header + nav stays on all pages

## Phase 4: Frontend — Pages and Data Hooks (5 tasks)

### Task 4.1: Add useProjects hook
- **File:** `frontend/src/hooks/useProjects.ts`
- Fetch `/-/api/projects` on mount
- Return projects list with loading/error state

### Task 4.2: Update useTracks hook
- **File:** `frontend/src/hooks/useTracks.ts`
- Add `project` field to track type
- Accept optional `project` param to filter: `/-/api/tracks?project=<slug>`

### Task 4.3: Create OverviewPage component
- **File:** `frontend/src/pages/OverviewPage.tsx`
- Shows agents grid, stat cards (existing content)
- Shows "Projects" section: list of projects with track count summary (pending/in-progress/complete)
- Each project is a link to `/projects/:slug`
- Empty state: "No projects registered — run `crelay add <remote>`"

### Task 4.4: Create ProjectPage component
- **File:** `frontend/src/pages/ProjectPage.tsx`
- Read `:slug` from URL params
- Fetch tracks filtered by project
- Show full track list with status indicators
- Back link to overview
- Show project metadata (origin remote, etc.)

### Task 4.5: Verify frontend builds and lint
- `cd frontend && npm run build` — no errors
- `cd frontend && npm run lint` — no warnings
- Test direct URL navigation to `/-/projects/myapp` works (SPA fallback)

## Phase 5: Verification (2 tasks)

### Task 5.1: End-to-end verification
- Start `crelay up` with no projects → dashboard shows empty projects section
- `crelay add <remote>` → dashboard shows the project
- Navigate to `/-/projects/<slug>` → see that project's tracks
- Browser back → returns to overview
- Direct URL paste → loads correct page

### Task 5.2: Run full test suite
- `make test` — all pass
- `make build` — full build succeeds (frontend embedded)
