# Implementation Plan: Project-Scoped Dashboard Tracks

## Phase 1: Backend — Replace projectDir with ProjectStore (5 tasks)

### Task 1.1: Define ProjectLister interface
- **File:** `backend/internal/adapter/dashboard/server.go`
- Add interface: `ProjectLister` with `List() []domain.Project` method
- Replace `projectDir string` field with `projects ProjectLister`

### Task 1.2: Update dashboard constructor and WithDashboard option
- **File:** `backend/internal/adapter/dashboard/server.go`
- Change `New()` to accept `ProjectLister` instead of `projectDir string`
- **File:** `backend/internal/adapter/rest/server.go`
- Update `WithDashboard()` option to accept `ProjectLister`

### Task 1.3: Update handleTracks to iterate projects
- **File:** `backend/internal/adapter/dashboard/handlers.go`
- `handleTracks()` calls `s.projects.List()`, discovers tracks from each project's `ProjectDir`
- Annotate each track with the project slug
- Return empty array when no projects are registered

### Task 1.4: Update serve.go to pass project store
- **File:** `backend/internal/adapter/cli/serve.go`
- Pass the already-loaded `reg` (ProjectStore) to `WithDashboard()` instead of `os.Getwd()`
- Remove `projectDir, _ := os.Getwd()` line

### Task 1.5: Update API response schema
- **File:** `backend/api/openapi.yaml`
- Add `project` field to track schema
- Regenerate with `make gen-api`

## Phase 2: Backend — Tests (3 tasks)

### Task 2.1: Update dashboard handler tests
- **File:** `backend/internal/adapter/dashboard/server_test.go`
- Create mock `ProjectLister` that returns test projects
- Test: no projects → empty tracks array
- Test: one project with tracks → tracks returned with project slug
- Test: multiple projects → tracks from all projects, each tagged

### Task 2.2: Update route registration tests
- **File:** `backend/internal/adapter/rest/routes_test.go`
- Update `buildMux()` and test helpers to pass ProjectLister instead of projectDir
- Verify all route tests still pass

### Task 2.3: Run full test suite
- `make test` — all pass
- `make test-smoke` — smoke tests pass

## Phase 3: Frontend — Display by Project (3 tasks)

### Task 3.1: Update track type and hook
- **File:** `frontend/src/hooks/useTracks.ts` (or equivalent)
- Add `project` field to track type
- No change to fetch URL

### Task 3.2: Add Projects section with per-project tracks
- **File:** `frontend/src/components/` (track list component)
- Dashboard shows a "Projects" section listing registered projects
- Each project expands to show its tracks with status indicators
- Show "No projects registered — run `crelay add <remote>`" when no projects exist

### Task 3.3: Verify frontend builds
- `cd frontend && npm run build` — no errors
- `cd frontend && npm run lint` — no warnings
