# Implementation Plan: Web UI CLI Integration and Gitea Links

**Track ID:** impl-webui-integration_20260308140001Z

## Phase 1: Config and CLI Command (3 tasks)

### Task 1.1: Add dashboard config fields
- [ ] Add `DashboardPort` (default: 3002) and `DashboardEnabled` (default: true) to config struct
- [ ] Add defaults adapter, JSON adapter, env adapter entries
- [ ] Support `CRELAY_DASHBOARD_PORT` and `CRELAY_DASHBOARD_ENABLED` env vars
- [ ] Add `--dashboard-port` and `--no-dashboard` flags

### Task 1.2: Create `crelay dashboard` command
- [ ] Create `internal/cli/dashboard.go`
- [ ] Standalone command that starts only the dashboard server
- [ ] Loads state and tracker from files, runs on configured port
- [ ] Graceful shutdown on SIGINT

### Task 1.3: Write config and command tests
- [ ] Test config resolution with new fields (defaults, env, flags)
- [ ] Test dashboard command initialization

## Phase 2: Startup Integration (3 tasks)

### Task 2.1: Start dashboard alongside relay in `crelay up`
- [ ] When `DashboardEnabled` is true, start dashboard server in goroutine
- [ ] Pass shared store and tracker instances to dashboard
- [ ] Both servers shut down on context cancellation
- [ ] Print dashboard URL in startup output

### Task 2.2: Update `crelay status` with dashboard info
- [ ] Show dashboard URL when enabled
- [ ] Show dashboard running/stopped status

### Task 2.3: Wire Gitea links in dashboard
- [ ] Pass `GiteaURL` to dashboard server
- [ ] Dashboard API includes Gitea PR URLs for each agent's PR
- [ ] Dashboard API includes Gitea repo URLs for each project
- [ ] Frontend renders clickable links to Gitea

## Phase 3: Verification (2 tasks)

### Task 3.1: Integration test
- [ ] Start `crelay up` with dashboard enabled
- [ ] Verify both relay and dashboard respond on their ports
- [ ] Verify dashboard shows Gitea links
- [ ] Verify `--no-dashboard` flag disables dashboard

### Task 3.2: Full build and test
- [ ] `go build ./...`
- [ ] `go test -race ./...`
- [ ] Verify no regressions

---

**Total: 8 tasks across 3 phases**
