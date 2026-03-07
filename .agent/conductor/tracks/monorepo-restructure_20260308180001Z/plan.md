# Implementation Plan: Restructure as Monorepo with Backend/Frontend Split

**Track ID:** monorepo-restructure_20260308180001Z

## Phase 1: Move Go Code to backend/ (5 tasks)

### Task 1.1: Create backend directory and move Go files
- [ ] Create `backend/` directory
- [ ] Move `cmd/`, `internal/`, `docs/` into `backend/`
- [ ] Move `go.mod`, `go.sum` into `backend/`
- [ ] Move `Makefile` to root (will be rewritten)
- [ ] Verify no import path changes needed (module stays `crelay`)

### Task 1.2: Update root go.work
- [ ] Create `go.work` at project root: `use ./backend`
- [ ] Verify `gopls` and IDE tooling work with the workspace

### Task 1.3: Remove old dashboard static files
- [ ] Delete `backend/internal/adapter/dashboard/static/` (index.html, app.js, style.css)
- [ ] Create `backend/internal/adapter/dashboard/dist/` with `.gitkeep`
- [ ] Add `backend/internal/adapter/dashboard/dist/` to `.gitignore` (except .gitkeep)

### Task 1.4: Update embed.go for dist/ directory
- [ ] Change `//go:embed static/*` to `//go:embed all:dist`
- [ ] Update `fs.Sub()` call: `fs.Sub(embeddedFiles, "dist")`
- [ ] Add fallback: if dist is empty (dev mode), serve a placeholder page saying "Run `make build-frontend` first"
- [ ] Handle SPA routing: serve `index.html` for non-file paths (React Router support)

### Task 1.5: Verify Go build and tests
- [ ] `cd backend && go build -buildvcs=false ./...`
- [ ] `cd backend && go test -buildvcs=false -race ./...`
- [ ] Fix any broken paths in tests (e.g., testdata references)
- [ ] Ensure `.agent/conductor/` path references in track_service still work

## Phase 2: Scaffold Frontend (4 tasks)

### Task 2.1: Initialize Vite + React + TypeScript project
- [ ] `cd frontend && npm create vite@latest . -- --template react-ts`
- [ ] Clean up default boilerplate (remove App.tsx demo content, default CSS)
- [ ] Verify `npm run dev` starts dev server
- [ ] Verify `npm run build` produces `dist/` output

### Task 2.2: Configure Vite for crelay
- [ ] Set `build.outDir` to `../backend/internal/adapter/dashboard/dist`
- [ ] Set `build.emptyOutDir: true`
- [ ] Configure dev proxy for all backend routes (`/api/*`, `/events`, `/webhook`, `/health`, `/gitea/*`, `/locks/*`)
- [ ] Set dev server port (e.g., 5173) to avoid conflict with backend 3001

### Task 2.3: Add minimal placeholder app
- [ ] Create a simple `App.tsx` that shows "crelay dashboard" heading
- [ ] Fetch `/api/status` on mount to verify proxy works
- [ ] Display connection status (connected/disconnected)
- [ ] This placeholder will be replaced by the react-dashboard track

### Task 2.4: Add frontend .gitignore and package config
- [ ] `frontend/.gitignore` — node_modules, dist (local dist only; output goes to backend)
- [ ] Verify `frontend/package.json` has correct scripts: `dev`, `build`, `preview`
- [ ] Pin Node version in `.nvmrc` or `package.json engines` (Node 18+)

## Phase 3: Build Pipeline (3 tasks)

### Task 3.1: Create root Makefile
- [ ] `make build` — builds frontend then backend, outputs single binary
- [ ] `make build-frontend` — `cd frontend && npm ci && npm run build`
- [ ] `make build-backend` — `cd backend && go build -buildvcs=false -o ../bin/crelay ./cmd/crelay`
- [ ] `make dev` — starts both frontend dev server and backend in parallel
- [ ] `make test` — runs Go tests
- [ ] `make clean` — removes build artifacts

### Task 3.2: Dev mode launcher
- [ ] `make dev` runs: `cd backend && go run ./cmd/crelay up` in background + `cd frontend && npm run dev`
- [ ] Or: use a simple shell script `scripts/dev.sh` that manages both processes
- [ ] Frontend dev server on 5173, backend on 3001
- [ ] Ctrl+C kills both processes

### Task 3.3: Verify single-binary deployment
- [ ] `make build` produces `bin/crelay`
- [ ] Run `bin/crelay up` — verify dashboard serves React app at `http://localhost:3001/`
- [ ] Verify all API endpoints work through the embedded app
- [ ] Verify Gitea proxy still works at `/gitea/`

## Phase 4: Final Verification (2 tasks)

### Task 4.1: Full test suite
- [ ] `cd backend && go build -buildvcs=false ./...`
- [ ] `cd backend && go test -buildvcs=false -race ./...`
- [ ] `cd frontend && npm run build` (no TypeScript errors)
- [ ] No regressions

### Task 4.2: Update documentation
- [ ] Update `README.md` with new project structure
- [ ] Document `make dev` workflow
- [ ] Document `make build` for production
- [ ] Update any `.agent/conductor/tech-stack.md` references

---

**Total: 14 tasks across 4 phases**
