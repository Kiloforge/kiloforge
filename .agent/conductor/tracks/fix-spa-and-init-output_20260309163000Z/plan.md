# Implementation Plan: Fix SPA Asset MIME Type, Init Output URLs, and Password Display

**Track ID:** fix-spa-and-init-output_20260309163000Z

## Phase 1: Fix SPA Asset Paths

- [ ] Task 1.1: Check and fix `frontend/vite.config.ts` — ensure `base` is set to `"/"` (not `"/-/"`)
- [ ] Task 1.2: Check `frontend/index.html` — ensure no `<base href="/-/">` tag or stale `/-/` references
- [ ] Task 1.3: Rebuild frontend (`npm run build`) and verify `dist/index.html` references `/assets/...` not `/-/assets/...`
- [ ] Task 1.4: Re-embed updated dist into Go binary (copy to `backend/internal/adapter/dashboard/dist/`)
- [ ] Task 1.5: Verify SPA handler serves JS/CSS with correct MIME types

## Phase 2: Fix Init Output

- [ ] Task 2.1: Update `backend/internal/adapter/cli/init.go` — change dashboard URL from `/-/` to `/`
- [ ] Task 2.2: Update init output — show Gitea URL as `http://localhost:<orchPort>/gitea/` instead of raw port
- [ ] Task 2.3: Remove admin password from init output — replace with note about auto-authentication
- [ ] Task 2.4: Update `backend/internal/adapter/cli/up.go` — fix dashboard URL in `kf up` output if it also shows `/-/`

## Phase 3: Verification

- [ ] Task 3.1: Verify `make build` succeeds (frontend + backend)
- [ ] Task 3.2: Verify dashboard loads at `/` without MIME errors
- [ ] Task 3.3: Verify `kf init` output shows correct URLs and no password
- [ ] Task 3.4: Verify `go test ./...` passes
