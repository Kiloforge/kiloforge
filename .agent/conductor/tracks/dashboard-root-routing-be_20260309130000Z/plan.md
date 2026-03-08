# Implementation Plan: Dashboard Root Routing and Kiloforge Rebrand Defaults (Backend)

**Track ID:** dashboard-root-routing-be_20260309130000Z

## Phase 1: Rebrand Defaults

- [x] Task 1.1: Update `backend/internal/adapter/config/defaults.go` — change `GiteaAdminUser` from `"conductor"` to `"kiloforger"`, `GiteaAdminEmail` from `"conductor@local.dev"` to `"kiloforger@local.dev"`
- [x] Task 1.2: Update `backend/internal/adapter/compose/template.go` — rename container from `conductor-gitea` to `kf-gitea`
- [x] Task 1.3: Update all test files that reference `"conductor"` username/email to use `"kiloforger"` — `defaults_test.go`, `json_adapter_test.go`, `client_test.go`, `issues_test.go`, `manager_test.go`, `routes_test.go`, `server_test.go`, `integration_test.go`, `resolve_test.go`, `merger_test.go`, `template_test.go`, `env_adapter_test.go`
- [x] Task 1.4: Verify `go test ./...` passes with new defaults

## Phase 2: OpenAPI Path Prefix Change

- [x] Task 2.1: Update `backend/api/openapi.yaml` — change all `/-/api/` prefixes to `/api/`
- [x] Task 2.2: Regenerate server code with `oapi-codegen` (or update generated files)
- [x] Task 2.3: Update any test files that reference `/-/api/` paths to `/api/`

## Phase 3: Route Restructuring

- [x] Task 3.1: Update `backend/internal/adapter/dashboard/server.go` — change `RegisterNonAPIRoutes` routes from `/-/` prefix to `/` prefix (events, tracks, pr, SPA catch-all)
- [x] Task 3.2: Update `backend/internal/adapter/proxy/gitea.go` — accept `authUser` param (from proxy-authn track), add `http.StripPrefix("/gitea", ...)` wrapping, inject `X-WEBAUTH-USER` header
- [x] Task 3.3: Update `backend/internal/adapter/rest/server.go` — mount Gitea proxy at `/gitea/` instead of `/`, pass admin user to proxy, mount dashboard SPA as catch-all at `/`
- [x] Task 3.4: Update `backend/internal/adapter/compose/template.go` — change `ROOT_URL` from `http://localhost:{{ .OrchestratorPort }}/` to `http://localhost:{{ .OrchestratorPort }}/gitea/`
- [x] Task 3.5: Update route tests and template tests for new paths

## Phase 4: Favicon and Polish

- [x] Task 4.1: Copy `icon.png` from kiloforge_site repo into `frontend/public/favicon.png`
- [x] Task 4.2: Update `frontend/index.html` — add `<link rel="icon" type="image/png" href="/favicon.png">`
- [x] Task 4.3: Verify favicon is included in the embedded static filesystem

## Phase 5: Verification

- [x] Task 5.1: Verify `go test ./...` passes
- [x] Task 5.2: Verify `make build` succeeds
- [x] Task 5.3: Verify dashboard loads at `localhost:4001/`
- [x] Task 5.4: Verify Gitea loads at `localhost:4001/gitea/`
- [x] Task 5.5: Verify API responds at `localhost:4001/api/agents`
