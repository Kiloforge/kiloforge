# Implementation Plan: Web UI CLI Integration and Gitea Links

**Track ID:** impl-webui-integration_20260308140001Z

## Phase 1: Config and CLI Command (3 tasks)

### Task 1.1: Add dashboard config fields
- [x] Add `DashboardPort` (default: 3002) and `DashboardEnabled` (default: true) to config struct
- [x] Add defaults adapter, JSON adapter, env adapter entries
- [x] Support `KF_DASHBOARD_PORT` and `KF_DASHBOARD_ENABLED` env vars
- [x] Add `--dashboard-port` and `--no-dashboard` flags

### Task 1.2: Create `kf dashboard` command
- [x] Create `internal/cli/dashboard.go`
- [x] Standalone command that starts only the dashboard server
- [x] Loads state and tracker from files, runs on configured port
- [x] Graceful shutdown on SIGINT

### Task 1.3: Write config and command tests
- [x] Test config resolution with new fields (defaults, env, flags)
- [x] Test dashboard command initialization

## Phase 2: Startup Integration (3 tasks)

### Task 2.1: Start dashboard alongside relay in `kf up`
- [x] When `DashboardEnabled` is true, start dashboard server in goroutine
- [x] Pass shared store and tracker instances to dashboard
- [x] Both servers shut down on context cancellation
- [x] Print dashboard URL in startup output

### Task 2.2: Update `kf status` with dashboard info
- [x] Show dashboard URL when enabled
- [x] Show dashboard running/stopped status

### Task 2.3: Wire Gitea links in dashboard
- [x] Pass `GiteaURL` to dashboard server
- [x] Dashboard API includes Gitea PR URLs for each agent's PR
- [x] Dashboard API includes Gitea repo URLs for each project
- [x] Frontend renders clickable links to Gitea

## Phase 3: Verification (2 tasks)

### Task 3.1: Integration test
- [x] Start `kf up` with dashboard enabled
- [x] Verify both relay and dashboard respond on their ports
- [x] Verify dashboard shows Gitea links
- [x] Verify `--no-dashboard` flag disables dashboard

### Task 3.2: Full build and test
- [x] `go build ./...`
- [x] `go test -race ./...`
- [x] Verify no regressions

---

**Total: 8 tasks across 3 phases**
