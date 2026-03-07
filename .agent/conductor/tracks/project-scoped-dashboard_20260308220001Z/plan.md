# Implementation Plan: Project-Scoped Dashboard Tracks

## Phase 1: Backend — Replace projectDir with ProjectStore (6 tasks)

### Task 1.1: Define ProjectLister interface
- **File:** `backend/internal/adapter/dashboard/server.go`
- Add interface: `ProjectLister` with `List() []domain.Project` method
- Replace `projectDir string` field with `projects ProjectLister`

### Task 1.2: Update dashboard constructor and WithDashboard option
- **File:** `backend/internal/adapter/dashboard/server.go`
- Change `New()` to accept `ProjectLister` instead of `projectDir string`
- **File:** `backend/internal/adapter/rest/server.go`
- Update `WithDashboard()` option to accept `ProjectLister`

### Task 1.3: Add projects API endpoint
- **File:** `backend/api/openapi.yaml`
- Add `GET /-/api/projects` endpoint returning list of registered projects (slug, repo name, origin remote, active status)
- **File:** `backend/internal/adapter/dashboard/handlers.go`
- Add `handleProjects()` that returns `s.projects.List()` as JSON
- Register route in `RegisterNonAPIRoutes()` or via OpenAPI gen

### Task 1.4: Update handleTracks to iterate projects
- **File:** `backend/internal/adapter/dashboard/handlers.go`
- `handleTracks()` calls `s.projects.List()`, discovers tracks from each project's `ProjectDir`
- Annotate each track with the project slug
- Return empty array when no projects are registered
- Support optional `?project=<slug>` query param to filter by project

### Task 1.5: Update serve.go to pass project store
- **File:** `backend/internal/adapter/cli/serve.go`
- Pass the already-loaded `reg` (ProjectStore) to `WithDashboard()` instead of `os.Getwd()`
- Remove `projectDir, _ := os.Getwd()` line

### Task 1.6: Update API response schema
- **File:** `backend/api/openapi.yaml`
- Add `project` field to track schema
- Add project list schema
- Regenerate with `make gen-api`

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
