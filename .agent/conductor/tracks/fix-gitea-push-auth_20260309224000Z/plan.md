# Implementation Plan: Fix Gitea Push Authentication — Embed Credentials in Remote URL

**Track ID:** fix-gitea-push-auth_20260309224000Z

## Phase 1: Fix

- [ ] Task 1.1: In `add.go` — embed API token in Gitea remote URL as password, display sanitized URL to user
- [ ] Task 1.2: In `project_service.go` — same fix for the REST API code path
- [ ] Task 1.3: Verify API token works as git HTTP password with Gitea (if not, fall back to admin password)

## Phase 2: Verification

- [ ] Task 2.1: `go test ./...` passes
- [ ] Task 2.2: Manual test — `kf add` pushes to Gitea without credential prompt
- [ ] Task 2.3: Verify credentials are not printed to stdout/stderr
