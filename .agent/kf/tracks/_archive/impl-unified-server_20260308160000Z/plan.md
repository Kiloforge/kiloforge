# Implementation Plan: Unified Server with Reverse Proxy to Gitea

**Track ID:** impl-unified-server_20260308160000Z

## Phase 1: Dashboard as Route Registrar (4 tasks)

### Task 1.1: Refactor dashboard.Server to export route registration
- [x] Add `RegisterRoutes(mux *http.ServeMux)` method to dashboard.Server
- [x] Move route registration from internal `routes()` into public method
- [x] Dashboard.Run() still works standalone (creates its own mux internally)

### Task 1.2: Mount dashboard routes in relay server
- [x] Import dashboard package in rest/server.go
- [x] Create dashboard.Server in relay's NewServer or Run
- [x] Call dashboard.RegisterRoutes on relay's mux
- [x] Start dashboard's watchState goroutine from relay's Run

### Task 1.3: Update `kf up` to use single server
- [x] Remove separate dashboard goroutine from up.go
- [x] Pass dashboard dependencies (AgentStore, QuotaTracker) to relay server constructor
- [x] Print single URL instead of separate relay/dashboard URLs
- [x] Keep `--no-dashboard` flag (skips mounting dashboard routes)

### Task 1.4: Update `kf dashboard` standalone command
- [x] Dashboard standalone still creates its own HTTP server
- [x] Uses dashboard.Server.Run() directly (unchanged behavior)

## Phase 2: Gitea Reverse Proxy (3 tasks)

### Task 2.1: Add reverse proxy handler
- [x] Create `internal/adapter/proxy/gitea.go`
- [x] `NewGiteaProxy(targetURL string) http.Handler` using httputil.ReverseProxy
- [x] Strip `/gitea` prefix, forward to Gitea
- [x] Handle WebSocket upgrade for Gitea live features

### Task 2.2: Configure Gitea sub-path in docker-compose
- [x] Set `ROOT_URL=http://localhost:3001/gitea/` in Gitea environment
- [x] Set `[server] ROOT_URL` via `GITEA__server__ROOT_URL` env var
- [x] Update docker-compose.yml template in compose adapter

### Task 2.3: Register proxy routes in relay server
- [x] Mount `/gitea/` prefix route on relay mux
- [x] Only mount if Gitea is running (health check first)
- [x] Update dashboard static UI links to use `/gitea/` prefix

## Phase 3: Cleanup and Verification (3 tasks)

### Task 3.1: Remove DashboardPort from config
- [x] Remove `DashboardPort` field and `IsDashboardEnabled` (keep enabled logic via --no-dashboard)
- [x] Update defaults, env adapter, JSON adapter
- [x] Update config tests

### Task 3.2: Update documentation and output
- [x] Update README architecture diagram (one server, one port)
- [x] Update `kf status` output to show single URL
- [x] Update `kf up` output messages

### Task 3.3: Full build and test
- [x] `go build -buildvcs=false ./...`
- [x] `go test -buildvcs=false -race ./...`
- [x] Manual verification: access dashboard, gitea proxy, webhook, lock API all on :3001

---

**Total: 10 tasks across 3 phases**
