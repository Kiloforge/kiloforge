# Implementation Plan: Passwordless Gitea Login via Reverse Proxy Authentication

**Track ID:** gitea-proxy-authn_20260309123000Z

## Phase 1: Enable Gitea Reverse Proxy Auth

- [ ] Task 1.1: Add `GITEA__service__ENABLE_REVERSE_PROXY_AUTHENTICATION=true` to docker-compose environment in `compose/template.go`
- [ ] Task 1.2: Update `compose/template_test.go` to verify the new env var is present

## Phase 2: Proxy Header Injection

- [ ] Task 2.1: Update `NewGiteaProxy()` in `proxy/gitea.go` to accept an `authUser` parameter and inject `X-WEBAUTH-USER` header on all requests
- [ ] Task 2.2: Update `server.go` — pass `s.cfg.GiteaAdminUser` to `NewGiteaProxy()`
- [ ] Task 2.3: Add test for proxy header injection — verify `X-WEBAUTH-USER` is set on forwarded requests

## Phase 3: Verification

- [ ] Task 3.1: Verify `kf init && kf up` works — Gitea starts with reverse proxy auth enabled
- [ ] Task 3.2: Verify accessing `localhost:4001/` shows Gitea as logged in
- [ ] Task 3.3: Verify `go test ./...` passes
- [ ] Task 3.4: Verify `make build` succeeds
