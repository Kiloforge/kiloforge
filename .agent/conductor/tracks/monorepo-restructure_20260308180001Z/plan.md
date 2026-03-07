# Implementation Plan: Restructure as Monorepo with Backend/Frontend Split

**Track ID:** monorepo-restructure_20260308180001Z

## Phase 1: Move Go Code to backend/ (5 tasks)

### Task 1.1: Create backend directory and move Go files
- [x] Create `backend/` directory
- [x] Move `cmd/`, `internal/`, `docs/` into `backend/`
- [x] Move `go.mod`, `go.sum` into `backend/`
- [x] Move `Makefile` to root (will be rewritten)
- [x] Verify no import path changes needed (module stays `crelay`)

### Task 1.2: Update root go.work
- [x] Create `go.work` at project root: `use ./backend`
- [x] Verify `gopls` and IDE tooling work with the workspace

### Task 1.3: Remove old dashboard static files
- [x] Delete `backend/internal/adapter/dashboard/static/` (index.html, app.js, style.css)
- [x] Create `backend/internal/adapter/dashboard/dist/` with `.gitkeep`
- [x] Add `backend/internal/adapter/dashboard/dist/` to `.gitignore` (except .gitkeep)

### Task 1.4: Update embed.go for dist/ directory
- [x] Change `//go:embed static/*` to `//go:embed all:dist`
- [x] Update `fs.Sub()` call: `fs.Sub(embeddedFiles, "dist")`
- [x] Add fallback: if dist is empty (dev mode), serve a placeholder page saying "Run `make build-frontend` first"
- [x] Handle SPA routing: serve `index.html` for non-file paths (React Router support)

### Task 1.5: Verify Go build and tests
- [x] `cd backend && go build -buildvcs=false ./...`
- [x] `cd backend && go test -buildvcs=false -race ./...`
- [x] Fix any broken paths in tests (e.g., testdata references)
- [x] Ensure `.agent/conductor/` path references in track_service still work

## Phase 2: Scaffold Frontend (4 tasks)

### Task 2.1: Initialize Vite + React + TypeScript project
- [x] `cd frontend && npm create vite@latest . -- --template react-ts`
- [x] Clean up default boilerplate (remove App.tsx demo content, default CSS)
- [x] Verify `npm run dev` starts dev server
- [x] Verify `npm run build` produces `dist/` output

### Task 2.2: Configure Vite for crelay
- [x] Set `build.outDir` to `../backend/internal/adapter/dashboard/dist`
- [x] Set `build.emptyOutDir: true`
- [x] Configure dev proxy for all backend routes (`/api/*`, `/events`, `/webhook`, `/health`, `/gitea/*`, `/locks/*`)
- [x] Set dev server port (e.g., 5173) to avoid conflict with backend 3001

### Task 2.3: Add minimal placeholder app
- [x] Create a simple `App.tsx` that shows "crelay dashboard" heading
- [x] Fetch `/api/status` on mount to verify proxy works
- [x] Display connection status (connected/disconnected)
- [x] This placeholder will be replaced by the react-dashboard track

### Task 2.4: Add frontend .gitignore and package config
- [x] `frontend/.gitignore` — node_modules, dist (local dist only; output goes to backend)
- [x] Verify `frontend/package.json` has correct scripts: `dev`, `build`, `preview`
- [x] Pin Node version in `.nvmrc` or `package.json engines` (Node 18+)

## Phase 3: Build Pipeline (3 tasks)

### Task 3.1: Create root Makefile
- [x] `make build` — builds frontend then backend, outputs single binary
- [x] `make build-frontend` — `cd frontend && npm ci && npm run build`
- [x] `make build-backend` — `cd backend && go build -buildvcs=false -o ../bin/crelay ./cmd/crelay`
- [x] `make dev` — starts both frontend dev server and backend in parallel
- [x] `make test` — runs Go tests
- [x] `make clean` — removes build artifacts

### Task 3.2: Dev mode launcher
- [x] `make dev` runs: `cd backend && go run ./cmd/crelay up` in background + `cd frontend && npm run dev`
- [x] Or: use a simple shell script `scripts/dev.sh` that manages both processes
- [x] Frontend dev server on 5173, backend on 3001
- [x] Ctrl+C kills both processes

### Task 3.3: Verify single-binary deployment
- [x] `make build` produces `bin/crelay`
- [x] Run `bin/crelay up` — verify dashboard serves React app at `http://localhost:3001/`
- [x] Verify all API endpoints work through the embedded app
- [x] Verify Gitea proxy still works at `/gitea/`

## Phase 4: Final Verification (2 tasks)

### Task 4.1: Full test suite
- [x] `cd backend && go build -buildvcs=false ./...`
- [x] `cd backend && go test -buildvcs=false -race ./...`
- [x] `cd frontend && npm run build` (no TypeScript errors)
- [x] No regressions

### Task 4.2: Update documentation
- [x] Update `README.md` with new project structure
- [x] Document `make dev` workflow
- [x] Document `make build` for production
- [x] Update any `.agent/conductor/tech-stack.md` references

---

**Total: 14 tasks across 4 phases**
