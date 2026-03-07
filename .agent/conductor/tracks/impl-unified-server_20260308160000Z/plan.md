# Implementation Plan: Unified Server with Reverse Proxy to Gitea

**Track ID:** impl-unified-server_20260308160000Z

## Phase 1: Dashboard as Route Registrar (4 tasks)

### Task 1.1: Refactor dashboard.Server to export route registration
- [ ] Add `RegisterRoutes(mux *http.ServeMux)` method to dashboard.Server
- [ ] Move route registration from internal `routes()` into public method
- [ ] Dashboard.Run() still works standalone (creates its own mux internally)

### Task 1.2: Mount dashboard routes in relay server
- [ ] Import dashboard package in rest/server.go
- [ ] Create dashboard.Server in relay's NewServer or Run
- [ ] Call dashboard.RegisterRoutes on relay's mux
- [ ] Start dashboard's watchState goroutine from relay's Run

### Task 1.3: Update `crelay up` to use single server
- [ ] Remove separate dashboard goroutine from up.go
- [ ] Pass dashboard dependencies (AgentStore, QuotaTracker) to relay server constructor
- [ ] Print single URL instead of separate relay/dashboard URLs
- [ ] Keep `--no-dashboard` flag (skips mounting dashboard routes)

### Task 1.4: Update `crelay dashboard` standalone command
- [ ] Dashboard standalone still creates its own HTTP server
- [ ] Uses dashboard.Server.Run() directly (unchanged behavior)

## Phase 2: Gitea Reverse Proxy (3 tasks)

### Task 2.1: Add reverse proxy handler
- [ ] Create `internal/adapter/proxy/gitea.go`
- [ ] `NewGiteaProxy(targetURL string) http.Handler` using httputil.ReverseProxy
- [ ] Strip `/gitea` prefix, forward to Gitea
- [ ] Handle WebSocket upgrade for Gitea live features

### Task 2.2: Configure Gitea sub-path in docker-compose
- [ ] Set `ROOT_URL=http://localhost:3001/gitea/` in Gitea environment
- [ ] Set `[server] ROOT_URL` via `GITEA__server__ROOT_URL` env var
- [ ] Update docker-compose.yml template in compose adapter

### Task 2.3: Register proxy routes in relay server
- [ ] Mount `/gitea/` prefix route on relay mux
- [ ] Only mount if Gitea is running (health check first)
- [ ] Update dashboard static UI links to use `/gitea/` prefix

## Phase 3: Cleanup and Verification (3 tasks)

### Task 3.1: Remove DashboardPort from config
- [ ] Remove `DashboardPort` field and `IsDashboardEnabled` (keep enabled logic via --no-dashboard)
- [ ] Update defaults, env adapter, JSON adapter
- [ ] Update config tests

### Task 3.2: Update documentation and output
- [ ] Update README architecture diagram (one server, one port)
- [ ] Update `crelay status` output to show single URL
- [ ] Update `crelay up` output messages

### Task 3.3: Full build and test
- [ ] `go build -buildvcs=false ./...`
- [ ] `go test -buildvcs=false -race ./...`
- [ ] Manual verification: access dashboard, gitea proxy, webhook, lock API all on :3001

---

**Total: 10 tasks across 3 phases**
